package prazos

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/webx"
)

// Service concentra as regras. Só toca na própria tabela.
type Service struct {
	Pool      *pgxpool.Pool
	Processos ProcessoLookup
	Users     UserLookup
}

func (s *Service) record(ctx context.Context, tx pgx.Tx, ac webx.Actor, action, id string, meta map[string]string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: action, TargetType: "prazo",
		TargetID: id, Result: audit.OK, RequestID: ac.RequestID, Meta: meta})
}

func (s *Service) checkRefs(ctx context.Context, n normalized) error {
	if n.processo != "" {
		ok, err := s.Processos.Existe(ctx, n.processo)
		if err != nil {
			return err
		}
		if !ok {
			return webx.Invalidf("Processo não encontrado.")
		}
	}
	if n.responsavel != "" {
		m, err := s.Users.Emails(ctx, []string{n.responsavel})
		if err != nil {
			return err
		}
		if _, ok := m[n.responsavel]; !ok {
			return webx.Invalidf("Responsável inválido.")
		}
	}
	return nil
}

// Create cadastra o prazo. Todo prazo nasce "a conferir", seja quem for que o digite (ADR 0012, item 3): só o ato
// explícito de confirmar (Confirmar, exclusivo do advogado) o torna válido e deixa "prazo.confirmado" na auditoria.
func (s *Service) Create(ctx context.Context, ac webx.Actor, in Input) (string, error) {
	n, err := in.normalize()
	if err != nil {
		return "", err
	}
	if err := s.checkRefs(ctx, n); err != nil {
		return "", err
	}
	situacao := AConferir
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO prazos (descricao, processo_id, data_fatal, fundamento, fonte, situacao, responsavel_id,
		created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8) RETURNING id`,
		n.descricao, webx.Nullable(n.processo), n.dataFatal, n.fundamento, n.fonte, situacao,
		webx.Nullable(n.responsavel), ac.UserID).Scan(&id)
	if err == nil {
		err = s.record(ctx, tx, ac, "prazo.criado", id, map[string]string{"etapa": situacao})
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	return id, err
}

// Update altera um prazo ABERTO. Ninguém confirma por efeito colateral: mudar a data derruba a confirmação (volta a
// "a conferir", sem quem/quando); manter a data mantém a situação. Confirmar é só pelo ato explícito.
func (s *Service) Update(ctx context.Context, ac webx.Actor, id string, version int, in Input) error {
	var daTarefa bool
	var antes time.Time
	if err := s.Pool.QueryRow(ctx, `SELECT tarefa_id IS NOT NULL, data_fatal FROM prazos WHERE id=$1`, id).Scan(&daTarefa, &antes); err == nil && daTarefa {
		return webx.Invalidf("Este prazo pertence a uma tarefa: altere o Prazo fatal na própria tarefa.")
	}
	n, err := in.normalize()
	if err != nil {
		return err
	}
	if err := s.checkRefs(ctx, n); err != nil {
		return err
	}
	etapa := "mesma_data"
	if !antes.Equal(n.dataFatal) {
		etapa = AConferir
	}
	return s.write(ctx, ac, id, "prazo.alterado", map[string]string{"etapa": etapa},
		`UPDATE prazos SET descricao=$3, processo_id=$4, fundamento=$6, fonte=$7, responsavel_id=$8,
		   situacao = CASE WHEN data_fatal = $5::date THEN situacao ELSE 'a_conferir' END,
		   confirmado_por = CASE WHEN data_fatal = $5::date THEN confirmado_por END,
		   confirmado_em = CASE WHEN data_fatal = $5::date THEN confirmado_em END,
		   data_fatal=$5, updated_at=now(), updated_by=$9, version=version+1
		 WHERE id=$1 AND version=$2 AND excluido_em IS NULL AND situacao IN ('a_conferir','confirmado')`,
		id, version, n.descricao, webx.Nullable(n.processo), n.dataFatal, n.fundamento, n.fonte, webx.Nullable(n.responsavel),
		ac.UserID)
}

// Confirmar: o advogado atesta o prazo (a conferir → confirmado).
func (s *Service) Confirmar(ctx context.Context, ac webx.Actor, id string, version int) error {
	return s.write(ctx, ac, id, "prazo.confirmado", nil,
		`UPDATE prazos SET situacao='confirmado', confirmado_por=$3, confirmado_em=now(), updated_at=now(), updated_by=$3, version=version+1
		 WHERE id=$1 AND version=$2 AND excluido_em IS NULL AND situacao='a_conferir'`, id, version, ac.UserID)
}

// Cumprir marca como cumprido (só prazo confirmado).
func (s *Service) Cumprir(ctx context.Context, ac webx.Actor, id string, version int) error {
	return s.write(ctx, ac, id, "prazo.cumprido", nil,
		`UPDATE prazos SET situacao='cumprido', cumprido_em=now(), updated_at=now(), updated_by=$3, version=version+1
		 WHERE id=$1 AND version=$2 AND excluido_em IS NULL AND situacao='confirmado'`, id, version, ac.UserID)
}

// Cancelar encerra um prazo aberto sem cumprimento (ex.: intimação tornada sem efeito).
func (s *Service) Cancelar(ctx context.Context, ac webx.Actor, id string, version int) error {
	return s.write(ctx, ac, id, "prazo.cancelado", nil,
		`UPDATE prazos SET situacao='cancelado', updated_at=now(), updated_by=$3, version=version+1
		 WHERE id=$1 AND version=$2 AND excluido_em IS NULL AND situacao IN ('a_conferir','confirmado')`, id, version, ac.UserID)
}

// Excluir oculta o registro (nunca apaga).
func (s *Service) Excluir(ctx context.Context, ac webx.Actor, id string, version int) error {
	return s.write(ctx, ac, id, "prazo.excluido", nil,
		`UPDATE prazos SET excluido_em=now(), updated_at=now(), updated_by=$3, version=version+1
		 WHERE id=$1 AND version=$2 AND excluido_em IS NULL`, id, version, ac.UserID)
}

// write roda o UPDATE e a auditoria na MESMA transação. 0 linhas = não existe, versão velha ou estado que não permite.
func (s *Service) write(ctx context.Context, ac webx.Actor, id, action string, meta map[string]string, sql string, args ...any) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var one int
		e := s.Pool.QueryRow(ctx, `SELECT 1 FROM prazos WHERE id=$1 AND excluido_em IS NULL`, id).Scan(&one)
		if errors.Is(e, pgx.ErrNoRows) {
			return webx.ErrNotFound
		}
		if e != nil {
			return e
		}
		return webx.ErrConflict
	}
	if err = s.record(ctx, tx, ac, action, id, meta); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
