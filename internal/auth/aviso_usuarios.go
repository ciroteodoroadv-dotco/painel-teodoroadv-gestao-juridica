package auth

import (
	"context"
	"strconv"
	"time"
)

// Alterações de usuários feitas pelo administrador que um advogado precisa saber (auditoria 05/11, item 2.3): o
// administrador não mexe em processos, mas pode criar um usuário "advogado" ou trocar papéis e fatores de acesso.
// A faixa na Agenda dos advogados torna isso visível a quem responde pelo escritório.
var acoesDeUsuarioAvisadas = []string{"usuario.criado", "usuario.papel_alterado", "usuario.senha_redefinida", "usuario.mfa_redefinido"}

// janelaAvisoUsuarios é quanto tempo a faixa continua aparecendo depois da alteração.
const janelaAvisoUsuarios = 72 * time.Hour

// faixaMudancasDeUsuarios conta as alterações recentes (só olha os últimos 5.000 eventos: a cadeia não tem índice
// por data e a faixa roda a cada abertura da Agenda).
func (a *Auth) faixaMudancasDeUsuarios(ctx context.Context, s *Session) (Faixa, error) {
	if s == nil || s.Role != Advogado {
		return Faixa{}, nil
	}
	var n int
	err := a.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events
		WHERE seq > (SELECT coalesce(max(seq), 0) - 5000 FROM audit_events) AND at >= $1 AND action = ANY($2)`,
		a.now().Add(-janelaAvisoUsuarios), acoesDeUsuarioAvisadas).Scan(&n)
	if err != nil || n == 0 {
		return Faixa{}, err
	}
	return Faixa{Strong: "Acessos alterados pelo administrador:",
		Text: strconv.Itoa(n) + " alteração(ões) de usuário (criação, papel, senha ou 2º fator) nas últimas 72 h. Se não reconhece, fale com o administrador e confira a auditoria."}, nil
}
