package app

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"painelteodoro/internal/auth"
)

func diaForm(cal, data, tipo, fonte string, conta bool) url.Values {
	v := url.Values{"calendario_id": {cal}, "data": {data}, "tipo": {tipo}, "descricao": {"Dia de teste"},
		"fonte_citacao": {fonte}, "fonte_url": {"https://exemplo.test/portaria"}}
	if conta {
		v.Set("conta_nao_util", "1")
	}
	return v
}

func (e *env) calendarioTeste(t *testing.T, b *browser, sigla string) string {
	t.Helper()
	r, _ := b.post("/calendario-forense/calendarios", url.Values{"sigla": {sigla}, "nome": {"Tribunal de " + sigla}, "ambito": {"estadual"}})
	if r.StatusCode != 200 {
		t.Fatalf("criar calendário = %d", r.StatusCode)
	}
	var id string
	if err := e.pool.QueryRow(context.Background(), `SELECT id::text FROM calendarios WHERE sigla=$1`, sigla).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestCalendarioForense(t *testing.T) {
	e := newEnv(t)
	t.Cleanup(func() { _, _ = e.pool.Exec(context.Background(), `TRUNCATE calendario_dias, calendarios CASCADE`) })
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	adm, _ := e.user("adm@exemplo.test", auth.Administrador)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)
	cal := e.calendarioTeste(t, adv, "TJTESTE")

	// Sem fonte não grava; com fonte grava e audita.
	if r, _ := adv.post("/calendario-forense/dias", diaForm(cal, "2027-02-10", "feriado", "", true)); r.StatusCode != 400 {
		t.Errorf("sem fonte = %d, quer 400", r.StatusCode)
	}
	if r, _ := adv.post("/calendario-forense/dias", diaForm(cal, "2027-02-10", "feriado", "ab", true)); r.StatusCode != 400 {
		t.Errorf("fonte curta = %d, quer 400", r.StatusCode)
	}
	if r, _ := adv.post("/calendario-forense/dias", diaForm(cal, "2027-02-10", "feriado", "Portaria TJTESTE 1/2027", true)); r.StatusCode != 200 {
		t.Fatalf("criar dia = %d", r.StatusCode)
	}
	// Duplicidade recusada.
	if r, _ := adv.post("/calendario-forense/dias", diaForm(cal, "2027-02-10", "feriado", "Portaria TJTESTE 2/2027", true)); r.StatusCode != 400 {
		t.Errorf("duplicado = %d, quer 400", r.StatusCode)
	}
	// Permissão: assistente lê mas não escreve (403); administrador escreve.
	if r, _ := ass.post("/calendario-forense/dias", diaForm(cal, "2027-02-11", "feriado", "Portaria TJTESTE 3/2027", true)); r.StatusCode != 403 {
		t.Errorf("assistente escrevendo = %d, quer 403", r.StatusCode)
	}
	if r, _ := ass.get("/calendario-forense/dias/novo"); r.StatusCode != 403 {
		t.Errorf("assistente abrindo drawer = %d, quer 403", r.StatusCode)
	}
	if r, _ := adm.post("/calendario-forense/dias", diaForm(cal, "2027-02-12", "feriado", "Portaria TJTESTE 4/2027", true)); r.StatusCode != 200 {
		t.Errorf("administrador escrevendo = %d, quer 200", r.StatusCode)
	}
	// Ponto facultativo como não útil: só o advogado marca.
	if r, _ := adm.post("/calendario-forense/dias", diaForm(cal, "2027-02-15", "ponto_facultativo", "Portaria TJTESTE 5/2027", true)); r.StatusCode != 400 {
		t.Errorf("administrador marcando ponto facultativo = %d, quer 400", r.StatusCode)
	}
	if r, _ := adv.post("/calendario-forense/dias", diaForm(cal, "2027-02-15", "ponto_facultativo", "Portaria TJTESTE 5/2027", false)); r.StatusCode != 200 {
		t.Errorf("ponto facultativo sem efeito = %d, quer 200", r.StatusCode)
	}
	// Leitura: todos os papéis; HTML sem inline, com fonte à vista e selo [VERIFICAR] no que ninguém validou.
	for _, b := range []*browser{adv, adm, ass} {
		r, html := b.get("/calendario-forense?calendario=" + cal + "&ano=2027")
		if r.StatusCode != 200 || !strings.Contains(html, "Portaria TJTESTE 1/2027") || !strings.Contains(html, "Calendário forense") {
			t.Fatalf("página = %d", r.StatusCode)
		}
		if regexp.MustCompile(`(?i)\son[a-z]+\s*=|\sstyle\s*=`).MatchString(html) {
			t.Error("HTML com inline")
		}
	}
	if _, html := adv.get("/calendario-forense?calendario=" + cal + "&ano=2027"); !strings.Contains(html, "Inativar") {
		t.Error("advogado deveria ver Inativar")
	}
	if _, html := ass.get("/calendario-forense?calendario=" + cal + "&ano=2027"); strings.Contains(html, "Inativar") || strings.Contains(html, "Novo dia") {
		t.Error("assistente não deveria ver os controles de escrita")
	}
	// O dia entra no calendário do cálculo; inativado sai, mas a linha permanece.
	svc := e.svc.Calendario
	ctx := context.Background()
	c, err := svc.Carregar(ctx, cal)
	if err != nil {
		t.Fatal(err)
	}
	d := time.Date(2027, 2, 10, 0, 0, 0, 0, time.UTC)
	if _, ok := c.NaoUtil(d); !ok {
		t.Fatal("dia cadastrado deveria ser não útil")
	}
	if _, ok := c.NaoUtil(time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC)); ok {
		t.Error("ponto facultativo sem efeito NÃO pode ser excluído do cálculo")
	}
	var id string
	_ = e.pool.QueryRow(ctx, `SELECT id::text FROM calendario_dias WHERE data='2027-02-10'`).Scan(&id)
	if r, _ := adv.post("/calendario-forense/dias/"+id+"/inativar", url.Values{"motivo": {""}}); r.StatusCode != 400 {
		t.Errorf("inativar sem motivo = %d, quer 400", r.StatusCode)
	}
	if r, _ := ass.post("/calendario-forense/dias/"+id+"/inativar", url.Values{"motivo": {"x"}}); r.StatusCode != 403 {
		t.Errorf("assistente inativando = %d, quer 403", r.StatusCode)
	}
	if r, _ := adv.post("/calendario-forense/dias/"+id+"/inativar", url.Values{"motivo": {"data digitada errada"}}); r.StatusCode != 200 {
		t.Fatalf("inativar = %d", r.StatusCode)
	}
	c, _ = svc.Carregar(ctx, cal) // o cache é invalidado ao gravar
	if _, ok := c.NaoUtil(d); ok {
		t.Error("dia inativado NÃO pode entrar no cálculo")
	}
	if n := e.count(`SELECT count(*) FROM calendario_dias WHERE id=$1 AND NOT ativo AND motivo_inativacao='data digitada errada'`, id); n != 1 {
		t.Error("o dia inativado deve permanecer com o motivo")
	}
	if _, html := adv.get("/calendario-forense?calendario=" + cal + "&ano=2027"); !strings.Contains(html, "Inativo") {
		t.Error("o dia inativado deve aparecer com o selo Inativo")
	}
	for _, a := range []string{"calendario.criado", "calendario.dia_criado", "calendario.dia_inativado"} {
		if e.count(`SELECT count(*) FROM audit_events WHERE action=$1`, a) == 0 {
			t.Errorf("sem auditoria %s", a)
		}
	}
}

