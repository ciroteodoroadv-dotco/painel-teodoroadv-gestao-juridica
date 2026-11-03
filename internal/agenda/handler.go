package agenda

import (
	"embed"
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

func page(name ...string) *template.Template {
	for i := range name {
		name[i] = "templates/" + name[i]
	}
	return template.Must(auth.BaseTemplate().ParseFS(tplFS, name...))
}

func fragmento(names ...string) *template.Template {
	for i := range names {
		names[i] = "templates/" + names[i]
	}
	return template.Must(template.ParseFS(tplFS, names...))
}

// Register declara as rotas, cada uma com a sua permissão.
func (h *Handler) Register(rt *auth.Router) {
	h.A.AddNavCount("Agenda", "/agenda", auth.PermAgendaRead, h.contar)
	w := auth.PermAgendaWrite
	rt.Handle("GET /agenda", auth.PermAgendaRead, h.pagina)
	rt.Handle("GET /agenda/novo", w, h.novoForm)
	rt.Handle("POST /agenda", w, h.criar)
	rt.Handle("GET /agenda/{id}/editar", w, h.editarForm)
	rt.Handle("POST /agenda/{id}", w, h.atualizar)
}

// Seletor é um <select> de vínculo: a opção marcada e as opções (a primeira, "Sem vínculo", o template põe).
type Seletor struct {
	Sel    string
	Opcoes []Opcao
}

type formView struct {
	Novo                              bool
	C                                 Compromisso
	Tipos                             []string
	Clientes, Atendimentos, Processos Seletor
	Criadores                         []Ref
}

func (h *Handler) montaForm(w http.ResponseWriter, r *http.Request, c Compromisso, novo bool) {
	ctx := r.Context()
	refs, err := h.S.Pessoas.Refs(ctx, nil)
	var ate, proc []Opcao
	if err == nil {
		ate, err = h.S.Vinculos.Opcoes(ctx, "atendimento")
	}
	if err == nil {
		proc, err = h.S.Vinculos.Opcoes(ctx, "processo")
	}
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	clientes := make([]Opcao, len(refs))
	for i, p := range refs {
		clientes[i] = Opcao{Valor: p.ID, Rotulo: p.Rotulo()}
	}
	f := formView{Novo: novo, C: c, Tipos: TiposCompromisso, Criadores: refs,
		Clientes: Seletor{c.ClienteID, clientes}, Atendimentos: Seletor{c.AtendimentoID, ate}, Processos: Seletor{c.ProcessoID, proc}}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("drawer.html").ExecuteTemplate(w, "form", f); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

func (h *Handler) novoForm(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	h.montaForm(w, r, Compromisso{}, true)
}

func (h *Handler) editarForm(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	c, err := h.S.Get(r.Context(), r.PathValue("id"))
	if webx.Falha(w, err, "Item de agenda não encontrado.") {
		return
	}
	h.montaForm(w, r, *c, false)
}

func inputFrom(r *http.Request) Input {
	v := r.PostFormValue
	return Input{Tipo: v("tipo"), Descricao: v("descricao"), Data: v("data"), Hora: v("hora"), Local: v("local"),
		Responsavel: v("responsavel"), ClienteID: v("cliente"), AtendimentoID: v("atendimento"), ProcessoID: v("processo"),
		CriadoPorID: v("criado_por")}
}

func (h *Handler) criar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	in, ac := inputFrom(r), webx.ActorOf(w, s.UserID)
	if strings.TrimSpace(in.Tipo) == TipoDiligencia {
		if !auth.Can(s.Role, auth.PermDiligenciaWrite) {
			http.Error(w, "Acesso negado.", http.StatusForbidden)
			return
		}
		codigo, err := h.S.CriarDiligencia(r.Context(), ac, in)
		if webx.Falha(w, err, "Registro não encontrado.", "Falha ao criar a diligência.") {
			return
		}
		webx.Ok(w, "Diligência criada — código "+codigo+".")
		return
	}
	aviso, err := h.S.CriarCompromisso(r.Context(), ac, in)
	if webx.Falha(w, err, "Registro não encontrado.", "Falha ao criar o compromisso.") {
		return
	}
	if aviso != "" {
		webx.Ok(w, "Compromisso criado — "+aviso)
		return
	}
	webx.Ok(w, "Compromisso criado.")
}

func (h *Handler) atualizar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.AtualizarCompromisso(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version")), inputFrom(r))
	if webx.Falha(w, err, "Item de agenda não encontrado.", "Falha ao atualizar o compromisso.") {
		return
	}
	webx.Ok(w, "Compromisso atualizado.")
}
