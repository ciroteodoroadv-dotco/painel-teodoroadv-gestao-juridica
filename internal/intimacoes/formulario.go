package intimacoes

import (
	"net/http"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/webx"
)

// linhaOAB é uma OAB na tabela da página Coleta DJEN (datas já em dd/mm/aaaa).
type linhaOAB struct {
	ID, Nome, Rotulo, Base, Data, UltimoSucesso string
	Ativa                                       bool
}

type paginaData struct {
	Ativa     bool
	Estado    string
	EstadoErr bool
	Linhas    []linhaOAB
}

func (h *Handler) pagina(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	ctx := r.Context()
	oabs, err := h.S.ListarOABs(ctx)
	var est string
	var estErr bool
	if err == nil {
		e, existe, e2 := h.S.Coletas.Obter(ctx, Modulo)
		err = e2
		if existe {
			est, estErr = e.Linha()
		} else {
			est = "Nenhuma coleta registrada ainda."
		}
	}
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	d := paginaData{Ativa: h.S.Cfg.Ativo, Estado: est, EstadoErr: estErr}
	for _, o := range oabs {
		l := linhaOAB{ID: o.ID, Nome: o.Nome, Rotulo: o.Rotulo(), Base: o.AutorizacaoBase, Data: webx.DataBR(o.AutorizacaoEm), Ativa: o.Ativa, UltimoSucesso: "—"}
		if o.UltimoSucesso != nil {
			l.UltimoSucesso = o.UltimoSucesso.In(webx.SP()).Format("02/01/2006 15:04")
		}
		d.Linhas = append(d.Linhas, l)
	}
	pd := auth.PageData{Title: "Coleta DJEN", Email: s.Email, Role: string(s.Role)}
	pd.Head = auth.Head{Title: "Coleta DJEN", Sub: "OABs monitoradas e estado da coleta de intimações do DJEN.",
		ActionLabel: "Nova OAB", ActionHref: "/coleta-djen/nova", ActionDrawer: true}
	pd.Data = d
	h.A.Render(w, r, http.StatusOK, page("oabs.html"), pd)
}

type formView struct {
	Novo bool
	O    OAB
	Data string // data da autorização no campo (AAAA-MM-DD)
}

func (h *Handler) montaForm(w http.ResponseWriter, o OAB, novo bool) {
	f := formView{Novo: novo, O: o, Data: o.AutorizacaoISO()}
	if novo {
		f.O = OAB{AutorizacaoBase: BaseAutorizacaoPadrao, Ativa: true}
		f.Data = webx.Hoje(h.S.now()).Format("2006-01-02")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("oab-drawer.html").ExecuteTemplate(w, "form", f); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

func (h *Handler) novaForm(w http.ResponseWriter, _ *http.Request, _ *auth.Session) {
	h.montaForm(w, OAB{}, true)
}

func (h *Handler) editarForm(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	o, err := h.S.ObterOAB(r.Context(), r.PathValue("id"))
	if webx.Falha(w, err, "OAB não encontrada.") {
		return
	}
	h.montaForm(w, o, false)
}

func oabIn(r *http.Request) OABIn {
	v := r.PostFormValue
	return OABIn{Nome: v("nome"), Numero: v("numero"), UF: v("uf"), AutorizacaoBase: v("autorizacao_base"),
		AutorizacaoEm: v("autorizacao_em"), Ativa: v("ativa") == "1"}
}

func (h *Handler) criar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	_, err := h.S.CadastrarOAB(r.Context(), webx.ActorOf(w, s.UserID), oabIn(r))
	if webx.Falha(w, err, "Registro não encontrado.", "Falha ao cadastrar a OAB.") {
		return
	}
	webx.Ok(w, "OAB cadastrada.")
}

func (h *Handler) atualizar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.AtualizarOAB(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), webx.Int(r.PostFormValue("version")), oabIn(r))
	if webx.Falha(w, err, "OAB não encontrada.", "Falha ao atualizar a OAB.") {
		return
	}
	webx.Ok(w, "OAB atualizada.")
}
