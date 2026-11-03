package processos

import (
	"net/http"
	"time"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/webx"
)

// acaoView alimenta os drawers das ações da lista (Mover estado, Andamento, Financeiro, Próxima providência, Excluir).
type acaoView struct {
	P       Processo
	Estados []string
	Hoje    string
	Abertos int
}

func (h *Handler) acao(w http.ResponseWriter, r *http.Request, nome string, comProcesso bool) {
	v := acaoView{Hoje: webx.Hoje(time.Now()).Format("2006-01-02"), Estados: []string{"ativo", "sobrestado", "extinto", "arquivado"}}
	if comProcesso {
		p, err := h.S.Get(r.Context(), r.PathValue("id"))
		if h.falha(w, err) {
			return
		}
		v.P = *p
		if nome == "excluir" {
			n, err := h.S.AbertosVinculados(r.Context(), p.ID)
			if h.falha(w, err) {
				return
			}
			v.Abertos = n
		}
	} else {
		v.P.ID = r.PathValue("id")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("acoes.html").ExecuteTemplate(w, nome, v); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

func (h *Handler) estadoForm(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	h.acao(w, r, "estado", true)
}
func (h *Handler) andamentoForm(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	h.acao(w, r, "andamento", false)
}
func (h *Handler) financeiroForm(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	h.acao(w, r, "financeiro", false)
}
func (h *Handler) providenciaForm(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	h.acao(w, r, "providencia", false)
}
func (h *Handler) removeForm(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	h.acao(w, r, "excluir", true)
}

func (h *Handler) estado(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.MoverEstado(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"),
		webx.Int(r.PostFormValue("version")), r.PostFormValue("novo_estado"), r.PostFormValue("motivo"))
	if h.falha(w, err) {
		return
	}
	ok(w, "Estado do processo atualizado.")
}

// dataOuVazio lê uma data AAAA-MM-DD; vazia devolve nil, inválida devolve erro de validação.
func dataOuVazio(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, webx.Invalidf("Data inválida.")
	}
	return &t, nil
}

func (h *Handler) andamento(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	d, err := dataOuVazio(r.PostFormValue("data"))
	if h.falha(w, err) {
		return
	}
	dia := webx.Hoje(time.Now()) // padrão do B: hoje
	if d != nil {
		dia = *d
	}
	err = h.S.RegistrarAndamento(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"), dia,
		r.PostFormValue("o_que"), r.PostFormValue("documento"))
	if h.falha(w, err) {
		return
	}
	ok(w, "Andamento registrado.")
}

func (h *Handler) financeiro(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.AtualizarFinanceiro(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"),
		r.PostFormValue("item"), r.PostFormValue("valor"), r.PostFormValue("situacao"))
	if h.falha(w, err) {
		return
	}
	ok(w, "Dado financeiro atualizado.")
}

func (h *Handler) providencia(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	d, err := dataOuVazio(r.PostFormValue("ate_quando"))
	if h.falha(w, err) {
		return
	}
	err = h.S.ProximaProvidencia(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"),
		r.PostFormValue("o_que"), d, r.PostFormValue("quem"))
	if h.falha(w, err) {
		return
	}
	ok(w, "Próxima providência registrada.")
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	err := h.S.Delete(r.Context(), webx.ActorOf(w, s.UserID), r.PathValue("id"),
		webx.Int(r.PostFormValue("version")), r.PostFormValue("confirmo") == "1")
	if h.falha(w, err) {
		return
	}
	ok(w, "Processo excluído.")
}
