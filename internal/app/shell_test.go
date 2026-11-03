package app

import (
	"net/url"
	"regexp"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
)

// ordem confere que os trechos aparecem na página, nesta ordem.
func ordem(t *testing.T, nome, html string, trechos ...string) {
	t.Helper()
	pos := 0
	for _, tr := range trechos {
		i := strings.Index(html[pos:], tr)
		if i < 0 {
			t.Errorf("%s: faltou (ou fora de ordem) %q", nome, tr)
			return
		}
		pos += i + len(tr)
	}
}

func TestCascaDoPainelB(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)

	adv.criarPessoa(t, "Ana Souza", "")
	if r, _ := adv.post("/tarefas", url.Values{"titulo": {"Tarefa da casca"}, "status": {"aberta"}}); r.StatusCode != 303 {
		t.Fatalf("criar tarefa = %d", r.StatusCode)
	}
	// Prazo registrado pela assistente fica "a conferir" e alimenta a faixa da Agenda.
	if r, _ := ass.post("/prazos", prazoForm("Prazo da faixa", 3)); r.StatusCode != 303 {
		t.Fatalf("criar prazo = %d", r.StatusCode)
	}

	for _, caminho := range []string{"/agenda", "/pessoas", "/tarefas", "/processos", "/prazos", "/conta", "/"} {
		_, html := adv.get(caminho)
		ordem(t, caminho, html, `id="app"`, `class="sidebar"`, `id="navTabs"`, `class="topbar"`, `id="stats"`,
			`id="content"`, `id="syncStatus"`, `id="idxTicker"`, `id="drawerOverlay"`, `id="drawer"`, `id="toast"`)
		if !strings.Contains(html, "<title>Teodoro Advocacia Estratégica — Painel Processual</title>") {
			t.Errorf("%s: título da aba diferente do painel B", caminho)
		}
		for _, banido := range []string{`<header class="topo">`, "cabeca", `class="pagina"`} {
			if strings.Contains(html, banido) {
				t.Errorf("%s: contém %q", caminho, banido)
			}
		}
		if !strings.Contains(html, `alt="Teodoro Advocacia Estratégica"`) || !strings.Contains(html, `class="seal-logo"`) {
			t.Errorf("%s: logo sem o alt do painel B", caminho)
		}
	}

	// Menu principal na ordem alfabética do B; itens do v2 num segundo grupo.
	_, html := adv.get("/agenda")
	ordem(t, "menu", html, ">Agenda<", ">Pessoas<", ">Processos<", ">Tarefas<", `class="tabs tabs-sistema"`, ">Prazos<", ">Documentos<", ">Minha conta<")
	if !regexp.MustCompile(`href="/pessoas"[^>]*>\s*<span>Pessoas</span><span class="count">1</span>`).MatchString(html) {
		t.Error("contador de Pessoas ausente ou errado")
	}
	if !regexp.MustCompile(`<span>Tarefas</span><span class="count">1</span>`).MatchString(html) {
		t.Error("contador de Tarefas ausente ou errado")
	}
	if !strings.Contains(html, `class="active"`) || !strings.Contains(html, `aria-current="page"`) {
		t.Error("item ativo do menu sem marcação")
	}
	if !regexp.MustCompile(`<h1>Agenda</h1>\s*<time class="agenda-date-chip" datetime="\d{4}-\d{2}-\d{2}">`).MatchString(html) {
		t.Error("Agenda sem o chip de data do painel B")
	}
	if !strings.Contains(html, `class="btn btn-primary"`) {
		t.Error("botão primário do topo ausente")
	}

	// Home = Agenda, com a faixa honesta de prazos (sem cálculo novo).
	_, home := adv.get("/")
	if !strings.Contains(home, `<h1>Agenda</h1>`) || !strings.Contains(home, `class="notice"`) ||
		!strings.Contains(home, "1 prazo(s) a conferir") || !strings.Contains(home, `href="/prazos"`) {
		t.Error("a Agenda inicial deveria trazer a faixa de prazos a conferir")
	}

	// Rodapé honesto (Plano 1B, Tarefa 4).
	if !strings.Contains(home, "Nenhuma coleta configurada neste painel. Não presuma que houve atualização; confira as fontes oficiais.") {
		t.Error("syncStatus sem o texto honesto")
	}
	if !strings.Contains(home, `<span class="idxt-status">Índices econômicos: não sincronizado</span>`) {
		t.Error("idxTicker sem o estado 'não sincronizado'")
	}

	// Sem sessão: cartão central, sem a casca.
	if r, _ := (&browser{e: e, sid: "x"}).get("/login"); r.StatusCode != 200 {
		t.Fatalf("login = %d", r.StatusCode)
	}
}
