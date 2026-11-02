package auth

import "testing"

func TestMatrizDePapeis(t *testing.T) {
	casos := []struct {
		r    Role
		p    Perm
		want bool
	}{
		{Assistente, PermPessoasRead, true},
		{Assistente, PermPessoasWrite, false},
		{Assistente, PermProcWrite, false},
		{Assistente, PermTarefaWrite, true},
		{Assistente, PermAgendaWrite, true},
		{Assistente, PermAuditRead, false},
		{Advogado, PermProcWrite, true},
		{Advogado, PermAuditRead, false},
		{Advogado, PermUsersManage, false},
		{Administrador, PermAuditRead, true},
		{Administrador, PermUsersManage, true},
		{Administrador, PermProcWrite, false},
		{Assistente, PermDocRead, true},
		{Assistente, PermDocWrite, false},
		{Advogado, PermDocWrite, true},
		{Assistente, PermAgendaWrite, true},
		{Role("visitante"), PermHome, false},
		{Advogado, Perm("inexistente:x"), false},
	}
	for _, c := range casos {
		if got := Can(c.r, c.p); got != c.want {
			t.Errorf("Can(%s,%s)=%v, esperado %v", c.r, c.p, got, c.want)
		}
	}
}
