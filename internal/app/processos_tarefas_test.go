package app

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/auth"
)

func TestTarefas(t *testing.T) {
	e := newEnv(t)
	adv, advID := e.user("adv@exemplo.test", auth.Advogado)
	ass, assID := e.user("ass@exemplo.test", auth.Assistente)
	ontem := time.Now().AddDate(0, 0, -2).Format("2006-01-02")
	amanha := time.Now().AddDate(0, 0, 3).Format("2006-01-02")
	f := func(titulo, venc, resp string) url.Values {
		return url.Values{"titulo": {titulo}, "vencimento": {venc}, "status": {"aberta"}, "responsavel_id": {resp}}
	}

	// Assistente escreve em tarefas (permitido), mas não em processos (já testado).
	if r, _ := ass.post("/tarefas", f("Juntar procuração", ontem, assID)); r.StatusCode != 303 {
		t.Fatalf("assistente criando tarefa = %d", r.StatusCode)
	}
	if r, body := adv.post("/tarefas", f("", amanha, advID)); r.StatusCode != 400 || !strings.Contains(body, "título") {
		t.Fatalf("título vazio = %d", r.StatusCode)
	}
	bad := f("Com processo fantasma", amanha, advID)
	bad.Set("processo_id", "00000000-0000-0000-0000-000000000000")
	if r, _ := adv.post("/tarefas", bad); r.StatusCode != 400 {
		t.Fatalf("processo inexistente = %d", r.StatusCode)
	}
	if r, _ := adv.post("/tarefas", f("Protocolar petição", amanha, advID)); r.StatusCode != 303 {
		t.Fatalf("criar = %d", r.StatusCode)
	}

	// Atraso é só exibição e usa a data informada; a tela "Hoje" mostra as do usuário.
	_, lista := ass.get("/tarefas")
	if !strings.Contains(lista, "Juntar procuração") || !strings.Contains(lista, "atrasada") {
		t.Error("tarefa vencida deveria aparecer como atrasada")
	}
	if _, home := ass.get("/"); !strings.Contains(home, "Juntar procuração") || strings.Contains(home, "Protocolar petição") {
		t.Error("a tela Hoje deveria mostrar só as tarefas do próprio usuário")
	}
	if _, home := adv.get("/"); !strings.Contains(home, "Protocolar petição") {
		t.Error("tarefa a vencer em 3 dias deveria estar na tela Hoje")
	}

	// Concluir: some da lista de abertas; versão velha conflita.
	var tid string
	var ver int
	if err := e.pool.QueryRow(context.Background(), `SELECT id, version FROM tarefas WHERE titulo='Juntar procuração'`).Scan(&tid, &ver); err != nil {
		t.Fatal(err)
	}
	if r, _ := ass.post("/tarefas/"+tid+"/status", url.Values{"status": {"concluida"}, "version": {"1"}}); r.StatusCode != 303 {
		t.Fatalf("concluir = %d", r.StatusCode)
	}
	if r, _ := ass.post("/tarefas/"+tid+"/status", url.Values{"status": {"cancelada"}, "version": {"1"}}); r.StatusCode != 409 {
		t.Fatalf("versão velha = %d, esperado 409", r.StatusCode)
	}
	if _, lista := ass.get("/tarefas"); strings.Contains(lista, "Juntar procuração") {
		t.Error("tarefa concluída não deveria estar entre as abertas")
	}
	if _, lista := ass.get("/tarefas?status=todas"); !strings.Contains(lista, "Juntar procuração") {
		t.Error("filtro 'todas' deveria mostrar a concluída")
	}
	if r, _ := ass.post("/tarefas/"+tid+"/excluir", url.Values{"version": {"2"}}); r.StatusCode != 303 {
		t.Fatalf("excluir = %d", r.StatusCode)
	}

	// Toda escrita tem ator, e a cadeia está íntegra.
	if n := e.count(`SELECT count(*) FROM audit_events WHERE action LIKE 'tarefa.%' AND actor IN ($1,$2)`, advID, assID); n < 4 {
		t.Errorf("eventos de tarefa com ator = %d", n)
	}
	if _, err := audit.Verify(context.Background(), e.pool); err != nil {
		t.Fatal(err)
	}
}

// Nenhuma rota de domínio pode ficar sem sessão: tudo redireciona ao login.
func TestRotasDeDominioExigemLogin(t *testing.T) {
	e := newEnv(t)
	anon := &browser{e: e, sid: "inexistente"}
	for _, p := range []string{"/pessoas", "/processos", "/tarefas", "/pessoas/nova", "/tarefas/nova"} {
		if r, _ := anon.get(p); r.StatusCode != 303 || r.Header.Get("Location") != "/login" {
			t.Errorf("GET %s sem sessão = %d %s", p, r.StatusCode, r.Header.Get("Location"))
		}
	}
	if r, _ := anon.post("/pessoas", pessoaForm("X Y", "")); r.StatusCode != 401 {
		t.Errorf("POST sem sessão = %d, esperado 401", r.StatusCode)
	}
}
