package app

import "testing"

// Nenhuma rota de domínio pode ficar sem sessão: tudo redireciona ao login.
func TestRotasDeDominioExigemLogin(t *testing.T) {
	e := newEnv(t)
	anon := &browser{e: e, sid: "inexistente"}
	for _, p := range []string{"/pessoas", "/processos", "/tarefas", "/pessoas/nova", "/tarefas/nova", "/atendimentos", "/atendimentos/novo"} {
		if r, _ := anon.get(p); r.StatusCode != 303 || r.Header.Get("Location") != "/login" {
			t.Errorf("GET %s sem sessão = %d %s", p, r.StatusCode, r.Header.Get("Location"))
		}
	}
	if r, _ := anon.post("/pessoas", pessoaForm("X Y", "")); r.StatusCode != 401 {
		t.Errorf("POST sem sessão = %d, esperado 401", r.StatusCode)
	}
}
