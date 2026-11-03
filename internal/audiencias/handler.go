package audiencias

import (
	"embed"
	"html/template"
	"net/http"
	"strings"

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
	h.A.AddNavCount("Audiências", "/audiencias", auth.PermAudienciaRead, h.S.Total)
	w := auth.PermAudienciaWrite
	rt.Handle("GET /audiencias", auth.PermAudienciaRead, h.lista)
	rt.Handle("GET /audiencias/nova", w, h.novaForm)
	rt.Handle("POST /audiencias", w, h.criar)
	rt.Handle("GET /audiencias/{id}/editar", w, h.editarForm)
	rt.Handle("POST /audiencias/{id}", w, h.atualizar)
	rt.Handle("GET /audiencias/{id}/cancelar", auth.PermAudienciaCancel, h.cancelarForm)
	rt.Handle("POST /audiencias/{id}/cancelar", auth.PermAudienciaCancel, h.cancelar)
}

// Painel é um tipo de ato com as suas audiências (uma pílula do acordeão).
type Painel struct {
	Ato    Ato
	Linhas []Linha
}

type listaData struct {
	Paineis             []Painel
	CanWrite, CanCancel bool
}

func (h *Handler) lista(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	as, err := h.S.Listar(r.Context())
	var linhas []Linha
	if err == nil {
		linhas, err = h.S.Linhas(r.Context(), as)
	}
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	d := listaData{CanWrite: auth.Can(s.Role, auth.PermAudienciaWrite), CanCancel: auth.Can(s.Role, auth.PermAudienciaCancel)}
	for _, a := range Atos {
		p := Painel{Ato: a}
		for _, l := range linhas {
			if l.TipoAto == a.Chave {
				p.Linhas = append(p.Linhas, l)
			}
		}
		d.Paineis = append(d.Paineis, p)
	}
	pd := auth.PageData{Title: "Audiências", Email: s.Email, Role: string(s.Role)}
	pd.Head = auth.Head{Title: "Audiências", Sub: "Audiências marcadas, agrupadas por tipo de ato."}
	if d.CanWrite {
		pd.Head.ActionLabel, pd.Head.ActionHref, pd.Head.ActionDrawer = "Nova audiência", "/audiencias/nova", true
	}
	pd.Data = d
	h.A.Render(w, r, http.StatusOK, page("pagina.html"), pd)
}

type formView struct {
	Novo     bool
	A        Audiencia
	Atos     []Ato
	Pessoas  []Opcao
	ClienteS string
}

func (h *Handler) montaForm(w http.ResponseWriter, r *http.Request, s *auth.Session, a Audiencia, novo bool) {
	pessoas, err := h.S.Pessoas.Opcoes(r.Context(), webx.ActorOf(w, s.UserID), auth.Can(s.Role, auth.PermPessoasWrite))
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("drawer.html").ExecuteTemplate(w, "form", formView{Novo: novo, A: a, Atos: Atos, Pessoas: pessoas, ClienteS: a.ClienteID}); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

func (h *Handler) novaForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	h.montaForm(w, r, s, Audiencia{}, true)
}

func (h *Handler) editarForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	a, err := h.S.Get(r.Context(), r.PathValue("id"))
	if webx.Falha(w, err, "Audiência não encontrada.") {
		return
	}
	if a.Cancelada {
		http.Error(w, "Audiência cancelada não pode ser editada.", http.StatusBadRequest)
		return
	}
	h.montaForm(w, r, s, *a, false)
}

func (h *Handler) cancelarForm(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	a, err := h.S.Get(r.Context(), r.PathValue("id"))
	if webx.Falha(w, err, "Audiência não encontrada.") {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("cancelar.html").ExecuteTemplate(w, "cancelar", a); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

func inputFrom(r *http.Request) Input {
	v := r.PostFormValue
	return Input{Tipo: v("tipo"), Titulo: v("titulo"), Processo: v("processo"), ClienteID: v("cliente"), Data: v("data"),
		Hora: v("hora"), Juizo: v("juizo"), Local: v("local"), Descricao: v("descricao")}
}

func (h *Handler) criar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	codigo, aviso, err := h.S.Create(r.Context(), webx.ActorOf(w, s.UserID), inputFrom(r))
	if webx.Falha(w, err, "Registro não encontrado.", "Falha ao criar a audiência.") {
		return
	}
	if aviso != "" {
		webx.Ok(w, "Audiência criada — "+aviso)
		return
	}
	webx.Ok(w, "Audiência criada — código "+codigo+".")
}

func (h *Handler) atualizar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.Update(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version")), inputFrom(r))
	if webx.Falha(w, err, "Audiência não encontrada.", "Falha ao atualizar a audiência.") {
		return
	}
	webx.Ok(w, "Audiência atualizada.")
}

func (h *Handler) cancelar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.Cancelar(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version")),
		strings.TrimSpace(r.PostFormValue("motivo")), r.PostFormValue("senha"))
	if webx.Falha(w, err, "Audiência não encontrada.", "Não foi possível cancelar a audiência.") {
		return
	}
	webx.Ok(w, "Audiência cancelada; a ficha foi preservada.")
}
