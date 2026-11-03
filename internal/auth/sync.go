package auth

import (
	"context"
	"net/http"

	"painelteodoro/internal/platform/coletas"
)

// SyncLinha é uma linha do rodapé "Estado das coletas"; Erro pinta a linha com sync-error (como no B).
type SyncLinha struct {
	Texto string
	Erro  bool
}

// SyncView alimenta o rodapé. Indisponivel = não deu para ler o histórico local (o rodapé diz isso, sem inventar estado).
type SyncView struct {
	Linhas       []SyncLinha
	Indisponivel bool
}

// SetSyncSource registra de onde vem o estado das coletas (o histórico local; nunca consulta fonte externa).
func (a *Auth) SetSyncSource(f func(context.Context) ([]coletas.Estado, error)) { a.sync = f }

func (a *Auth) syncView(ctx context.Context) SyncView {
	if a.sync == nil {
		return SyncView{}
	}
	es, err := a.sync(ctx)
	if err != nil {
		a.Log.Error("falha ao ler o estado das coletas")
		return SyncView{Indisponivel: true}
	}
	v := SyncView{}
	for _, e := range es {
		t, erro := e.Linha()
		v.Linhas = append(v.Linhas, SyncLinha{Texto: t, Erro: erro})
	}
	return v
}

// syncFragmento devolve só o miolo do rodapé; o JS (data-refresh) o recarrega a cada 60 s e ao voltar à aba.
func (a *Auth) syncFragmento(w http.ResponseWriter, r *http.Request, _ *Session) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Painel-Fragmento", "sync")
	if err := BaseTemplate().ExecuteTemplate(w, "syncConteudo", a.syncView(r.Context())); err != nil {
		a.Log.Error("falha ao renderizar o rodapé de coletas")
	}
}
