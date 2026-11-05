package app

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/webx"
)

func dia(delta int) string { return webx.Hoje(time.Now()).AddDate(0, 0, delta).Format("2006-01-02") }

func prazoForm(desc string, delta int) url.Values {
	return url.Values{"descricao": {desc}, "data_fatal": {dia(delta)}, "fonte": {"manual"}, "fundamento": {"CPC art. 335"}}
}

func prazoID(t *testing.T, e *env, desc string) (string, int) {
	t.Helper()
	var id string
	var v int
	if err := e.pool.QueryRow(context.Background(), `SELECT id, version FROM prazos WHERE descricao=$1`, desc).Scan(&id, &v); err != nil {
		t.Fatalf("prazo %q não encontrado: %v", desc, err)
	}
	return id, v
}

func situacao(t *testing.T, e *env, id string) string {
	var s string
	_ = e.pool.QueryRow(context.Background(), `SELECT situacao FROM prazos WHERE id=$1`, id).Scan(&s)
	return s
}

func TestPrazosRegraDeOuro(t *testing.T) {
	e := newEnv(t)
	adv, advID := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)
	ctx := context.Background()

	// Todo prazo nasce A CONFERIR, inclusive o digitado pelo advogado (ADR 0012, item 3; auditoria 05/11, item 1.2).
	// O advogado confirma pelo ato explícito, que grava quem e quando.
	if r, _ := adv.post("/prazos", prazoForm("Contestação do advogado", 4)); r.StatusCode != 303 {
		t.Fatalf("advogado criando prazo = %d", r.StatusCode)
	}
	if r, _ := ass.post("/prazos", prazoForm("Recurso digitado pela assistente", 2)); r.StatusCode != 303 {
		t.Fatalf("assistente criando prazo = %d", r.StatusCode)
	}
	idAdv, _ := prazoID(t, e, "Contestação do advogado")
	idAss, vAss := prazoID(t, e, "Recurso digitado pela assistente")
	if situacao(t, e, idAdv) != "a_conferir" {
		t.Fatalf("prazo do advogado deveria nascer a conferir: %s", situacao(t, e, idAdv))
	}
	if r, _ := adv.post("/prazos/"+idAdv+"/confirmar", url.Values{"version": {"1"}}); r.StatusCode != 303 {
		t.Fatalf("advogado confirmando o próprio prazo = %d", r.StatusCode)
	}
	var por string
	_ = e.pool.QueryRow(ctx, `SELECT coalesce(confirmado_por::text,'') FROM prazos WHERE id=$1`, idAdv).Scan(&por)
	if situacao(t, e, idAdv) != "confirmado" || por != advID {
		t.Fatalf("depois do ato explícito, confirmado por ele: %s / %s", situacao(t, e, idAdv), por)
	}
	if situacao(t, e, idAss) != "a_conferir" {
		t.Fatalf("prazo da assistente deveria nascer a conferir: %s", situacao(t, e, idAss))
	}

	// Assistente NÃO confirma, cumpre, cancela nem exclui.
	for _, acao := range []string{"confirmar", "cumprir", "cancelar", "excluir"} {
		if r, _ := ass.post("/prazos/"+idAss+"/"+acao, url.Values{"version": {"1"}}); r.StatusCode != 403 {
			t.Errorf("assistente em /%s = %d, esperado 403", acao, r.StatusCode)
		}
	}
	if situacao(t, e, idAss) != "a_conferir" {
		t.Fatal("o estado não pode ter mudado")
	}

	// A tela "Hoje" avisa o que está a conferir e mostra o prazo próximo.
	_, home := adv.get("/agenda")
	if !strings.Contains(home, "prazo(s) a conferir") || !strings.Contains(home, "Recurso digitado pela assistente") {
		t.Error("a tela Hoje deveria avisar sobre prazo a conferir")
	}

	// Cumprir exige estar confirmado; depois de confirmar, funciona; versão velha conflita.
	if r, _ := adv.post("/prazos/"+idAss+"/cumprir", url.Values{"version": {"1"}}); r.StatusCode != 409 {
		t.Fatalf("cumprir prazo a conferir = %d, esperado 409", r.StatusCode)
	}
	if r, _ := adv.post("/prazos/"+idAss+"/confirmar", url.Values{"version": {"1"}}); r.StatusCode != 303 {
		t.Fatalf("confirmar = %d", r.StatusCode)
	}
	if r, _ := adv.post("/prazos/"+idAss+"/confirmar", url.Values{"version": {"1"}}); r.StatusCode != 409 {
		t.Fatalf("segunda confirmação com versão velha = %d, esperado 409", r.StatusCode)
	}
	_ = vAss
	if r, _ := adv.post("/prazos/"+idAss+"/cumprir", url.Values{"version": {"2"}}); r.StatusCode != 303 {
		t.Fatalf("cumprir prazo confirmado = %d", r.StatusCode)
	}
	if _, l := adv.get("/prazos"); strings.Contains(l, "Recurso digitado pela assistente") {
		t.Error("prazo cumprido não deveria estar entre os abertos")
	}
	if _, l := adv.get("/prazos?situacao=cumpridos"); !strings.Contains(l, "Recurso digitado pela assistente") {
		t.Error("prazo cumprido deveria estar em 'cumpridos'")
	}
	if r, _ := adv.get("/prazos/" + idAss + "/editar"); r.StatusCode == 200 {
		t.Error("prazo cumprido não pode ser editado")
	}

	// Mudar a data DERRUBA a confirmação, seja quem for que edite; nada confirma por efeito colateral.
	f := prazoForm("Contestação do advogado", 7)
	f.Set("version", "2")
	if r, _ := ass.post("/prazos/"+idAdv, f); r.StatusCode != 303 {
		t.Fatalf("assistente editando = %d", r.StatusCode)
	}
	if situacao(t, e, idAdv) != "a_conferir" {
		t.Fatal("data confirmada alterada por assistente deveria voltar a 'a conferir'")
	}
	_ = e.pool.QueryRow(ctx, `SELECT coalesce(confirmado_por::text,'') FROM prazos WHERE id=$1`, idAdv).Scan(&por)
	if por != "" {
		t.Fatal("a confirmação anterior deveria ter sido apagada")
	}
	f2 := prazoForm("Contestação do advogado", 8)
	f2.Set("version", "3")
	adv.post("/prazos/"+idAdv, f2)
	if situacao(t, e, idAdv) != "a_conferir" {
		t.Fatal("editar (mesmo como advogado) não pode confirmar")
	}
	adv.post("/prazos/"+idAdv+"/confirmar", url.Values{"version": {"4"}})
	if situacao(t, e, idAdv) != "confirmado" {
		t.Fatal("advogado deveria conseguir reconfirmar")
	}

	// Toda escrita tem ator e a cadeia está íntegra.
	for _, a := range []string{"prazo.criado", "prazo.confirmado", "prazo.cumprido", "prazo.alterado"} {
		if e.count(`SELECT count(*) FROM audit_events WHERE action=$1`, a) == 0 {
			t.Errorf("evento %q não foi auditado", a)
		}
	}
	if _, err := audit.Verify(ctx, e.pool); err != nil {
		t.Fatal(err)
	}
}

