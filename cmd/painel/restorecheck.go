package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/documentos"
	"painelteodoro/internal/platform/config"
)

// restoreCheck prova que uma cópia restaurada está inteira: cadeia de auditoria íntegra, contagens por
// tabela e TODOS os documentos decifrados e conferidos. Só lê; usado pelo teste mensal de restauração.
func restoreCheck(ctx context.Context, cfg config.Config, pool *pgxpool.Pool) error {
	var falhas []string

	n, err := audit.Verify(ctx, pool)
	if err != nil {
		falhas = append(falhas, fmt.Sprintf("auditoria comprometida após %d eventos: %v", n, err))
	}
	fmt.Printf("auditoria: %d eventos, cadeia %s\n", n, map[bool]string{true: "ÍNTEGRA", false: "QUEBRADA"}[err == nil])

	for _, t := range []string{"users", "pessoas", "processos", "processo_partes", "tarefas", "agenda_eventos", "documentos", "prazos"} {
		var c int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+t).Scan(&c); err != nil {
			falhas = append(falhas, fmt.Sprintf("tabela %s ilegível: %v", t, err))
			continue
		}
		fmt.Printf("tabela %-16s %d linha(s)\n", t, c)
	}

	store, err := documentos.NewStore(cfg.DocsDir, cfg.FieldKey)
	if err != nil {
		return fmt.Errorf("pasta de documentos: %w", err)
	}
	v, err := (&documentos.Service{Pool: pool, Store: store}).Verificar(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("documentos: %d registrado(s), %d conferido(s) com sucesso\n", v.Total, v.OK)
	falhas = append(falhas, v.Problemas...)

	if len(falhas) > 0 {
		for _, f := range falhas {
			fmt.Println("FALHA:", f)
		}
		return errors.New("a cópia restaurada NÃO passou na conferência")
	}
	fmt.Println("RESULTADO: cópia restaurada íntegra e utilizável.")
	return nil
}
