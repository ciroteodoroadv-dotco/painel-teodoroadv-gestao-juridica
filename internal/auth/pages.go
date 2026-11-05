package auth

import (
	"context"
	"net/http"
	"sync"
	"time"

	"painelteodoro/internal/audit"
)

// RegisterPages liga a tela inicial (a Agenda) e a visão de auditoria (somente administrador).
func (a *Auth) RegisterPages(rt *Router) {
	rt.Handle("GET /{$}", PermHome, a.home)
	rt.Handle("GET /admin/auditoria", PermAuditRead, a.auditPage)
	rt.Handle("GET /sync/estado", AnyLoggedIn, a.syncFragmento)
}

// home leva à Agenda, que é a página inicial do painel (como no B).
func (a *Auth) home(w http.ResponseWriter, r *http.Request, _ *Session) {
	http.Redirect(w, r, "/agenda", http.StatusFound)
}

// HomeExtras reúne a faixa de aviso e os blocos (lista de prazos) que os domínios registraram para o topo da Agenda.
func (a *Auth) HomeExtras(ctx context.Context, s *Session) (faixas []Faixa, widgets []Widget) {
	for _, f := range a.faixas {
		if fx, err := f(ctx, s); err == nil && (fx.Strong != "" || fx.Text != "") {
			faixas = append(faixas, fx)
		} else if err != nil {
			a.Log.Error("faixa da tela inicial falhou")
		}
	}
	for _, f := range a.widgets {
		if wd, err := f(ctx, s); err == nil && wd.Body != "" {
			widgets = append(widgets, wd)
		} else if err != nil {
			a.Log.Error("widget da tela inicial falhou")
		}
	}
	return faixas, widgets
}

func (a *Auth) auditPage(w http.ResponseWriter, r *http.Request, s *Session) {
	rows, err := audit.Recent(r.Context(), a.Pool, 50)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	d := PageData{Title: "Auditoria", Email: s.Email, Role: string(s.Role), Events: rows, CanAudit: true}
	if err := a.conferirCadeia(r.Context()); err != nil {
		d.ChainMsg = err.Error()
	} else {
		d.ChainOK, d.ChainMsg = true, "Cadeia íntegra."
	}
	a.render(w, r, http.StatusOK, "audit.html", d)
}

// conferirCadeia confere a cadeia de forma incremental a partir do último ponto conferido nesta execução; uma vez por
// dia (e a cada reinício) confere desde o começo. Antes recalculava a cadeia INTEIRA a cada visita (auditoria 05/11,
// item 4.2). Um defeito encontrado zera o ponto: a próxima visita volta a conferir tudo.
func (a *Auth) conferirCadeia(ctx context.Context) error {
	a.cadeia.mu.Lock()
	defer a.cadeia.mu.Unlock()
	agora := a.now()
	if agora.Sub(a.cadeia.completa) >= 24*time.Hour {
		a.cadeia.ponto = audit.Ponto{}
	}
	_, novo, err := audit.VerifyFrom(ctx, a.Pool, a.cadeia.ponto)
	if err != nil {
		a.cadeia.ponto, a.cadeia.completa = audit.Ponto{}, time.Time{}
		return err
	}
	if a.cadeia.ponto.Seq == 0 {
		a.cadeia.completa = agora
	}
	a.cadeia.ponto = novo
	return nil
}

// pontoDaCadeia guarda o último elo conferido pela tela de auditoria.
type pontoDaCadeia struct {
	mu       sync.Mutex
	ponto    audit.Ponto
	completa time.Time // quando foi a última conferência desde o começo
}
