package intimacoes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/webx"
)

// PrazoInternoDias é a antecedência do prazo interno de confirmação (E3 do Plano 6): data da criação + 2 dias corridos,
// data civil de São Paulo. NÃO é o prazo processual.
const PrazoInternoDias = 2

// Nova é uma intimação/movimentação a registrar (de qualquer origem: DJEN, DataJud, DJEN (sigiloso)).
type Nova struct {
	Hash, Origem, OabID, Tribunal, Orgao, ProcessoNumero, ProcessoID, Tipo string
	Data                                                                   *time.Time
	Texto, Resumo, Link                                                    string
	TituloAnalise, DescricaoAnalise, LinhaConsolidada                      string
	CriadoPor, Ator, RequestID                                             string
}

// ConsolidadaIn é o pedido da tarefa consolidada do dia (espelha tarefas.ConsolidadaIn; o adaptador liga os dois).
type ConsolidadaIn struct {
	ProcessoID, Titulo               string
	Dia                              time.Time
	Linhas                           []string
	CriadoPor, AuditActor, RequestID string
	PrazoFatal                       *time.Time
}

func (s *Service) prazoInterno() *time.Time {
	d := webx.Hoje(s.now()).AddDate(0, 0, PrazoInternoDias)
	return &d
}

// RegistrarNaTx grava a intimação nova, abre a tarefa de análise individual e, havendo processo cadastrado, a consolidada do dia
// (ambas com o prazo interno), tudo na transação do chamador. repetida=true quando o hash já existia (nada é gravado).
func (s *Service) RegistrarNaTx(ctx context.Context, tx pgx.Tx, n Nova) (id string, repetida bool, err error) {
	var data any
	if n.Data != nil && !n.Data.IsZero() {
		data = *n.Data
	}
	resumo := n.Resumo
	if resumo == "" {
		resumo = n.Texto
	}
	err = tx.QueryRow(ctx, `INSERT INTO intimacoes (hash, origem, oab_id, tribunal, orgao, processo_numero, processo_id, tipo, data_publicacao, resumo, texto, link)
		VALUES ($1,$2,$3::uuid,$4,$5,$6,$7::uuid,$8,$9,left($10,2000),$11,$12) ON CONFLICT (hash) DO NOTHING RETURNING id::text`,
		n.Hash, n.Origem, nullable(n.OabID), n.Tribunal, n.Orgao, n.ProcessoNumero, nullable(n.ProcessoID), n.Tipo, data, resumo, n.Texto, n.Link).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", true, nil
	}
	if err != nil {
		return "", false, err
	}
	if s.Tarefas == nil {
		return "", false, errors.New("intimacoes: tarefa de análise não configurada")
	}
	prazo := s.prazoInterno()
	if _, err = s.Tarefas.CriarAnaliseDeIntimacao(ctx, tx, AnaliseIn{IntimacaoID: id, ProcessoID: n.ProcessoID, Titulo: n.TituloAnalise,
		Descricao: n.DescricaoAnalise, CriadoPor: n.CriadoPor, AuditActor: n.Ator, RequestID: n.RequestID, PrazoFatal: prazo, Fonte: n.Origem}); err != nil {
		return "", false, err
	}
	if n.ProcessoID != "" {
		linha := n.LinhaConsolidada
		if linha == "" {
			linha = fmt.Sprintf("%s: %s", n.Origem, strings.TrimSpace(n.Tipo))
		}
		if _, err = s.Tarefas.CriarConsolidadaDoDia(ctx, tx, ConsolidadaIn{ProcessoID: n.ProcessoID, Titulo: "Novidades do dia — " + n.ProcessoNumero,
			Dia: webx.Hoje(s.now()), Linhas: []string{linha}, CriadoPor: n.CriadoPor, AuditActor: n.Ator, RequestID: n.RequestID, PrazoFatal: prazo}); err != nil {
			return "", false, err
		}
	}
	err = audit.Record(ctx, tx, audit.Event{Actor: n.Ator, Action: "intimacao.registrada", TargetType: "intimacao", TargetID: id,
		Result: audit.OK, RequestID: n.RequestID})
	return id, false, err
}
