package indices

import (
	"html/template"
	"net/http"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/coletas"
	"painelteodoro/internal/platform/webx"
)

// Handler liga o rodapé e a página do administrador ao serviço.
type Handler struct {
	A *auth.Auth
	S *Service
}

var paginaTpl = template.Must(auth.BaseTemplate().ParseFS(tplFS, "templates/pagina.html"))

// Register declara as rotas. O fragmento só lê o último estado gravado (nunca coleta); a página e a coleta manual são do administrador.
func (h *Handler) Register(rt *auth.Router) {
	h.A.SetTickerSource(h.S.Fragmento)
	h.A.AddNavSistema("Índices", "/admin/indices", auth.PermColetaManage)
	rt.Handle("GET /fragmentos/indices", auth.AnyLoggedIn, h.fragmento)
	rt.Handle("GET /admin/indices", auth.PermColetaManage, h.pagina)
	rt.Handle("POST /admin/indices/atualizar", auth.PermColetaManage, h.atualizar)
}

func (h *Handler) fragmento(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	html, err := h.S.Fragmento(r.Context())
	if err != nil {
		http.Error(w, "Índices indisponíveis.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Painel-Fragmento", "indices")
	_, _ = w.Write([]byte(html))
}

type linha struct{ Rotulo, Valor, Ref, Obtido, Situacao string }
type paginaView struct {
	Ativa  bool
	Estado string
	Linhas []linha
}

func (h *Handler) pagina(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	vs, err := h.S.Repo().Listar(r.Context())
	e, existe, err2 := h.S.Coletas.Obter(r.Context(), Modulo)
	if err != nil || err2 != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	v := paginaView{Ativa: h.S.Ativo, Estado: "Nenhuma coleta registrada ainda."}
	if existe {
		v.Estado, _ = e.Linha()
	}
	t := Montar(vs, coletas.Estado{}, false, h.S.now(), h.S.validade())
	for i, x := range vs {
		sit := "atual"
		if t.Itens[i].Velho {
			sit = "desatualizado"
		}
		v.Linhas = append(v.Linhas, linha{Rotulo: x.Rotulo, Valor: t.Itens[i].Valor + " (" + x.Unidade + ")", Ref: x.Referencia.Format("02/01/2006"),
			Obtido: x.ObtidoEm.In(webx.SP()).Format("02/01/2006 15:04"), Situacao: sit})
	}
	pd := auth.PageData{Title: "Índices econômicos", Email: s.Email, Role: string(s.Role), Data: v}
	pd.Head = auth.Head{Title: "Índices econômicos", Sub: "Faixa de índices do rodapé: valores do Banco Central (SGS), sempre datados."}
	h.A.Render(w, r, http.StatusOK, paginaTpl, pd)
}

func (h *Handler) atualizar(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	res, err := h.S.Coletar(r.Context())
	if webx.Falha(w, err, "", "Falha ao atualizar os índices.") {
		return
	}
	switch res.Status {
	case "bloqueado", "falha":
		webx.Falha(w, webx.Invalidf(res.Mensagem), "")
	case "parcial":
		webx.Ok(w, "Índices parciais. "+res.Mensagem)
	default:
		webx.Ok(w, "Índices atualizados. Para cálculo judicial, confira sempre a fonte oficial.")
	}
}
