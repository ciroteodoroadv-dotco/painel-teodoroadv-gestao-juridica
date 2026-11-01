package pessoas

import (
	"embed"
	"errors"
	"html/template"
	"net/http"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/br"
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
	h.A.AddNav("Pessoas", "/pessoas", auth.PermPessoasRead)
	rt.Handle("GET /pessoas", auth.PermPessoasRead, h.list)
	rt.Handle("GET /pessoas/nova", auth.PermPessoasWrite, h.newForm)
	rt.Handle("POST /pessoas", auth.PermPessoasWrite, h.create)
	rt.Handle("GET /pessoas/{id}", auth.PermPessoasRead, h.detail)
	rt.Handle("GET /pessoas/{id}/editar", auth.PermPessoasWrite, h.editForm)
	rt.Handle("POST /pessoas/{id}", auth.PermPessoasWrite, h.update)
	rt.Handle("POST /pessoas/{id}/excluir", auth.PermPessoasWrite, h.remove)
	rt.Handle("GET /pessoas/{id}/relatorio", auth.PermPessoasWrite, h.relatorio)
}

type listData struct {
	Q        string
	Pessoas  []Pessoa
	Pag      int
	Anterior int
	Proxima  int
	CanWrite bool
}

type formData struct {
	IsNew   bool
	ID      string
	Version int
	In      Input
	Papeis  []Opcao
}

func base(s *auth.Session, title string) auth.PageData {
	return auth.PageData{Title: title, Email: s.Email, Role: string(s.Role)}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	q := r.URL.Query().Get("q")
	pag, limit, off := webx.Page(r)
	rows, err := h.S.List(r.Context(), q, limit, off)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	d := listData{Q: q, Pag: pag, CanWrite: auth.Can(s.Role, auth.PermPessoasWrite)}
	if len(rows) > webx.PageSize {
		rows, d.Proxima = rows[:webx.PageSize], pag+1
	}
	if pag > 1 {
		d.Anterior = pag - 1
	}
	d.Pessoas = rows
	t := page("lista.html")
	if webx.IsHX(r) { // busca ao vivo: troca só a tabela, o campo de busca (e o foco) ficam intactos
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = t.ExecuteTemplate(w, "resultado", d)
		return
	}
	pd := base(s, "Pessoas")
	pd.Data = d
	h.A.Render(w, r, http.StatusOK, t, pd)
}

func (h *Handler) renderForm(w http.ResponseWriter, r *http.Request, s *auth.Session, status int, f formData, msg string) {
	f.Papeis = Papeis
	pd := base(s, "Pessoa")
	pd.Data, pd.Error = f, msg
	h.A.Render(w, r, status, page("form.html"), pd)
}

func inputFrom(r *http.Request) Input {
	return Input{Tipo: r.PostFormValue("tipo"), Nome: r.PostFormValue("nome"), Documento: r.PostFormValue("documento"),
		Email: r.PostFormValue("email"), Telefone: r.PostFormValue("telefone"), Papel: r.PostFormValue("papel"),
		RetencaoAte: r.PostFormValue("retencao_ate")}
}

func (h *Handler) newForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	h.renderForm(w, r, s, http.StatusOK, formData{IsNew: true, In: Input{Tipo: "PF", Papel: "cliente"}}, "")
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
		http.Redirect(w, r, "/pessoas/"+id+"?aviso=criada", http.StatusSeeOther)
	}
}

func (h *Handler) detail(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	p, err := h.S.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	var refs []ProcessoRef
	if h.S.Processos != nil {
		refs, _ = h.S.Processos.ProcessosDaPessoa(r.Context(), p.ID)
	}
	pd := base(s, p.Nome)
	pd.Data = map[string]any{"P": p, "Refs": refs, "CanWrite": auth.Can(s.Role, auth.PermPessoasWrite),
		"Aviso": r.URL.Query().Get("aviso") == "criada"}
	h.A.Render(w, r, http.StatusOK, page("detalhe.html"), pd)
}

func (h *Handler) editForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	id := r.PathValue("id")
	p, err := h.S.Get(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	doc, err := h.S.Reveal(r.Context(), webx.ActorOf(w, s.UserID), id) // auditado
	if err != nil {
		h.fail(w, err)
		return
	}
	in := Input{Tipo: p.Tipo, Nome: p.Nome, Documento: br.FormatDocumento(doc), Email: p.Email, Telefone: p.Telefone, Papel: p.Papel}
	if p.RetencaoAte != nil {
		in.RetencaoAte = p.RetencaoAte.Format("2006-01-02")
	}
	h.renderForm(w, r, s, http.StatusOK, formData{ID: id, Version: p.Version, In: in}, "")
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
		http.Redirect(w, r, "/pessoas/"+id, http.StatusSeeOther)
	}
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.Delete(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version")))
	var inv webx.Invalid
	if errors.As(err, &inv) {
		http.Error(w, inv.Msg, http.StatusConflict)
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/pessoas", http.StatusSeeOther)
}

func (h *Handler) relatorio(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	rel, err := h.S.Relatorio(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	pd := base(s, "Dados do titular")
	pd.Data = rel
	h.A.Render(w, r, http.StatusOK, page("relatorio.html"), pd)
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
