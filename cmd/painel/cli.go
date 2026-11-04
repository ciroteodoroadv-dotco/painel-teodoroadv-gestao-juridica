package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/app"
	"painelteodoro/internal/audit"
	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/config"
	"painelteodoro/internal/platform/migrate"
)

const uso = `Uso:
  painel migrate up|down          aplica ou reverte a última migração
  painel audit verify             confere a cadeia de auditoria
  painel restore-check            confere uma cópia restaurada (auditoria + todos os documentos decifrados)
  painel user create EMAIL PAPEL  cria usuário (administrador|advogado|assistente); senha lida da entrada padrão
  painel user reset-password EMAIL  nova senha lida da entrada padrão; revoga as sessões
  painel user unlock EMAIL        libera conta bloqueada por tentativas
  painel user reset-2fa EMAIL     apaga o segundo fator (aparelho perdido); a pessoa cadastra de novo no próximo acesso
  painel seed-demo EMAIL          (só em modo dev) cria dados fictícios de demonstração, em nome desse usuário`

func runCLI(args []string) error {
	cfg, err := config.Load(config.OSEnv())
	if err != nil {
		return fmt.Errorf("configuração incompleta:\n%w", err)
	}
	ctx := context.Background()
	owner, pool, err := openDBs(ctx, cfg)
	if err != nil {
		return err
	}
	defer owner.Close()
	defer pool.Close()
	st := &auth.Store{Pool: pool}

	switch {
	case len(args) == 2 && args[0] == "migrate" && args[1] == "up":
		return migrate.Up(ctx, owner)
	case len(args) == 2 && args[0] == "migrate" && args[1] == "down":
		return migrate.Down(ctx, owner)
	case len(args) == 2 && args[0] == "audit" && args[1] == "verify":
		n, err := audit.Verify(ctx, pool)
		if err != nil {
			return fmt.Errorf("CADEIA COMPROMETIDA após %d eventos: %w", n, err)
		}
		fmt.Printf("Cadeia íntegra: %d eventos conferidos.\n", n)
		return nil
	case len(args) == 4 && args[0] == "user" && args[1] == "create":
		return userCreate(ctx, st, args[2], auth.Role(args[3]))
	case len(args) == 3 && args[0] == "user" && args[1] == "reset-password":
		return userReset(ctx, st, args[2])
	case len(args) == 1 && args[0] == "restore-check":
		return restoreCheck(ctx, cfg, pool)
	case len(args) == 2 && args[0] == "seed-demo":
		return seedDemo(ctx, cfg, st, pool, args[1])
	case len(args) == 3 && args[0] == "user" && args[1] == "reset-2fa":
		return userReset2FA(ctx, st, args[2])
	case len(args) == 3 && args[0] == "user" && args[1] == "unlock":
		return userUnlock(ctx, st, args[2])
	}
	return fmt.Errorf("comando desconhecido ou incompleto.\n%s", uso)
}

func readPassword() (string, error) {
	fmt.Fprintln(os.Stderr, "Digite a senha e tecle Enter (ela não será gravada em lugar nenhum):")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", errors.New("nenhuma senha recebida na entrada padrão")
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func cliAudit(ctx context.Context, tx pgx.Tx, action, userID string, meta map[string]string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: "sistema:cli", Action: action, TargetType: "usuario",
		TargetID: userID, Result: audit.OK, Meta: meta})
}

func userCreate(ctx context.Context, st *auth.Store, email string, role auth.Role) error {
	if !auth.ValidRole(role) {
		return fmt.Errorf("papel inválido %q: use administrador, advogado ou assistente", role)
	}
	pw, err := readPassword()
	if err != nil {
		return err
	}
	if err := auth.CheckPolicy(pw, email); err != nil {
		return err
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		return err
	}
	tx, err := st.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id, err := st.CreateUser(ctx, tx, email, role, hash, true)
	if err != nil {
		return fmt.Errorf("não consegui criar (e-mail já existe?): %w", err)
	}
	if err := cliAudit(ctx, tx, "usuario.criado", id, map[string]string{"papel": string(role)}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Println("Usuário criado com senha PROVISÓRIA: no primeiro acesso ele troca a senha e cadastra o segundo fator.")
	return nil
}

func userReset(ctx context.Context, st *auth.Store, email string) error {
	u, err := st.UserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("usuário não encontrado")
	}
	pw, err := readPassword()
	if err != nil {
		return err
	}
	if err := auth.CheckPolicy(pw, u.Email); err != nil {
		return err
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		return err
	}
	tx, err := st.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := st.SetPassword(ctx, tx, u.ID, hash, true); err != nil {
		return err
	}
	if err := cliAudit(ctx, tx, "usuario.senha_redefinida", u.ID, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func userUnlock(ctx context.Context, st *auth.Store, email string) error {
	u, err := st.UserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("usuário não encontrado")
	}
	tx, err := st.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := st.Unlock(ctx, tx, u.ID); err != nil {
		return err
	}
	if err := cliAudit(ctx, tx, "usuario.desbloqueado", u.ID, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func seedDemo(ctx context.Context, cfg config.Config, st *auth.Store, pool *pgxpool.Pool, email string) error {
	if !cfg.Dev {
		return errors.New("seed-demo só roda com PAINEL_ENV=dev: nunca coloque dados de demonstração em produção")
	}
	u, err := st.UserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("usuário não encontrado")
	}
	if err := app.SeedDemo(ctx, pool, cfg.FieldKey, st, u.ID); err != nil {
		return err
	}
	if err := app.SeedDemoDocumentos(ctx, pool, cfg.FieldKey, cfg.DocsDir, u.ID); err != nil {
		return err
	}
	fmt.Println("Dados fictícios criados: 4 pessoas, 3 processos, 3 atendimentos, tarefas, agenda, intimações, calendários e 1 documento cifrado (todos marcados como exemplo).")
	return nil
}

func userReset2FA(ctx context.Context, st *auth.Store, email string) error {
	u, err := st.UserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("usuário não encontrado")
	}
	tx, err := st.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := st.ResetMFA(ctx, tx, u.ID); err != nil {
		return err
	}
	if err := cliAudit(ctx, tx, "usuario.mfa_redefinido", u.ID, nil); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Println("Segundo fator apagado. No próximo acesso a pessoa cadastra de novo (aplicativo ou chave).")
	return nil
}
