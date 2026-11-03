package tarefas

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/platform/webx"
)

const cols = `id, codigo, titulo, subtitulo, estado, origem_tipo, coalesce(atendimento_id, processo_id)::text,
	coalesce(tarefa_pai_id::text,''), coalesce(criado_por_pessoa_id::text,''), descricao, observacoes,
	data_base, prazo, prazo_fatal, prioridade, version`

func scan(row pgx.Row) (*Tarefa, error) {
	var t Tarefa
	err := row.Scan(&t.ID, &t.Codigo, &t.Titulo, &t.Subtitulo, &t.Estado, &t.OrigemTipo, &t.OrigemID, &t.TarefaPaiID,
		&t.CriadoPorID, &t.Descricao, &t.Observacoes, &t.DataBase, &t.Prazo, &t.PrazoFatal, &t.Prioridade, &t.Version)
	return &t, err
}

func (s *Service) existe(ctx context.Context, id string) bool {
	var one int
	return s.Pool.QueryRow(ctx, `SELECT 1 FROM tarefas WHERE id=$1 AND excluido_em IS NULL`, id).Scan(&one) == nil
}

// Get devolve uma tarefa não excluída, já com vínculo, pessoas e situação do prazo fatal.
func (s *Service) Get(ctx context.Context, id string) (*Tarefa, error) {
	if !webx.UUID(id) {
		return nil, webx.ErrNotFound
	}
	t, err := scan(s.Pool.QueryRow(ctx, `SELECT `+cols+` FROM tarefas WHERE id=$1 AND excluido_em IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, webx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.enrich(ctx, []*Tarefa{t}); err != nil {
		return nil, err
	}
	return t, nil
}

// OpcoesVinculo monta o seletor "Vínculo específico" de um tipo (atendimento ou processo).
func (s *Service) OpcoesVinculo(ctx context.Context, tipo string) ([]Opcao, error) {
	return s.Vinculos.Opcoes(ctx, tipo)
}

// OpcoesPessoa monta os seletores de pessoa (Partes e Responsáveis): "NOME — Física (123.456.789-09)".
func (s *Service) OpcoesPessoa(ctx context.Context, ac webx.Actor, comDocumento bool) ([]Opcao, error) {
	return s.Pessoas.Opcoes(ctx, ac, comDocumento)
}

// Refs devolve código e nome das pessoas (seletor "Criado por").
func (s *Service) Refs(ctx context.Context) ([]Ref, error) { return s.Pessoas.Refs(ctx, nil) }
