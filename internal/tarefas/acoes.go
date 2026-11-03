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

// documento recebe o PDF do campo "Documentos do caso" (envio imediato ao escolher o arquivo).
func (h *Handler) documento(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
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

type excluirView struct {
	T Tarefa
}

func (h *Handler) excluirForm(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	t, err := h.S.Get(r.Context(), r.PathValue("id"))
	if webx.Falha(w, err, "Tarefa não encontrada.") {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("excluir.html").ExecuteTemplate(w, "excluir", excluirView{T: *t}); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

// excluir reautentica a senha do usuário logado, exige PermTarefaDelete (já na rota) e faz a exclusão lógica.
func (h *Handler) excluir(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.Excluir(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version")), r.PostFormValue("senha"))
	if err != nil {
		if webx.Falha(w, prefixo(err), "Tarefa não encontrada.", "Não foi possível excluir a tarefa.") {
			return
		}
	}
	webx.Ok(w, "Tarefa excluída.")
}

// prefixo acrescenta o texto do B ("Não foi possível excluir a tarefa: ") às recusas de validação.
func prefixo(err error) error {
	if inv, ok := err.(webx.Invalid); ok {
		return webx.Invalidf("Não foi possível excluir a tarefa: " + inv.Msg)
	}
	return err
}

// maxUpload é o limite do corpo de um envio de PDF (25 MiB do domínio Documentos, mais o cabeçalho do multipart).
const maxUpload = 26 << 20
