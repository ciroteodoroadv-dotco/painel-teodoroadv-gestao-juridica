package tarefas

import (
	"net/http"
	"time"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/webx"
)

// ComOutro é um <select> com a opção "Outro", que revela um campo de texto (como o papel das Partes do B).
type ComOutro struct{ Sel, Outro string }

func papelComOutro(atual string) ComOutro {
	for _, p := range PapeisParte[:len(PapeisParte)-1] {
		if p == atual {
			return ComOutro{Sel: atual}
		}
	}
	if atual != "" {
		return ComOutro{Sel: "Outro", Outro: atual}
	}
	return ComOutro{}
}

// LinhaParte é uma linha do repetidor "Partes"; LinhaResp, uma de "Responsáveis existentes".
type LinhaParte struct {
	Pessoa  string
	Papel   ComOutro
	Papeis  []string
	Pessoas []Opcao
}
type LinhaResp struct {
	Pessoa  string
	Pessoas []Opcao
}

type formView struct {
	Novo, Filha                    bool
	T                              Tarefa
	Vinculo                        string // vínculo inicial: "atendimento" | "processo" | ""
	OrigemID                       string
	TemAtend, TemProc              bool
	OpAtend, OpProc                []Opcao
	Pessoas                        []Opcao
	Criadores                      []Ref
	Partes                         []LinhaParte
	Resps                          []LinhaResp
	VaziaParte                     LinhaParte
	VaziaResp                      LinhaResp
	Estados                        []Estado
	Prioridades, Papeis, Subtitulo []string
	EstadoSel, PrioridadeSel       string
	Data                           string
	AConferir                      bool
}

func (h *Handler) montaForm(w http.ResponseWriter, r *http.Request, s *auth.Session, t Tarefa, pai *Tarefa, novo bool) {
	ctx := r.Context()
	pessoas, err := h.S.OpcoesPessoa(ctx, webx.ActorOf(w, s.UserID), auth.Can(s.Role, auth.PermPessoasWrite))
	var criad []Ref
	var oa, op []Opcao
	if err == nil {
		criad, err = h.S.Refs(ctx)
	}
	if err == nil && novo {
		if oa, err = h.S.OpcoesVinculo(ctx, "atendimento"); err == nil {
			op, err = h.S.OpcoesVinculo(ctx, "processo")
		}
	}
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	f := formView{Novo: novo, T: t, Pessoas: pessoas, Criadores: criad, OpAtend: oa, OpProc: op, TemAtend: len(oa) > 0, TemProc: len(op) > 0,
		Estados: Estados, Prioridades: Prioridades, Papeis: PapeisParte, Subtitulo: Subtitulos, EstadoSel: "pendente", PrioridadeSel: "Normal",
		VaziaParte: LinhaParte{Pessoas: pessoas, Papeis: PapeisParte}, VaziaResp: LinhaResp{Pessoas: pessoas}, Data: webx.Hoje(time.Now()).Format("2006-01-02")}
	src := &t
	if novo {
		switch {
		case pai != nil:
			f.Filha, f.Vinculo, f.OrigemID, src = true, pai.OrigemTipo, pai.OrigemID, pai
			if pai.OrigemTipo == OrigemIntimacao { // a filha de uma tarefa do DJEN escolhe o próprio vínculo
				f.Vinculo, f.OrigemID = "", ""
			}
			f.T = Tarefa{TarefaPaiID: pai.ID}
		case f.TemAtend:
			f.Vinculo = "atendimento"
		case f.TemProc:
			f.Vinculo = "processo"
		}
	} else {
		f.EstadoSel, f.PrioridadeSel, f.AConferir = t.Estado, t.Prioridade, t.AConferir()
	}
	if novo && pai == nil {
		src = &Tarefa{}
	}
	for _, p := range src.Partes {
		f.Partes = append(f.Partes, LinhaParte{Pessoa: p.PessoaID, Papel: papelComOutro(p.Papel), Papeis: PapeisParte, Pessoas: pessoas})
	}
	for _, p := range src.Responsaveis {
		f.Resps = append(f.Resps, LinhaResp{Pessoa: p.ID, Pessoas: pessoas})
	}
	if len(f.Partes) == 0 {
		f.Partes = []LinhaParte{f.VaziaParte}
	}
	if len(f.Resps) == 0 {
		f.Resps = []LinhaResp{f.VaziaResp}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("drawer.html").ExecuteTemplate(w, "form", f); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

func (h *Handler) newForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	h.montaForm(w, r, s, Tarefa{}, nil, true)
}

// filhaForm abre "Nova tarefa" com vínculo, partes e responsáveis da tarefa-pai (só pré-preenchimento).
func (h *Handler) filhaForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	pai, err := h.S.Get(r.Context(), r.PathValue("id"))
	if webx.Falha(w, err, "Tarefa não encontrada.") {
		return
	}
	h.montaForm(w, r, s, Tarefa{}, pai, true)
}

func (h *Handler) editForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	t, err := h.S.Get(r.Context(), r.PathValue("id"))
	if webx.Falha(w, err, "Tarefa não encontrada.") {
		return
	}
	h.montaForm(w, r, s, *t, nil, false)
}

// inputFrom lê o formulário. Partes e Responsáveis chegam em listas paralelas (uma entrada por linha).
func inputFrom(r *http.Request) (Input, bool) {
	_ = r.ParseForm()
	v := r.PostFormValue
	pes, pap, outro := r.PostForm["parte_pessoa"], r.PostForm["parte_papel"], r.PostForm["parte_papel_outro"]
	if len(pap) != len(pes) || len(outro) != len(pes) {
		return Input{}, false
	}
	in := Input{OrigemTipo: v("origem_tipo"), OrigemID: v("origem_id"), TarefaPai: v("tarefa_pai"), Titulo: v("titulo"),
		Subtitulo: v("subtitulo"), Descricao: v("descricao"), CriadoPorID: v("criado_por"), DataBase: v("data_base"), Prazo: v("prazo"),
		PrazoFatal: v("prazo_fatal"), Estado: v("estado"), Prioridade: v("prioridade"), Observacoes: v("observacoes"),
		Responsaveis: r.PostForm["responsavel"]}
	for i := range pes {
		papel := pap[i]
		if papel == "Outro" {
			papel = outro[i]
		}
		in.Partes = append(in.Partes, ParteIn{PessoaID: pes[i], Papel: papel})
	}
	return in, true
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	in, ok := inputFrom(r)
	if !ok {
		http.Error(w, "Formulário inválido.", http.StatusBadRequest)
		return
	}
	id, _, err := h.S.Create(r.Context(), webx.ActorOf(w, s.UserID), in, auth.Can(s.Role, auth.PermPrazoConfirm))
	if webx.Falha(w, err, "Tarefa não encontrada.", "Falha ao criar a tarefa.") {
		return
	}
	w.Header().Set("X-Painel-Id", id)
	webx.Ok(w, "Tarefa criada.")
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	in, ok := inputFrom(r)
	if !ok {
		http.Error(w, "Formulário inválido.", http.StatusBadRequest)
		return
	}
	err := h.S.Update(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version")), in,
		auth.Can(s.Role, auth.PermPrazoConfirm))
	if webx.Falha(w, err, "Tarefa não encontrada.", "Falha ao atualizar a tarefa.") {
		return
	}
	webx.Ok(w, "Tarefa atualizada.")
}
