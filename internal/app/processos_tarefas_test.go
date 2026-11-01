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
)

func TestProcessosEPartes(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)
	num := br.MakeCNJ("0001234", "2024", "8", "26", "0100")
	form := func(n string) url.Values {
		return url.Values{"numero": {n}, "orgao": {"1ª Vara Cível"}, "situacao": {"ativo"}}
	}

	// Número inválido, assistente sem escrita.
	bad := []byte(num)
	bad[1] = '9'
	if r, body := adv.post("/processos", form(string(bad))); r.StatusCode != 400 || !strings.Contains(body, "inválido") {
		t.Fatalf("CNJ inválido = %d", r.StatusCode)
	}
	if r, _ := ass.post("/processos", form(num)); r.StatusCode != 403 {
		t.Fatalf("assistente criando processo = %d, esperado 403", r.StatusCode)
	}
	r, _ := adv.post("/processos", form(strings.NewReplacer(".", "", "-", "").Replace(num))) // sem pontuação
	pid := idFrom(t, r, "/processos/")
	if e.count(`SELECT count(*) FROM processos WHERE numero_cnj=$1`, num) != 1 {
		t.Fatal("o número deveria ser gravado normalizado")
	}
	if r, body := adv.post("/processos", form(num)); r.StatusCode != 400 || !strings.Contains(body, "Já existe") {
		t.Fatalf("processo duplicado = %d", r.StatusCode)
	}

	// Partes: pessoa precisa existir; vínculo entra com auditoria; a pessoa vinculada não pode ser excluída.
	r, _ = adv.post("/pessoas", pessoaForm("Beatriz Lima", "111.444.777-35"))
	pes := idFrom(t, r, "/pessoas/")
	if r, _ := adv.post("/processos/"+pid+"/partes", url.Values{"pessoa_id": {"00000000-0000-0000-0000-000000000000"}, "qualidade": {"autor"}}); r.StatusCode != 303 ||
		!strings.Contains(r.Header.Get("Location"), "erro=parte") {
		t.Fatalf("parte inexistente deveria ser recusada: %d %s", r.StatusCode, r.Header.Get("Location"))
	}
	if r, _ := adv.post("/processos/"+pid+"/partes", url.Values{"pessoa_id": {pes}, "qualidade": {"autor"}}); r.StatusCode != 303 {
		t.Fatalf("adicionar parte = %d", r.StatusCode)
	}
	if _, body := adv.get("/processos/" + pid); !strings.Contains(body, "Beatriz Lima") {
		t.Error("a parte deveria aparecer no processo")
	}
	if _, body := adv.get("/pessoas/" + pes); !strings.Contains(body, num) {
		t.Error("o processo deveria aparecer na ficha da pessoa (via interface entre domínios)")
	}
	if r, body := adv.post("/pessoas/"+pes+"/excluir", url.Values{"version": {"1"}}); r.StatusCode != 409 || !strings.Contains(body, "parte em processo") {
		t.Fatalf("excluir pessoa vinculada = %d", r.StatusCode)
	}
	if r, _ := adv.post("/processos/"+pid+"/partes/remover", url.Values{"pessoa_id": {pes}, "qualidade": {"autor"}}); r.StatusCode != 303 {
		t.Fatalf("retirar parte = %d", r.StatusCode)
	}
	if r, _ := adv.post("/pessoas/"+pes+"/excluir", url.Values{"version": {"1"}}); r.StatusCode != 303 {
		t.Fatalf("excluir pessoa sem vínculos = %d", r.StatusCode)
	}

	// Busca por trecho do número e por órgão; conflito de versão.
	if _, body := adv.get("/processos?q=1234"); !strings.Contains(body, num) {
		t.Error("busca por trecho do número falhou")
	}
	if _, body := adv.get("/processos?q=cível"); !strings.Contains(body, num) {
		t.Error("busca por órgão falhou")
	}
	f := form(num)
	f.Set("version", "1")
	f.Set("situacao", "suspenso")
	if r, _ := adv.post("/processos/"+pid, f); r.StatusCode != 303 {
		t.Fatalf("alteração = %d", r.StatusCode)
	}
	if r, _ := adv.post("/processos/"+pid, f); r.StatusCode != 409 {
		t.Fatalf("versão velha = %d, esperado 409", r.StatusCode)
	}
}

// Operação multi-entidade: se a auditoria falhar no meio, NADA fica gravado (transação única).
func TestFalhaNoMeioNaoDeixaEstadoParcial(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	num := br.MakeCNJ("0000077", "2024", "8", "26", "0100")
	r, _ := adv.post("/processos", url.Values{"numero": {num}, "situacao": {"ativo"}})
	pid := idFrom(t, r, "/processos/")
	r, _ = adv.post("/pessoas", pessoaForm("Carlos Dias", ""))
	pes := idFrom(t, r, "/pessoas/")

	ctx := context.Background()
	// Simula a falha: a tabela de auditoria some entre a escrita da parte e a gravação do evento.
	if _, err := e.pool.Exec(ctx, `ALTER TABLE audit_events RENAME TO audit_events_off`); err != nil {
		t.Fatal(err)
	}
	r, _ = adv.post("/processos/"+pid+"/partes", url.Values{"pessoa_id": {pes}, "qualidade": {"autor"}})
	if _, err := e.pool.Exec(ctx, `ALTER TABLE audit_events_off RENAME TO audit_events`); err != nil {
		t.Fatal(err)
	}
	if r.StatusCode < 500 {
		t.Fatalf("a falha deveria virar erro interno, veio %d", r.StatusCode)
	}
	if n := e.count(`SELECT count(*) FROM processo_partes`); n != 0 {
		t.Fatalf("a parte ficou gravada sem auditoria (%d linhas): estado parcial", n)
	}
	// Repetir a operação depois funciona e não duplica.
	for i := 0; i < 2; i++ {
		if r, _ := adv.post("/processos/"+pid+"/partes", url.Values{"pessoa_id": {pes}, "qualidade": {"autor"}}); r.StatusCode != 303 {
			t.Fatalf("repetição %d = %d", i, r.StatusCode)
		}
	}
	if n := e.count(`SELECT count(*) FROM processo_partes`); n != 1 {
		t.Fatalf("repetir a mesma parte não pode duplicar: %d linhas", n)
	}
}

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
