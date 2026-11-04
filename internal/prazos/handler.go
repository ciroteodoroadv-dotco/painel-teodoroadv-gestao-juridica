package prazos

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
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
	// Calculadora (Plano 7).
	Cal      Calendarios
	Validada bool             // PAINEL_CALCULADORA_REGRAS_VALIDADAS: só o advogado liga
	Agora    func() time.Time // relógio (teste); nil = time.Now
}

func fragmento(name string) *template.Template {
	return template.Must(template.ParseFS(tplFS, "templates/"+name))
}

func page(name string) *template.Template {
	return template.Must(auth.BaseTemplate().ParseFS(tplFS, "templates/"+name))
}

// Register declara as rotas e o bloco de prazos críticos da tela "Hoje".
func (h *Handler) Register(rt *auth.Router) {
	h.A.AddNavSistema("Prazos", "/prazos", auth.PermPrazoRead)
	h.A.AddWidget(h.widget)
	h.A.AddFaixa(h.faixa)
	if h.Cal != nil {
		h.A.AddNavAcao("Calculadora de prazo", "/prazos/calculadora", auth.PermPrazoRead)
		rt.Handle("GET /prazos/calculadora", auth.PermPrazoRead, h.calculadoraDrawer)
		rt.Handle("POST /prazos/calcular", auth.PermPrazoRead, h.calcular)
	}
	rt.Handle("GET /prazos", auth.PermPrazoRead, h.list)
	rt.Handle("GET /prazos/novo", auth.PermPrazoWrite, h.newForm)
	rt.Handle("POST /prazos", auth.PermPrazoWrite, h.create)
	rt.Handle("GET /prazos/processos", auth.PermPrazoWrite, h.processoOptions)
	rt.Handle("GET /prazos/{id}/editar", auth.PermPrazoWrite, h.editForm)
	rt.Handle("POST /prazos/{id}", auth.PermPrazoWrite, h.update)
	rt.Handle("POST /prazos/{id}/confirmar", auth.PermPrazoConfirm, h.acao("confirmar"))
	rt.Handle("POST /prazos/{id}/cumprir", auth.PermPrazoConfirm, h.acao("cumprir"))
	rt.Handle("POST /prazos/{id}/cancelar", auth.PermPrazoConfirm, h.acao("cancelar"))
	rt.Handle("POST /prazos/{id}/excluir", auth.PermPrazoConfirm, h.acao("excluir"))
}

type listData struct {
	F             Filtro
	Prazos        []Prazo
	Pag, Anterior int
	Proxima       int
	CanWrite      bool
	CanConfirm    bool
	CSRF          string
}

type formData struct {
	IsNew        bool
	ID           string
	Version      int
	In           Input
	Fontes       []Opcao
	Responsaveis []Opcao
	ProcessoNum  string
	Aviso        string
}

func base(s *auth.Session, title string) auth.PageData {
	return auth.PageData{Title: title, Email: s.Email, Role: string(s.Role)}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	qs := r.URL.Query()
	f := Filtro{Situacao: qs.Get("situacao"), Q: qs.Get("q"), ProcessoID: qs.Get("processo")}
	if qs.Get("quem") == "meus" {
		f.Responsavel = s.UserID
	}
	pag, limit, off := webx.Page(r)
	rows, err := h.S.List(r.Context(), f, limit, off)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	d := listData{F: f, Pag: pag, CanWrite: auth.Can(s.Role, auth.PermPrazoWrite),
		CanConfirm: auth.Can(s.Role, auth.PermPrazoConfirm), CSRF: h.A.CSRFFor(r)}
	if len(rows) > webx.PageSize {
		rows, d.Proxima = rows[:webx.PageSize], pag+1
	}
	if pag > 1 {
		d.Anterior = pag - 1
	}
	d.Prazos = rows
	t := page("lista.html")
	if webx.IsHX(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = t.ExecuteTemplate(w, "resultado", d)
		return
	}
	pd := base(s, "Prazos")
	pd.Head = auth.Head{Title: "Prazos", Sub: "Prazos informados pelo advogado; só valem depois de confirmados."}
	if d.CanWrite {
		pd.Head.ActionLabel, pd.Head.ActionHref = "Novo prazo", "/prazos/novo"
	}
	pd.Data = d
	h.A.Render(w, r, http.StatusOK, t, pd)
}

