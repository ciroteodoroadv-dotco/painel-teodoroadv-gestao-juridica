package intimacoes

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/coletas"
	"painelteodoro/internal/platform/webx"
)

// Service concentra as regras de OABs, coleta e decisão. Só toca nas próprias tabelas.
type Service struct {
	Pool      *pgxpool.Pool
	Coletas   *coletas.Store
	Coletor   Coletor
	Tarefas   TarefaAutomatica
	TarefasDe TarefasDeIntimacao
	Processos ProcessoPorNumero
	Cfg       Config
	Now       func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

var (
	reOAB = regexp.MustCompile(`^[0-9]{1,10}$`)
	reUF  = regexp.MustCompile(`^[A-Z]{2}$`)
)

func (s *Service) recusa(ctx context.Context, ac webx.Actor, acao, motivo string) {
	_ = audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ac.UserID, Action: acao, TargetType: "oab", Result: audit.Negado,
		RequestID: ac.RequestID, Meta: map[string]string{"motivo": motivo}})
}

type oabNorm struct {
	nome, numero, uf, base string
	em                     time.Time
}

// normalizaOAB confere o formulário. Sem a base e a data da autorização o cadastro é recusado e a recusa é auditada.
func (s *Service) normalizaOAB(ctx context.Context, ac webx.Actor, in OABIn) (n oabNorm, err error) {
	n.nome, n.numero = strings.TrimSpace(in.Nome), strings.TrimSpace(in.Numero)
	n.uf, n.base = strings.ToUpper(strings.TrimSpace(in.UF)), strings.TrimSpace(in.AutorizacaoBase)
	falha := func(motivo, msg string) (oabNorm, error) {
		s.recusa(ctx, ac, "oab.cadastro_recusado", motivo)
		return n, webx.Invalidf(msg)
	}
	if utf8.RuneCountInString(n.nome) < 2 || utf8.RuneCountInString(n.nome) > 200 {
		return falha("nome", "Informe o nome do advogado.")
	}
	if !reOAB.MatchString(n.numero) {
		return falha("numero", "Informe o número da OAB (só dígitos).")
	}
	if !reUF.MatchString(n.uf) {
		return falha("uf", "Informe a UF da OAB com duas letras.")
	}
	if utf8.RuneCountInString(n.base) < 3 || utf8.RuneCountInString(n.base) > 500 {
		return falha("autorizacao_base", "Informe a base da autorização.")
	}
	em, derr := webx.DataISO(in.AutorizacaoEm)
	if derr != nil {
		return falha("autorizacao_em", "Informe a data da autorização.")
	}
	if em.After(webx.Hoje(s.now())) {
		return falha("autorizacao_futura", "A data da autorização não pode ser futura.")
	}
	n.em = em
	return n, nil
}

// CadastrarOAB cadastra uma OAB para a coleta e audita quem cadastrou. Base e data da autorização são obrigatórias.
func (s *Service) CadastrarOAB(ctx context.Context, ac webx.Actor, in OABIn) (string, error) {
	n, err := s.normalizaOAB(ctx, ac, in)
	if err != nil {
		return "", err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO oab_monitoradas (advogado_nome, numero_oab, uf, autorizacao_base, autorizacao_em, ativa, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7::uuid,$7::uuid) RETURNING id::text`, n.nome, n.numero, n.uf, n.base, n.em, in.Ativa, ac.UserID).Scan(&id)
	if webx.IsUniqueViolation(err) {
		return "", webx.Invalidf("Esta OAB já está cadastrada.")
	}
	if err == nil {
		err = audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: "oab.cadastrada", TargetType: "oab", TargetID: id, Result: audit.OK, RequestID: ac.RequestID})
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	return id, err
}

// AtualizarOAB altera os dados da OAB (concorrência otimista). Número e UF não mudam: seria outra OAB.
func (s *Service) AtualizarOAB(ctx context.Context, ac webx.Actor, id string, version int, in OABIn) error {
	if !webx.UUID(id) {
		return webx.ErrNotFound
	}
	atual, err := s.ObterOAB(ctx, id)
	if err != nil {
		return err
	}
	in.Numero, in.UF = atual.Numero, atual.UF
	n, err := s.normalizaOAB(ctx, ac, in)
	if err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE oab_monitoradas SET advogado_nome=$3, autorizacao_base=$4, autorizacao_em=$5, ativa=$6,
		updated_at=now(), updated_by=$7::uuid, version=version+1 WHERE id=$1::uuid AND version=$2`, id, version, n.nome, n.base, n.em, in.Ativa, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return webx.ErrConflict
	}
	if err = audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: "oab.alterada", TargetType: "oab", TargetID: id, Result: audit.OK, RequestID: ac.RequestID}); err == nil {
		err = tx.Commit(ctx)
	}
	return err
}

