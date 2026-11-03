package processos

import (
	"embed"
	"errors"
	"html/template"
	"net/http"

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

// Register declara as rotas, cada uma com a sua permissão.
func (h *Handler) Register(rt *auth.Router) {
	h.A.AddNavCount("Processos", "/processos", auth.PermProcRead, webx.ContarAtivos(h.S.Pool, "processos"))
	rt.Handle("GET /processos", auth.PermProcRead, h.list)
	rt.Handle("GET /processos/novo", auth.PermProcWrite, h.newForm)
	rt.Handle("POST /processos", auth.PermProcWrite, h.create)
	rt.Handle("GET /processos/{id}", auth.PermProcRead, h.detail)
	rt.Handle("GET /processos/{id}/editar", auth.PermProcWrite, h.editForm)
	rt.Handle("POST /processos/{id}", auth.PermProcWrite, h.update)
	rt.Handle("POST /processos/{id}/excluir", auth.PermProcWrite, h.remove)
	rt.Handle("GET /processos/{id}/pessoas", auth.PermProcWrite, h.pessoaOptions)
	rt.Handle("POST /processos/{id}/partes", auth.PermProcWrite, h.addParte)
	rt.Handle("POST /processos/{id}/partes/remover", auth.PermProcWrite, h.removeParte)
}

type listData struct {
	Q, Situacao   string
	Rows          []Row
	Situacoes     []Opcao
	Pag, Anterior int
	Proxima       int
	CanWrite      bool
}

type formData struct {
	IsNew        bool
	ID           string
	Version      int
	In           Input
	Situacoes    []Opcao
	Responsaveis []Opcao
}

func base(s *auth.Session, title string) auth.PageData {
	return auth.PageData{Title: title, Email: s.Email, Role: string(s.Role)}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	q, sit := r.URL.Query().Get("q"), r.URL.Query().Get("situacao")
	pag, limit, off := webx.Page(r)
	rows, err := h.S.List(r.Context(), q, sit, limit, off)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	d := listData{Q: q, Situacao: sit, Pag: pag, Situacoes: Situacoes, CanWrite: auth.Can(s.Role, auth.PermProcWrite)}
	if len(rows) > webx.PageSize {
		rows, d.Proxima = rows[:webx.PageSize], pag+1
	}
	if pag > 1 {
		d.Anterior = pag - 1
	}
	d.Rows = rows
	t := page("lista.html")
	if webx.IsHX(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = t.ExecuteTemplate(w, "resultado", d)
		return
	}
	pd := base(s, "Processos")
	pd.Head = auth.Head{Title: "Processos", Sub: "Cadastro processual: partes, órgão, situação e responsável."}
	if d.CanWrite {
		pd.Head.ActionLabel, pd.Head.ActionHref = "Novo processo", "/processos/novo"
	}
	pd.Data = d
	h.A.Render(w, r, http.StatusOK, t, pd)
}

func (h *Handler) renderForm(w http.ResponseWriter, r *http.Request, s *auth.Session, status int, f formData, msg string) {
	f.Situacoes = Situacoes
	f.Responsaveis, _ = h.S.Users.Options(r.Context())
	pd := base(s, "Editar processo")
	if f.IsNew {
		pd.Title = "Novo processo"
	}
	pd.Data, pd.Error = f, msg
	h.A.Render(w, r, status, page("form.html"), pd)
}

func inputFrom(r *http.Request) Input {
	return Input{Numero: r.PostFormValue("numero"), Orgao: r.PostFormValue("orgao"),
		Situacao: r.PostFormValue("situacao"), ResponsavelID: r.PostFormValue("responsavel_id")}
}

func (h *Handler) newForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	h.renderForm(w, r, s, http.StatusOK, formData{IsNew: true, In: Input{Situacao: "ativo", ResponsavelID: s.UserID}}, "")
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	in := inputFrom(r)
	id, err := h.S.Create(r.Context(), webx.ActorOf(w, s.UserID), in)
	var inv webx.Invalid
	switch {
	case errors.As(err, &inv):
		h.renderForm(w, r, s, http.StatusBadRequest, formData{IsNew: true, In: in}, inv.Msg)
	case err != nil:
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	default:
		http.Redirect(w, r, "/processos/"+id+"?aviso=criado", http.StatusSeeOther)
	}
}

func (h *Handler) detail(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	p, err := h.S.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	pd := base(s, p.Numero)
	if auth.Can(s.Role, auth.PermProcWrite) {
		pd.Head.ActionLabel, pd.Head.ActionHref = "Editar", "/processos/"+p.ID+"/editar"
	}
	pd.Data = map[string]any{"P": p, "Qualidades": Qualidades, "CanWrite": auth.Can(s.Role, auth.PermProcWrite),
		"Aviso": r.URL.Query().Get("aviso") == "criado", "Erro": r.URL.Query().Get("erro") != ""}
	h.A.Render(w, r, http.StatusOK, page("detalhe.html"), pd)
}

func (h *Handler) editForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	p, err := h.S.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	in := Input{Numero: p.Numero, Orgao: p.Orgao, Situacao: p.Situacao, ResponsavelID: p.ResponsavelID}
	h.renderForm(w, r, s, http.StatusOK, formData{ID: p.ID, Version: p.Version, In: in}, "")
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	id, in := r.PathValue("id"), inputFrom(r)
	v := webx.Int(r.PostFormValue("version"))
	err := h.S.Update(r.Context(), webx.ActorOf(w, s.UserID), id, v, in)
	var inv webx.Invalid
	switch {
	case errors.As(err, &inv):
		h.renderForm(w, r, s, http.StatusBadRequest, formData{ID: id, Version: v, In: in}, inv.Msg)
	case err != nil:
		h.fail(w, err)
	default:
		http.Redirect(w, r, "/processos/"+id, http.StatusSeeOther)
	}
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	if err := h.S.Delete(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version"))); err != nil {
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/processos", http.StatusSeeOther)
}

// pessoaOptions devolve <option>s para o seletor de partes (busca ao vivo via htmx).
func (h *Handler) pessoaOptions(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	opts, err := h.S.Pessoas.Buscar(r.Context(), r.URL.Query().Get("busca"), 20)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.Must(template.New("o").Parse(`{{range .}}<option value="{{.Valor}}">{{.Rotulo}}</option>{{else}}<option value="">Nenhuma pessoa encontrada</option>{{end}}`))
	_ = t.Execute(w, opts)
}

func (h *Handler) addParte(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	id := r.PathValue("id")
	err := h.S.AddParte(r.Context(), webx.ActorOf(w, s.UserID), id, r.PostFormValue("pessoa_id"), r.PostFormValue("qualidade"))
	var inv webx.Invalid
	switch {
	case errors.As(err, &inv):
		http.Redirect(w, r, "/processos/"+id+"?erro=parte", http.StatusSeeOther)
	case err != nil:
		h.fail(w, err)
	default:
		http.Redirect(w, r, "/processos/"+id, http.StatusSeeOther)
	}
}

func (h *Handler) removeParte(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	id := r.PathValue("id")
	if err := h.S.RemoveParte(r.Context(), webx.ActorOf(w, s.UserID), id, r.PostFormValue("pessoa_id"), r.PostFormValue("qualidade")); err != nil {
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/processos/"+id, http.StatusSeeOther)
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, webx.ErrNotFound):
		http.Error(w, "Registro não encontrado.", http.StatusNotFound)
	case errors.Is(err, webx.ErrConflict):
		http.Error(w, "Este registro foi alterado por outra pessoa. Volte, recarregue a página e refaça a alteração.", http.StatusConflict)
	default:
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}
