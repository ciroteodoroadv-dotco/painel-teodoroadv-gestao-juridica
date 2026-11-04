package diligencias

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/codigos"
	"painelteodoro/internal/platform/webx"
)

// Service concentra as regras de diligências. Só toca na própria tabela; os outros domínios entram por interfaces.
type Service struct {
	Pool      *pgxpool.Pool
	Pessoas   PessoaLookup
	Processos ProcessoLookup
	Conflitos ConflitoLookup
}

func (s *Service) record(ctx context.Context, tx pgx.Tx, ac webx.Actor, action, id string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: action, TargetType: "diligencia", TargetID: id,
		Result: audit.OK, RequestID: ac.RequestID})
}

// vinculos confere o cliente e resolve o processo: pelo id escolhido (vem da Agenda; o texto passa a ser o rótulo do
// processo) ou pelo número digitado (existindo em Processos, grava o vínculo real além do texto).
func (s *Service) vinculos(ctx context.Context, n *normalized) (processoID string, err error) {
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
	if n.procID != "" {
		rotulo, ok := "", false
		if webx.UUID(n.procID) {
			if rotulo, ok, err = s.Processos.Rotulo(ctx, n.procID); err != nil {
				return "", err
			}
		}
		if !ok {
			return "", webx.Invalidf("Processo vinculado não encontrado.")
		}
		n.processo = rotulo
		return n.procID, nil
	}
	if n.processo == "" {
		return "", nil
	}
	return s.Processos.PorNumero(ctx, n.processo)
}

// Criar cadastra a diligência (código DIL. 0000001, status Pendente). Diligência é item flexível: é recusada quando cai
// no mesmo dia e na mesma hora exata de uma audiência (item crítico).
func (s *Service) Criar(ctx context.Context, ac webx.Actor, in Input) (string, error) {
	n, err := in.normalize()
	if err != nil {
		return "", err
	}
	proc, err := s.vinculos(ctx, &n)
	if err != nil {
		return "", err
	}
	if n.hora != "" {
		cs, err := s.Conflitos.NoHorario(ctx, n.data, n.hora)
		if err != nil {
			return "", err
		}
		if len(cs) > 0 {
			return "", webx.Invalidf(fmt.Sprintf("Bloqueado: conflito de horário com Audiência \"%s\" em %s às %s.", cs[0].Titulo, cs[0].Data, cs[0].Hora))
		}
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	codigo, err := codigos.Proximo(ctx, tx, "DIL")
	if err != nil {
		return "", err
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO diligencias (codigo, titulo, processo_numero, processo_id, cliente_pessoa_id, local, data_limite, hora,
		responsavel, status, created_by, updated_by) VALUES ($1,$2,$3,$4::uuid,$5::uuid,$6,$7,$8::time,$9,$10,$11,$11) RETURNING id::text`,
		codigo, n.titulo, n.processo, webx.Nullable(proc), webx.Nullable(n.cliente), n.local, n.data, webx.Nullable(n.hora),
		n.responsavel, StatusPendente, ac.UserID).Scan(&id)
	if err == nil {
		err = s.record(ctx, tx, ac, "diligencia.criada", id)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return "", err
	}
	return codigo, nil
}

// Atualizar altera título, processo, cliente, data-limite, hora, local e responsável. O status não se edita por aqui.
func (s *Service) Atualizar(ctx context.Context, ac webx.Actor, id string, version int, in Input) error {
	atual, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if atual.Status == StatusCancelada {
		return webx.Invalidf("Diligência cancelada não pode ser editada.")
	}
	in.ProcessoID = ""
	n, err := in.normalize()
	if err != nil {
		return err
	}
	proc, err := s.vinculos(ctx, &n)
	if err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE diligencias SET titulo=$3, processo_numero=$4, processo_id=$5::uuid, cliente_pessoa_id=$6::uuid, local=$7,
		data_limite=$8, hora=$9::time, responsavel=$10, updated_at=now(), updated_by=$11, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL`, atual.ID, version, n.titulo, n.processo, webx.Nullable(proc),
		webx.Nullable(n.cliente), n.local, n.data, webx.Nullable(n.hora), n.responsavel, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return webx.ErrConflict
	}
	if err = s.record(ctx, tx, ac, "diligencia.atualizada", atual.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Cancelar preserva a diligência com o motivo, quem cancelou e quando (nada se apaga). O status passa a Cancelada.
func (s *Service) Cancelar(ctx context.Context, ac webx.Actor, id string, version int, motivo string) error {
	if motivo = strings.TrimSpace(motivo); motivo == "" {
		return webx.Invalidf("Informe o motivo do cancelamento.")
	}
	if utf8.RuneCountInString(motivo) > 1000 {
		return webx.Invalidf("Texto longo demais.")
	}
	atual, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if atual.Status == StatusCancelada {
		return webx.Invalidf("Esta diligência já está cancelada.")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE diligencias SET status=$3, cancelado=true, cancelamento_motivo=$4, cancelado_em=now(), cancelado_por=$5::uuid,
		updated_at=now(), updated_by=$5::uuid, version=version+1 WHERE id=$1::uuid AND version=$2 AND excluido_em IS NULL AND NOT cancelado`,
		atual.ID, version, StatusCancelada, motivo, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return webx.ErrConflict
	}
	if err = s.record(ctx, tx, ac, "diligencia.cancelada", atual.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
