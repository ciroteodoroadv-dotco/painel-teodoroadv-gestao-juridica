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

func TestAgenda(t *testing.T) {
	e := newEnv(t)
	adv, advID := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)
	ctx := context.Background()

	proc := adv.criarProcesso(t, br.MakeCNJ("0000500", "2024", "8", "26", "0100"))
	pes := adv.criarPessoa(t, "Dora Lopes", "")

	amanha := time.Now().In(time.FixedZone("BRT", -3*3600)).AddDate(0, 0, 1).Format("2006-01-02")
	form := func(titulo, ini, fim string) url.Values {
		return url.Values{"tipo": {"audiencia"}, "titulo": {titulo}, "inicio": {ini}, "fim": {fim}, "local": {"Sala 3"},
			"processo_id": {proc}, "responsavel_id": {advID}}
	}

	// Validações.
	if r, body := adv.post("/agenda", form("Audiência", "", "")); r.StatusCode != 400 || !strings.Contains(body, "data e a hora") {
		t.Fatalf("sem início = %d", r.StatusCode)
	}
	if r, body := adv.post("/agenda", form("Audiência", amanha+"T15:00", amanha+"T14:00")); r.StatusCode != 400 || !strings.Contains(body, "antes do início") {
		t.Fatalf("término antes do início = %d", r.StatusCode)
	}
	bad := form("Audiência", amanha+"T14:00", "")
	bad.Set("processo_id", "00000000-0000-0000-0000-000000000000")
	if r, _ := adv.post("/agenda", bad); r.StatusCode != 400 {
		t.Fatalf("processo inexistente = %d", r.StatusCode)
	}

	// Assistente escreve na agenda (permitido). Horário digitado em Brasília é gravado em UTC.
	r, _ := ass.post("/agenda", form("Audiência de instrução", amanha+"T14:00", amanha+"T16:30"))
	id := idFrom(t, r, "/agenda/")
	var h int
	if err := e.pool.QueryRow(ctx, `SELECT extract(hour from inicio at time zone 'UTC')::int FROM agenda_eventos WHERE id=$1`, id).Scan(&h); err != nil || h != 17 {
		t.Fatalf("14:00 de Brasília deveria ser 17:00 UTC, veio %d (%v)", h, err)
	}
	_, det := adv.get("/agenda/" + id)
	if !strings.Contains(det, "14:00") || !strings.Contains(det, "16:30") || !strings.Contains(det, "Sala 3") {
		t.Error("o detalhe deveria mostrar o horário no fuso de Brasília")
	}

	// Participantes: pessoa precisa existir; entra e sai com auditoria.
	if r, _ := adv.post("/agenda/"+id+"/participantes", url.Values{"pessoa_id": {"00000000-0000-0000-0000-000000000000"}}); !strings.Contains(r.Header.Get("Location"), "erro=1") {
		t.Fatalf("participante inexistente deveria ser recusado: %s", r.Header.Get("Location"))
	}
	if r, _ := adv.post("/agenda/"+id+"/participantes", url.Values{"pessoa_id": {pes}}); r.StatusCode != 303 {
		t.Fatalf("adicionar participante = %d", r.StatusCode)
	}
	if _, det := adv.get("/agenda/" + id); !strings.Contains(det, "DORA LOPES") {
		t.Error("o participante deveria aparecer")
	}
	if r, _ := adv.post("/agenda/"+id+"/participantes/remover", url.Values{"pessoa_id": {pes}}); r.StatusCode != 303 {
		t.Fatalf("retirar participante = %d", r.StatusCode)
	}

	// Listas: futuro aparece em "de hoje em diante", não em "já passaram"; tela Hoje mostra os 7 próximos dias.
	if _, l := adv.get("/agenda"); !strings.Contains(l, "Audiência de instrução") {
		t.Error("evento de amanhã deveria estar na agenda")
	}
	if _, l := adv.get("/agenda?periodo=passados"); strings.Contains(l, "Audiência de instrução") {
		t.Error("evento futuro não deveria estar em 'já passaram'")
	}
	if _, l := adv.get("/agenda?tipo=reuniao"); strings.Contains(l, "Audiência de instrução") {
		t.Error("filtro de tipo deveria excluir a audiência")
	}
	if _, home := ass.get("/"); !strings.Contains(home, "Audiência de instrução") {
		t.Error("a tela Hoje deveria listar a audiência de amanhã")
	}

	// Concorrência otimista e exclusão.
	f := form("Audiência (remarcada)", amanha+"T15:00", "")
	f.Set("version", "1")
	if r, _ := adv.post("/agenda/"+id, f); r.StatusCode != 303 {
		t.Fatalf("alteração = %d", r.StatusCode)
	}
	if r, _ := adv.post("/agenda/"+id, f); r.StatusCode != 409 {
		t.Fatalf("versão velha = %d, esperado 409", r.StatusCode)
	}
	if r, _ := adv.post("/agenda/"+id+"/excluir", url.Values{"version": {"2"}}); r.StatusCode != 303 {
		t.Fatalf("exclusão = %d", r.StatusCode)
	}
	if _, l := adv.get("/agenda"); strings.Contains(l, "remarcada") {
		t.Error("evento excluído ainda aparece")
	}
	for _, a := range []string{"evento.criado", "evento.alterado", "evento.excluido", "evento.participante_adicionado", "evento.participante_removido"} {
		if e.count(`SELECT count(*) FROM audit_events WHERE action=$1`, a) == 0 {
			t.Errorf("evento %q não foi auditado", a)
		}
	}
	if _, err := audit.Verify(ctx, e.pool); err != nil {
		t.Fatal(err)
	}
}
