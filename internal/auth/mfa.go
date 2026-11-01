package auth

import (
	"image/png"
	"net/http"
	"net/url"
	"strings"

	"github.com/pquerna/otp"

	"painelteodoro/internal/audit"
)

const msgCodigoInvalido = "Código inválido ou já utilizado. Confira o aplicativo e tente de novo."

func (a *Auth) mfaGet(w http.ResponseWriter, r *http.Request, s *Session) {
	u, err := a.Store.UserByID(r.Context(), s.UserID)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	keys, _ := a.Store.Keys(r.Context(), u.ID)
	if !u.TOTPConfirmed && len(keys) == 0 {
		http.Redirect(w, r, "/2fa/setup", http.StatusSeeOther)
		return
	}
	a.render(w, r, http.StatusOK, "mfa.html", PageData{Title: "Segundo fator", Email: s.Email,
		HasTOTP: u.TOTPConfirmed, HasKey: len(keys) > 0 && a.WA != nil})
}

// mfaPost aceita o código do aplicativo (6 dígitos) ou um código de recuperação (xxxxx-xxxxx).
func (a *Auth) mfaPost(w http.ResponseWriter, r *http.Request, s *Session) {
	ctx := r.Context()
	code := strings.TrimSpace(r.PostFormValue("codigo"))
	var hasTOTP, hasKey bool
	bad := func(motivo string, status int) {
		_ = a.audit(r, w, nil, audit.Event{Actor: s.UserID, Action: "mfa.falha", Result: audit.Falha,
			Meta: map[string]string{"motivo": motivo, "etapa": "2fa"}})
		a.render(w, r, status, "mfa.html", PageData{Title: "Segundo fator", Email: s.Email, Error: msgCodigoInvalido,
			HasTOTP: hasTOTP, HasKey: hasKey})
	}
	if !a.Limiter.Allow("mfa|" + clientIP(r)) {
		bad("limite_ip", http.StatusTooManyRequests)
		return
	}
	u, err := a.Store.UserByID(ctx, s.UserID)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	keys, _ := a.Store.Keys(ctx, u.ID)
	hasTOTP, hasKey = u.TOTPConfirmed, len(keys) > 0 && a.WA != nil
	if !u.TOTPConfirmed && len(keys) == 0 {
		http.Redirect(w, r, "/2fa/setup", http.StatusSeeOther)
		return
	}
	now := a.now()
	if u.IsLocked(now) {
		bad("conta_bloqueada", http.StatusUnauthorized)
		return
	}
	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ok, via := false, "totp"
	if strings.Contains(code, "-") {
		via = "recuperacao"
		ok, err = a.Store.UseRecoveryCode(ctx, tx, u.ID, code)
	} else if secret, e := TOTPSecret(a.FieldKey, u); e == nil && u.TOTPConfirmed {
		var step int64
		if step, ok = CheckTOTP(secret, code, u.TOTPLastStep, now); ok {
			err = a.Store.UseTOTPStep(ctx, tx, u.ID, step)
			ok = err == nil
		}
	}
	if !ok {
		_ = tx.Rollback(ctx)
		n, _ := a.Store.RegisterFailure(ctx, u.ID, now)
		if d, perm := LockSchedule(n); d > 0 || perm {
			_ = a.audit(r, w, nil, audit.Event{Actor: u.ID, Action: "conta.bloqueada", TargetType: "usuario",
				TargetID: u.ID, Result: audit.OK})
		}
		bad("codigo_"+via, http.StatusUnauthorized)
		return
	}
	tok, err := a.Store.PromoteSession(ctx, tx, s.Token, u.ID, now)
	if err == nil {
		err = a.audit(r, w, tx, audit.Event{Actor: u.ID, Action: "login.completo", TargetType: "usuario",
			TargetID: u.ID, Result: audit.OK, Meta: map[string]string{"etapa": via, "papel": string(u.Role)}})
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	_ = a.Store.ResetFailures(ctx, u.ID)
	setCookie(w, sidCookie, tok, int(AbsoluteTimeout.Seconds()))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// setupGet mostra o segredo (e o QR) para cadastrar o aplicativo autenticador.
func (a *Auth) setupGet(w http.ResponseWriter, r *http.Request, s *Session) {
	u, secret, ok := a.pendingSecret(w, r, s)
	if !ok {
		return
	}
	if a.hasFactor(r, u) {
		http.Redirect(w, r, "/2fa", http.StatusSeeOther)
		return
	}
	a.render(w, r, http.StatusOK, "setup.html", PageData{Title: "Ativar segundo fator", Email: s.Email,
		Secret: groups4(secret), HasKey: a.WA != nil})
}

func (a *Auth) setupQR(w http.ResponseWriter, r *http.Request, s *Session) {
	u, secret, ok := a.pendingSecret(w, r, s)
	if !ok || a.hasFactor(r, u) {
		http.NotFound(w, r)
		return
	}
	uri := (&url.URL{Scheme: "otpauth", Host: "totp", Path: "/" + issuer + ":" + s.Email,
		RawQuery: url.Values{"secret": {secret}, "issuer": {issuer}}.Encode()}).String()
	key, err := otp.NewKeyFromURL(uri)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	img, err := key.Image(220, 220)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_ = png.Encode(w, img)
}

// pendingSecret devolve o segredo ainda não confirmado, criando-o na primeira visita.
func (a *Auth) pendingSecret(w http.ResponseWriter, r *http.Request, s *Session) (*User, string, bool) {
	u, err := a.Store.UserByID(r.Context(), s.UserID)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return nil, "", false
	}
	if a.hasFactor(r, u) {
		return u, "", true
	}
	if secret, err := TOTPSecret(a.FieldKey, u); err == nil {
		return u, secret, true
	}
	key, err := NewTOTPKey(u.Email)
	if err == nil {
		err = a.Store.SaveTOTPSecret(r.Context(), a.FieldKey, u.ID, key.Secret())
	}
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return nil, "", false
	}
	return u, key.Secret(), true
}

