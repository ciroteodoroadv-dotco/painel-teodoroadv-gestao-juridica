package app

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"painelteodoro/internal/auth"
)

var hx = map[string]string{"HX-Request": "true"}

func relogio(y int, m time.Month, d int) Option {
	return WithRelogio(func() time.Time { return time.Date(y, m, d, 12, 0, 0, 0, time.UTC) })
}

func calcForm(base, modo, tipo, dias string) url.Values {
	return url.Values{"base": {base}, "modo": {modo}, "tipo": {tipo}, "dias": {dias}}
}

// O drawer repete os rótulos do B, NA ORDEM do B; os extras do v2 vêm depois do "Número de dias".
func TestCalculadoraDrawerComOsRotulosDoB(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	r, html := adv.do("GET", "/prazos/calculadora", nil, hx)
	if r.StatusCode != 200 {
		t.Fatalf("drawer = %d", r.StatusCode)
	}
	ordem(t, "drawer", html, "<h2>Calculadora de prazo</h2>",
		"Ferramenta solta, não vinculada a ficha nenhuma — só calcula e mostra o resultado.",
		"Data do evento", "Data da disponibilização no Diário ou da ciência/intimação direta.",
		"Modo de contagem", "Diário oficial (DJe)", "Ciência / intimação direta",
		"Regra do prazo", "CPC (dias úteis)", "CPP (prazo processual penal — dias corridos)",
		"Prazo material cível (dias corridos)", "Prazo material penal (dias corridos)",
		"Número de dias", "Calendário", "Nacional (sem calendário local verificado)",
		`id="resultadoPrazo"`, "Fechar", "Calcular")
	for _, banido := range []string{"em dobro", "Fazenda Pública", "Defensoria"} {
		if strings.Contains(html, banido) {
			t.Errorf("emenda de 04/11/2026: o drawer não pode ter %q", banido)
		}
	}
	for _, trecho := range []string{`name="base"`, `name="modo" value="dje" checked`, `name="tipo" value="uteis" checked`, `name="dias"`, `hx-post="/prazos/calcular"`, `hx-target="#resultadoPrazo"`, `class="radio-row"`} {
		if !strings.Contains(html, trecho) {
			t.Errorf("faltou %s", trecho)
		}
	}
	if regexp.MustCompile(`(?i)\son[a-z]+\s*=|\sstyle\s*=`).MatchString(html) {
		t.Error("drawer com inline")
	}
}

