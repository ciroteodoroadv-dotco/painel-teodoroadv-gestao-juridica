package auth

import (
	"context"
	"net/http"

	"painelteodoro/internal/audit"
)

// RegisterPages liga a tela inicial (a Agenda) e a visão de auditoria (somente administrador).
func (a *Auth) RegisterPages(rt *Router) {
	rt.Handle("GET /{$}", PermHome, a.home)
	rt.Handle("GET /admin/auditoria", PermAuditRead, a.auditPage)
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
	if n, err := audit.Verify(r.Context(), a.Pool); err != nil {
		d.ChainMsg = err.Error()
	} else {
		d.ChainOK, d.ChainMsg = true, "Cadeia íntegra."
		_ = n
	}
	a.render(w, r, http.StatusOK, "audit.html", d)
}
