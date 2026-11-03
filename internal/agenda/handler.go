package agenda

import (
	"bytes"
	"context"
	"embed"
	"errors"
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

// Register declara as rotas, cada uma com a sua permissão, e o bloco da tela "Hoje".
func (h *Handler) Register(rt *auth.Router) {
	h.A.AddNav("Agenda", "/agenda", auth.PermAgendaRead)
	h.A.AddWidget(h.widget)
	rt.Handle("GET /agenda", auth.PermAgendaRead, h.list)
	rt.Handle("GET /agenda/novo", auth.PermAgendaWrite, h.newForm)
	rt.Handle("POST /agenda", auth.PermAgendaWrite, h.create)
	rt.Handle("GET /agenda/processos", auth.PermAgendaWrite, h.processoOptions)
	rt.Handle("GET /agenda/{id}", auth.PermAgendaRead, h.detail)
	rt.Handle("GET /agenda/{id}/editar", auth.PermAgendaWrite, h.editForm)
	rt.Handle("POST /agenda/{id}", auth.PermAgendaWrite, h.update)
	rt.Handle("POST /agenda/{id}/excluir", auth.PermAgendaWrite, h.remove)
	rt.Handle("GET /agenda/{id}/pessoas", auth.PermAgendaWrite, h.pessoaOptions)
	rt.Handle("POST /agenda/{id}/participantes", auth.PermAgendaWrite, h.addParticipante)
	rt.Handle("POST /agenda/{id}/participantes/remover", auth.PermAgendaWrite, h.removeParticipante)
}

type listData struct {
	Tipo, Periodo string
	Eventos       []Evento
	Tipos         []Opcao
	Pag, Anterior int
	Proxima       int
	CanWrite      bool
}

type formData struct {
	IsNew        bool
	ID           string
	Version      int
	In           Input
	Tipos        []Opcao
	Responsaveis []Opcao
	ProcessoNum  string
}

func base(s *auth.Session, title string) auth.PageData {
	return auth.PageData{Title: title, Email: s.Email, Role: string(s.Role)}
}

func hojeSP() time.Time {
	n := time.Now().In(webx.SP())
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, webx.SP()).UTC()
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	qs := r.URL.Query()
	f := Filtro{Tipo: qs.Get("tipo")}
	per := qs.Get("periodo")
	switch per {
	case "passados":
		f.Ate = hojeSP()
	case "todos":
	default:
		per, f.De = "proximos", hojeSP()
	}
	pag, limit, off := webx.Page(r)
	rows, err := h.S.List(r.Context(), f, limit, off)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	d := listData{Tipo: f.Tipo, Periodo: per, Pag: pag, Tipos: Tipos, CanWrite: auth.Can(s.Role, auth.PermAgendaWrite)}
	if len(rows) > webx.PageSize {
		rows, d.Proxima = rows[:webx.PageSize], pag+1
	}
	if pag > 1 {
		d.Anterior = pag - 1
	}
	d.Eventos = rows
	t := page("lista.html")
	if webx.IsHX(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = t.ExecuteTemplate(w, "resultado", d)
		return
	}
	pd := base(s, "Agenda")
	pd.Data = d
	h.A.Render(w, r, http.StatusOK, t, pd)
}

func (h *Handler) renderForm(w http.ResponseWriter, r *http.Request, s *auth.Session, status int, f formData, msg string) {
	f.Tipos = Tipos
	f.Responsaveis, _ = h.S.Users.Options(r.Context())
	if f.In.ProcessoID != "" {
		if m, err := h.S.Processos.Resumos(r.Context(), []string{f.In.ProcessoID}); err == nil {
			f.ProcessoNum = m[f.In.ProcessoID]
		}
	}
	pd := base(s, "Evento")
	pd.Data, pd.Error = f, msg
	h.A.Render(w, r, status, page("form.html"), pd)
}

func inputFrom(r *http.Request) Input {
	return Input{Tipo: r.PostFormValue("tipo"), Titulo: r.PostFormValue("titulo"), Inicio: r.PostFormValue("inicio"),
		Fim: r.PostFormValue("fim"), Local: r.PostFormValue("local"), ProcessoID: r.PostFormValue("processo_id"),
		ResponsavelID: r.PostFormValue("responsavel_id")}
}

func (h *Handler) newForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	h.renderForm(w, r, s, http.StatusOK, formData{IsNew: true,
		In: Input{Tipo: "audiencia", ResponsavelID: s.UserID, ProcessoID: r.URL.Query().Get("processo")}}, "")
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
		http.Redirect(w, r, "/agenda/"+id+"?aviso=criado", http.StatusSeeOther)
	}
}

