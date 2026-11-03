package app

import (
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"painelteodoro/internal/auth"
)

var inlineHandler = regexp.MustCompile(`(?i)\son[a-z]+\s*=|\sstyle\s*=`)

func TestAgendaPainelCategorias(t *testing.T) {
	e := newEnv(t)
	ass, assID := e.user("ass@exemplo.test", auth.Assistente)
	ro, _ := e.user("adm@exemplo.test", auth.Administrador) // só leitura na agenda

	dia := time.Now().In(time.FixedZone("BRT", -3*3600)).AddDate(0, 0, 3).Format("2006-01-02")
	if r, _ := ass.post("/agenda", url.Values{"tipo": {"aula"}, "titulo": {"Aula de processo civil"}, "inicio": {dia + "T19:00"}, "responsavel_id": {assID}}); r.StatusCode != 303 {
		t.Fatalf("criar aula = %d", r.StatusCode)
	}
	if r, _ := ass.post("/tarefas", url.Values{"titulo": {"Preparar slides"}, "subtitulo": {"Aula"}, "prioridade": {"alta"}, "status": {"aberta"}}); r.StatusCode != 303 {
		t.Fatalf("criar tarefa = %d", r.StatusCode)
	}

	r, body := ass.get("/agenda?cat=aulas")
	if r.StatusCode != 200 || !strings.Contains(body, "Aulas") || !strings.Contains(body, "Aula de processo civil") ||
		!strings.Contains(body, "Preparar slides") || !strings.Contains(body, "Novo compromisso") {
		t.Fatalf("painel aulas = %d", r.StatusCode)
	}
	if m := inlineHandler.FindString(body); m != "" {
		t.Errorf("atributo inline proibido na página: %q", m)
	}
	for _, rot := range []string{"Tarefas", "Intimações", "Audiências", "Diligências", "Reuniões", "Ligações", "Aulas"} {
		if !strings.Contains(body, rot) {
			t.Errorf("pílula %q ausente", rot)
		}
	}
	if !strings.Contains(body, `<span class="count">2</span>`) {
		t.Error("o contador de Aulas deveria ser 2 (1 evento + 1 tarefa)")
	}
	if _, b := ass.get("/agenda?cat=intimacoes"); !strings.Contains(b, "Plano 5") {
		t.Error("intimações deveria trazer o aviso do Plano 5")
	}
	if _, b := ass.get("/agenda?cat=xyz"); strings.Contains(b, "Aula de processo civil") {
		t.Error("categoria inválida não abre nada")
	}

	// Sem sessão: login. Só leitura: sem o botão de escrita.
	if r, _ := (&browser{e: e, sid: "x"}).get("/agenda"); r.StatusCode != 303 && r.StatusCode != 302 {
		t.Errorf("sem sessão = %d, esperado redirecionamento ao login", r.StatusCode)
	}
	if r, b := ro.get("/agenda?cat=aulas"); r.StatusCode != 200 || strings.Contains(b, "Novo compromisso") {
		t.Errorf("somente leitura: %d / botão visível=%v", r.StatusCode, strings.Contains(b, "Novo compromisso"))
	}
}
