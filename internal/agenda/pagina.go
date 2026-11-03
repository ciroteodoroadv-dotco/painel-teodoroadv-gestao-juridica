package agenda

import (
	"context"
	"net/http"
	"strings"
	"time"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/webx"
)

// categoriaDef é uma das 7 pílulas da Agenda, na ordem do B (CATEGORIAS_AGENDA). Tipo é o tipo de compromisso
// que a categoria mostra; Subtitulos são os subtítulos de tarefa (normalizados) que também entram nela.
type categoriaDef struct {
	Chave, Rotulo, Tipo string
	Subtitulos          []string
}

var categorias = []categoriaDef{
	{"tarefas", "Tarefas", "", nil},
	{"intimacoes", "Intimações", "", nil},
	{"audiencias", "Audiências", "", nil},
	{"diligencias", "Diligências", "", nil},
	{"reunioes", "Reuniões", "Reunião", []string{"reuniao", "reunioes"}},
	{"ligacoes", "Ligações", "Ligação", []string{"ligacao", "ligacoes"}},
	{"aulas", "Aulas", "Aula", []string{"aula", "aulas"}},
}

// CompromissoLinha é um compromisso na tabela da Agenda (linhaAgenda do B).
type CompromissoLinha struct {
	ID, Tipo, Descricao, Data, Hora, Local, Processo, CriadoPor, Cliente, Atendimento string
	Busca                                                                             string
}

// Categoria é uma pílula com o seu painel.
type Categoria struct {
	Chave, Rotulo string
	Count         int
	Tarefas       []TarefaLinha
	Intimacoes    []IntimacaoLinha
	IntimOK       bool
	IntimMsg      string
	Audiencias    []AudienciaLinha
	Diligencias   []DiligenciaLinha
	Compromissos  []CompromissoLinha

	CanWrite, CanAudWrite, CanAudCancel, CanDilWrite, CanDecidir bool
}

type paginaData struct{ Cats []Categoria }

// contar é o contador do menu: compromissos + audiências + diligências (renderNav do B).
func (h *Handler) contar(ctx context.Context) (int, error) {
	n, err := h.S.Contar(ctx)
	if err != nil {
		return 0, err
	}
	for _, f := range []func(context.Context) (int, error){h.S.Audiencias.Total, h.S.Diligencias.Total} {
		m, err := f(ctx)
		if err != nil {
			return 0, err
		}
		n += m
	}
	return n, nil
}

func (h *Handler) pagina(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	ctx := r.Context()
	cats, err := h.S.Categorias(ctx, h.S.now(), func(c *Categoria) {
		c.CanWrite = auth.Can(s.Role, auth.PermAgendaWrite)
		c.CanAudWrite = auth.Can(s.Role, auth.PermAudienciaWrite)
		c.CanAudCancel = auth.Can(s.Role, auth.PermAudienciaCancel)
		c.CanDilWrite = auth.Can(s.Role, auth.PermDiligenciaWrite)
		c.CanDecidir = auth.Can(s.Role, auth.PermIntimacaoDecide)
	})
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	pd := auth.PageData{Title: "Agenda", Email: s.Email, Role: string(s.Role)}
	pd.Head = auth.Head{Title: "Agenda", Sub: "Painel principal: compromissos e tarefas em aberto, por categoria."}
	if auth.Can(s.Role, auth.PermAgendaWrite) {
		pd.Head.ActionLabel, pd.Head.ActionHref, pd.Head.ActionDrawer = "Novo compromisso", "/agenda/novo", true
	}
	pd.Faixas, pd.Widgets = h.A.HomeExtras(ctx, s)
	pd.Data = paginaData{Cats: cats}
	h.A.Render(w, r, http.StatusOK, page("pagina.html"), pd)
}

