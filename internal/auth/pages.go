package auth

import (
	"net/http"

	"painelteodoro/internal/audit"
)

// RegisterPages liga a tela inicial e a visão de auditoria (somente administrador).
func (a *Auth) RegisterPages(rt *Router) {
	rt.Handle("GET /{$}", PermHome, a.home)
	rt.Handle("GET /admin/auditoria", PermAuditRead, a.auditPage)
}

func (a *Auth) home(w http.ResponseWriter, r *http.Request, s *Session) {
	a.render(w, r, http.StatusOK, "home.html", pageData{Title: "Hoje", Email: s.Email, Role: string(s.Role),
		CanAudit: Can(s.Role, PermAuditRead)})
}

func (a *Auth) auditPage(w http.ResponseWriter, r *http.Request, s *Session) {
	rows, err := audit.Recent(r.Context(), a.Pool, 50)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	d := pageData{Title: "Auditoria", Email: s.Email, Role: string(s.Role), Events: rows, CanAudit: true}
	if n, err := audit.Verify(r.Context(), a.Pool); err != nil {
		d.ChainMsg = err.Error()
	} else {
		d.ChainOK, d.ChainMsg = true, "Cadeia íntegra."
		_ = n
	}
	a.render(w, r, http.StatusOK, "audit.html", d)
}
