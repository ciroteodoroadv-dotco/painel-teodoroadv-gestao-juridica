package auth

import (
	"net/http"

	"painelteodoro/internal/audit"
)

var avisos = map[string]string{
	"senha":          "Senha alterada. As outras sessões foram encerradas.",
	"chave_removida": "Chave de segurança removida.",
}

// RegisterAccount liga a área "Conta": senha, chaves de segurança e códigos de recuperação.
func (a *Auth) RegisterAccount(rt *Router) {
	rt.Handle("GET /conta", AnyLoggedIn, a.contaGet)
	rt.Handle("GET /conta/senha", AnyLoggedIn, a.senhaGet)
	rt.Handle("POST /conta/senha", AnyLoggedIn, a.senhaPost)
	rt.Handle("POST /conta/chave/remover", AnyLoggedIn, a.chaveRemover)
	rt.Handle("POST /conta/recuperacao", AnyLoggedIn, a.recuperacaoNova)
}

func (a *Auth) contaGet(w http.ResponseWriter, r *http.Request, s *Session) {
	u, err := a.Store.UserByID(r.Context(), s.UserID)
	keys, err2 := a.Store.Keys(r.Context(), s.UserID)
	if err != nil || err2 != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	a.render(w, r, http.StatusOK, "conta.html", PageData{Title: "Minha conta", Email: s.Email, Role: string(s.Role),
		HasTOTP: u.TOTPConfirmed, HasKey: a.WA != nil, Keys: keys, Notice: avisos[r.URL.Query().Get("aviso")]})
}

func (a *Auth) senhaGet(w http.ResponseWriter, r *http.Request, s *Session) {
	a.render(w, r, http.StatusOK, "senha.html", PageData{Title: "Trocar senha", Email: s.Email,
		Role: string(s.Role), MustChange: s.MustChange})
}

func (a *Auth) senhaPost(w http.ResponseWriter, r *http.Request, s *Session) {
	ctx := r.Context()
	fail := func(msg string, status int) {
		a.render(w, r, status, "senha.html", PageData{Title: "Trocar senha", Email: s.Email,
			Role: string(s.Role), MustChange: s.MustChange, Error: msg})
	}
	u, err := a.Store.UserByID(ctx, s.UserID)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	now := a.now()
	if !a.Limiter.Allow("senha|"+s.UserID) || u.IsLocked(now) {
		fail("Muitas tentativas. Aguarde alguns minutos.", http.StatusTooManyRequests)
		return
	}
	atual, nova := r.PostFormValue("atual"), r.PostFormValue("nova")
	if !VerifyPassword(atual, u.PasswordHash) {
		_, _ = a.Store.RegisterFailure(ctx, u.ID, now)
		_ = a.audit(r, w, nil, audit.Event{Actor: u.ID, Action: "senha.troca_falha", Result: audit.Falha})
		fail("A senha atual não confere.", http.StatusUnauthorized)
		return
	}
	if nova != r.PostFormValue("confirmar") {
		fail("A confirmação não é igual à nova senha.", http.StatusBadRequest)
		return
	}
	if nova == atual {
		fail("A nova senha precisa ser diferente da atual.", http.StatusBadRequest)
		return
	}
	if err := CheckPolicy(nova, u.Email); err != nil {
		fail(capitalize(err.Error())+".", http.StatusBadRequest)
		return
	}
	hash, err := HashPassword(nova)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var tok string
	err = a.Store.SetPassword(ctx, tx, u.ID, hash, false) // encerra TODAS as sessões, inclusive esta
	if err == nil {
		tok, err = a.Store.CreateSession(ctx, tx, u.ID, true, now)
	}
	if err == nil {
		err = a.audit(r, w, tx, audit.Event{Actor: u.ID, Action: "senha.alterada", TargetType: "usuario",
			TargetID: u.ID, Result: audit.OK})
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	setCookie(w, sidCookie, tok, int(AbsoluteTimeout.Seconds()))
	http.Redirect(w, r, "/conta?aviso=senha", http.StatusSeeOther)
}

func (a *Auth) chaveRemover(w http.ResponseWriter, r *http.Request, s *Session) {
	ctx := r.Context()
	u, err := a.Store.UserByID(ctx, s.UserID)
	keys, err2 := a.Store.Keys(ctx, s.UserID)
	if err != nil || err2 != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	if len(keys) <= 1 && !u.TOTPConfirmed {
		http.Error(w, "Não é possível remover o único segundo fator.", http.StatusConflict)
		return
	}
	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ok, err := a.Store.DeleteCredential(ctx, tx, s.UserID, r.PostFormValue("id"))
	if err == nil && ok {
		err = a.audit(r, w, tx, audit.Event{Actor: s.UserID, Action: "mfa.chave_removida", TargetType: "usuario",
			TargetID: s.UserID, Result: audit.OK})
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil || !ok {
		http.Error(w, "Chave não encontrada.", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, "/conta?aviso=chave_removida", http.StatusSeeOther)
}

func (a *Auth) recuperacaoNova(w http.ResponseWriter, r *http.Request, s *Session) {
	ctx := r.Context()
	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	codes, err := a.Store.NewRecoveryCodes(ctx, tx, s.UserID)
	if err == nil {
		err = a.audit(r, w, tx, audit.Event{Actor: s.UserID, Action: "mfa.recuperacao_regerada", TargetType: "usuario",
			TargetID: s.UserID, Result: audit.OK})
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	a.render(w, r, http.StatusOK, "recovery.html", PageData{Title: "Códigos de recuperação", Email: s.Email,
		Role: string(s.Role), Codes: codes, Next: "/conta"})
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 32
	}
	return string(r)
}
