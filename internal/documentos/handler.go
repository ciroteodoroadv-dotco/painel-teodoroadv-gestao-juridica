package documentos

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/url"

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

var funcs = template.FuncMap{"tam": func(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KiB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}}

func page(name string) *template.Template {
	return template.Must(auth.BaseTemplate().Funcs(funcs).ParseFS(tplFS, "templates/"+name))
}

// Register declara as rotas, cada uma com a sua permissão.
func (h *Handler) Register(rt *auth.Router) {
	h.A.AddNavSistema("Documentos", "/documentos", auth.PermDocRead)
	rt.Handle("GET /documentos", auth.PermDocRead, h.list)
	rt.Handle("GET /documentos/novo", auth.PermDocWrite, h.newForm)
	rt.Handle("POST /documentos", auth.PermDocWrite, h.upload)
	rt.Handle("GET /documentos/processos", auth.PermDocWrite, h.processoOptions)
	rt.Handle("GET /documentos/pessoas", auth.PermDocWrite, h.pessoaOptions)
	rt.Handle("GET /documentos/{id}/baixar", auth.PermDocRead, h.download)
	rt.Handle("POST /documentos/{id}/excluir", auth.PermDocWrite, h.remove)
}

type listData struct {
	F             Filtro
	Docs          []Doc
	Pag, Anterior int
	Proxima       int
	CanWrite      bool
	CSRF          string
}

type formData struct{ ProcessoID, ProcessoNum, PessoaID, PessoaNome string }

func base(s *auth.Session, title string) auth.PageData {
	return auth.PageData{Title: title, Email: s.Email, Role: string(s.Role)}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	qs := r.URL.Query()
	f := Filtro{Q: qs.Get("q"), ProcessoID: qs.Get("processo"), PessoaID: qs.Get("pessoa")}
	pag, limit, off := webx.Page(r)
	rows, err := h.S.List(r.Context(), f, limit, off)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	d := listData{F: f, Pag: pag, CanWrite: auth.Can(s.Role, auth.PermDocWrite), CSRF: h.A.CSRFFor(r)}
	if len(rows) > webx.PageSize {
		rows, d.Proxima = rows[:webx.PageSize], pag+1
	}
	if pag > 1 {
		d.Anterior = pag - 1
	}
	d.Docs = rows
	t := page("lista.html")
	if webx.IsHX(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = t.ExecuteTemplate(w, "resultado", d)
		return
	}
	pd := base(s, "Documentos")
	pd.Head = auth.Head{Title: "Documentos", Sub: "Arquivos cifrados no servidor; cada leitura e cada envio ficam na auditoria."}
	if d.CanWrite {
		pd.Head.ActionLabel, pd.Head.ActionHref = "Enviar documento", "/documentos/novo"
		if d.F.ProcessoID != "" {
			pd.Head.ActionHref += "?processo=" + url.QueryEscape(d.F.ProcessoID)
		} else if d.F.PessoaID != "" {
			pd.Head.ActionHref += "?pessoa=" + url.QueryEscape(d.F.PessoaID)
		}
	}
	pd.Data = d
	h.A.Render(w, r, http.StatusOK, t, pd)
}

func (h *Handler) renderForm(w http.ResponseWriter, r *http.Request, s *auth.Session, status int, f formData, msg string) {
	if f.ProcessoID != "" {
		if m, err := h.S.Processos.Resumos(r.Context(), []string{f.ProcessoID}); err == nil {
			f.ProcessoNum = m[f.ProcessoID]
		}
	}
	if f.PessoaID != "" {
		if m, err := h.S.Pessoas.Names(r.Context(), []string{f.PessoaID}); err == nil {
			f.PessoaNome = m[f.PessoaID]
		}
	}
	pd := base(s, "Enviar documento")
	pd.Data, pd.Error = f, msg
	h.A.Render(w, r, status, page("form.html"), pd)
}

func (h *Handler) newForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	h.renderForm(w, r, s, http.StatusOK, formData{ProcessoID: r.URL.Query().Get("processo"), PessoaID: r.URL.Query().Get("pessoa")}, "")
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	f := formData{ProcessoID: r.PostFormValue("processo_id"), PessoaID: r.PostFormValue("pessoa_id")}
	file, hdr, err := r.FormFile("arquivo")
	if err != nil {
		h.renderForm(w, r, s, http.StatusBadRequest, f, "Escolha um arquivo.")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxArquivo+1))
	if err != nil {
		http.Error(w, "Erro ao ler o arquivo.", http.StatusBadRequest)
		return
	}
	id, err := h.S.Enviar(r.Context(), webx.ActorOf(w, s.UserID), Upload{Nome: hdr.Filename, Data: data,
		Descricao: r.PostFormValue("descricao"), ProcessoID: f.ProcessoID, PessoaID: f.PessoaID})
	var inv webx.Invalid
	switch {
	case errors.As(err, &inv):
		h.renderForm(w, r, s, http.StatusBadRequest, f, inv.Msg)
	case err != nil:
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	default:
		_ = id
		to := "/documentos"
		if f.ProcessoID != "" {
			to += "?processo=" + f.ProcessoID
		} else {
			to += "?pessoa=" + f.PessoaID
		}
		http.Redirect(w, r, to, http.StatusSeeOther)
	}
}

// download entrega o arquivo SEMPRE como anexo, com o tipo detectado pelo conteúdo e sem execução no navegador.
func (h *Handler) download(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	d, data, err := h.S.Baixar(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	hd := w.Header()
	hd.Set("Content-Type", MimeOf(d.Tipo))
	hd.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": d.Nome}))
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Content-Security-Policy", "sandbox; default-src 'none'")
	hd.Set("Content-Length", fmt.Sprint(len(data)))
	// #nosec G705 -- falso positivo: o arquivo é cifrado em repouso e sai como anexo (attachment) com nosniff e CSP "sandbox; default-src 'none'"; nunca é interpretado como HTML.
	_, _ = w.Write(data)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	if err := h.S.Excluir(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version"))); err != nil {
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/documentos", http.StatusSeeOther)
}

func opts(w http.ResponseWriter, list []Opcao, vazio string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = template.Must(template.New("o").Parse(vazio+`{{range .}}<option value="{{.Valor}}">{{.Rotulo}}</option>{{end}}`)).Execute(w, list)
}

func (h *Handler) processoOptions(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	l, err := h.S.Processos.Buscar(r.Context(), r.URL.Query().Get("busca"), 20)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	opts(w, l, `<option value="">— sem processo —</option>`)
}

func (h *Handler) pessoaOptions(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	l, err := h.S.Pessoas.Buscar(r.Context(), r.URL.Query().Get("busca"), 20)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	opts(w, l, `<option value="">— sem pessoa —</option>`)
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, webx.ErrNotFound):
		http.Error(w, "Registro não encontrado.", http.StatusNotFound)
	case errors.Is(err, webx.ErrConflict):
		http.Error(w, "Este documento foi alterado por outra pessoa. Recarregue a página.", http.StatusConflict)
	default:
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}
