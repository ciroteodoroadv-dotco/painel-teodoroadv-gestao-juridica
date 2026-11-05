package auth

import (
	"context"
	"errors"

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

// msgSenhaAtual é o pedido de confirmação mostrado quando a senha atual não veio ou não confere.
const msgSenhaAtual = "Para esta ação, confirme com a SUA senha atual."

// confirmaSenha exige a senha atual de quem está logado antes de ações que permitem tomar uma conta (auditoria 05/11,
// item 2.2): criar usuário, mudar papel, redefinir senha ou 2º fator de outra pessoa, cadastrar/remover chave e regerar
// códigos de recuperação. Senha errada conta tentativa (mesma trava do login) e vai para a auditoria.
// Devolve "" quando confirmou; senão a mensagem para a tela (ok=false) ou erro interno.
func (a *Auth) confirmaSenha(ctx context.Context, s *Session, senha string) (msg string, err error) {
	if senha == "" {
		return msgSenhaAtual, nil
	}
	err = a.Reautenticar(ctx, s.UserID, senha)
	var inv webx.Invalid
	switch {
	case err == nil:
		return "", nil
	case errors.As(err, &inv):
		return inv.Msg + " " + msgSenhaAtual, nil
	}
	return "", err
}

// tentativasPorOrigem é o teto de tentativas por minuto de UMA origem (IP) somando todas as contas. O limite por
// origem+conta é o do Limiter (10/min). Antes era só por origem com 10/min: num escritório que sai à internet por um
// único IP, uma pessoa errando a senha travava o login de todos (auditoria 05/11, item 2.8).
const tentativasPorOrigem = 60

// limiteDeTentativas aplica os dois limites (origem+conta e origem). Os dois contadores andam sempre.
func (a *Auth) limiteDeTentativas(etapa, ip, conta string) bool {
	porConta := a.Limiter.Allow(etapa + "|" + ip + "|" + conta)
	porOrigem := a.Limiter.AllowMax(etapa+"-origem|"+ip, tentativasPorOrigem)
	return porConta && porOrigem
}
