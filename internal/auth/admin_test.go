package auth

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

const senhaBoa = "cavalo-bateria-grampo-7"

func TestGestaoDeUsuariosESenhaProvisoria(t *testing.T) {
	e := newEnv(t)
	e.user("adm@exemplo.test", Administrador, senhaBoa)
	e.user("adv@exemplo.test", Advogado, senhaBoa)
	adm := e.fullLogin("adm@exemplo.test", senhaBoa)
	adv := e.fullLogin("adv@exemplo.test", senhaBoa)
	tok := adm.csrfFrom("/admin/usuarios")

	// Quem não é administrador não gere usuários.
	if r, _ := adv.do("POST", "/admin/usuarios", url.Values{"csrf_token": {adv.csrfFrom("/conta")}, "email": {"x@exemplo.test"},
		"papel": {"assistente"}, "senha": {senhaBoa}}); r.StatusCode != 403 {
		t.Fatalf("advogado criando usuário = %d, esperado 403", r.StatusCode)
	}
	// Senha fraca é recusada; boa cria.
	if r, _ := adm.do("POST", "/admin/usuarios", url.Values{"csrf_token": {tok}, "email": {"novo@exemplo.test"},
		"papel": {"assistente"}, "senha": {"curta"}}); r.StatusCode != 400 {
		t.Fatalf("senha curta = %d, esperado 400", r.StatusCode)
	}
	// Sem a senha do administrador (ou com a errada), não cria (auditoria 05/11, item 2.2).
	for _, atual := range []string{"", "senha-errada-do-admin"} {
		if r, _ := adm.do("POST", "/admin/usuarios", url.Values{"csrf_token": {tok}, "email": {"novo@exemplo.test"},
			"papel": {"assistente"}, "senha": {"provisoria-do-escritorio-1"}, "senha_atual": {atual}}); r.StatusCode != 401 {
			t.Fatalf("criar usuário sem confirmar a senha (%q) = %d, esperado 401", atual, r.StatusCode)
		}
	}
	if r, _ := adm.do("POST", "/admin/usuarios", url.Values{"csrf_token": {tok}, "email": {"novo@exemplo.test"},
		"papel": {"assistente"}, "senha": {"provisoria-do-escritorio-1"}, "senha_atual": {senhaBoa}}); r.StatusCode != 303 {
		t.Fatalf("criar usuário = %d", r.StatusCode)
	}

	// O novo usuário: senha provisória → 2FA → obrigado a trocar a senha antes de qualquer coisa.
	novo := e.fullLogin("novo@exemplo.test", "provisoria-do-escritorio-1")
	if r, _ := novo.do("GET", "/", nil); r.StatusCode != 303 || r.Header.Get("Location") != "/conta/senha" {
		t.Fatalf("senha provisória deveria forçar a troca: %d %s", r.StatusCode, r.Header.Get("Location"))
	}
	tk := novo.csrfFrom("/conta/senha")
	if r, _ := novo.do("POST", "/conta/senha", url.Values{"csrf_token": {tk}, "atual": {"errada-errada-1"},
		"nova": {"outra-frase-bem-longa-9"}, "confirmar": {"outra-frase-bem-longa-9"}}); r.StatusCode != 401 {
		t.Fatalf("senha atual errada = %d", r.StatusCode)
	}
	if r, _ := novo.do("POST", "/conta/senha", url.Values{"csrf_token": {tk}, "atual": {"provisoria-do-escritorio-1"},
		"nova": {"password1234"}, "confirmar": {"password1234"}}); r.StatusCode != 400 {
		t.Fatalf("senha comum = %d, esperado 400", r.StatusCode)
	}
	if r, _ := novo.do("POST", "/conta/senha", url.Values{"csrf_token": {tk}, "atual": {"provisoria-do-escritorio-1"},
		"nova": {"outra-frase-bem-longa-9"}, "confirmar": {"outra-frase-bem-longa-9"}}); r.StatusCode != 303 {
		t.Fatalf("troca válida = %d", r.StatusCode)
	}
	if r, _ := novo.do("GET", "/", nil); r.StatusCode != 302 {
		t.Fatalf("após trocar a senha a home deveria abrir: %d", r.StatusCode)
	}

	// Administrador muda o papel: as sessões do alvo caem; não pode mexer em si mesmo.
	var novoID, admID string
	ctx := context.Background()
	_ = e.pool.QueryRow(ctx, "SELECT id FROM users WHERE email='novo@exemplo.test'").Scan(&novoID)
	_ = e.pool.QueryRow(ctx, "SELECT id FROM users WHERE email='adm@exemplo.test'").Scan(&admID)
	if r, _ := adm.do("POST", "/admin/usuarios/"+novoID+"/papel", url.Values{"csrf_token": {tok}, "papel": {"advogado"}, "senha_atual": {senhaBoa}}); r.StatusCode != 303 {
		t.Fatalf("mudar papel = %d", r.StatusCode)
	}
	if r, _ := novo.do("GET", "/", nil); r.StatusCode != 303 {
		t.Fatalf("sessão do usuário deveria cair após mudança de papel: %d", r.StatusCode)
	}
	if r, _ := adm.do("POST", "/admin/usuarios/"+admID+"/ativo", url.Values{"csrf_token": {tok}, "ativo": {"0"}}); r.StatusCode != 409 {
		t.Fatalf("administrador desativando a si mesmo = %d, esperado 409", r.StatusCode)
	}
	// Desativado não entra mais.
	adm.do("POST", "/admin/usuarios/"+novoID+"/ativo", url.Values{"csrf_token": {tok}, "ativo": {"0"}})
	b := e.browser()
	if r := b.password("novo@exemplo.test", "outra-frase-bem-longa-9"); r.StatusCode != 401 {
		t.Fatalf("usuário desativado entrou: %d", r.StatusCode)
	}
	_, page := adm.do("GET", "/admin/usuarios", nil)
	if !strings.Contains(page, "novo@exemplo.test") || !strings.Contains(page, "desativado") {
		t.Error("a lista deveria mostrar o usuário desativado")
	}
	for _, a := range []string{"usuario.criado", "usuario.papel_alterado", "usuario.desativado", "senha.alterada"} {
		if e.events(a) == 0 {
			t.Errorf("evento %q não foi auditado", a)
		}
	}
}

