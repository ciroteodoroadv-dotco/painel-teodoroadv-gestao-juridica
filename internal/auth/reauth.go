package auth

import (
	"context"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/webx"
)

// Reautenticar confere a senha do USUÁRIO LOGADO antes de uma ação destrutiva (exclusão de tarefa). Não existe
// "senha de administrador" única: cada pessoa confirma com a própria senha (Argon2id). Senha errada conta como
// tentativa falha (mesma trava de bloqueio do login) e vai para a auditoria.
func (a *Auth) Reautenticar(ctx context.Context, userID, senha string) error {
	u, err := a.Store.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	now := a.now()
	if !a.Limiter.Allow("reauth|"+userID) || u.IsLocked(now) {
		return webx.Invalidf("Muitas tentativas. Aguarde alguns minutos.")
	}
	if !VerifyPassword(senha, u.PasswordHash) {
		_, _ = a.Store.RegisterFailure(ctx, u.ID, now)
		_ = audit.RecordPool(ctx, a.Pool, audit.Event{Actor: u.ID, Action: "senha.reautenticacao_falha", Result: audit.Falha})
		return webx.Invalidf("Senha incorreta.")
	}
	return nil
}
