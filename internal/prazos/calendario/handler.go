package calendario

import (
	"embed"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/webx"
)

//go:embed templates/*.html
var tplFS embed.FS

// Handler liga a página "Calendário forense" ao serviço.
type Handler struct {
	A *auth.Auth
	S *Service
}

func fragmento(name string) *template.Template {
	return template.Must(template.ParseFS(tplFS, "templates/"+name))
}

// Register declara as rotas (página no grupo "sistema").
func (h *Handler) Register(rt *auth.Router) {
	h.A.AddNavSistema("Calendário forense", "/calendario-forense", auth.PermCalendarioRead)
	rt.Handle("GET /calendario-forense", auth.PermCalendarioRead, h.pagina)
	rt.Handle("GET /calendario-forense/dias/novo", auth.PermCalendarioWrite, h.diaForm)
	rt.Handle("POST /calendario-forense/dias", auth.PermCalendarioWrite, h.diaCriar)
	rt.Handle("GET /calendario-forense/novo", auth.PermCalendarioWrite, h.calForm)
	rt.Handle("POST /calendario-forense/calendarios", auth.PermCalendarioWrite, h.calCriar)
	rt.Handle("GET /calendario-forense/dias/{id}/inativar", auth.PermCalendarioWrite, h.inativarForm)
	rt.Handle("POST /calendario-forense/dias/{id}/inativar", auth.PermCalendarioWrite, h.inativar)
}

type paginaData struct {
	Calendarios []Calendario
	Sel         Calendario
	Ano         int
	Anos        []int
	Dias        []Dia
	CanWrite    bool
}

func (h *Handler) pagina(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	cals, err := h.S.Calendarios(r.Context())
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	d := paginaData{Calendarios: cals, CanWrite: auth.Can(s.Role, auth.PermCalendarioWrite)}
	hoje := webx.Hoje(time.Now())
	d.Ano = hoje.Year()
	if a, err := strconv.Atoi(r.URL.Query().Get("ano")); err == nil && a >= 2000 && a <= 2100 {
		d.Ano = a
	}
	for y := hoje.Year() - 1; y <= hoje.Year()+3; y++ {
		d.Anos = append(d.Anos, y)
	}
	want := r.URL.Query().Get("calendario")
	for _, c := range cals {
		if c.ID == want || (want == "" && d.Sel.ID == "") {
			d.Sel = c
		}
	}
	if d.Sel.ID != "" {
		if d.Dias, err = h.S.Dias(r.Context(), d.Sel.ID, d.Ano); err != nil {
			http.Error(w, "Erro interno.", http.StatusInternalServerError)
			return
		}
	}
	pd := auth.PageData{Title: "Calendário forense", Email: s.Email, Role: string(s.Role)}
	pd.Head = auth.Head{Title: "Calendário forense", Sub: "Feriados e suspensões com a fonte de cada data. Sem fonte, não entra no cálculo."}
	if d.CanWrite {
		pd.Head.ActionLabel, pd.Head.ActionHref, pd.Head.ActionDrawer = "Novo dia", "/calendario-forense/dias/novo", true
	}
	pd.Data = d
	h.A.Render(w, r, http.StatusOK, template.Must(auth.BaseTemplate().ParseFS(tplFS, "templates/pagina.html")), pd)
}

func (h *Handler) servir(w http.ResponseWriter, arq, nome string, dados any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento(arq).ExecuteTemplate(w, nome, dados); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

func (h *Handler) diaForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	cals, err := h.S.Calendarios(r.Context())
	if webx.Falha(w, err, "") {
		return
	}
	h.servir(w, "dia.html", "dia", map[string]any{"Calendarios": cals, "Tipos": Tipos, "Sel": r.URL.Query().Get("calendario"),
		"Advogado": auth.Can(s.Role, auth.PermPrazoConfirm)})
}

func (h *Handler) diaCriar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	v := r.PostFormValue
	in := DiaInput{CalendarioID: v("calendario_id"), Data: v("data"), Tipo: v("tipo"), Descricao: v("descricao"),
		FonteCitacao: v("fonte_citacao"), FonteURL: v("fonte_url"), Observacao: v("observacao"), ContaNaoUtil: v("conta_nao_util") == "1"}
	_, err := h.S.CriarDia(r.Context(), webx.ActorOf(w, s.UserID), in, auth.Can(s.Role, auth.PermPrazoConfirm))
	if webx.Falha(w, err, "Calendário não encontrado.", "Falha ao salvar o dia do calendário.") {
		return
	}
	webx.Ok(w, "Dia de calendário salvo.")
}

func (h *Handler) calForm(w http.ResponseWriter, _ *http.Request, _ *auth.Session) {
	h.servir(w, "calendario.html", "calendario", map[string]any{"Ambitos": Ambitos})
}

func (h *Handler) calCriar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	v := r.PostFormValue
	_, err := h.S.CriarCalendario(r.Context(), webx.ActorOf(w, s.UserID), CalInput{Sigla: v("sigla"), Nome: v("nome"), Ambito: v("ambito")})
	if webx.Falha(w, err, "", "Falha ao salvar o calendário.") {
		return
	}
	webx.Ok(w, "Calendário salvo.")
}

func (h *Handler) inativarForm(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	d, err := h.S.Dia(r.Context(), r.PathValue("id"))
	if webx.Falha(w, err, "Dia não encontrado.") {
		return
	}
	h.servir(w, "inativar.html", "inativar", d)
}

func (h *Handler) inativar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.InativarDia(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), r.PostFormValue("motivo"))
	if webx.Falha(w, err, "Dia não encontrado ou já inativado.", "Falha ao inativar o dia.") {
		return
	}
	webx.Ok(w, "Dia inativado. A linha permanece no calendário.")
}
