package audiencias

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/codigos"
	"painelteodoro/internal/platform/webx"
)

// Service concentra as regras de audiências. Só toca na própria tabela; os outros domínios entram por interfaces.
type Service struct {
	Pool      *pgxpool.Pool
	Pessoas   PessoaLookup
	Processos ProcessoLookup
	Reauth    Reauth
}

func (s *Service) record(ctx context.Context, tx pgx.Tx, ac webx.Actor, action, id string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: action, TargetType: "audiencia", TargetID: id,
		Result: audit.OK, RequestID: ac.RequestID})
}

// vinculos confere o cliente e resolve o número do processo: existindo em Processos, grava o vínculo real além do texto.
func (s *Service) vinculos(ctx context.Context, n normalized) (processoID string, err error) {
	if n.cliente != "" {
		ok := map[string]bool{}
		if webx.UUID(n.cliente) {
			if ok, err = s.Pessoas.Exist(ctx, []string{n.cliente}); err != nil {
				return "", err
			}
		}
		if !ok[n.cliente] {
			return "", webx.Invalidf("Pessoa não encontrada.")
		}
	}
	if n.processo == "" {
		return "", nil
	}
	return s.Processos.PorNumero(ctx, n.processo)
}

// Create cadastra a audiência (código AUD. 0000001). Audiência é item crítico: nunca é bloqueada por conflito de
// horário; só devolve o aviso do B quando coincide com outra audiência.
func (s *Service) Create(ctx context.Context, ac webx.Actor, in Input) (codigo, aviso string, err error) {
	n, err := in.normalize(false)
	if err != nil {
		return "", "", err
	}
	proc, err := s.vinculos(ctx, n)
	if err != nil {
		return "", "", err
	}
	outras, err := s.NoHorario(ctx, n.data, n.hora)
	if err != nil {
		return "", "", err
	}
	if len(outras) > 0 {
		o := outras[0]
		aviso = fmt.Sprintf("Atenção: coincide com Audiência \"%s\" em %s às %s — imposição externa, verifique manualmente.", o.Titulo, o.Data, o.Hora)
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if codigo, err = codigos.Proximo(ctx, tx, "AUD"); err != nil {
		return "", "", err
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO audiencias (codigo, tipo_ato, titulo, processo_numero, processo_id, cliente_pessoa_id, data, hora,
		juizo, local, descricao, created_by, updated_by) VALUES ($1,$2,$3,$4,$5::uuid,$6::uuid,$7,$8::time,$9,$10,$11,$12,$12) RETURNING id::text`,
		codigo, n.tipo, n.titulo, n.processo, webx.Nullable(proc), webx.Nullable(n.cliente), n.data, n.hora, n.juizo, n.local,
		n.descricao, ac.UserID).Scan(&id)
	if err == nil {
		err = s.record(ctx, tx, ac, "audiencia.criada", id)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return "", "", err
	}
	return codigo, aviso, nil
}

// Update altera título, processo, cliente, data, hora, juízo e local (o tipo do ato não se edita; nem as observações,
// como no B). Audiência cancelada fica como está.
func (s *Service) Update(ctx context.Context, ac webx.Actor, id string, version int, in Input) error {
	atual, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if atual.Cancelada {
		return webx.Invalidf("Audiência cancelada não pode ser editada.")
	}
	n, err := in.normalize(true)
	if err != nil {
		return err
	}
	proc, err := s.vinculos(ctx, n)
	if err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE audiencias SET titulo=$3, processo_numero=$4, processo_id=$5::uuid, cliente_pessoa_id=$6::uuid, data=$7,
		hora=$8::time, juizo=$9, local=$10, updated_at=now(), updated_by=$11, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL AND NOT cancelada`, atual.ID, version, n.titulo, n.processo,
		webx.Nullable(proc), webx.Nullable(n.cliente), n.data, n.hora, n.juizo, n.local, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return webx.ErrConflict
	}
	if err = s.record(ctx, tx, ac, "audiencia.atualizada", atual.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Cancelar preserva a ficha com o motivo, quem cancelou e quando. Reautentica o usuário logado (a senha dele) e audita.
func (s *Service) Cancelar(ctx context.Context, ac webx.Actor, id string, version int, motivo, senha string) error {
	if motivo = strings.TrimSpace(motivo); motivo == "" {
		return webx.Invalidf("Informe o motivo do cancelamento.")
	}
	atual, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if atual.Cancelada {
		return webx.Invalidf("Esta audiência já está cancelada.")
	}
	if err := s.Reauth.Confirmar(ctx, ac, senha); err != nil {
		_ = audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ac.UserID, Action: "audiencia.cancelamento_negado", TargetType: "audiencia",
			TargetID: id, Result: audit.Falha, RequestID: ac.RequestID})
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE audiencias SET cancelada=true, cancelamento_motivo=$3, cancelada_em=now(), cancelada_por=$4,
		updated_at=now(), updated_by=$4, version=version+1 WHERE id=$1 AND version=$2 AND excluido_em IS NULL AND NOT cancelada`,
		id, version, motivo, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return webx.ErrConflict
	}
	if err = s.record(ctx, tx, ac, "audiencia.cancelada", id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
