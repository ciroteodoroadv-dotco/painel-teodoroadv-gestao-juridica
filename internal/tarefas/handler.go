package tarefas

import (
	"bytes"
	"context"
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

// Register declara as rotas, cada uma com a sua permissão, e o bloco da tela "Hoje".
func (h *Handler) Register(rt *auth.Router) {
	h.A.AddNav("Tarefas", "/tarefas", auth.PermTarefaRead)
	h.A.AddWidget(h.widget)
	rt.Handle("GET /tarefas", auth.PermTarefaRead, h.list)
	rt.Handle("GET /tarefas/nova", auth.PermTarefaWrite, h.newForm)
	rt.Handle("POST /tarefas", auth.PermTarefaWrite, h.create)
	rt.Handle("GET /tarefas/{id}/editar", auth.PermTarefaWrite, h.editForm)
	rt.Handle("POST /tarefas/{id}", auth.PermTarefaWrite, h.update)
	rt.Handle("POST /tarefas/{id}/status", auth.PermTarefaWrite, h.status)
	rt.Handle("POST /tarefas/{id}/excluir", auth.PermTarefaWrite, h.remove)
	rt.Handle("GET /tarefas/processos", auth.PermTarefaWrite, h.processoOptions)
}

type listData struct {
	F             Filtro
	Tarefas       []Tarefa
	Statuses      []Opcao
	Pag, Anterior int
	Proxima       int
	CanWrite      bool
	SoMinhas      bool
	CSRF          string
}

type formData struct {
	IsNew        bool
	ID           string
	Version      int
	In           Input
	Statuses     []Opcao
	Prioridades  []Opcao
	Responsaveis []Opcao
	ProcessoNum  string
}

func base(s *auth.Session, title string) auth.PageData {
	return auth.PageData{Title: title, Email: s.Email, Role: string(s.Role)}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	qs := r.URL.Query()
	f := Filtro{Q: qs.Get("q"), Status: qs.Get("status")}
	minhas := qs.Get("quem") == "minhas"
	if minhas {
		f.Responsavel = s.UserID
	}
	pag, limit, off := webx.Page(r)
	rows, err := h.S.List(r.Context(), f, limit, off)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	d := listData{F: f, Pag: pag, Statuses: Statuses, SoMinhas: minhas, CanWrite: auth.Can(s.Role, auth.PermTarefaWrite)}
	if len(rows) > webx.PageSize {
		rows, d.Proxima = rows[:webx.PageSize], pag+1
	}
	if pag > 1 {
		d.Anterior = pag - 1
	}
	d.Tarefas, d.CSRF = rows, h.A.CSRFFor(r)
	t := page("lista.html")
	if webx.IsHX(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = t.ExecuteTemplate(w, "resultado", d)
		return
	}
	pd := base(s, "Tarefas")
	pd.Data = d
	h.A.Render(w, r, http.StatusOK, t, pd)
}

func (h *Handler) renderForm(w http.ResponseWriter, r *http.Request, s *auth.Session, status int, f formData, msg string) {
	f.Statuses, f.Prioridades = Statuses, Prioridades
	f.Responsaveis, _ = h.S.Users.Options(r.Context())
	if f.In.ProcessoID != "" {
		if m, err := h.S.Processos.Resumos(r.Context(), []string{f.In.ProcessoID}); err == nil {
			f.ProcessoNum = m[f.In.ProcessoID]
		}
	}
	pd := base(s, "Tarefa")
	pd.Data, pd.Error = f, msg
	if webx.IsHX(r) { // drawer: só o fragmento; erro de validação volta com 200 para o htmx trocar o conteúdo
		pd.CSRF = h.A.CSRFFor(r)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page("drawer.html").ExecuteTemplate(w, "drawer", pd)
		return
	}
	h.A.Render(w, r, status, page("form.html"), pd)
}

func inputFrom(r *http.Request) Input {
	return Input{Titulo: r.PostFormValue("titulo"), ProcessoID: r.PostFormValue("processo_id"),
		ResponsavelID: r.PostFormValue("responsavel_id"), Vencimento: r.PostFormValue("vencimento"), Status: r.PostFormValue("status"),
		Subtitulo: r.PostFormValue("subtitulo"), Prioridade: r.PostFormValue("prioridade")}
}

func (h *Handler) newForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	in := Input{Status: "aberta", Prioridade: "normal", ResponsavelID: s.UserID, ProcessoID: r.URL.Query().Get("processo")}
	h.renderForm(w, r, s, http.StatusOK, formData{IsNew: true, In: in}, "")
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	in := inputFrom(r)
	_, err := h.S.Create(r.Context(), webx.ActorOf(w, s.UserID), in)
	var inv webx.Invalid
	switch {
	case errors.As(err, &inv):
		h.renderForm(w, r, s, statusErro(r), formData{IsNew: true, In: in}, inv.Msg)
	case err != nil:
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	default:
		h.saved(w, r, "/tarefas?quem=minhas")
	}
}

