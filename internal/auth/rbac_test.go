package auth

import (
	"strings"
	"testing"
)

func TestMatrizDePapeis(t *testing.T) {
	casos := []struct {
		r    Role
		p    Perm
		want bool
	}{
		{Advogado, PermIntimacaoDecide, true},
		{Assistente, PermIntimacaoDecide, false},
		{Administrador, PermIntimacaoDecide, false},
		{Administrador, PermColetaManage, true},
		{Advogado, PermColetaManage, false},
		{Assistente, PermColetaManage, false},
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
		{Assistente, PermPrazoRead, true},
		{Assistente, PermPrazoWrite, true},
		{Assistente, PermPrazoConfirm, false},
		{Advogado, PermPrazoConfirm, true},
		{Administrador, PermPrazoConfirm, false},
		{Assistente, PermDocRead, true},
		{Assistente, PermDocWrite, false},
		{Advogado, PermDocWrite, true},
		{Assistente, PermAgendaWrite, true},
		{Assistente, PermAtendimentoRead, true},
		{Assistente, PermAtendimentoWrite, true},
		{Advogado, PermAtendimentoWrite, true},
		{Administrador, PermAtendimentoRead, true},
		{Administrador, PermAtendimentoWrite, false},
		{Role("visitante"), PermHome, false},
		{Advogado, Perm("inexistente:x"), false},
	}
	for _, c := range casos {
		if got := Can(c.r, c.p); got != c.want {
			t.Errorf("Can(%s,%s)=%v, esperado %v", c.r, c.p, got, c.want)
		}
	}
}

// Nada se apaga (determinação de 04/11/2026): nenhum papel tem permissão de excluir tarefa, audiência, compromisso, diligência ou intimação.
func TestNinguemExcluiRegistrosDeAta(t *testing.T) {
	for _, r := range []Role{Administrador, Advogado, Assistente} {
		for p := range matrix[r] {
			s := string(p)
			if strings.Contains(s, "excluir") && (strings.HasPrefix(s, "tarefas") || strings.HasPrefix(s, "audiencias") ||
				strings.HasPrefix(s, "agenda") || strings.HasPrefix(s, "diligencias") || strings.HasPrefix(s, "intimacoes")) {
				t.Errorf("%s tem a permissão %q", r, p)
			}
		}
	}
}
