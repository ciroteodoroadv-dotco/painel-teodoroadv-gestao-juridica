package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"painelteodoro/internal/audit"
)

const msgChaveFalhou = "Não foi possível validar a chave de segurança. Tente de novo ou use outro método."

// RegisterWebAuthn liga as rotas da chave física (passkey). Sem a.WA configurado, ficam desligadas.
func (a *Auth) RegisterWebAuthn(rt *Router) {
	rt.Handle("POST /2fa/webauthn/register/begin", AnySession, a.waRegisterBegin)
	rt.Handle("POST /2fa/webauthn/register/finish", AnySession, a.waRegisterFinish)
	rt.Handle("POST /2fa/webauthn/login/begin", MFAPending, a.waLoginBegin)
	rt.Handle("POST /2fa/webauthn/login/finish", MFAPending, a.waLoginFinish)
	rt.Handle("GET /2fa/recovery", AnyLoggedIn, a.recoveryGet)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"erro": msg})
}

func (a *Auth) loadWA(ctx context.Context, userID string) (*waUser, error) {
	u, err := a.Store.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	creds, err := a.Store.Credentials(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &waUser{u: u, creds: creds}, nil
}

func (a *Auth) waRegisterBegin(w http.ResponseWriter, r *http.Request, s *Session) {
	if a.WA == nil {
		jsonErr(w, http.StatusNotFound, "Chave de segurança indisponível.")
		return
	}
	wu, err := a.loadWA(r.Context(), s.UserID)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "Erro interno.")
		return
	}
	// Sessão ainda sem 2FA só pode cadastrar o PRIMEIRO fator; quem já tem fator precisa antes validá-lo.
	if !s.MFAVerified && (wu.u.TOTPConfirmed || len(wu.creds) > 0) {
		a.deny(w, r, s, "cadastro_de_fator_sem_2fa", http.StatusForbidden)
		return
	}
	creation, sd, err := a.WA.BeginRegistration(wu,
		webauthn.WithExclusions(webauthn.Credentials(wu.creds).CredentialDescriptors()),
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementDiscouraged))
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "Erro interno.")
		return
	}
	a.cer.put("reg|"+s.Token, sd, a.now())
	writeJSON(w, http.StatusOK, creation)
}

func (a *Auth) waRegisterFinish(w http.ResponseWriter, r *http.Request, s *Session) {
	ctx := r.Context()
	sd, err := a.cer.take("reg|"+s.Token, a.now())
	if a.WA == nil || err != nil {
		jsonErr(w, http.StatusBadRequest, msgChaveFalhou)
		return
	}
	wu, err := a.loadWA(ctx, s.UserID)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "Erro interno.")
		return
	}
	if !s.MFAVerified && (wu.u.TOTPConfirmed || len(wu.creds) > 0) {
		a.deny(w, r, s, "cadastro_de_fator_sem_2fa", http.StatusForbidden)
		return
	}
	cred, err := a.WA.FinishRegistration(wu, *sd, r)
	if err != nil {
		_ = a.audit(r, w, nil, audit.Event{Actor: s.UserID, Action: "mfa.chave_cadastro_falha", Result: audit.Falha})
		jsonErr(w, http.StatusBadRequest, msgChaveFalhou)
		return
	}
	name := cleanName(r.URL.Query().Get("nome"))
	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "Erro interno.")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	err = a.Store.AddCredential(ctx, tx, s.UserID, name, cred)
	first := !s.MFAVerified
	var tok string
	var codes []string
	if err == nil && first {
		if codes, err = a.Store.NewRecoveryCodes(ctx, tx, s.UserID); err == nil {
			tok, err = a.Store.PromoteSession(ctx, tx, s.Token, s.UserID, a.now())
		}
	}
	if err == nil {
		err = a.audit(r, w, tx, audit.Event{Actor: s.UserID, Action: "mfa.chave_cadastrada", TargetType: "usuario",
			TargetID: s.UserID, Result: audit.OK, Meta: map[string]string{"etapa": "cadastro"}})
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "Erro interno.")
		return
	}
	if first {
		setCookie(w, sidCookie, tok, int(AbsoluteTimeout.Seconds()))
		a.fl.put(tok, codes)
		writeJSON(w, http.StatusOK, map[string]string{"redirect": "/2fa/recovery"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"redirect": "/conta"})
}

