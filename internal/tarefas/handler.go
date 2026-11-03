package tarefas

import (
	"embed"
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

// fragmento carrega arquivos de templates sem a casca da página (miolo do drawer).
func fragmento(names ...string) *template.Template {
	for i := range names {
		names[i] = "templates/" + names[i]
	}
	return template.Must(template.ParseFS(tplFS, names...))
}

// Register declara as rotas, cada uma com a sua permissão.
func (h *Handler) Register(rt *auth.Router) {
	h.A.AddNavCount("Tarefas", "/tarefas", auth.PermTarefaRead, webx.ContarAtivos(h.S.Pool, "tarefas"))
	w := auth.PermTarefaWrite
	rt.Handle("GET /tarefas", auth.PermTarefaRead, h.list)
	rt.Handle("GET /tarefas/nova", w, h.newForm)
	rt.Handle("POST /tarefas", w, h.create)
	rt.Handle("GET /tarefas/{id}/editar", w, h.editForm)
	rt.Handle("GET /tarefas/{id}/filha", w, h.filhaForm)
	rt.Handle("POST /tarefas/{id}", w, h.update)
	rt.Handle("POST /tarefas/{id}/mover", w, h.mover)
	rt.Handle("POST /tarefas/{id}/documentos", w, h.documento)
	rt.Handle("POST /tarefas/{id}/confirmar-prazo", auth.PermPrazoConfirm, h.confirmarPrazo)
	rt.Handle("GET /tarefas/{id}/excluir", auth.PermTarefaDelete, h.excluirForm)
	rt.Handle("POST /tarefas/{id}/excluir", auth.PermTarefaDelete, h.excluir)
}

// Grupo é um painel do acordeão: um dos 7 estados com as suas tarefas.
type Grupo struct {
	Estado Estado
	Linhas []Linha
}

// Linha é uma tarefa na tabela, com o que a linha precisa para decidir os botões.
type Linha struct {
	T                                 *Tarefa
	Outros                            []Estado // opções de "Mover para..." (os outros 6 estados)
	CanWrite, CanDelete, CanConfirmar bool
}

// Quantidade é o número do botão do grupo.
func (g Grupo) Quantidade() int { return len(g.Linhas) }

type listData struct {
	Grupos []Grupo
	Total  int
	Q      string
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	q := r.URL.Query().Get("q")
	l, err := h.S.Listar(r.Context(), q)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	d := listData{Total: len(l), Q: q}
	escreve := auth.Can(s.Role, auth.PermTarefaWrite)
	for _, e := range Estados {
		g := Grupo{Estado: e}
		for _, t := range l {
			if t.Estado != e.Chave {
				continue
			}
			ln := Linha{T: t, CanWrite: escreve, CanDelete: escreve && auth.Can(s.Role, auth.PermTarefaDelete),
				CanConfirmar: auth.Can(s.Role, auth.PermPrazoConfirm)}
			for _, x := range Estados {
				if x.Chave != e.Chave {
					ln.Outros = append(ln.Outros, x)
				}
			}
			g.Linhas = append(g.Linhas, ln)
		}
		d.Grupos = append(d.Grupos, g)
	}
	pd := auth.PageData{Title: "Tarefas", Email: s.Email, Role: string(s.Role)}
	pd.Head = auth.Head{Title: "Tarefas", Sub: "Toda tarefa nasce de um atendimento (ordem de serviço) ou de um processo."}
	if escreve {
		pd.Head.ActionLabel, pd.Head.ActionHref, pd.Head.ActionDrawer = "Nova tarefa", "/tarefas/nova", true
	}
	pd.Data = d
	h.A.Render(w, r, http.StatusOK, page("lista.html"), pd)
}
