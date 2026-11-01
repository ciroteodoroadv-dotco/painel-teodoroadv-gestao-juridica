package auth

import (
	"net/http"
	"strings"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/cfaccess"
)

const msgLoginFalhou = "Não foi possível entrar. Confira e-mail e senha. Após várias tentativas a conta é bloqueada por um tempo."

// Register liga as rotas de acesso ao roteador.
func (a *Auth) Register(rt *Router) {
	rt.Handle("GET /login", Public, a.loginGet)
	rt.Handle("POST /login", Public, a.loginPost)
	rt.Handle("POST /logout", Public, a.logout)
	rt.Handle("GET /2fa", MFAPending, a.mfaGet)
	rt.Handle("POST /2fa", MFAPending, a.mfaPost)
	rt.Handle("GET /2fa/setup", MFAPending, a.setupGet)
	rt.Handle("GET /2fa/setup/qr.png", MFAPending, a.setupQR)
	rt.Handle("POST /2fa/setup", MFAPending, a.setupPost)
}

func (a *Auth) loginGet(w http.ResponseWriter, r *http.Request, _ *Session) {
	if s := a.session(r); s != nil && s.MFAVerified {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	a.render(w, r, http.StatusOK, "login.html", PageData{Title: "Entrar"})
}

func (a *Auth) loginPost(w http.ResponseWriter, r *http.Request, _ *Session) {
	ctx := r.Context()
	email := strings.TrimSpace(r.PostFormValue("email"))
	pw := r.PostFormValue("senha")
	fail := func(actor, motivo string, status int) {
		_ = a.audit(r, w, nil, audit.Event{Actor: actor, Action: "login.falha", Result: audit.Falha,
			Meta: map[string]string{"motivo": motivo, "email_hash": emailHash(email)}})
		a.render(w, r, status, "login.html", PageData{Title: "Entrar", Error: msgLoginFalhou, Email: email})
	}
	if !a.Limiter.Allow("login|" + clientIP(r)) {
		fail("anonimo", "limite_ip", http.StatusTooManyRequests)
		return
	}
	if a.RequireAccessEmail && !strings.EqualFold(cfaccess.Email(ctx), email) {
		burnPassword(pw)
		fail("anonimo", "email_diferente_do_access", http.StatusUnauthorized)
		return
	}
	u, err := a.Store.UserByEmail(ctx, email)
	if err != nil || !u.Active {
		burnPassword(pw)
		fail("anonimo", "credenciais", http.StatusUnauthorized)
		return
	}
	now := a.now()
	if u.IsLocked(now) {
		fail(u.ID, "conta_bloqueada", http.StatusUnauthorized)
		return
	}
	if !VerifyPassword(pw, u.PasswordHash) {
		n, _ := a.Store.RegisterFailure(ctx, u.ID, now)
		if d, perm := LockSchedule(n); d > 0 || perm {
			_ = a.audit(r, w, nil, audit.Event{Actor: u.ID, Action: "conta.bloqueada", TargetType: "usuario",
				TargetID: u.ID, Result: audit.OK})
		}
		fail(u.ID, "senha", http.StatusUnauthorized)
		return
	}
	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tok, err := a.Store.CreateSession(ctx, tx, u.ID, false, now)
	if err == nil {
		err = a.audit(r, w, tx, audit.Event{Actor: u.ID, Action: "login.senha_ok", TargetType: "usuario",
			TargetID: u.ID, Result: audit.OK, Meta: map[string]string{"etapa": "senha"}})
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	setCookie(w, sidCookie, tok, int(AbsoluteTimeout.Seconds()))
	next := "/2fa"
	if !u.TOTPConfirmed {
		next = "/2fa/setup"
	}
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (a *Auth) logout(w http.ResponseWriter, r *http.Request, _ *Session) {
	if s := a.session(r); s != nil {
		_ = a.Store.DeleteSession(r.Context(), s.Token)
		_ = a.audit(r, w, nil, audit.Event{Actor: s.UserID, Action: "logout", TargetType: "usuario",
			TargetID: s.UserID, Result: audit.OK})
	}
	setCookie(w, sidCookie, "", -1)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