func TestPrazosUrgenciaEValidacao(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)

	casos := []struct {
		desc   string
		delta  int
		rotulo string
	}{
		{"Vencido ontem", -1, "ATRASADO"}, {"Vence hoje", 0, "VENCE HOJE"}, {"Em três dias", 3, "até 3 dias"},
		{"Em cinco dias", 5, "até 5 dias"}, {"Em dez dias", 10, "até 10 dias"}, {"Daqui a um mês", 30, ""},
	}
	for _, c := range casos {
		if r, _ := adv.post("/prazos", prazoForm(c.desc, c.delta)); r.StatusCode != 303 {
			t.Fatalf("%s: criar = %d", c.desc, r.StatusCode)
		}
	}
	_, lista := adv.get("/prazos")
	// Ordem: do mais urgente ao menos urgente.
	if !(strings.Index(lista, "Vencido ontem") < strings.Index(lista, "Vence hoje") &&
		strings.Index(lista, "Vence hoje") < strings.Index(lista, "Daqui a um mês")) {
		t.Error("a lista deveria vir ordenada pela data fatal")
	}
	for _, c := range casos {
		if c.rotulo != "" && !strings.Contains(lista, c.rotulo) {
			t.Errorf("a lista deveria exibir %q para %q", c.rotulo, c.desc)
		}
	}
	if !strings.Contains(lista, "1 dia de atraso") || !strings.Contains(lista, "faltam 3 dias") {
		t.Error("o texto de dias corridos deveria aparecer")
	}
	// Widget: só até 5 dias (e atrasados).
	_, home := adv.get("/agenda")
	for _, quer := range []string{"Vencido ontem", "Vence hoje", "Em três dias", "Em cinco dias"} {
		if !strings.Contains(home, quer) {
			t.Errorf("a tela Hoje deveria listar %q", quer)
		}
	}
	for _, nao := range []string{"Em dez dias", "Daqui a um mês"} {
		if strings.Contains(home, nao) {
			t.Errorf("a tela Hoje não deveria listar %q", nao)
		}
	}

	// Validações.
	invalidos := map[string]url.Values{
		"sem data":        {"descricao": {"x prazo"}, "fonte": {"manual"}},
		"data impossível": {"descricao": {"x prazo"}, "data_fatal": {"2026-02-31"}, "fonte": {"manual"}},
		"descrição curta": {"descricao": {"a"}, "data_fatal": {dia(3)}, "fonte": {"manual"}},
		"fonte inválida":  {"descricao": {"x prazo"}, "data_fatal": {dia(3)}, "fonte": {"adivinhei"}},
		"processo falso":  {"descricao": {"x prazo"}, "data_fatal": {dia(3)}, "fonte": {"manual"}, "processo_id": {"00000000-0000-0000-0000-000000000000"}},
	}
	for nome, f := range invalidos {
		if r, _ := adv.post("/prazos", f); r.StatusCode != 400 {
			t.Errorf("%s: status %d, esperado 400", nome, r.StatusCode)
		}
	}
	// Vínculo com processo existente funciona e aparece na lista.
	proc := adv.criarProcesso(t, br.MakeCNJ("0000900", "2024", "8", "26", "0100"))
	f := prazoForm("Com processo", 6)
	f.Set("processo_id", proc)
	if r, _ := adv.post("/prazos", f); r.StatusCode != 303 {
		t.Fatalf("prazo com processo = %d", r.StatusCode)
	}
	if _, l := adv.get("/prazos?processo=" + proc); !strings.Contains(l, "Com processo") || strings.Contains(l, "Vence hoje") {
		t.Error("o filtro por processo deveria trazer só o prazo vinculado")
	}
}

// O próprio banco impede um prazo "confirmado" sem quem confirmou (defesa em profundidade).
func TestPrazosBancoExigeConfirmador(t *testing.T) {
	e := newEnv(t)
	_, id := e.user("adv@exemplo.test", auth.Advogado)
	_, err := e.pool.Exec(context.Background(), `INSERT INTO prazos (descricao, data_fatal, situacao, created_by, updated_by)
		VALUES ('sem confirmador', current_date, 'confirmado', $1, $1)`, id)
	if err == nil {
		t.Fatal("o banco deveria recusar prazo confirmado sem confirmado_por")
	}
}
