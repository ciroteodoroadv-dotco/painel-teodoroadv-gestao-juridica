package tarefas

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/codigos"
	"painelteodoro/internal/platform/webx"
)

// OrigemIntimacao é o vínculo das tarefas que nasceram de uma intimação do DJEN sem processo cadastrado.
const OrigemIntimacao = "intimacao"

// AnaliseIn é o que o domínio Intimações entrega para abrir a tarefa de análise. Os textos já vêm prontos (o sistema
// não calcula prazo: a tarefa nasce SEM prazo, para o advogado apurar). CriadoPor é o usuário dono do registro
// (created_by); AuditActor é quem a auditoria registra (o usuário que mandou coletar, ou "sistema").
type AnaliseIn struct {
	IntimacaoID, ProcessoID, Titulo, Descricao string
	CriadoPor, AuditActor, RequestID           string
}

// CriarAnaliseDeIntimacao abre, DENTRO da transação do chamador, a tarefa de análise em "A confirmar", sem nenhuma data.
// É idempotente por intimação: se já existe tarefa para ela, devolve a existente. Com processo conhecido, o vínculo é o
// processo; sem ele, a origem é a própria intimação (exibida como "DJEN").
func (s *Service) CriarAnaliseDeIntimacao(ctx context.Context, tx pgx.Tx, in AnaliseIn) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT id::text FROM tarefas WHERE intimacao_id=$1::uuid`, in.IntimacaoID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	codigo, err := codigos.Proximo(ctx, tx, "TAR")
	if err != nil {
		return "", err
	}
	origem, proc := OrigemIntimacao, any(nil)
	if in.ProcessoID != "" {
		origem, proc = "processo", in.ProcessoID
	}
	err = tx.QueryRow(ctx, `INSERT INTO tarefas (codigo, titulo, subtitulo, estado, origem_tipo, processo_id, intimacao_id,
		descricao, prioridade, created_by, updated_by)
		VALUES ($1,$2,'Análise de decisão','a-confirmar',$3,$4::uuid,$5::uuid,$6,'Normal',$7::uuid,$7::uuid) RETURNING id::text`,
		codigo, in.Titulo, origem, proc, in.IntimacaoID, in.Descricao, in.CriadoPor).Scan(&id)
	if err != nil {
		return "", err
	}
	err = audit.Record(ctx, tx, audit.Event{Actor: in.AuditActor, Action: "tarefa.criada_de_intimacao", TargetType: "tarefa",
		TargetID: id, Result: audit.OK, RequestID: in.RequestID})
	return id, err
}

// ResumoDeIntimacao é a tarefa de análise como a tabela de Intimações a mostra: título e estado. O B mostra o estado
// pela chave da pasta ("a-confirmar"), não pelo rótulo; o v2 repete (decisão em NOTAS-PLANO-5.md).
type ResumoDeIntimacao struct{ Titulo, Estado string }

// PorIntimacoes devolve, por id de intimação, a tarefa de análise (não excluída).
func (s *Service) PorIntimacoes(ctx context.Context, ids []string) (map[string]ResumoDeIntimacao, error) {
	out := map[string]ResumoDeIntimacao{}
	var ok []string
	for _, id := range ids {
		if webx.UUID(id) {
			ok = append(ok, id)
		}
	}
	if len(ok) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT intimacao_id::text, titulo, estado FROM tarefas
		WHERE intimacao_id = ANY($1::uuid[]) AND excluido_em IS NULL`, ok)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var iid, titulo, estado string
		if err := rows.Scan(&iid, &titulo, &estado); err != nil {
			return nil, err
		}
		out[iid] = ResumoDeIntimacao{Titulo: titulo, Estado: estado}
	}
	return out, rows.Err()
}