func (h *Handler) detail(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	e, err := h.S.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	pd := base(s, e.Titulo)
	pd.Data = map[string]any{"E": e, "CanWrite": auth.Can(s.Role, auth.PermAgendaWrite),
		"Aviso": r.URL.Query().Get("aviso") == "criado", "Erro": r.URL.Query().Get("erro") != "", "Conflito": h.conflitoDe(r.Context(), e)}
	h.A.Render(w, r, http.StatusOK, page("detalhe.html"), pd)
}

func (h *Handler) editForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	e, err := h.S.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	in := Input{Tipo: e.Tipo, Titulo: e.Titulo, Inicio: webx.FormatLocal(e.Inicio), Local: e.Local,
		ProcessoID: e.ProcessoID, ResponsavelID: e.ResponsavelID}
	if e.Fim != nil {
		in.Fim = webx.FormatLocal(*e.Fim)
	}
	h.renderForm(w, r, s, http.StatusOK, formData{ID: e.ID, Version: e.Version, In: in}, "")
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
		http.Redirect(w, r, "/agenda/"+id, http.StatusSeeOther)
	}
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	if err := h.S.Delete(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version"))); err != nil {
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/agenda", http.StatusSeeOther)
}

func optionsTpl() *template.Template {
	return template.Must(template.New("o").Parse(`{{range .}}<option value="{{.Valor}}">{{.Rotulo}}</option>{{else}}<option value="">Nenhum resultado</option>{{end}}`))
}

func (h *Handler) pessoaOptions(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	opts, err := h.S.Pessoas.Buscar(r.Context(), r.URL.Query().Get("busca"), 20)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = optionsTpl().Execute(w, opts)
}

func (h *Handler) processoOptions(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	opts, err := h.S.Processos.Buscar(r.Context(), r.URL.Query().Get("busca"), 20)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = template.Must(template.New("o").Parse(`<option value="">— sem processo —</option>{{range .}}<option value="{{.Valor}}">{{.Rotulo}}</option>{{end}}`)).Execute(w, opts)
}

func (h *Handler) addParticipante(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	id := r.PathValue("id")
	err := h.S.AddParticipante(r.Context(), webx.ActorOf(w, s.UserID), id, r.PostFormValue("pessoa_id"))
	var inv webx.Invalid
	switch {
	case errors.As(err, &inv):
		http.Redirect(w, r, "/agenda/"+id+"?erro=1", http.StatusSeeOther)
	case err != nil:
		h.fail(w, err)
	default:
		http.Redirect(w, r, "/agenda/"+id, http.StatusSeeOther)
	}
}

func (h *Handler) removeParticipante(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	id := r.PathValue("id")
	if err := h.S.RemoveParticipante(r.Context(), webx.ActorOf(w, s.UserID), id, r.PostFormValue("pessoa_id")); err != nil {
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/agenda/"+id, http.StatusSeeOther)
}

// widget alimenta a tela "Hoje" com os eventos dos próximos 7 dias.
func (h *Handler) widget(ctx context.Context, _ *auth.Session) (auth.Widget, error) {
	list, err := h.S.Proximos(ctx, 7, 10)
	if err != nil || len(list) == 0 {
		return auth.Widget{}, err
	}
	var b bytes.Buffer
	t := template.Must(template.New("w").Parse(`<ul class="lista">{{range .}}<li><span><strong>{{.TipoRotulo}}</strong> · <a href="/agenda/{{.ID}}">{{.Titulo}}</a>
{{if .ProcessoNumero}}<small class="apoio">· {{.ProcessoNumero}}</small>{{end}}</span><small>{{.InicioFmt}}</small></li>{{end}}</ul>
<p><a href="/agenda">Abrir a agenda</a></p>`))
	if err := t.Execute(&b, list); err != nil {
		return auth.Widget{}, err
	}
	return auth.Widget{Title: "Agenda dos próximos 7 dias", Body: template.HTML(b.String())}, nil // #nosec G203 -- saída já escapada pelo html/template
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, webx.ErrNotFound):
		http.Error(w, "Registro não encontrado.", http.StatusNotFound)
	case errors.Is(err, webx.ErrConflict):
		http.Error(w, "Este evento foi alterado por outra pessoa. Volte, recarregue a página e refaça a alteração.", http.StatusConflict)
	default:
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

// conflitoDe devolve o aviso de horário ocupado de uma audiência (que nunca é bloqueada); vazio nos demais tipos.
func (h *Handler) conflitoDe(ctx context.Context, e *Evento) string {
	if e.Tipo != "audiencia" {
		return ""
	}
	com, err := h.S.ExisteNoInstante(ctx, e.Inicio, e.ResponsavelID, e.ID)
	if err != nil || com == "" {
		return ""
	}
	return "Atenção: já existe " + com + " para este responsável no mesmo horário."
}
