package processos

import (
	"embed"
	"errors"
	"html/template"
	"net/http"
	"strings"

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

// fragmento carrega um arquivo de templates sem a casca da página (miolo do drawer).
func fragmento(names ...string) *template.Template {
	for i := range names {
		names[i] = "templates/" + names[i]
	}
	return template.Must(template.ParseFS(tplFS, names...))
}

// Register declara as rotas, cada uma com a sua permissão.
func (h *Handler) Register(rt *auth.Router) {
	h.A.AddNavCount("Processos", "/processos", auth.PermProcRead, webx.ContarAtivos(h.S.Pool, "processos"))
	w := auth.PermProcWrite
	rt.Handle("GET /processos", auth.PermProcRead, h.list)
	rt.Handle("GET /processos/novo", w, h.newForm)
	rt.Handle("POST /processos", w, h.create)
	rt.Handle("GET /processos/{id}", auth.PermProcRead, h.detail)
	rt.Handle("GET /processos/{id}/editar", w, h.editForm)
	rt.Handle("POST /processos/{id}", w, h.update)
	rt.Handle("GET /processos/{id}/estado", w, h.estadoForm)
	rt.Handle("POST /processos/{id}/estado", w, h.estado)
	rt.Handle("GET /processos/{id}/andamento", w, h.andamentoForm)
	rt.Handle("POST /processos/{id}/andamento", w, h.andamento)
	rt.Handle("GET /processos/{id}/financeiro", w, h.financeiroForm)
	rt.Handle("POST /processos/{id}/financeiro", w, h.financeiro)
	rt.Handle("GET /processos/{id}/providencia", w, h.providenciaForm)
	rt.Handle("POST /processos/{id}/providencia", w, h.providencia)
	rt.Handle("GET /processos/{id}/excluir", w, h.removeForm)
	rt.Handle("POST /processos/{id}/excluir", w, h.remove)
}

func base(s *auth.Session, title string) auth.PageData {
	return auth.PageData{Title: title, Email: s.Email, Role: string(s.Role)}
}

// Grupo é um painel do acordeão.
type Grupo struct {
	Estado    Estado
	Processos []*Processo
}

// Quantidade é o número do botão do grupo.
func (g Grupo) Quantidade() int { return len(g.Processos) }

type listData struct {
	Grupos   []Grupo
	Total    int
	CanWrite bool
}

// Busca é o texto que o campo de busca da lista consulta: número, classe, matéria, cliente e partes.
func (p *Processo) Busca() string {
	return strings.Join([]string{p.Numero, p.Classe, p.Materia, p.ClienteNome(), strings.Join(p.ParteContraria(), ", ")}, " ")
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	ps, err := h.S.Listar(r.Context())
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	d := listData{Total: len(ps), CanWrite: auth.Can(s.Role, auth.PermProcWrite)}
	for _, e := range Estados {
		g := Grupo{Estado: e}
		for _, p := range ps {
			if p.Estado == e.Valor {
				g.Processos = append(g.Processos, p)
			}
		}
		d.Grupos = append(d.Grupos, g)
	}
	pd := base(s, "Processos")
	pd.Head = auth.Head{Title: "Processos", Sub: "Cadastro processual: partes, classe, valor da causa e marcos."}
	if d.CanWrite {
		pd.Head.ActionLabel, pd.Head.ActionHref, pd.Head.ActionDrawer = "Novo processo", "/processos/novo", true
	}
	pd.Data = d
	h.A.Render(w, r, http.StatusOK, page("lista.html"), pd)
}

func (h *Handler) detail(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	p, err := h.S.Get(r.Context(), r.PathValue("id"))
	if h.falha(w, err) {
		return
	}
	and, err1 := h.S.Andamentos(r.Context(), p.ID)
	fin, err2 := h.S.Financeiro(r.Context(), p.ID)
	if err1 != nil || err2 != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	pd := base(s, p.NumeroOuTitulo())
	pd.Head = auth.Head{Title: p.NumeroOuTitulo(), Sub: p.Codigo}
	canWrite := auth.Can(s.Role, auth.PermProcWrite)
	if canWrite {
		pd.Head.ActionLabel, pd.Head.ActionHref, pd.Head.ActionDrawer = "Editar", "/processos/"+p.ID+"/editar", true
	}
	pd.Data = map[string]any{"P": p, "Andamentos": and, "Financeiro": fin, "CanWrite": canWrite}
	h.A.Render(w, r, http.StatusOK, page("detalhe.html"), pd)
}

// ok responde sucesso ao formulário do drawer: o JS fecha o drawer, mostra o toast e recarrega o conteúdo.
func ok(w http.ResponseWriter, msg string) {
	w.Header().Set("X-Painel-Ok", msg)
	w.WriteHeader(http.StatusOK)
}

// falha traduz o erro em texto simples (o JS do drawer mostra como toast de erro, como o showToast(..., true) do B).
func (h *Handler) falha(w http.ResponseWriter, err error, msgFalha ...string) bool {
	if err == nil {
		return false
	}
	var inv webx.Invalid
	switch {
	case errors.As(err, &inv):
		http.Error(w, inv.Msg, http.StatusBadRequest)
	case errors.Is(err, webx.ErrNotFound):
		http.Error(w, "Processo não encontrado.", http.StatusNotFound)
	case errors.Is(err, webx.ErrConflict):
		http.Error(w, "Este registro foi alterado por outra pessoa. Feche, recarregue a página e refaça a alteração.", http.StatusConflict)
	default:
		msg := "Erro interno."
		if len(msgFalha) > 0 {
			msg = msgFalha[0] // texto de falha do painel B
		}
		http.Error(w, msg, http.StatusInternalServerError)
	}
	return true
}
