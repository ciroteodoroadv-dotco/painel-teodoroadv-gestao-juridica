package monitor

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/coletas"
	"painelteodoro/internal/platform/webx"
)

//go:embed templates/*.html
var tplFS embed.FS

// Handler liga as telas do monitor ao serviço. As seções aparecem na página "Coleta DJEN" (só administrador).
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

// Register declara as rotas, todas só para o administrador (PermColetaManage).
func (h *Handler) Register(rt *auth.Router) {
	g := auth.PermColetaManage
	rt.Handle("POST /coleta-djen/monitor/executar", g, h.executar)
	rt.Handle("GET /coleta-djen/sigilosos/novo", g, h.novoSigiloso)
	rt.Handle("POST /coleta-djen/sigilosos", g, h.criarSigiloso)
	rt.Handle("POST /coleta-djen/sigilosos/{id}/desativar", g, h.desativarSigiloso)
}

type secaoView struct {
	Ativa, Admin      bool
	Estado, EstadoSig string
	Linhas            []LinhaProcesso
	Sigilosos         []LinhaSigiloso
}

func linhaEstado(e coletas.Estado, existe bool) string {
	if !existe {
		return "Nenhuma coleta registrada ainda."
	}
	t, _ := e.Linha()
	return t
}

// Secao devolve as duas seções (monitor e sigilosos) já renderizadas, para a página Coleta DJEN.
func (h *Handler) Secao(r *http.Request) (template.HTML, error) {
	ctx := r.Context()
	linhas, err := h.S.Linhas(ctx)
	if err != nil {
		return "", err
	}
	sig, err := h.S.LinhasSigilosos(ctx)
	if err != nil {
		return "", err
	}
	e1, ok1, err := h.S.Coletas.Obter(ctx, Modulo)
	if err != nil {
		return "", err
	}
	e2, ok2, err := h.S.Coletas.Obter(ctx, ModuloSigilosos)
	if err != nil {
		return "", err
	}
	v := secaoView{Ativa: h.S.Cfg.Ativo, Admin: true, Estado: linhaEstado(e1, ok1), EstadoSig: linhaEstado(e2, ok2), Linhas: linhas, Sigilosos: sig}
	var b bytes.Buffer
	if err := fragmento("secao.html").ExecuteTemplate(&b, "secao", v); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil // #nosec G203 -- saída de html/template, já escapada
}

func (h *Handler) executar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	res, err := h.S.ColetarAgora(r.Context(), webx.ActorOf(w, s.UserID))
	if webx.Falha(w, err, "Coleta não encontrada.", "Falha ao executar o monitor.") {
		return
	}
	switch res.Status {
	case "bloqueado", "falha":
		msg := res.Mensagem
		if res.Status == "falha" {
			msg = "O monitor falhou. " + res.Mensagem
		}
		webx.Falha(w, webx.Invalidf(msg), "")
	case "parcial":
		webx.Ok(w, fmt.Sprintf("Monitor parcial: %d movimentação(ões) nova(s). Veja o estado de cada processo e confira as fontes oficiais.", res.Novas))
	default:
		webx.Ok(w, fmt.Sprintf("Monitor concluído: %d movimentação(ões) nova(s). A conferência nas fontes oficiais continua necessária.", res.Novas))
	}
}

func (h *Handler) novoSigiloso(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	o, err := h.S.OpcoesSigiloso(r.Context())
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("sigiloso-drawer.html").ExecuteTemplate(w, "sigiloso", o); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

func (h *Handler) criarSigiloso(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.CadastrarSigiloso(r.Context(), webx.ActorOf(w, s.UserID), r.PostFormValue("processo_id"), r.PostFormValue("oab_id"))
	if webx.Falha(w, err, "Registro não encontrado.", "Falha ao cadastrar o processo em segredo de justiça.") {
		return
	}
	webx.Ok(w, "Processo em segredo de justiça cadastrado.")
}

func (h *Handler) desativarSigiloso(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.DesativarSigiloso(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"))
	if webx.Falha(w, err, "Registro não encontrado.", "Falha ao desativar.") {
		return
	}
	webx.Ok(w, "Processo em segredo de justiça desativado e mantido no histórico.")
}
