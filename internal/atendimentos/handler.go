package atendimentos

import (
	"embed"
	"html/template"
	"net/http"
	"time"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/webx"
)

//go:embed templates/*.html
var tplFS embed.FS

// Handler liga as telas ao serviço.
type Handler struct {
	A *auth.Auth
	S *Service
}

func page(name string) *template.Template {
	return template.Must(auth.BaseTemplate().ParseFS(tplFS, "templates/"+name))
}

func fragmento(names ...string) *template.Template {
	for i := range names {
		names[i] = "templates/" + names[i]
	}
	return template.Must(template.ParseFS(tplFS, names...))
}

// Register declara as rotas, cada uma com a sua permissão.
func (h *Handler) Register(rt *auth.Router) {
	h.A.AddNavCount("Atendimentos", "/atendimentos", auth.PermAtendimentoRead, webx.ContarAtivos(h.S.Pool, "atendimentos"))
	w := auth.PermAtendimentoWrite
	rt.Handle("GET /atendimentos", auth.PermAtendimentoRead, h.list)
	rt.Handle("GET /atendimentos/novo", w, h.newForm)
	rt.Handle("GET /atendimentos/pessoas", w, h.pessoasOpcoes)
	rt.Handle("POST /atendimentos", w, h.create)
	rt.Handle("GET /atendimentos/{id}/editar", w, h.editForm)
	rt.Handle("POST /atendimentos/{id}", w, h.update)
}

type listData struct {
	Lista    []*Atendimento
	Total    int
	CanWrite bool
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	l, err := h.S.Listar(r.Context(), "")
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	d := listData{Lista: l, Total: len(l), CanWrite: auth.Can(s.Role, auth.PermAtendimentoWrite)}
	pd := auth.PageData{Title: "Atendimentos", Email: s.Email, Role: string(s.Role)}
	pd.Head = auth.Head{Title: "Atendimentos", Sub: "Registro das demandas e dos atendimentos do escritório."}
	if d.CanWrite {
		pd.Head.ActionLabel, pd.Head.ActionHref, pd.Head.ActionDrawer = "Novo atendimento", "/atendimentos/novo", true
	}
	pd.Data = d
	h.A.Render(w, r, http.StatusOK, page("lista.html"), pd)
}

// inputFrom lê o formulário do drawer.
func inputFrom(r *http.Request) Input {
	_ = r.ParseForm()
	v := r.PostFormValue
	return Input{CriadoPorID: v("criado_por"), Clientes: r.PostForm["clientes"], PartesAdversas: r.PostForm["partes_adversas"],
		Descricao: v("descricao"), Observacoes: v("observacoes"), DataAbertura: v("data_abertura")}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	id, _, err := h.S.Create(r.Context(), webx.ActorOf(w, s.UserID), inputFrom(r))
	if webx.Falha(w, err, "Atendimento não encontrado.", "Falha ao salvar atendimento.") {
		return
	}
	w.Header().Set("X-Painel-Id", id)
	webx.Ok(w, "Atendimento criado.")
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.Update(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version")), inputFrom(r))
	if webx.Falha(w, err, "Atendimento não encontrado.", "Falha ao salvar atendimento.") {
		return
	}
	webx.Ok(w, "Atendimento atualizado.")
}

// LinhaPessoa é uma linha de cliente ou de parte adversa no drawer.
type LinhaPessoa struct {
	Papel, Singular string // Papel: "clientes" | "partes_adversas"
	Sel             string
	Pessoas         []Ref
}

// Grupo é um dos dois fieldsets de pessoas.
type Grupo struct {
	Papel, Legenda, Singular, Plural, Adicionar, Novo string
	Linhas                                            []LinhaPessoa
	Vazia                                             LinhaPessoa
}

type formView struct {
	Novo        bool
	A           Atendimento
	Data        string
	Pessoas     []Ref
	Grupos      []Grupo
	CanNovaPess bool
}

func (h *Handler) montaForm(w http.ResponseWriter, r *http.Request, s *auth.Session, a Atendimento, novo bool) {
	refs, err := h.S.Pessoas.Refs(r.Context(), nil)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	linhas := func(papel, sing string, rs []Ref) []LinhaPessoa {
		out := []LinhaPessoa{}
		for _, x := range rs {
			out = append(out, LinhaPessoa{Papel: papel, Singular: sing, Sel: x.ID, Pessoas: refs})
		}
		if len(out) == 0 {
			out = append(out, LinhaPessoa{Papel: papel, Singular: sing, Pessoas: refs})
		}
		return out
	}
	data := a.DataISO()
	if novo {
		data = webx.Hoje(time.Now()).Format("2006-01-02")
	}
	v := formView{Novo: novo, A: a, Data: data, Pessoas: refs, CanNovaPess: auth.Can(s.Role, auth.PermPessoasWrite),
		Grupos: []Grupo{
			{"clientes", "Cliente(s)", "Cliente", "clientes", "Adicionar cliente", "Novo cliente",
				linhas("clientes", "Cliente", a.Clientes), LinhaPessoa{Papel: "clientes", Singular: "Cliente", Pessoas: refs}},
			{"partes_adversas", "Parte(s) adversa(s)", "Parte adversa", "partes adversas", "Adicionar parte adversa", "Nova parte adversa",
				linhas("partes_adversas", "Parte adversa", a.PartesAdversas), LinhaPessoa{Papel: "partes_adversas", Singular: "Parte adversa", Pessoas: refs}},
		}}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("drawer.html").ExecuteTemplate(w, "form", v); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

func (h *Handler) newForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	h.montaForm(w, r, s, Atendimento{}, true)
}

func (h *Handler) editForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	a, err := h.S.Get(r.Context(), r.PathValue("id"))
	if webx.Falha(w, err, "Atendimento não encontrado.") {
		return
	}
	h.montaForm(w, r, s, *a, false)
}

// pessoasOpcoes devolve as opções do seletor de pessoa (usado depois de "Novo cliente" criar uma pessoa nova).
func (h *Handler) pessoasOpcoes(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	refs, err := h.S.Pessoas.Refs(r.Context(), nil)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("drawer.html").ExecuteTemplate(w, "opcoesSel", LinhaPessoa{Pessoas: refs}); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}
