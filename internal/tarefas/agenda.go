package tarefas

import (
	"context"
	"sort"
	"strings"
	"time"

	"painelteodoro/internal/platform/webx"
)

// AgendaLinha é uma tarefa na tabela da Agenda (tabelaTarefasCategoriaAgenda do B): datas em dd/mm/aaaa ou "—".
type AgendaLinha struct {
	Codigo, Titulo, CriadoPor, Prazo, PrazoFatal string
	Clientes                                     []Ref
	StatusRotulo, StatusClasse                   string
	FatalAConferir                               bool
	Busca                                        string
}

// statusAgenda é o computeTarefaStatus do B: o selo do estado, trocado por "· atrasada" (vermelho) quando o prazo de
// entrega já passou e a tarefa ainda corre.
func statusAgenda(t *Tarefa, hoje time.Time) (rotulo, classe string) {
	rotulo, classe = t.Estado, "badge-neutral"
	if rotulo == "" {
		rotulo = "Sem estado"
	}
	for _, e := range Estados {
		if e.Chave == t.Estado {
			rotulo, classe = e.Selo, e.Classe
		}
	}
	if t.Prazo != nil && t.Prazo.Before(hoje) && t.Aberta() {
		return rotulo + " · atrasada", "badge-danger"
	}
	return rotulo, classe
}

// agendaLinhas completa as tarefas com os clientes do vínculo (nome e código PES) e monta as linhas.
func (s *Service) agendaLinhas(ctx context.Context, ts []*Tarefa, hoje time.Time) ([]AgendaLinha, error) {
	porTipo := map[string][]string{}
	for _, t := range ts {
		porTipo[t.OrigemTipo] = append(porTipo[t.OrigemTipo], t.OrigemID)
	}
	clientes := map[string][]Ref{}
	for tipo, ids := range porTipo {
		m, err := s.Vinculos.Clientes(ctx, tipo, ids)
		if err != nil {
			return nil, err
		}
		for id, rs := range m {
			clientes[tipo+"|"+id] = rs
		}
	}
	out := make([]AgendaLinha, len(ts))
	for i, t := range ts {
		l := AgendaLinha{Codigo: t.Codigo, Titulo: t.Titulo, CriadoPor: t.CriadoPor, Prazo: t.PrazoBR(), PrazoFatal: t.PrazoFatalBR(),
			Clientes: clientes[t.OrigemTipo+"|"+t.OrigemID], FatalAConferir: t.PrazoFatal != nil && t.AConferir()}
		l.StatusRotulo, l.StatusClasse = statusAgenda(t, hoje)
		var nomes []string
		for _, c := range l.Clientes {
			nomes = append(nomes, c.Nome, c.Codigo)
		}
		l.Busca = webx.SemAcento(t.busca() + " " + strings.Join(nomes, " ") + " " + t.CriadoPor + " " + t.PrazoISO() + " " + l.Prazo + " " + l.PrazoFatal)
		out[i] = l
	}
	return out, nil
}

func emAberto(l []*Tarefa) (out []*Tarefa) {
	for _, t := range l {
		if t.Aberta() {
			out = append(out, t)
		}
	}
	return out
}

// AgendaEmAberto devolve as tarefas em aberto (categoria Tarefas), por código.
func (s *Service) AgendaEmAberto(ctx context.Context, hoje time.Time) ([]AgendaLinha, error) {
	l, err := s.Listar(ctx, "")
	if err != nil {
		return nil, err
	}
	return s.agendaLinhas(ctx, emAberto(l), hoje)
}

// AgendaPorSubtitulo devolve as em aberto cujo subtítulo, sem acento e em minúsculas, é um dos normalizados, por
// prazo de entrega (ou data base), como tarefasDaCategoriaAgenda do B.
func (s *Service) AgendaPorSubtitulo(ctx context.Context, hoje time.Time, normalizados []string) ([]AgendaLinha, error) {
	l, err := s.Listar(ctx, "")
	if err != nil {
		return nil, err
	}
	var sel []*Tarefa
	for _, t := range emAberto(l) {
		for _, n := range normalizados {
			if webx.SemAcento(strings.TrimSpace(t.Subtitulo)) == n {
				sel = append(sel, t)
				break
			}
		}
	}
	chave := func(t *Tarefa) string {
		if t.Prazo != nil {
			return t.PrazoISO()
		}
		return t.DataBaseISO()
	}
	sort.SliceStable(sel, func(i, j int) bool { return chave(sel[i]) < chave(sel[j]) })
	return s.agendaLinhas(ctx, sel, hoje)
}

// TitulosComPrazoEm devolve os títulos das tarefas em aberto cujo prazo de entrega é esse dia (aviso do B).
func (s *Service) TitulosComPrazoEm(ctx context.Context, data time.Time) ([]string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT titulo FROM tarefas WHERE excluido_em IS NULL AND prazo=$1
		AND estado NOT IN ('concluidas-com-sucesso','concluidas-sem-sucesso','canceladas') ORDER BY codigo`, data)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
