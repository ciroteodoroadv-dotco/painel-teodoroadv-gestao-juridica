package pessoas

import (
	"embed"
	"errors"
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

// page é a página inteira (com a casca do painel); fragmento é só o miolo do drawer.
func page(name string) *template.Template {
	return template.Must(auth.BaseTemplate().ParseFS(tplFS, "templates/"+name))
}

func fragmento(name string) *template.Template {
	return template.Must(template.ParseFS(tplFS, "templates/"+name))
}

// Register declara as rotas, cada uma com a sua permissão.
func (h *Handler) Register(rt *auth.Router) {
	h.A.AddNavCount("Pessoas", "/pessoas", auth.PermPessoasRead, webx.ContarAtivos(h.S.Pool, "pessoas"))
	rt.Handle("GET /pessoas", auth.PermPessoasRead, h.list)
	rt.Handle("GET /pessoas/nova", auth.PermPessoasWrite, h.newForm)
	rt.Handle("POST /pessoas", auth.PermPessoasWrite, h.create)
	rt.Handle("GET /pessoas/{id}", auth.PermPessoasRead, h.detail)
	rt.Handle("GET /pessoas/{id}/editar", auth.PermPessoasWrite, h.editForm)
	rt.Handle("POST /pessoas/{id}", auth.PermPessoasWrite, h.update)
	rt.Handle("GET /pessoas/{id}/excluir", auth.PermPessoasWrite, h.removeForm)
	rt.Handle("POST /pessoas/{id}/excluir", auth.PermPessoasWrite, h.remove)
	rt.Handle("GET /pessoas/{id}/relatorio", auth.PermPessoasWrite, h.relatorio)
}

func base(s *auth.Session, title string) auth.PageData {
	return auth.PageData{Title: title, Email: s.Email, Role: string(s.Role)}
}

func (h *Handler) papeis(r *http.Request, ids []string) (map[string][]PapelEmProcesso, error) {
	if h.S.Papeis == nil {
		return map[string][]PapelEmProcesso{}, nil
	}
	return h.S.Papeis.PapeisPorPessoa(r.Context(), ids)
}

type listData struct {
	Grupos   []Grupo
	Total    int
	CanWrite bool
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	canWrite := auth.Can(s.Role, auth.PermPessoasWrite)
	rows, err := h.S.Listar(r.Context(), webx.ActorOf(w, s.UserID), canWrite) // documento inteiro só a quem escreve
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	ids := make([]string, len(rows))
	for i, p := range rows {
		ids[i] = p.ID
	}
	pap, err := h.papeis(r, ids)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	pd := base(s, "Pessoas")
	pd.Head = auth.Head{Title: "Pessoas", Sub: "Cadastro de clientes, partes adversas e terceiros interessados — art. 319 do CPC."}
	if canWrite {
		pd.Head.ActionLabel, pd.Head.ActionHref, pd.Head.ActionDrawer = "Nova pessoa", "/pessoas/nova", true
	}
	pd.Data = listData{Grupos: Agrupar(rows, pap), Total: len(rows), CanWrite: canWrite}
	h.A.Render(w, r, http.StatusOK, page("lista.html"), pd)
}

type formView struct {
	Novo    bool
	P       Pessoa
	Estados []string
}

func (h *Handler) drawer(w http.ResponseWriter, f formView) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("form.html").ExecuteTemplate(w, "form", f); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

func (h *Handler) newForm(w http.ResponseWriter, _ *http.Request, _ *auth.Session) {
	h.drawer(w, formView{Novo: true, Estados: EstadosCivis,
		P: Pessoa{TipoPessoa: Fisica, Nacionalidade: "Brasileiro(a)"}})
}

func inputFrom(r *http.Request) Input {
	v := r.PostFormValue
	return Input{TipoPessoa: v("tipo_pessoa"), NomeCompleto: v("nome_completo"), CPF: v("cpf"), RG: v("rg"),
		EstadoCivil: v("estado_civil"), Profissao: v("profissao"), Nacionalidade: v("nacionalidade"),
		DataNascimento: v("data_nascimento"), RazaoSocial: v("razao_social"), NomeFantasia: v("nome_fantasia"),
		CNPJ: v("cnpj"), RepresentanteNome: v("representante_nome"), RepresentanteCPF: v("representante_cpf"),
		RepresentanteCargo: v("representante_cargo"), Logradouro: v("logradouro"), Numero: v("numero"),
		Complemento: v("complemento"), Bairro: v("bairro"), CEP: v("cep"), Cidade: v("cidade"), UF: v("uf"),
		Email: v("email"), Telefone: v("telefone"), Observacoes: v("observacoes")}
}

// ok responde sucesso ao formulário do drawer: o JS fecha o drawer, mostra o toast e recarrega o conteúdo.
func ok(w http.ResponseWriter, msg string) {
	w.Header().Set("X-Painel-Ok", msg)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	id, codigo, err := h.S.Create(r.Context(), webx.ActorOf(w, s.UserID), inputFrom(r))
	if h.falha(w, err) {
		return
	}
	w.Header().Set("X-Painel-Id", id)
	ok(w, "Pessoa criada — código "+codigo+".")
}

func (h *Handler) editForm(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	p, err := h.S.GetCompleto(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id")) // revela e audita
	if h.falha(w, err) {
		return
	}
	h.drawer(w, formView{P: *p, Estados: EstadosCivis})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.Update(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version")), inputFrom(r))
	if h.falha(w, err) {
		return
	}
	ok(w, "Pessoa atualizada.")
}

type excluirView struct {
	P         Pessoa
	Processos []PapelEmProcesso
}

// removeForm decide, como no B, o que o botão "Excluir" faz: pessoa em um processo abre esse processo;
// em vários, pergunta qual; sem processo, pede a confirmação da exclusão lógica.
func (h *Handler) removeForm(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	id := r.PathValue("id")
	p, err := h.S.Get(r.Context(), id)
	if h.falha(w, err) {
		return
	}
	pap, err := h.papeis(r, []string{id})
	if h.falha(w, err) {
		return
	}
	switch refs := pap[id]; len(refs) {
	case 0:
		h.fragmentoTpl(w, "confirmar", excluirView{P: *p})
	case 1:
		http.Redirect(w, r, "/processos/"+refs[0].ProcessoID+"/editar?da_pessoa="+id, http.StatusSeeOther)
	default:
		h.fragmentoTpl(w, "escolha", excluirView{P: *p, Processos: refs})
	}
}

func (h *Handler) fragmentoTpl(w http.ResponseWriter, nome string, d any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("excluir.html").ExecuteTemplate(w, nome, d); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.Delete(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version")))
	if h.falha(w, err) {
		return
	}
	ok(w, "Pessoa excluída.")
}

func (h *Handler) detail(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	p, err := h.S.Get(r.Context(), r.PathValue("id"))
	if h.falha(w, err) {
		return
	}
	pap, err := h.papeis(r, []string{p.ID})
	if h.falha(w, err) {
		return
	}
	pd := base(s, p.NomeLista())
	pd.Head = auth.Head{Title: p.NomeLista(), Sub: p.Codigo}
	canWrite := auth.Can(s.Role, auth.PermPessoasWrite)
	if canWrite {
		pd.Head.ActionLabel, pd.Head.ActionHref, pd.Head.ActionDrawer = "Editar", "/pessoas/"+p.ID+"/editar", true
	}
	pd.Data = map[string]any{"P": p, "Refs": pap[p.ID], "CanWrite": canWrite}
	h.A.Render(w, r, http.StatusOK, page("detalhe.html"), pd)
}

func (h *Handler) relatorio(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	rel, err := h.S.Relatorio(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"))
	if h.falha(w, err) {
		return
	}
	pd := base(s, "Dados do titular")
	pd.Data = rel
	h.A.Render(w, r, http.StatusOK, page("relatorio.html"), pd)
}

// falha traduz o erro em resposta de texto simples (o JS do drawer a mostra como toast de erro,
// como o showToast(..., true) do B). Devolve true se respondeu.
func (h *Handler) falha(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var inv webx.Invalid
	switch {
	case errors.As(err, &inv):
		http.Error(w, inv.Msg, http.StatusBadRequest)
	case errors.Is(err, webx.ErrNotFound):
		http.Error(w, "Pessoa não encontrada.", http.StatusNotFound)
	case errors.Is(err, webx.ErrConflict):
		http.Error(w, "Este registro foi alterado por outra pessoa. Feche, recarregue a página e refaça a alteração.", http.StatusConflict)
	default:
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
	return true
}
