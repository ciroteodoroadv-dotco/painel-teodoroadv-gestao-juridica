package app

import (
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/webx"
)

// contagemPilha lê o número da pílula (acc-head) da categoria.
func contagemPilha(html, chave string) string {
	m := regexp.MustCompile(`data-acc="agenda:` + chave + `"><span>[^<]*</span><span class="count">(\d+)</span>`).FindStringSubmatch(html)
	if m == nil {
		return "ausente"
	}
	return m[1]
}

func TestPaginaAgendaSeteCategorias(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)
	adm, _ := e.user("adm@exemplo.test", auth.Administrador)

	amanha := webx.Hoje(time.Now()).AddDate(0, 0, 1).Format("2006-01-02")
	depois := webx.Hoje(time.Now()).AddDate(0, 0, 2).Format("2006-01-02")
	pes := adv.criarPessoa(t, "Dora Lopes", "")
	ate := adv.criarAtendimento(t, "Demanda da agenda", []string{pes})

	// 2 aulas futuras + 1 passada (não conta) + 1 tarefa "aula" aberta + 1 tarefa "aula" concluída (não conta) = 3.
	for i, d := range []string{amanha, depois, "2020-01-01"} {
		adv.okPostTeste(t, "/agenda", compromissoForm("Aula", "Aula "+string(rune('A'+i)), d, "19:00"))
	}
	adv.okPostTeste(t, "/agenda", compromissoForm("Reunião", "Reunião com a cliente", amanha, "10:00"))
	adv.okPostTeste(t, "/agenda", compromissoForm("Ligação", "Ligação de retorno", amanha, "11:00"))
	f := tarefaForm("atendimento", ate, "Preparar a aula de processo")
	f.Set("subtitulo", "Aula")
	adv.criarTarefa(t, f)
	f2 := tarefaForm("atendimento", ate, "Aula já dada")
	f2.Set("subtitulo", "aulas")
	f2.Set("estado", "concluidas-com-sucesso")
	adv.criarTarefa(t, f2)
	f3 := tarefaForm("atendimento", ate, "Outra tarefa")
	f3.Set("subtitulo", "Ligações")
	f3.Set("prazo_fatal", amanha)
	ass.criarTarefa(t, f3) // assistente: o prazo fatal fica "a conferir"
	// Audiência futura, audiência passada e diligência.
	adv.okPostTeste(t, "/audiencias", audienciaForm("sustentacao-oral", "Sustentação futura", amanha, "08:00"))
	adv.okPostTeste(t, "/audiencias", audienciaForm("sustentacao-oral", "Sustentação passada", "2020-02-02", "08:00"))
	adv.okPostTeste(t, "/agenda", diligenciaForm("Protocolo físico", amanha, "15:00"))
	adv.okPostTeste(t, "/agenda", diligenciaForm("Diligência antiga", "2020-03-03", "15:00"))

	_, pg := adv.get("/agenda")
	// As 7 pílulas, nesta ordem e com estes rótulos.
	pos := 0
	for _, r := range []string{"Tarefas", "Intimações", "Audiências", "Diligências", "Reuniões", "Ligações", "Aulas"} {
		i := strings.Index(pg[pos:], "<span>"+r+"</span>")
		if i < 0 {
			t.Fatalf("pílula %q ausente ou fora de ordem", r)
		}
		pos += i + 1
	}
	for chave, want := range map[string]string{"tarefas": "2", "intimacoes": "0", "audiencias": "1", "diligencias": "1", "reunioes": "1", "ligacoes": "2", "aulas": "3"} {
		if got := contagemPilha(pg, chave); got != want {
			t.Errorf("contador de %s = %s, esperado %s", chave, got, want)
		}
	}
	// Cabeçalho, busca, botão, chip (pt-BR) e acordeão fechado de início.
	hoje := time.Now().In(webx.SP())
	chip := regexp.MustCompile(`<time class="agenda-date-chip" datetime="` + hoje.Format("2006-01-02") + `">[^<]+, ` + hoje.Format("02/01/2006") + `</time>`)
	for nome, ok := range map[string]bool{
		"h1":      strings.Contains(pg, "<h1>Agenda</h1>"),
		"sub":     strings.Contains(pg, "Painel principal: compromissos e tarefas em aberto, por categoria."),
		"chip":    chip.MatchString(pg),
		"busca":   strings.Contains(pg, `id="buscaAgenda"`) && strings.Contains(pg, `placeholder="Buscar por título, tipo, cliente ou responsável…"`),
		"novo":    strings.Contains(pg, `data-drawer="/agenda/novo"`) && strings.Contains(pg, ">Novo compromisso<"),
		"fechado": !strings.Contains(pg, `acc-head open`) && !strings.Contains(pg, `acc-item open`),
		"menu":    regexp.MustCompile(`<span>Agenda</span><span class="count">9</span>`).MatchString(pg), // 5 compromissos futuros+passada (6) + 2 audiências + 2 diligências = 10? ver abaixo
	} {
		if !ok && nome != "menu" {
			t.Errorf("%s ausente na Agenda", nome)
		}
	}
	// Menu: Agenda = compromissos (5) + audiências (2) + diligências (2); Audiências = 2.
	if !regexp.MustCompile(`<span>Agenda</span><span class="count">9</span>`).MatchString(pg) ||
		!regexp.MustCompile(`<span>Audiências</span><span class="count">2</span>`).MatchString(pg) {
		t.Error("contadores do menu (Agenda = compromissos + audiências + diligências) errados")
	}
	// Tabelas e textos do B: Aulas mostra compromissos e a tarefa de subtítulo "Aula" (a concluída não).
	for _, s := range []string{"Aula A", "Aula B", "Preparar a aula de processo", `<th>Cliente(s) / PES</th>`, `<th>Prazo de entrega</th>`, `<th>Prazo fatal</th>`,
		"Sustentação futura", "Protocolo físico", "Reunião com a cliente", "Ligação de retorno", `data-drawer="/agenda/`, `/cancelar">Cancelar</button>`} {
		if !strings.Contains(pg, s) {
			t.Errorf("Agenda sem %q", s)
		}
	}
	for _, s := range []string{"Aula C", "Aula já dada", "Sustentação passada", "Diligência antiga"} {
		if strings.Contains(pg, s) {
			t.Errorf("Agenda mostra %q, que não está em aberto", s)
		}
	}
	// Cliente no formato do B ("NOME · PES. 0000001") e prazo fatal a conferir no mesmo campo.
	if !strings.Contains(pg, "DORA LOPES · PES. 0000001") {
		t.Error("cliente da tarefa sem o código PES")
	}
	if !regexp.MustCompile(`tabular">` + regexp.QuoteMeta(webx.DataBR(webx.Hoje(time.Now()).AddDate(0, 0, 1))) + ` <span class="badge badge-warn">A CONFERIR</span>`).MatchString(pg) {
		t.Error("o prazo fatal a conferir deveria levar o selo A CONFERIR no mesmo campo")
	}
	// Intimações sem nenhuma coleta: o bloco do B, com o aviso de conferência manual ("sem resultado" nunca é "sem prazo").
	if !strings.Contains(pg, `<div class="form-block"><strong>Intimações indisponíveis.</strong><br>Nenhuma sincronização de intimações foi executada. Confira o DJEN manualmente.</div>`) {
		t.Error("Intimações deveria mostrar o bloco de indisponibilidade do B")
	}
	// Sem script nem estilo inline.
	if regexp.MustCompile(`\son[a-z]+=|\sstyle=`).MatchString(pg) || strings.Contains(pg, "javascript:") {
		t.Error("a Agenda não pode ter script ou estilo inline")
	}
	// A faixa de prazos do v2 continua no topo.
	ass.post("/prazos", prazoForm("Prazo da faixa", 3))
	if _, p2 := adv.get("/agenda"); !strings.Contains(p2, `class="notice"`) {
		t.Error("a faixa de prazos a conferir deveria estar no topo da Agenda")
	}
	// Quem só lê vê a Agenda sem botões de escrita; a raiz leva à Agenda; sem sessão vai ao login.
	_, pa := adm.get("/agenda")
	if strings.Contains(pa, "Novo compromisso") || strings.Contains(pa, `data-drawer="/agenda/`) || strings.Contains(pa, `/audiencias/`) {
		t.Error("o administrador só lê: sem botões de escrita")
	}
	if r, _ := adv.get("/"); r.StatusCode != 302 || r.Header.Get("Location") != "/agenda" {
		t.Errorf("/ = %d %q", r.StatusCode, r.Header.Get("Location"))
	}
	if r, _ := (&browser{e: e, sid: "x"}).get("/agenda"); r.StatusCode == 200 {
		t.Error("a Agenda exige sessão")
	}
	_ = url.Values{}
}