func (h *Handler) renderForm(w http.ResponseWriter, r *http.Request, s *auth.Session, status int, f formData, msg string) {
	f.Fontes = Fontes
	f.Responsaveis, _ = h.S.Users.Options(r.Context())
	if f.In.ProcessoID != "" {
		if m, err := h.S.Processos.Resumos(r.Context(), []string{f.In.ProcessoID}); err == nil {
			f.ProcessoNum = m[f.In.ProcessoID]
		}
	}
	if !auth.Can(s.Role, auth.PermPrazoConfirm) {
		f.Aviso = "Você pode registrar o prazo, mas ele só passa a valer quando um advogado confirmar."
	}
	pd := base(s, "Editar prazo")
	if f.IsNew {
		pd.Title = "Novo prazo"
	}
	pd.Data, pd.Error = f, msg
	h.A.Render(w, r, status, page("form.html"), pd)
}

func inputFrom(r *http.Request) Input {
	return Input{Descricao: r.PostFormValue("descricao"), ProcessoID: r.PostFormValue("processo_id"),
		DataFatal: r.PostFormValue("data_fatal"), Fundamento: r.PostFormValue("fundamento"), Fonte: r.PostFormValue("fonte"),
		ResponsavelID: r.PostFormValue("responsavel_id")}
}

func (h *Handler) newForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	h.renderForm(w, r, s, http.StatusOK, formData{IsNew: true,
		In: prefill(r, s.UserID)}, "")
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	in := inputFrom(r)
	_, err := h.S.Create(r.Context(), webx.ActorOf(w, s.UserID), in, auth.Can(s.Role, auth.PermPrazoConfirm))
	var inv webx.Invalid
	switch {
	case errors.As(err, &inv):
		h.renderForm(w, r, s, http.StatusBadRequest, formData{IsNew: true, In: in}, inv.Msg)
	case err != nil:
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	default:
		http.Redirect(w, r, "/prazos", http.StatusSeeOther)
	}
}

func (h *Handler) editForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	p, err := h.S.Get(r.Context(), r.PathValue("id"))
	if err != nil || !p.Aberto() {
		h.fail(w, errOr(err, webx.ErrConflict))
		return
	}
	in := Input{Descricao: p.Descricao, ProcessoID: p.ProcessoID, DataFatal: p.DataFatal.Format("2006-01-02"),
		Fundamento: p.Fundamento, Fonte: p.Fonte, ResponsavelID: p.ResponsavelID}
	h.renderForm(w, r, s, http.StatusOK, formData{ID: p.ID, Version: p.Version, In: in}, "")
}

func errOr(err, def error) error {
	if err != nil {
		return err
	}
	return def
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	id, in := r.PathValue("id"), inputFrom(r)
	v := webx.Int(r.PostFormValue("version"))
	err := h.S.Update(r.Context(), webx.ActorOf(w, s.UserID), id, v, in, auth.Can(s.Role, auth.PermPrazoConfirm))
	var inv webx.Invalid
	switch {
	case errors.As(err, &inv):
		h.renderForm(w, r, s, http.StatusBadRequest, formData{ID: id, Version: v, In: in}, inv.Msg)
	case err != nil:
		h.fail(w, err)
	default:
		http.Redirect(w, r, "/prazos", http.StatusSeeOther)
	}
}