func (h *Handler) editForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	t, err := h.S.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	in := Input{Titulo: t.Titulo, ProcessoID: t.ProcessoID, ResponsavelID: t.ResponsavelID, Status: t.Status,
		Subtitulo: t.Subtitulo, Prioridade: t.Prioridade}
	if t.Vencimento != nil {
		in.Vencimento = t.Vencimento.Format("2006-01-02")
	}
	h.renderForm(w, r, s, http.StatusOK, formData{ID: t.ID, Version: t.Version, In: in}, "")
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	id, in := r.PathValue("id"), inputFrom(r)
	v := webx.Int(r.PostFormValue("version"))
	err := h.S.Update(r.Context(), webx.ActorOf(w, s.UserID), id, v, in)
	var inv webx.Invalid
	switch {
	case errors.As(err, &inv):
		h.renderForm(w, r, s, statusErro(r), formData{ID: id, Version: v, In: in}, inv.Msg)
	case err != nil:
		h.fail(w, err)
	default:
		h.saved(w, r, "/tarefas")
	}
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.SetStatus(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"),
		webx.Int(r.PostFormValue("version")), r.PostFormValue("status"))
	var inv webx.Invalid
	if errors.As(err, &inv) {
		http.Error(w, inv.Msg, http.StatusBadRequest)
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/tarefas", http.StatusSeeOther)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	if err := h.S.Delete(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version"))); err != nil {
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/tarefas", http.StatusSeeOther)
}

func (h *Handler) processoOptions(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	opts, err := h.S.Processos.Buscar(r.Context(), r.URL.Query().Get("busca"), 20)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.Must(template.New("o").Parse(`<option value="">— sem processo —</option>{{range .}}<option value="{{.Valor}}">{{.Rotulo}}</option>{{end}}`))
	_ = t.Execute(w, opts)
}

// widget alimenta a tela "Hoje" com as tarefas do usuário vencidas ou vencendo em 7 dias.
func (h *Handler) widget(ctx context.Context, s *auth.Session) (auth.Widget, error) {
	list, err := h.S.Proximas(ctx, s.UserID, 7, 10)
	if err != nil || len(list) == 0 {
		return auth.Widget{}, err
	}
	var b bytes.Buffer
	t := template.Must(template.New("w").Parse(`<ul class="lista">{{range .}}<li><span>{{if .Atrasada}}<strong class="erro">Atrasada</strong> · {{end}}{{.Titulo}}
{{if .ProcessoNumero}}<small class="apoio">· {{.ProcessoNumero}}</small>{{end}}</span>
<small>{{if .Vencimento}}{{.Vencimento.Format "02/01/2006"}}{{end}}</small></li>{{end}}</ul>
<p><a href="/tarefas?quem=minhas">Ver todas as minhas tarefas</a></p>`))
	if err := t.Execute(&b, list); err != nil {
		return auth.Widget{}, err
	}
	return auth.Widget{Title: "Minhas tarefas vencidas ou a vencer (7 dias)", Body: template.HTML(b.String())}, nil // #nosec: saída do html/template
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, webx.ErrNotFound):
		http.Error(w, "Registro não encontrado.", http.StatusNotFound)
	case errors.Is(err, webx.ErrConflict):
		http.Error(w, "Esta tarefa foi alterada por outra pessoa. Volte, recarregue a página e refaça a alteração.", http.StatusConflict)
	default:
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

// statusErro: o htmx só troca o conteúdo em resposta 2xx; fora dele, erro de validação segue 400.
func statusErro(r *http.Request) int {
	if webx.IsHX(r) {
		return http.StatusOK
	}
	return http.StatusBadRequest
}

// saved encerra um POST bem-sucedido: no drawer manda o navegador para a lista com aviso "Salvo.".
func (h *Handler) saved(w http.ResponseWriter, r *http.Request, normal string) {
	if webx.IsHX(r) {
		w.Header().Set("HX-Redirect", "/tarefas?salvo=1")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, normal, http.StatusSeeOther)
}
