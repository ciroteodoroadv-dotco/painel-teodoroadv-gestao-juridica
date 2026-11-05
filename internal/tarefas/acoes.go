package tarefas

import (
	"net/http"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/webx"
)

// mover é o "Mover para..." da linha: troca só o estado.
func (h *Handler) mover(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.Mover(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version")),
		r.PostFormValue("estado"), auth.Can(s.Role, auth.PermPrazoConfirm))
	if webx.Falha(w, err, "Tarefa não encontrada.", "Falha ao mover o estado da tarefa.") {
		return
	}
	webx.Ok(w, "Estado da tarefa atualizado.")
}

// confirmarPrazo é o botão "Confirmar prazo" (só quem tem PermPrazoConfirm): usa o fluxo de confirmação do domínio Prazos.
func (h *Handler) confirmarPrazo(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.ConfirmarPrazo(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"))
	if webx.Falha(w, err, "Tarefa não encontrada.", "Falha ao confirmar o prazo.") {
		return
	}
	webx.Ok(w, "Prazo confirmado.")
}

// documento recebe o PDF do campo "Documentos do caso" (envio imediato ao escolher o arquivo). O limite de 25 MiB
// vem da rota (auth.Router.HandleUpload).
func (h *Handler) documento(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	f, hdr, err := r.FormFile("documento")
	if err != nil {
		http.Error(w, "Escolha um arquivo.", http.StatusBadRequest)
		return
	}
	defer f.Close()
	err = h.S.AnexarDocumento(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), hdr.Filename, f)
	if webx.Falha(w, err, "Tarefa não encontrada.", "Erro ao guardar o documento.") {
		return
	}
	webx.Ok(w, "Documento guardado.")
}
