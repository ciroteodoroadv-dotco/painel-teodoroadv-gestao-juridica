package auth

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"painelteodoro/internal/platform/ratelimit"
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

// Auditoria 05/11, item 2.2: com a sessão completa, cadastrar uma chave ADICIONAL exige a senha atual.
func TestChaveAdicionalExigeSenhaAtual(t *testing.T) {
	e := newEnv(t)
	e.user("adv@exemplo.test", Advogado, senhaBoa)
	b := e.fullLogin("adv@exemplo.test", senhaBoa)
	tok := b.csrfFrom("/conta")
	if r, body := b.jsonPost("/2fa/webauthn/register/begin", tok, nil); r.StatusCode != 401 || !strings.Contains(body, "senha atual") {
		t.Fatalf("chave adicional sem senha = %d %s, esperado 401", r.StatusCode, body)
	}
	if r, _ := b.jsonPost("/2fa/webauthn/register/begin", tok, []byte(`{"senha":"errada-errada-errada"}`)); r.StatusCode != 401 {
		t.Fatalf("chave adicional com senha errada = %d, esperado 401", r.StatusCode)
	}
	if r, body := b.jsonPost("/2fa/webauthn/register/begin", tok, []byte(`{"senha":"`+senhaBoa+`"}`)); r.StatusCode != 200 || !strings.Contains(body, "publicKey") {
		t.Fatalf("chave adicional com a senha = %d %s", r.StatusCode, body)
	}
}

// Auditoria 05/11, item 2.8: no mesmo IP, alguém esgotando as tentativas da PRÓPRIA conta não trava a dos colegas;
// o teto por origem (60/min) continua valendo para tentativas espalhadas por muitas contas.
func TestLimiteDeLoginPorContaNaoTravaColegas(t *testing.T) {
	e := newEnv(t)
	e.a.Limiter = ratelimit.New(10, time.Minute)
	e.user("ana@exemplo.test", Advogado, senhaBoa)
	e.user("bia@exemplo.test", Advogado, senhaBoa)
	b := e.browser()
	for i := 0; i < 11; i++ {
		b.password("ana@exemplo.test", "senha-errada-numero-x")
	}
	if r := b.password("ana@exemplo.test", senhaBoa); r.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("a própria conta deveria estar no limite: %d", r.StatusCode)
	}
	if r := e.browser().password("bia@exemplo.test", senhaBoa); r.StatusCode != http.StatusSeeOther {
		t.Fatalf("colega no mesmo IP = %d, esperado 303 (login segue)", r.StatusCode)
	}
	for i := 0; i < 60; i++ {
		e.browser().password("x"+strconv.Itoa(i)+"@exemplo.test", "qualquer-coisa-aqui")
	}
	if r := e.browser().password("bia@exemplo.test", senhaBoa); r.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("passado o teto por origem = %d, esperado 429", r.StatusCode)
	}
}

// Auditoria 05/11, item 2.3: o advogado vê na Agenda que o administrador criou/alterou usuários; o assistente não.
func TestAdvogadoVeAlteracoesDeUsuarios(t *testing.T) {
	e := newEnv(t)
	e.user("adm@exemplo.test", Administrador, senhaBoa)
	e.user("adv@exemplo.test", Advogado, senhaBoa)
	e.user("ass@exemplo.test", Assistente, senhaBoa)
	f, err := e.a.faixaMudancasDeUsuarios(t.Context(), &Session{Role: Advogado})
	if err != nil || f.Strong != "" {
		t.Fatalf("sem alteração, sem faixa: %+v %v", f, err)
	}
	adm := e.fullLogin("adm@exemplo.test", senhaBoa)
	if r, _ := adm.do("POST", "/admin/usuarios", url.Values{"csrf_token": {adm.csrfFrom("/admin/usuarios")}, "email": {"novo@exemplo.test"},
		"papel": {"advogado"}, "senha": {"provisoria-do-escritorio-1"}, "senha_atual": {senhaBoa}}); r.StatusCode != 303 {
		t.Fatalf("criar = %d", r.StatusCode)
	}
	if f, _ := e.a.faixaMudancasDeUsuarios(t.Context(), &Session{Role: Advogado}); !strings.Contains(f.Text, "1 alteração") {
		t.Fatalf("advogado deveria ver 1 alteração: %+v", f)
	}
	if f, _ := e.a.faixaMudancasDeUsuarios(t.Context(), &Session{Role: Assistente}); f.Strong != "" {
		t.Fatalf("assistente não recebe a faixa: %+v", f)
	}
}