// Categorias monta as 7 categorias com a regra de "em aberto" do B (data de hoje em diante, no fuso de Brasília).
func (s *Service) Categorias(ctx context.Context, agora time.Time, marca func(*Categoria)) ([]Categoria, error) {
	hoje := webx.Hoje(agora)
	comps, err := s.EmAberto(ctx, hoje)
	if err != nil {
		return nil, err
	}
	linhas, err := s.linhasCompromisso(ctx, comps)
	if err != nil {
		return nil, err
	}
	var out []Categoria
	for _, d := range categorias {
		c := Categoria{Chave: d.Chave, Rotulo: d.Rotulo}
		switch d.Chave {
		case "tarefas":
			c.Tarefas, err = s.Tarefas.EmAberto(ctx, hoje)
			c.Count = len(c.Tarefas)
		case "intimacoes":
			c.IntimOK, c.IntimMsg = false, "Nenhuma sincronização de intimações foi executada."
			if s.Intimacoes != nil {
				var msg string
				if c.Intimacoes, c.IntimOK, msg, err = s.Intimacoes.Lista(ctx); err == nil && msg != "" {
					c.IntimMsg = msg
				}
			}
			c.Count = len(c.Intimacoes)
		case "audiencias":
			c.Audiencias, err = s.Audiencias.EmAberto(ctx, hoje)
			c.Count = len(c.Audiencias)
		case "diligencias":
			c.Diligencias, err = s.Diligencias.EmAberto(ctx, hoje)
			c.Count = len(c.Diligencias)
		default:
			for i, cp := range comps {
				if cp.Tipo == d.Tipo {
					c.Compromissos = append(c.Compromissos, linhas[i])
				}
			}
			c.Tarefas, err = s.Tarefas.PorSubtitulo(ctx, hoje, d.Subtitulos)
			c.Count = len(c.Compromissos) + len(c.Tarefas)
		}
		if err != nil {
			return nil, err
		}
		if marca != nil {
			marca(&c)
		}
		out = append(out, c)
	}
	return out, nil
}

// linhasCompromisso resolve nomes e códigos dos vínculos (uma leitura por domínio) e monta as linhas.
func (s *Service) linhasCompromisso(ctx context.Context, comps []*Compromisso) ([]CompromissoLinha, error) {
	var pessoas, atend, proc []string
	for _, c := range comps {
		pessoas = append(pessoas, c.ClienteID, c.CriadoPorID)
		atend, proc = append(atend, c.AtendimentoID), append(proc, c.ProcessoID)
	}
	refs := map[string]Ref{}
	if ids := naoVazios(pessoas); len(ids) > 0 {
		rs, err := s.Pessoas.Refs(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, r := range rs {
			refs[r.ID] = r
		}
	}
	ate, err := s.Vinculos.Resumos(ctx, "atendimento", naoVazios(atend))
	if err != nil {
		return nil, err
	}
	pro, err := s.Vinculos.Resumos(ctx, "processo", naoVazios(proc))
	if err != nil {
		return nil, err
	}
	out := make([]CompromissoLinha, len(comps))
	for i, c := range comps {
		l := CompromissoLinha{ID: c.ID, Tipo: c.Tipo, Descricao: c.Descricao, Data: webx.DataBR(c.Data), Hora: c.Hora, Local: c.Local,
			Processo: pro[c.ProcessoID], Atendimento: ate[c.AtendimentoID]}
		if r, ok := refs[c.ClienteID]; ok && c.ClienteID != "" {
			l.Cliente = r.Rotulo()
		}
		if r, ok := refs[c.CriadoPorID]; ok && c.CriadoPorID != "" {
			l.CriadoPor = r.Nome
		}
		l.Busca = webx.SemAcento(strings.Join([]string{c.Tipo, c.Descricao, c.Local, c.DataISO(), l.Data, c.Hora, l.CriadoPor,
			l.Cliente, l.Atendimento, l.Processo}, " "))
		out[i] = l
	}
	return out, nil
}

func naoVazios(ids []string) []string {
	var out []string
	vistos := map[string]bool{}
	for _, id := range ids {
		if id != "" && !vistos[id] {
			vistos[id] = true
			out = append(out, id)
		}
	}
	return out
}
