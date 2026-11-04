package tarefas

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/codigos"
	"painelteodoro/internal/platform/webx"
)

// Service concentra as regras de tarefas. Só toca nas próprias tabelas; os outros domínios entram por interfaces.
type Service struct {
	Pool     *pgxpool.Pool
	Pessoas  PessoaLookup
	Vinculos VinculoLookup
	Prazos   PrazoSync
	Docs     DocumentoSink
}

func (s *Service) record(ctx context.Context, tx pgx.Tx, ac webx.Actor, action, id string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: action, TargetType: "tarefa",
		TargetID: id, Result: audit.OK, RequestID: ac.RequestID})
}

func (s *Service) recordDePara(ctx context.Context, tx pgx.Tx, ac webx.Actor, action, id, de, para string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: action, TargetType: "tarefa", TargetID: id,
		Result: audit.OK, RequestID: ac.RequestID, Meta: map[string]string{"de": de, "para": para}})
}

// msgSoAdvogado é a recusa a quem não é advogado e tenta tirar uma tarefa de "A confirmar".
const msgSoAdvogado = "Somente advogado confirma tarefas a confirmar."

func gravarListas(ctx context.Context, tx pgx.Tx, id string, n normalized) error {
	for _, t := range []string{"tarefa_partes", "tarefa_responsaveis"} {
		if _, err := tx.Exec(ctx, `DELETE FROM `+t+` WHERE tarefa_id=$1`, id); err != nil { // #nosec G202 -- tabela constante
			return err
		}
	}
	for i, p := range n.partes {
		if _, err := tx.Exec(ctx, `INSERT INTO tarefa_partes (tarefa_id, pessoa_id, papel, ordem) VALUES ($1,$2,$3,$4)`, id, p.PessoaID, p.Papel, i); err != nil {
			return err
		}
	}
	for i, r := range n.resp {
		if _, err := tx.Exec(ctx, `INSERT INTO tarefa_responsaveis (tarefa_id, pessoa_id, ordem) VALUES ($1,$2,$3)`, id, r, i); err != nil {
			return err
		}
	}
	return nil
}

// Create cadastra a tarefa vinculada e devolve o id e o código (TAR. 0000001). podeConfirmar diz se o ator é advogado
// (PermPrazoConfirm): só ele grava o prazo fatal como confirmado.
func (s *Service) Create(ctx context.Context, ac webx.Actor, in Input, podeConfirmar bool) (id, codigo string, err error) {
	n, err := s.normalize(ctx, in, false)
	if err != nil {
		return "", "", err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if codigo, err = codigos.Proximo(ctx, tx, "TAR"); err != nil {
		return "", "", err
	}
	var atend, proc any
	if n.origemTipo == "atendimento" {
		atend = n.origemID
	} else {
		proc = n.origemID
	}
	err = tx.QueryRow(ctx, `INSERT INTO tarefas (codigo, titulo, subtitulo, estado, origem_tipo, atendimento_id, processo_id, tarefa_pai_id,
		criado_por_pessoa_id, descricao, observacoes, data_base, prazo, prazo_fatal, prioridade, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$16) RETURNING id`,
		codigo, n.titulo, n.subtitulo, n.estado, n.origemTipo, atend, proc, webx.Nullable(n.pai), webx.Nullable(n.criadoPor),
		n.descricao, n.obs, n.dataBase, n.prazo, n.prazoFatal, n.prioridade, ac.UserID).Scan(&id)
	if err == nil {
		err = gravarListas(ctx, tx, id, n)
	}
	if err == nil {
		err = s.record(ctx, tx, ac, "tarefa.criada", id)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return "", "", err
	}
	novo := &Tarefa{ID: id, Codigo: codigo, Titulo: n.titulo, Estado: n.estado, OrigemTipo: n.origemTipo, OrigemID: n.origemID}
	return id, codigo, s.efeitoNoPrazo(ctx, ac, novo, nil, n.prazoFatal, podeConfirmar)
}

// Update altera com concorrência otimista. O vínculo (atendimento ou processo) não muda depois de criada a tarefa.
func (s *Service) Update(ctx context.Context, ac webx.Actor, id string, version int, in Input, podeConfirmar bool) error {
	atual, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	n, err := s.normalize(ctx, in, true)
	if err != nil {
		return err
	}
	if atual.Estado == "a-confirmar" && n.estado != "a-confirmar" && !podeConfirmar {
		return webx.Invalidf(msgSoAdvogado)
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE tarefas SET titulo=$3, subtitulo=$4, estado=$5, criado_por_pessoa_id=$6, descricao=$7, observacoes=$8,
		data_base=$9, prazo=$10, prazo_fatal=$11, prioridade=$12, updated_at=now(), updated_by=$13, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL`,
		id, version, n.titulo, n.subtitulo, n.estado, webx.Nullable(n.criadoPor), n.descricao, n.obs, n.dataBase, n.prazo, n.prazoFatal,
		n.prioridade, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return s.whyNoRows(ctx, id)
	}
	if err = gravarListas(ctx, tx, id, n); err == nil {
		err = s.record(ctx, tx, ac, "tarefa.alterada", id)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return err
	}
	novo := *atual
	novo.Titulo, novo.Estado = n.titulo, n.estado
	return s.efeitoNoPrazo(ctx, ac, &novo, atual, n.prazoFatal, podeConfirmar)
}

// Mover troca só o estado ("Mover para..."). A regra de "A confirmar" vale aqui também.
func (s *Service) Mover(ctx context.Context, ac webx.Actor, id string, version int, novoEstado string, podeConfirmar bool) error {
	if !EstadoValido(novoEstado) {
		return webx.Invalidf("Estado inválido.")
	}
	atual, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if atual.Estado == novoEstado {
		return webx.Invalidf("A tarefa já está neste estado.")
	}
	if atual.Estado == "a-confirmar" && !podeConfirmar {
		return webx.Invalidf(msgSoAdvogado)
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE tarefas SET estado=$3, updated_at=now(), updated_by=$4, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL`, id, version, novoEstado, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return s.whyNoRows(ctx, id)
	}
	if err = s.recordDePara(ctx, tx, ac, "tarefa.estado_alterado", id, atual.Estado, novoEstado); err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return err
	}
	novo := *atual
	novo.Estado = novoEstado
	return s.efeitoNoPrazo(ctx, ac, &novo, atual, atual.PrazoFatal, podeConfirmar)
}

// whyNoRows distingue tarefa inexistente de versão velha.
func (s *Service) whyNoRows(ctx context.Context, id string) error {
	if !s.existe(ctx, id) {
		return webx.ErrNotFound
	}
	return webx.ErrConflict
}

// ConfirmarPrazo confirma o prazo fatal da tarefa pelo fluxo do domínio Prazos (a rota exige PermPrazoConfirm).
func (s *Service) ConfirmarPrazo(ctx context.Context, ac webx.Actor, id string) error {
	t, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if t.PrazoFatal == nil || t.PrazoFatalSituacao != "a_conferir" {
		return webx.Invalidf("Esta tarefa não tem prazo fatal a conferir.")
	}
	return s.Prazos.Confirmar(ctx, ac, id)
}
