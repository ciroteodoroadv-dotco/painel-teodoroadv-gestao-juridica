package processos

import (
	"net/http"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/webx"
)

// ComOutro é um <select> com a opção "Outro", que revela um campo de texto (selectComOutroHtml do B).
type ComOutro struct {
	Nome, Sel, Outro string
	Opcoes           []string
}

func novoComOutro(nome string, lista []string, atual string) ComOutro {
	c := ComOutro{Nome: nome, Opcoes: lista}
	switch {
	case naLista(lista[:len(lista)-1], atual):
		c.Sel = atual
	case atual != "":
		c.Sel, c.Outro = "Outro", atual
	}
	return c
}

// LinhaParte é uma linha do repetidor "Partes".
type LinhaParte struct {
	Pessoa  string
	Papel   ComOutro
	Cliente bool
	Pessoas []Opcao
}

// RespSel é um dos quatro seletores fixos de "Responsáveis pelo cadastro".
type RespSel struct {
	N   int
	Sel string
}

type formView struct {
	Novo     bool
	P        Processo
	Aviso    string
	Pessoas  []Opcao
	Partes   []LinhaParte
	Vazia    LinhaParte
	Resp     []RespSel
	Sistema  ComOutro
	Instan   ComOutro
	Classes  []string
	Materias []string
}

// montaForm monta o que o formulário precisa. p é zero na criação.
func (h *Handler) montaForm(w http.ResponseWriter, r *http.Request, s *auth.Session, novo bool, p Processo) (formView, bool) {
	ops, err := h.S.Pessoas.Opcoes(r.Context(), webx.ActorOf(w, s.UserID), ComDocumento(s))
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return formView{}, false
	}
	linha := func(pt Parte) LinhaParte {
		return LinhaParte{Pessoa: pt.PessoaID, Papel: novoComOutro("parte_papel", PapeisProcessuais, pt.Papel), Cliente: pt.Cliente, Pessoas: ops}
	}
	f := formView{Novo: novo, P: p, Pessoas: ops, Classes: Classes, Materias: Materias, Vazia: linha(Parte{}),
		Sistema: novoComOutro("sistema", Sistemas, p.Sistema), Instan: novoComOutro("instancia", Instancias, p.Instancia)}
	for _, pt := range p.Partes {
		f.Partes = append(f.Partes, linha(pt))
	}
	if len(f.Partes) == 0 {
		f.Partes = []LinhaParte{linha(Parte{})}
	}
	for i := 0; i < 4; i++ {
		sel := ""
		if i < len(p.Responsaveis) {
			sel = p.Responsaveis[i]
		}
		f.Resp = append(f.Resp, RespSel{N: i + 1, Sel: sel})
	}
	return f, true
}

func (h *Handler) mostra(w http.ResponseWriter, f formView) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("form.html").ExecuteTemplate(w, "form", f); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

func (h *Handler) newForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	if f, ok := h.montaForm(w, r, s, true, Processo{}); ok {
		h.mostra(w, f)
	}
}

// avisoDaPessoa monta o aviso que o B mostra ao abrir o processo a partir da aba Pessoas.
func avisoDaPessoa(p *Processo, pessoaID string) string {
	for _, pt := range p.Partes {
		if pt.PessoaID == pessoaID {
			rot := pt.Papel
			switch rot {
			case "cliente", "parte_contraria", "terceiro_interessado":
				rot = map[string]string{"cliente": "cliente", "parte_contraria": "parte contrária", "terceiro_interessado": "terceiro interessado"}[rot]
			}
			return "Esta pessoa é " + rot + " neste processo — edite ou apague o campo correspondente abaixo."
		}
	}
	for _, r := range p.Responsaveis {
		if r == pessoaID {
			return "Esta pessoa é Responsável pelo cadastro neste processo — edite ou apague o campo correspondente abaixo."
		}
	}
	return ""
}

func (h *Handler) editForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	p, err := h.S.Get(r.Context(), r.PathValue("id"))
	if h.falha(w, err) {
		return
	}
	f, ok := h.montaForm(w, r, s, false, *p)
	if !ok {
		return
	}
	if da := r.URL.Query().Get("da_pessoa"); da != "" {
		f.Aviso = avisoDaPessoa(p, da)
	}
	h.mostra(w, f)
}

// inputFrom lê o formulário. As linhas de Partes chegam em listas paralelas (uma entrada por linha).
func inputFrom(r *http.Request) (Input, bool) {
	_ = r.ParseForm()
	v := r.PostFormValue
	pes, pap, outro, cli := r.PostForm["parte_pessoa"], r.PostForm["parte_papel"], r.PostForm["parte_papel_outro"], r.PostForm["parte_cliente"]
	if len(pap) != len(pes) || len(outro) != len(pes) || len(cli) != len(pes) {
		return Input{}, false
	}
	in := Input{Responsaveis: r.PostForm["responsavel"], Numero: v("numero_processo"), ValorCausa: v("valor_causa"),
		Sistema: comOutro(v("sistema"), v("sistema_outro")), Instancia: comOutro(v("instancia"), v("instancia_outro")),
		Classe: v("classe"), Materia: v("ramo"), Comarca: v("comarca"), Juizo: v("juizo"), Contingencia: v("tipo_contingencia"),
		Distribuicao: v("distribuido_em"), Citacao: v("citacao"), Audiencia: v("audiencia"), Sentenca: v("sentenca"),
		Recursos: v("recursos"), FaseExecucao: v("fase_execucao"), Observacoes: v("observacoes"),
		Titulo: v("titulo"), Polo: v("polo"), Projeto: v("projeto_vinculado")}
	for i := range pes {
		in.Partes = append(in.Partes, ParteIn{PessoaID: pes[i], Papel: comOutro(pap[i], outro[i]), Cliente: cli[i] == "1"})
	}
	return in, true
}

func comOutro(sel, outro string) string {
	if sel == "Outro" {
		return outro
	}
	return sel
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	in, ok2 := inputFrom(r)
	if !ok2 {
		http.Error(w, "Formulário inválido.", http.StatusBadRequest)
		return
	}
	id, _, err := h.S.Create(r.Context(), webx.ActorOf(w, s.UserID), in)
	if h.falha(w, err) {
		return
	}
	w.Header().Set("X-Painel-Id", id)
	ok(w, "Processo criado.")
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	in, ok2 := inputFrom(r)
	if !ok2 {
		http.Error(w, "Formulário inválido.", http.StatusBadRequest)
		return
	}
	err := h.S.Update(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version")), in)
	if h.falha(w, err) {
		return
	}
	ok(w, "Processo atualizado.")
}
