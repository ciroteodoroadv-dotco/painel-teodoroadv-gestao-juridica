package app

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/tarefas"
)

func tarefaForm(titulo, sub, prio, status string) url.Values {
	return url.Values{"titulo": {titulo}, "subtitulo": {sub}, "prioridade": {prio}, "status": {status}}
}

func TestTarefasSubtituloPrioridadeRevisao(t *testing.T) {
	e := newEnv(t)
	adv, advID := e.user("adv@exemplo.test", auth.Advogado)
	ctx := context.Background()

	if r, _ := adv.post("/tarefas", tarefaForm("Ligar ao cliente", "Ligação", "alta", "em_revisao")); r.StatusCode != 303 {
		t.Fatalf("criar com subtítulo/prioridade = %d", r.StatusCode)
	}
	var sub, prio, st string
	if err := e.pool.QueryRow(ctx, `SELECT subtitulo, prioridade, status FROM tarefas WHERE titulo='Ligar ao cliente'`).Scan(&sub, &prio, &st); err != nil ||
		sub != "Ligação" || prio != "alta" || st != "em_revisao" {
		t.Fatalf("lido de volta: %q %q %q (%v)", sub, prio, st, err)
	}
	if r, body := adv.post("/tarefas", tarefaForm("Outra", "", "urgente", "aberta")); r.StatusCode != 400 || !strings.Contains(body, "rioridade") {
		t.Fatalf("prioridade inválida = %d", r.StatusCode)
	}
	if r, _ := adv.post("/tarefas", tarefaForm("Outra", strings.Repeat("x", 81), "normal", "aberta")); r.StatusCode != 400 {
		t.Fatalf("subtítulo longo = %d", r.StatusCode)
	}
	_, lista := adv.get("/tarefas")
	if !strings.Contains(lista, "Ligar ao cliente") || !strings.Contains(lista, "prioridade-alta") || !strings.Contains(lista, "Ligação") {
		t.Error("a lista deveria mostrar a tarefa em revisão com prioridade e subtítulo")
	}
	if !strings.Contains(lista, `data-drawer="/tarefas/nova"`) {
		t.Error("o link Nova tarefa deveria abrir o drawer")
	}

	// AbertasPorSubtitulo: só abertas (inclui em revisão), nunca concluídas, canceladas ou excluídas.
	adv.post("/tarefas", tarefaForm("Concluída", "Ligação", "normal", "concluida"))
	adv.post("/tarefas", tarefaForm("Cancelada", "Ligação", "normal", "cancelada"))
	adv.post("/tarefas", tarefaForm("Excluída", "Ligação", "normal", "aberta"))
	if _, err := e.pool.Exec(ctx, `UPDATE tarefas SET excluido_em=now() WHERE titulo='Excluída'`); err != nil {
		t.Fatal(err)
	}
	_ = advID
	svc := &tarefas.Service{Pool: e.pool}
	got, err := svc.AbertasPorSubtitulo(ctx, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Titulo != "Ligar ao cliente" {
		t.Errorf("AbertasPorSubtitulo = %+v", got)
	}
	if got, _ := svc.AbertasPorSubtitulo(ctx, "inexistente", 100); len(got) != 0 {
		t.Error("filtro q deveria excluir")
	}
}

func TestTarefasDrawer(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ro, _ := e.user("adm@exemplo.test", auth.Administrador) // sem escrita em tarefas
	hx := map[string]string{"HX-Request": "true"}

	r, body := adv.do("GET", "/tarefas/nova", nil, hx)
	if r.StatusCode != 200 || strings.Contains(body, "<html") || !strings.HasPrefix(strings.TrimSpace(body), `<h2 id="drawer-titulo">`) ||
		!strings.Contains(body, "data-drawer-fechar") || !strings.Contains(body, `hx-target="#drawer-corpo"`) {
		t.Fatalf("fragmento do drawer inválido (%d): %.200s", r.StatusCode, body)
	}
	if _, full := adv.get("/tarefas/nova"); !strings.Contains(full, "<html") {
		t.Error("sem HX-Request deve vir a página completa")
	}

	r, _ = adv.do("POST", "/tarefas", tarefaForm("Pelo drawer", "Aula", "media", "aberta"), hx)
	if r.StatusCode != 200 || r.Header.Get("HX-Redirect") != "/tarefas?salvo=1" {
		t.Fatalf("POST drawer = %d, HX-Redirect=%q", r.StatusCode, r.Header.Get("HX-Redirect"))
	}
	r, body = adv.do("POST", "/tarefas", tarefaForm("", "", "normal", "aberta"), hx)
	if r.StatusCode != 200 || r.Header.Get("HX-Redirect") != "" || !strings.Contains(body, "título") || strings.Contains(body, "<html") {
		t.Fatalf("erro no drawer = %d: %.200s", r.StatusCode, body)
	}

	var id string
	if err := e.pool.QueryRow(context.Background(), `SELECT id FROM tarefas WHERE titulo='Pelo drawer'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	r, body = adv.do("GET", "/tarefas/"+id+"/editar", nil, hx)
	if r.StatusCode != 200 || strings.Contains(body, "<html") || !strings.Contains(body, "Pelo drawer") {
		t.Fatalf("editar no drawer = %d", r.StatusCode)
	}

	if r, _ := ro.do("POST", "/tarefas", tarefaForm("Não pode", "", "normal", "aberta"), hx); r.StatusCode != 403 {
		t.Errorf("sem permissão de escrita = %d, esperado 403", r.StatusCode)
	}
	if _, l := ro.get("/tarefas"); strings.Contains(l, "Nova tarefa") {
		t.Error("quem só lê não deveria ver Nova tarefa")
	}
}