func TestCalculadoraResultado(t *testing.T) {
	e := newEnvOpts(t, relogio(2026, 11, 27))
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)
	// Caso dourado 1 do pacote de cálculo, agora pela tela: vence 01/12/2026; hoje 27/11 -> faltam 2 dias úteis.
	for _, b := range []*browser{adv, ass} {
		r, html := b.do("POST", "/prazos/calcular", calcForm("2026-11-06", "dje", "uteis", "15"), hx)
		if r.StatusCode != 200 {
			t.Fatalf("calcular = %d", r.StatusCode)
		}
		for _, quer := range []string{"Vencimento calculado: 01/12/2026", "A CONFERIR", "badge-warn", "faltam 2 dia(s) útil(eis)",
			"Início da contagem", "20/11/2026", "Lei 14.759/2023", "Sem calendário verificado deste tribunal: considerados só os feriados nacionais",
			"prazo em dobro (CPC arts. 180, 183, 186 e 229): o painel não o aplica; se o caso o exigir, o advogado ajusta manualmente",
			"Cálculo de apoio. O advogado confere o prazo no processo e no calendário do tribunal.",
			"Criar prazo com este cálculo", "/prazos/novo?", "data_fatal=2026-12-01", "fonte=calculadora",
			"Regras de cálculo ainda não validadas pelo advogado — não use para prazo real."} {
			if !strings.Contains(html, quer) {
				t.Errorf("resultado sem %q", quer)
			}
		}
		if strings.Contains(html, "Prazo fatal") || strings.Contains(html, "<html") {
			t.Error("o resultado é só o fragmento e nunca diz 'Prazo fatal'")
		}
		if regexp.MustCompile(`(?i)\son[a-z]+\s*=|\sstyle\s*=`).MatchString(html) {
			t.Error("resultado com inline")
		}
	}
	// "venceu há" e "vence hoje".
	_, html := adv.do("POST", "/prazos/calcular", calcForm("2026-10-01", "ciencia", "uteis", "5"), hx)
	if !strings.Contains(html, "venceu há") {
		t.Errorf("faltou 'venceu há': %s", html)
	}
	// Erros: 400 com a MENSAGEM do motivo (corrige o B, que perdia o motivo).
	casos := []struct {
		f   url.Values
		msg string
	}{
		{calcForm("2026-11-06", "dje", "uteis", "0"), "Número de dias inválido."},
		{calcForm("2026-11-06", "dje", "uteis", "abc"), "Número de dias inválido."},
		{calcForm("2026-11-06", "dje", "xyz", "5"), "Regra do prazo desconhecida."},
		{calcForm("2026-11-06", "xyz", "uteis", "5"), "Modo de contagem desconhecido."},
		{calcForm("", "dje", "uteis", "5"), "Data do evento inválida."},
	}
	for _, c := range casos {
		r, html := adv.do("POST", "/prazos/calcular", c.f, hx)
		if r.StatusCode != 400 || !strings.Contains(html, c.msg) || !strings.Contains(html, "form-block") {
			t.Errorf("%v: status %d, corpo %q", c.f, r.StatusCode, html)
		}
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='prazo.calculado'`) < 2 {
		t.Error("sem auditoria prazo.calculado")
	}
	// Sem sessão: não calcula.
	resp, err := http.Post(e.srv.URL+"/prazos/calcular", "application/x-www-form-urlencoded", strings.NewReader("dias=5"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode == 200 {
		t.Error("sem sessão não pode calcular")
	}
}

func TestCalculadoraRegrasValidadasLigaOuDesliga(t *testing.T) {
	aviso := "Regras de cálculo ainda não validadas pelo advogado"
	e := newEnvOpts(t, relogio(2026, 11, 27), WithCalculadoraValidada(true))
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	_, html := adv.do("POST", "/prazos/calcular", calcForm("2026-11-06", "dje", "uteis", "15"), hx)
	if strings.Contains(html, aviso) || !strings.Contains(html, "A CONFERIR") {
		t.Error("validada=true: sem o aviso vermelho, mas sempre A CONFERIR")
	}
}

func TestCalculadoraComCalendarioDoTribunal(t *testing.T) {
	e := newEnvOpts(t, relogio(2026, 11, 6))
	t.Cleanup(func() { _, _ = e.pool.Exec(context.Background(), `TRUNCATE calendario_dias, calendarios CASCADE`) })
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	cal := e.calendarioTeste(t, adv, "TJCALC")
	if r, _ := adv.post("/calendario-forense/dias", diaForm(cal, "2026-11-10", "feriado", "Portaria TJCALC 9/2026", true)); r.StatusCode != 200 {
		t.Fatal("cadastrar dia")
	}
	_, html := adv.do("GET", "/prazos/calculadora", nil, hx)
	if !strings.Contains(html, "TJCALC") || strings.Contains(html, "NACIONAL —") {
		t.Error("o select traz os calendários do tribunal (e não o nacional, sempre aplicado)")
	}
	// Ciência sex 06/11, 3 dias úteis: 09, [10 = dia do tribunal], 11, 12 -> 12/11/2026.
	f := calcForm("2026-11-06", "ciencia", "uteis", "3")
	f.Set("calendario", cal)
	_, html = adv.do("POST", "/prazos/calcular", f, hx)
	for _, quer := range []string{"Vencimento calculado: 12/11/2026", "Portaria TJCALC 9/2026", "Calendário do tribunal"} {
		if !strings.Contains(html, quer) {
			t.Errorf("sem %q", quer)
		}
	}
	f.Set("calendario", "nao-e-uuid")
	if r, _ := adv.do("POST", "/prazos/calcular", f, hx); r.StatusCode != 400 {
		t.Error("calendário inválido deve ser recusado")
	}
}

// "Criar prazo com este cálculo" só leva ao formulário; nada nasce sozinho e o prazo NÃO nasce confirmado,
// nem pelo advogado (fonte "calculadora" = a conferir até o clique em Confirmar).
func TestCriarPrazoComEsteCalculo(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	antes := e.count(`SELECT count(*) FROM prazos`)
	_, html := adv.get("/prazos/novo?data_fatal=2026-12-01&fonte=calculadora&fundamento=Calculadora+%28A+CONFERIR%29%3A+CPC")
	for _, quer := range []string{`value="2026-12-01"`, `value="Calculadora (A CONFERIR): CPC"`, `<option value="calculadora" selected>`} {
		if !strings.Contains(html, quer) {
			t.Errorf("formulário sem %q", quer)
		}
	}
	if e.count(`SELECT count(*) FROM prazos`) != antes {
		t.Fatal("abrir o formulário não pode criar prazo")
	}
	f := prazoForm("Contestação (calculadora)", 5)
	f.Set("fonte", "calculadora")
	if r, _ := adv.post("/prazos", f); r.StatusCode != 303 {
		t.Fatalf("criar = %d", r.StatusCode)
	}
	if n := e.count(`SELECT count(*) FROM prazos WHERE fonte='calculadora' AND situacao='a_conferir' AND confirmado_por IS NULL`); n != 1 {
		t.Error("prazo vindo da calculadora deve nascer A CONFERIR mesmo quando o advogado salva")
	}
}

func TestBotaoDaCalculadoraNaSidebar(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	_, html := adv.get("/agenda")
	ordem(t, "sidebar", html, `id="navTabs"`, "Audiências", `data-drawer="/prazos/calculadora"`, "Calculadora de prazo", "Pessoas")
	_, html = adv.get("/prazos")
	for _, quer := range []string{"Calculadora de prazo", "o resultado é sempre", "A CONFERIR"} {
		if !strings.Contains(html, quer) {
			t.Errorf("tela Prazos sem %q", quer)
		}
	}
	if strings.Contains(html, "ainda não calcula dias úteis") {
		t.Error("o aviso antigo da tela Prazos deveria ter sido trocado")
	}
}