const colsOAB = `id::text, advogado_nome, numero_oab, uf, autorizacao_base, autorizacao_em, ativa, ultimo_sucesso, created_by::text, version`

func varreOAB(row pgx.Row) (OAB, error) {
	var o OAB
	err := row.Scan(&o.ID, &o.Nome, &o.Numero, &o.UF, &o.AutorizacaoBase, &o.AutorizacaoEm, &o.Ativa, &o.UltimoSucesso, &o.CriadoPor, &o.Version)
	return o, err
}

// ObterOAB devolve uma OAB cadastrada.
func (s *Service) ObterOAB(ctx context.Context, id string) (OAB, error) {
	if !webx.UUID(id) {
		return OAB{}, webx.ErrNotFound
	}
	o, err := varreOAB(s.Pool.QueryRow(ctx, `SELECT `+colsOAB+` FROM oab_monitoradas WHERE id=$1::uuid`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return OAB{}, webx.ErrNotFound
	}
	return o, err
}

// ListarOABs devolve todas as OABs (ativas ou não), por nome.
func (s *Service) ListarOABs(ctx context.Context) ([]OAB, error) {
	return s.oabs(ctx, false)
}

func (s *Service) oabs(ctx context.Context, soAtivas bool) ([]OAB, error) {
	q := `SELECT ` + colsOAB + ` FROM oab_monitoradas`
	if soAtivas {
		q += ` WHERE ativa`
	}
	rows, err := s.Pool.Query(ctx, q+` ORDER BY advogado_nome, numero_oab`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OAB
	for rows.Next() {
		o, err := varreOAB(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// Decidir confirma a intimação (só a partir de "triagem"). Cancelar tem motivo e vai por Cancelar. Quem pode decidir é
// decidido pela rota (só advogado); aqui ficam a regra do dado e a auditoria.
func (s *Service) Decidir(ctx context.Context, ac webx.Actor, id, status string) error {
	if status != "confirmada" {
		return webx.Invalidf("Decisão inválida.")
	}
	return s.mudaStatus(ctx, ac, id, status, "", `status=$2, decidida_por=$3::uuid, decidida_em=now()`)
}

// Cancelar cancela a intimação com o motivo, registrando quem e quando; ela permanece no histórico (nada se apaga).
func (s *Service) Cancelar(ctx context.Context, ac webx.Actor, id, motivo string) error {
	if motivo = strings.TrimSpace(motivo); motivo == "" {
		return webx.Invalidf("Informe o motivo do cancelamento.")
	}
	if utf8.RuneCountInString(motivo) > 1000 {
		return webx.Invalidf("Texto longo demais.")
	}
	return s.mudaStatus(ctx, ac, id, "cancelada", motivo, `status=$2, decidida_por=$3::uuid, decidida_em=now(), cancelado_por=$3::uuid, cancelado_em=now(), cancelamento_motivo=$4`)
}

func (s *Service) mudaStatus(ctx context.Context, ac webx.Actor, id, status, motivo, set string) error {
	if !webx.UUID(id) {
		return webx.ErrNotFound
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	args := []any{id, status, ac.UserID}
	if motivo != "" {
		args = append(args, motivo)
	}
	tag, err := tx.Exec(ctx, `UPDATE intimacoes SET `+set+` WHERE id=$1::uuid AND status='triagem'`, args...) // #nosec G202 -- fragmento constante
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var um int
		if s.Pool.QueryRow(ctx, `SELECT 1 FROM intimacoes WHERE id=$1::uuid`, id).Scan(&um) != nil {
			return webx.ErrNotFound
		}
		return webx.Invalidf("Esta intimação já foi decidida.")
	}
	if err = audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: "intimacao." + status, TargetType: "intimacao",
		TargetID: id, Result: audit.OK, RequestID: ac.RequestID}); err == nil {
		err = tx.Commit(ctx)
	}
	return err
}
