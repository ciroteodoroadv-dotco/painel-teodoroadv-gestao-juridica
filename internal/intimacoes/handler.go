package intimacoes

import (
	"embed"
	"fmt"
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

func fragmento(names ...string) *template.Template {
	for i := range names {
		names[i] = "templates/" + names[i]
	}
	return template.Must(template.ParseFS(tplFS, names...))
}

// Register declara as rotas, cada uma com a sua permissão. A tabela de Intimações em si mora na Agenda (ela monta a
// página); aqui ficam a decisão (só advogado) e a página "Coleta DJEN" (só administrador).
func (h *Handler) Register(rt *auth.Router) {
	h.A.AddNavSistema("Coleta DJEN", "/coleta-djen", auth.PermColetaManage)
	g := auth.PermColetaManage
	rt.Handle("POST /intimacoes/{id}/decidir", auth.PermIntimacaoDecide, h.decidir)
	rt.Handle("GET /coleta-djen", g, h.pagina)
	rt.Handle("GET /coleta-djen/nova", g, h.novaForm)
	rt.Handle("POST /coleta-djen", g, h.criar)
	rt.Handle("POST /coleta-djen/executar", g, h.executar)
	rt.Handle("GET /coleta-djen/{id}/editar", g, h.editarForm)
	rt.Handle("POST /coleta-djen/{id}", g, h.atualizar)
}

// Textos do B (App.atualizarStatusIntimacao).
const (
	msgConfirmada = "Intimação confirmada."
	msgDescartada = "Intimação descartada e mantida no histórico."
	msgFalhaDecid = "Falha ao atualizar a intimação."
)

func (h *Handler) decidir(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	status := r.PostFormValue("status")
	err := h.S.Decidir(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), status)
	if webx.Falha(w, err, "Intimação não encontrada.", msgFalhaDecid) {
		return
	}
	if status == "descartada" {
		webx.Ok(w, msgDescartada)
		return
	}
	webx.Ok(w, msgConfirmada)
}

// executar roda a coleta a pedido do administrador. O resultado nunca é apresentado como garantia de "nada novo".
func (h *Handler) executar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	res, err := h.S.ColetarAgora(r.Context(), webx.ActorOf(w, s.UserID))
	if webx.Falha(w, err, "Coleta não encontrada.", "Falha ao executar a coleta.") {
		return
	}
	switch res.Status {
	case "bloqueado", "falha":
		msg := res.Mensagem
		if res.Status == "falha" {
			msg = "A coleta falhou. " + res.Mensagem
		}
		webx.Falha(w, webx.Invalidf(msg), "")
	case "parcial":
		webx.Ok(w, fmt.Sprintf("Coleta parcial: %d comunicação(ões) nova(s). Veja o estado das coletas e confira o DJEN.", res.Novas))
	default:
		webx.Ok(w, fmt.Sprintf("Coleta concluída: %d comunicação(ões) nova(s). A conferência nas fontes oficiais continua necessária.", res.Novas))
	}
}
