package auth

import (
	"net/url"
	"testing"
)

// Auditoria 05/11, item 3.5: a gestão de usuários só responde 404 quando o usuário não existe. Antes, desbloquear um
// id inexistente "dava certo" (303) e gravava auditoria de um usuário fantasma; e qualquer falha virava 404.
func TestGestaoDeUsuarioInexistente(t *testing.T) {
	e := newEnv(t)
	e.user("adm@exemplo.test", Administrador, senhaBoa)
	adm := e.fullLogin("adm@exemplo.test", senhaBoa)
	tok := adm.csrfFrom("/admin/usuarios")
	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "nao-e-uuid"} {
		if r, _ := adm.do("POST", "/admin/usuarios/"+id+"/desbloquear", url.Values{"csrf_token": {tok}}); r.StatusCode != 404 {
			t.Errorf("desbloquear %q = %d, esperado 404", id, r.StatusCode)
		}
	}
	var n int
	_ = e.pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_events WHERE action='usuario.desbloqueado'`).Scan(&n)
	if n != 0 {
		t.Errorf("auditoria de usuário inexistente: %d evento(s)", n)
	}
}