func (a *Auth) waLoginBegin(w http.ResponseWriter, r *http.Request, s *Session) {
	wu, err := a.loadWA(r.Context(), s.UserID)
	if a.WA == nil || err != nil || len(wu.creds) == 0 {
		jsonErr(w, http.StatusBadRequest, msgChaveFalhou)
		return
	}
	assertion, sd, err := a.WA.BeginLogin(wu)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "Erro interno.")
		return
	}
	a.cer.put("login|"+s.Token, sd, a.now())
	writeJSON(w, http.StatusOK, assertion)
}

func (a *Auth) waLoginFinish(w http.ResponseWriter, r *http.Request, s *Session) {
	ctx := r.Context()
	now := a.now()
	bad := func(motivo string, status int) {
		n, _ := a.Store.RegisterFailure(ctx, s.UserID, now)
		if d, perm := LockSchedule(n); d > 0 || perm {
			_ = a.audit(r, w, nil, audit.Event{Actor: s.UserID, Action: "conta.bloqueada", TargetType: "usuario",
				TargetID: s.UserID, Result: audit.OK})
		}
		_ = a.audit(r, w, nil, audit.Event{Actor: s.UserID, Action: "mfa.falha", Result: audit.Falha,
			Meta: map[string]string{"motivo": motivo, "etapa": "chave"}})
		jsonErr(w, status, msgChaveFalhou)
	}
	if !a.Limiter.Allow("mfa|" + clientIP(r)) {
		bad("limite_ip", http.StatusTooManyRequests)
		return
	}
	sd, err := a.cer.take("login|"+s.Token, now)
	wu, err2 := a.loadWA(ctx, s.UserID)
	if a.WA == nil || err != nil || err2 != nil || wu.u.IsLocked(now) {
		bad("cerimonia_ou_bloqueio", http.StatusUnauthorized)
		return
	}
	cred, err := a.WA.FinishLogin(wu, *sd, r)
	if err != nil {
		bad("assinatura", http.StatusUnauthorized)
		return
	}
	if cred.Authenticator.CloneWarning {
		bad("contador_regrediu_chave_clonada", http.StatusUnauthorized)
		return
	}
	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "Erro interno.")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	err = a.Store.TouchCredential(ctx, tx, s.UserID, cred, now)
	var tok string
	if err == nil {
		tok, err = a.Store.PromoteSession(ctx, tx, s.Token, s.UserID, now)
	}
	if err == nil {
		err = a.audit(r, w, tx, audit.Event{Actor: s.UserID, Action: "login.completo", TargetType: "usuario",
			TargetID: s.UserID, Result: audit.OK, Meta: map[string]string{"etapa": "chave", "papel": string(wu.u.Role)}})
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "Erro interno.")
		return
	}
	_ = a.Store.ResetFailures(ctx, s.UserID)
	setCookie(w, sidCookie, tok, int(AbsoluteTimeout.Seconds()))
	writeJSON(w, http.StatusOK, map[string]string{"redirect": "/"})
}

func (a *Auth) recoveryGet(w http.ResponseWriter, r *http.Request, s *Session) {
	codes := a.fl.take(s.Token)
	if len(codes) == 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	a.render(w, r, http.StatusOK, "recovery.html", pageData{Title: "Códigos de recuperação", Email: s.Email,
		Role: string(s.Role), Codes: codes, Next: "/"})
}

func cleanName(n string) string {
	n = strings.TrimSpace(n)
	if n == "" {
		return "Chave de segurança"
	}
	if utf8.RuneCountInString(n) > 60 {
		n = string([]rune(n)[:60])
	}
	return n
}