func (a *Auth) setupPost(w http.ResponseWriter, r *http.Request, s *Session) {
	ctx := r.Context()
	u, secret, ok := a.pendingSecret(w, r, s)
	if !ok {
		return
	}
	if a.hasFactor(r, u) {
		http.Redirect(w, r, "/2fa", http.StatusSeeOther)
		return
	}
	step, valid := CheckTOTP(secret, r.PostFormValue("codigo"), 0, a.now())
	if !valid {
		_ = a.audit(r, w, nil, audit.Event{Actor: u.ID, Action: "mfa.cadastro_falha", Result: audit.Falha,
			Meta: map[string]string{"etapa": "cadastro"}})
		a.render(w, r, http.StatusUnauthorized, "setup.html", PageData{Title: "Ativar segundo fator",
			Email: s.Email, Secret: groups4(secret), Error: msgCodigoInvalido, HasKey: a.WA != nil})
		return
	}
	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var codes []string
	var tok string
	err = a.Store.ConfirmTOTP(ctx, tx, u.ID, step)
	if err == nil {
		codes, err = a.Store.NewRecoveryCodes(ctx, tx, u.ID)
	}
	if err == nil {
		tok, err = a.Store.PromoteSession(ctx, tx, s.Token, u.ID, a.now())
	}
	if err == nil {
		err = a.audit(r, w, tx, audit.Event{Actor: u.ID, Action: "mfa.ativado", TargetType: "usuario",
			TargetID: u.ID, Result: audit.OK, Meta: map[string]string{"etapa": "cadastro"}})
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	setCookie(w, sidCookie, tok, int(AbsoluteTimeout.Seconds()))
	// O token CSRF desta resposta deve acompanhar a NOVA sessão.
	r = withValue(r, sidKey{}, tok)
	a.render(w, r, http.StatusOK, "recovery.html", PageData{Title: "Códigos de recuperação", Email: s.Email, Codes: codes})
}

// hasFactor diz se o usuário já tem algum segundo fator (TOTP confirmado ou chave física).
func (a *Auth) hasFactor(r *http.Request, u *User) bool {
	if u.TOTPConfirmed {
		return true
	}
	keys, _ := a.Store.Keys(r.Context(), u.ID)
	return len(keys) > 0
}

func groups4(s string) string {
	var b strings.Builder
	for i, c := range s {
		if i > 0 && i%4 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(c)
	}
	return b.String()
}