// acao devolve o handler de uma transição de estado (confirmar, cumprir, cancelar, excluir): só advogado.
func (h *Handler) acao(nome string) auth.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, s *auth.Session) {
		ac, id, v := webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version"))
		var err error
		switch nome {
		case "confirmar":
			err = h.S.Confirmar(r.Context(), ac, id, v)
		case "cumprir":
			err = h.S.Cumprir(r.Context(), ac, id, v)
		case "cancelar":
			err = h.S.Cancelar(r.Context(), ac, id, v)
		case "excluir":
			err = h.S.Excluir(r.Context(), ac, id, v)
		}
		if err != nil {
			h.fail(w, err)
			return
		}
		http.Redirect(w, r, "/prazos", http.StatusSeeOther)
	}
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

// widget mostra, na Agenda inicial, os prazos vencidos ou a vencer em 5 dias e quantos aguardam conferência.
func (h *Handler) widget(ctx context.Context, _ *auth.Session) (auth.Widget, error) {
	list, aConferir, err := h.S.Criticos(ctx, 5, 15)
	if err != nil || (len(list) == 0 && aConferir == 0) {
		return auth.Widget{}, err
	}
	var b bytes.Buffer
	t := template.Must(template.New("w").Parse(`<ul class="lista">{{range .L}}<li class="nivel-{{.Nivel}}"><span><strong>{{.Rotulo}}</strong> {{.Descricao}}
{{if .ProcessoNumero}}<small class="apoio">· {{.ProcessoNumero}}</small>{{end}}{{if eq .Situacao "a_conferir"}} <span class="selo">a conferir</span>{{end}}</span>
<small>{{.DataFatal.Format "02/01/2006"}} · {{.Texto}}</small></li>{{end}}</ul>
<p><a href="/prazos">Abrir todos os prazos</a></p>`))
	if err := t.Execute(&b, map[string]any{"L": list, "A": aConferir}); err != nil {
		return auth.Widget{}, err
	}
	return auth.Widget{Title: "PRAZOS — vencidos ou a vencer em 5 dias", Body: template.HTML(b.String())}, nil // #nosec G203 -- saída do html/template
}

// faixa resume, na Agenda inicial, quantos prazos aguardam conferência e quantos vencem em até 5 dias.
// Só conta o que já está registrado: nenhum cálculo novo.
func (h *Handler) faixa(ctx context.Context, _ *auth.Session) (auth.Faixa, error) {
	list, aConferir, err := h.S.Criticos(ctx, 5, 15)
	if err != nil || (len(list) == 0 && aConferir == 0) {
		return auth.Faixa{}, err
	}
	return auth.Faixa{Strong: fmt.Sprintf("%d prazo(s) a conferir", aConferir),
		Text: fmt.Sprintf("· %d vencido(s) ou a vencer em até 5 dias —", len(list)), Href: "/prazos", Link: "ver Prazos"}, nil
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, webx.ErrNotFound):
		http.Error(w, "Registro não encontrado.", http.StatusNotFound)
	case errors.Is(err, webx.ErrConflict):
		http.Error(w, "Este prazo foi alterado por outra pessoa ou não está em situação que permita esta ação. Volte e recarregue a página.", http.StatusConflict)
	default:
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

// prefill preenche o formulário a partir do link "Criar prazo com este cálculo". Só preenche o formulário:
// o prazo só nasce se a pessoa salvar, e vindo da calculadora nasce A CONFERIR.
func prefill(r *http.Request, userID string) Input {
	q := r.URL.Query()
	in := Input{Fonte: "manual", ResponsavelID: userID, ProcessoID: q.Get("processo")}
	if q.Get("fonte") == "calculadora" {
		in.Fonte = "calculadora"
		if _, err := time.Parse("2006-01-02", q.Get("data_fatal")); err == nil {
			in.DataFatal = q.Get("data_fatal")
		}
		if f := []rune(q.Get("fundamento")); len(f) <= 300 {
			in.Fundamento = string(f)
		}
	}
	return in
}
