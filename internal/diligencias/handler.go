package diligencias

import (
	"embed"
	"html/template"
	"net/http"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/webx"
)

//go:embed templates/*.html
var tplFS embed.FS

// Handler liga as telas ao serviço. A diligência nasce no "Novo compromisso" da Agenda; aqui só há a edição.
type Handler struct {
	A *auth.Auth
	S *Service
}

func fragmento(names ...string) *template.Template {
	for i := range names {
		names[i] = "templates/" + names[i]
	}
	return template.Must(template.ParseFS(tplFS, names...))
}

// Register declara as rotas, cada uma com a sua permissão.
func (h *Handler) Register(rt *auth.Router) {
	rt.Handle("GET /diligencias/{id}/editar", auth.PermDiligenciaWrite, h.editarForm)
	rt.Handle("POST /diligencias/{id}", auth.PermDiligenciaWrite, h.atualizar)
	rt.Handle("GET /diligencias/{id}/cancelar", auth.PermDiligenciaCancel, h.cancelarForm)
	rt.Handle("POST /diligencias/{id}/cancelar", auth.PermDiligenciaCancel, h.cancelar)
}

type formView struct {
	D       Diligencia
	Pessoas []Opcao
}

func (h *Handler) editarForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	d, err := h.S.Get(r.Context(), r.PathValue("id"))
	if webx.Falha(w, err, "Diligência não encontrada.") {
		return
	}
	if d.Status == StatusCancelada {
		http.Error(w, "Diligência cancelada não pode ser editada.", http.StatusBadRequest)
		return
	}
	pessoas, err := h.S.Pessoas.Opcoes(r.Context(), webx.ActorOf(w, s.UserID), auth.Can(s.Role, auth.PermPessoasWrite))
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("editar.html").ExecuteTemplate(w, "editar", formView{D: *d, Pessoas: pessoas}); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

func (h *Handler) atualizar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	v := r.PostFormValue
	in := Input{Titulo: v("titulo"), Processo: v("processo"), ClienteID: v("cliente"), Local: v("local"), DataLimite: v("data_limite"),
		Hora: v("hora"), Responsavel: v("responsavel")}
	err := h.S.Atualizar(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(v("version")), in)
	if webx.Falha(w, err, "Diligência não encontrada.", "Falha ao atualizar a diligência.") {
		return
	}
	webx.Ok(w, "Diligência atualizada.")
}

func (h *Handler) cancelarForm(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	d, err := h.S.Get(r.Context(), r.PathValue("id"))
	if webx.Falha(w, err, "Diligência não encontrada.") {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("cancelar.html").ExecuteTemplate(w, "cancelar", d); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

func (h *Handler) cancelar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.Cancelar(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version")), r.PostFormValue("motivo"))
	if webx.Falha(w, err, "Diligência não encontrada.", "Falha ao cancelar a diligência.") {
		return
	}
	webx.Ok(w, "Diligência cancelada; o registro foi preservado.")
}
