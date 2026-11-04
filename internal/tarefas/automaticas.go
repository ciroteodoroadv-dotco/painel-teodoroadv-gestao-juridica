package tarefas

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/codigos"
	"painelteodoro/internal/platform/webx"
)

// TextoPrazoInterno é a frase obrigatória na descrição das tarefas de análise (E3 do Plano 6): o prazo fatal dessas
// tarefas é INTERNO, de confirmação pelo advogado, e nunca o prazo processual (que o sistema não calcula nem presume).
const TextoPrazoInterno = "Prazo interno de confirmação (2 dias corridos). NÃO é o prazo processual: o advogado apura o prazo da intimação."

const maxDescricao = 20000

func descricaoComPrazoInterno(d string, prazo *time.Time) string {
	if prazo == nil || strings.Contains(d, TextoPrazoInterno) {
		return d
	}
	return strings.TrimRight(d, "\n") + "\n" + TextoPrazoInterno
}

// prazoInterno grava o prazo fatal interno da tarefa recém-criada e o espelha no domínio Prazos, na MESMA transação,
// pelo mesmo caminho do prazo fatal da tarefa (PrazoSync): fica "a conferir" (o selo A CONFERIR aparece no mesmo campo).
func (s *Service) prazoInterno(ctx context.Context, tx pgx.Tx, id, processoID, titulo string, prazo *time.Time, criadoPor, reqID string) error {
	if prazo == nil {
		return nil
	}
	if _, err := tx.Exec(ctx, `UPDATE tarefas SET prazo_fatal=$2, prazo_interno=true WHERE id=$1::uuid`, id, prazo); err != nil {
		return err
	}
	desc := titulo
	if utf8.RuneCountInString(desc) > 480 {
		desc = string([]rune(desc)[:480])
	}
	_, err := s.Prazos.SincronizarTx(ctx, tx, webx.Actor{UserID: criadoPor, RequestID: reqID}, id, processoID, "Prazo interno de confirmação — "+desc, prazo, false)
	return err
}

// ConsolidadaIn é o pedido da tarefa consolidada do dia: uma por processo e por dia de coleta com novidades.
type ConsolidadaIn struct {
	ProcessoID, Titulo               string
	Dia                              time.Time // data civil de São Paulo
	Linhas                           []string  // as novidades, uma por linha
	CriadoPor, AuditActor, RequestID string
	PrazoFatal                       *time.Time
}

// CriarConsolidadaDoDia abre, dentro da transação do chamador, a tarefa consolidada em "A confirmar" ou, se o processo já tem
// uma do dia, acrescenta as novas linhas à descrição (idempotente por processo e dia). Nasce com o prazo interno.
func (s *Service) CriarConsolidadaDoDia(ctx context.Context, tx pgx.Tx, in ConsolidadaIn) (string, error) {
	dia := time.Date(in.Dia.Year(), in.Dia.Month(), in.Dia.Day(), 0, 0, 0, 0, time.UTC)
	texto := strings.Join(in.Linhas, "\n")
	var id string
	err := tx.QueryRow(ctx, `SELECT id::text FROM tarefas WHERE processo_id=$1::uuid AND consolidada_dia=$2 FOR UPDATE`, in.ProcessoID, dia).Scan(&id)
	switch {
	case err == nil:
		_, err = tx.Exec(ctx, `UPDATE tarefas SET descricao=left(descricao || E'\n' || $2, $3), version=version+1, updated_at=now() WHERE id=$1::uuid`, id, texto, maxDescricao)
		return id, err
	case !errors.Is(err, pgx.ErrNoRows):
		return "", err
	}
	codigo, err := codigos.Proximo(ctx, tx, "TAR")
	if err != nil {
		return "", err
	}
	desc := "Novidades coletadas no dia. Cada uma tem a sua tarefa de análise individual.\n" + texto
	err = tx.QueryRow(ctx, `INSERT INTO tarefas (codigo, titulo, subtitulo, estado, origem_tipo, processo_id, consolidada_dia, descricao,
		prioridade, created_by, updated_by) VALUES ($1,$2,'Análise consolidada do dia','a-confirmar','processo',$3::uuid,$4,$5,'Normal',$6::uuid,$6::uuid)
		RETURNING id::text`, codigo, in.Titulo, in.ProcessoID, dia, descricaoComPrazoInterno(left(desc, maxDescricao-300), in.PrazoFatal), in.CriadoPor).Scan(&id)
	if err != nil {
		return "", err
	}
	if err = s.prazoInterno(ctx, tx, id, in.ProcessoID, in.Titulo, in.PrazoFatal, in.CriadoPor, in.RequestID); err != nil {
		return "", err
	}
	return id, audit.Record(ctx, tx, audit.Event{Actor: in.AuditActor, Action: "tarefa.criada_consolidada", TargetType: "tarefa",
		TargetID: id, Result: audit.OK, RequestID: in.RequestID})
}

func left(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// encerrarInterno tira o prazo fatal interno da tarefa e o encerra no domínio Prazos (a conferir nunca valeu: vira cancelado).
func (s *Service) encerrarInterno(ctx context.Context, ac webx.Actor, id string) error {
	if _, err := s.Pool.Exec(ctx, `UPDATE tarefas SET prazo_fatal=NULL, prazo_interno=false WHERE id=$1::uuid AND prazo_interno`, id); err != nil {
		return err
	}
	return s.encerraPrazoInterno(ctx, ac, id)
}

// EncerrarPrazoInternoDaIntimacao encerra o prazo interno da tarefa de análise quando a intimação é decidida (Confirmar/Cancelar).
// Repetir não é erro.
func (s *Service) EncerrarPrazoInternoDaIntimacao(ctx context.Context, ac webx.Actor, intimacaoID string) error {
	var id string
	err := s.Pool.QueryRow(ctx, `SELECT id::text FROM tarefas WHERE intimacao_id=$1::uuid AND prazo_interno`, intimacaoID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.encerrarInterno(ctx, ac, id)
}

// saiDoPrazoInterno: a tarefa tinha prazo interno e está saindo de "A confirmar" (só o advogado move).
func saiDoPrazoInterno(atual *Tarefa, novoEstado string) bool {
	return atual.PrazoInterno && atual.Estado == "a-confirmar" && novoEstado != "a-confirmar"
}

func (s *Service) encerraPrazoInterno(ctx context.Context, ac webx.Actor, id string) error {
	if err := s.Prazos.Encerrar(ctx, ac, id, "cancelada"); err != nil {
		return webx.Invalidf("A tarefa foi salva, mas o prazo interno não foi encerrado no domínio Prazos. Abra a tarefa e salve de novo.")
	}
	return nil
}
