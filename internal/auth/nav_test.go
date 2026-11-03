package auth

import "testing"

func TestMenuFiltraPorPermissaoEOrdena(t *testing.T) {
	a := &Auth{}
	a.AddNavCount("Tarefas", "/tarefas", PermTarefaRead, nil)
	a.AddNavCount("Agenda", "/agenda", PermAgendaRead, nil)
	a.AddNavCount("Segredo", "/segredo", PermUsersManage, nil)
	a.AddNavSistema("Prazos", "/prazos", PermPrazoRead)

	principal, sistema := a.menu(nil, Assistente, "/tarefas/abc")
	if len(principal) != 2 || principal[0].Label != "Agenda" || principal[1].Label != "Tarefas" {
		t.Fatalf("menu principal: %+v", principal)
	}
	if !principal[1].Active || principal[0].Active {
		t.Errorf("item ativo errado: %+v", principal)
	}
	if len(sistema) != 1 || sistema[0].Label != "Prazos" {
		t.Errorf("menu do sistema: %+v", sistema)
	}
	if p, _ := a.menu(nil, Administrador, "/"); len(p) != 3 || !p[0].Active {
		t.Errorf("a raiz deve marcar a Agenda: %+v", p)
	}
}