func TestCodigosDeRecuperacaoNovos(t *testing.T) {
	e := newEnv(t)
	e.user("rec@exemplo.test", Advogado, senhaBoa)
	b := e.fullLogin("rec@exemplo.test", senhaBoa)
	if r, _ := b.do("POST", "/conta/recuperacao", url.Values{"csrf_token": {b.csrfFrom("/conta")}}); r.StatusCode != 401 {
		t.Fatalf("regerar códigos sem a senha atual = %d, esperado 401", r.StatusCode)
	}
	_, body := b.do("POST", "/conta/recuperacao", url.Values{"csrf_token": {b.csrfFrom("/conta")}, "senha_atual": {senhaBoa}})
	if n := len(reCodes.FindAllString(body, -1)); n != 8 {
		t.Fatalf("novos códigos = %d, esperado 8", n)
	}
}

func TestAdminRedefineSegundoFator(t *testing.T) {
	e := newEnv(t)
	e.user("adm@exemplo.test", Administrador, senhaBoa)
	e.user("perdeu@exemplo.test", Advogado, senhaBoa)
	adm := e.fullLogin("adm@exemplo.test", senhaBoa)
	vitima := e.fullLogin("perdeu@exemplo.test", senhaBoa) // tem TOTP e sessão ativa
	var id string
	_ = e.pool.QueryRow(context.Background(), "SELECT id FROM users WHERE email='perdeu@exemplo.test'").Scan(&id)

	if r, _ := adm.do("POST", "/admin/usuarios/"+id+"/mfa/redefinir", url.Values{"csrf_token": {adm.csrfFrom("/admin/usuarios")}}); r.StatusCode != 401 {
		t.Fatalf("redefinir 2º fator sem a senha do administrador = %d, esperado 401", r.StatusCode)
	}
	if r, _ := adm.do("POST", "/admin/usuarios/"+id+"/mfa/redefinir", url.Values{"csrf_token": {adm.csrfFrom("/admin/usuarios")}, "senha_atual": {senhaBoa}}); r.StatusCode != 303 {
		t.Fatalf("redefinir 2º fator = %d", r.StatusCode)
	}
	if r, _ := vitima.do("GET", "/", nil); r.StatusCode != 303 {
		t.Fatalf("a sessão da pessoa deveria cair: %d", r.StatusCode)
	}
	// Próximo acesso: senha ok → volta ao CADASTRO (não ao código).
	b := e.browser()
	if r := b.password("perdeu@exemplo.test", senhaBoa); r.Header.Get("Location") != "/2fa/setup" {
		t.Fatalf("depois da redefinição deveria recadastrar: %s", r.Header.Get("Location"))
	}
	if e.events("usuario.mfa_redefinido") != 1 {
		t.Error("a redefinição deveria estar auditada")
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), "SELECT count(*) FROM recovery_codes WHERE user_id=$1", id).Scan(&n)
	if n != 0 {
		t.Errorf("códigos de recuperação antigos deveriam ter sido apagados: %d", n)
	}
}