func TestCalendarioSemente(t *testing.T) {
	e := newEnv(t)
	t.Cleanup(func() { _, _ = e.pool.Exec(context.Background(), `TRUNCATE calendario_dias, calendarios CASCADE`) })
	ctx := context.Background()
	if err := e.svc.Calendario.Semear(ctx, 2022); err != nil {
		t.Fatal(err)
	}
	antes := e.count(`SELECT count(*) FROM calendario_dias`)
	if err := e.svc.Calendario.Semear(ctx, 2022); err != nil { // segunda execução não duplica
		t.Fatal(err)
	}
	if depois := e.count(`SELECT count(*) FROM calendario_dias`); depois != antes {
		t.Errorf("a semente duplicou: %d -> %d", antes, depois)
	}
	if n := e.count(`SELECT count(*) FROM calendario_dias WHERE fonte_citacao = ''`); n != 0 {
		t.Error("registro da semente sem fonte")
	}
	// 20/11: não em 2022 nem 2023; sim em 2024 (Lei 14.759/2023).
	for ano, quer := range map[string]int{"2022-11-20": 0, "2023-11-20": 0, "2024-11-20": 1} {
		if n := e.count(`SELECT count(*) FROM calendario_dias WHERE data=$1::date`, ano); n != quer {
			t.Errorf("%s: %d registros, quer %d", ano, n, quer)
		}
	}
	// Tudo o que a semente grava cita a norma e nasce sem verificador (aparece como [VERIFICAR]).
	if n := e.count(`SELECT count(*) FROM calendario_dias WHERE created_by IS NOT NULL OR verificado_por IS NOT NULL`); n != 0 {
		t.Error("a semente não tem autor nem verificador")
	}
}
