package intimacoes

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/coletas"
	"painelteodoro/internal/platform/webx"
)

//go:embed templates/*.html
var tplFS embed.FS

// Handler liga as telas ao serviço.
type Handler struct {
	A *auth.Auth
	S *Service
	// Extra devolve seções adicionais da página Coleta DJEN (o monitor do DataJud e os processos em segredo de justiça).
	Extra func(r *http.Request) (template.HTML, error)
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
	rt.Handle("GET /intimacoes/{id}/detalhe", auth.PermAgendaRead, h.detalhe)
	rt.Handle("POST /intimacoes/{id}/decidir", auth.PermIntimacaoDecide, h.decidir)
	rt.Handle("GET /intimacoes/{id}/cancelar", auth.PermIntimacaoDecide, h.cancelarForm)
	rt.Handle("POST /intimacoes/{id}/cancelar", auth.PermIntimacaoDecide, h.cancelar)
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
	msgCancelada  = "Intimação cancelada e mantida no histórico."
	msgFalhaDecid = "Falha ao atualizar a intimação."
)

func (h *Handler) decidir(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.Decidir(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), r.PostFormValue("status"))
	if webx.Falha(w, err, "Intimação não encontrada.", msgFalhaDecid) {
		return
	}
	webx.Ok(w, msgConfirmada)
}

type cancelarView struct{ ID string }

func (h *Handler) cancelarForm(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	id := r.PathValue("id")
	if !webx.UUID(id) {
		http.Error(w, "Intimação não encontrada.", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("cancelar.html").ExecuteTemplate(w, "cancelar", cancelarView{ID: id}); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

func (h *Handler) cancelar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.Cancelar(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), r.PostFormValue("motivo"))
	if webx.Falha(w, err, "Intimação não encontrada.", msgFalhaDecid) {
		return
	}
	webx.Ok(w, msgCancelada)
}

// executar roda a coleta a pedido do administrador. O resultado nunca é apresentado como garantia de "nada novo".
// EsperaNaTela é quanto o pedido "executar agora" espera pela rodada antes de responder "em andamento".
// Fica abaixo do WriteTimeout do servidor (60 s) e do limite do Cloudflare (~100 s).
var EsperaNaTela = 20 * time.Second

func (h *Handler) executar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	ac := webx.ActorOf(w, s.UserID)
	res, pronto, err := coletas.EmSegundoPlano(r.Context(), EsperaNaTela, func(ctx context.Context) (Resumo, error) { return h.S.ColetarAgora(ctx, ac) })
	if err == nil && !pronto {
		webx.Ok(w, "Coleta iniciada e ainda em andamento: o resultado aparece no estado das coletas. Recarregue a página em alguns minutos.")
		return
	}
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
