package app

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"painelteodoro/internal/auth"
)

func TestAgendaTiposEConflito(t *testing.T) {
	e := newEnv(t)
	adv, advID := e.user("adv@exemplo.test", auth.Advogado)
	_, outroID := e.user("outro@exemplo.test", auth.Assistente)

	dia := time.Now().In(time.FixedZone("BRT", -3*3600)).AddDate(0, 0, 2).Format("2006-01-02")
	f := func(tipo, titulo, hora, resp string) url.Values {
		return url.Values{"tipo": {tipo}, "titulo": {titulo}, "inicio": {dia + "T" + hora}, "responsavel_id": {resp}}
	}

	for tipo, hora := range map[string]string{"ligacao": "08:00", "aula": "07:00"} {
		if r, _ := adv.post("/agenda", f(tipo, "Compromisso "+tipo, hora, advID)); r.StatusCode != 303 {
			t.Fatalf("tipo %s deveria ser aceito: %d", tipo, r.StatusCode)
		}
	}
	// Mesmo instante e mesmo responsável: reunião é barrada com a mensagem do conflito.
	r, body := adv.post("/agenda", f("reuniao", "Reunião com cliente", "08:00", advID))
	if r.StatusCode != 400 || !strings.Contains(body, "Já existe") || !strings.Contains(body, "08:00") {
		t.Fatalf("conflito = %d: %.300s", r.StatusCode, body)
	}
	if n := e.count(`SELECT count(*) FROM agenda_eventos WHERE titulo='Reunião com cliente'`); n != 0 {
		t.Error("o evento em conflito não deveria ser criado")
	}
	// Outro responsável, mesma hora: ok.
	if r, _ := adv.post("/agenda", f("reuniao", "Reunião do outro", "08:00", outroID)); r.StatusCode != 303 {
		t.Fatalf("outro responsável = %d", r.StatusCode)
	}
	// Audiência nunca é bloqueada.
	r, _ = adv.post("/agenda", f("audiencia", "Audiência no mesmo horário", "08:00", advID))
	idAud := idFrom(t, r, "/agenda/")
	if _, det := adv.get("/agenda/" + idAud); !strings.Contains(det, "Atenção") {
		t.Error("a audiência em horário já ocupado deveria exibir aviso no detalhe")
	}
	// Editar o próprio evento sem mudar horário: ok (ignora a si mesmo).
	var id, v string
	if err := e.pool.QueryRow(t.Context(), `SELECT id, version::text FROM agenda_eventos WHERE titulo='Compromisso aula'`).Scan(&id, &v); err != nil {
		t.Fatal(err)
	}
	g := f("aula", "Compromisso aula (editado)", "07:00", advID)
	g.Set("version", v)
	if r, _ := adv.post("/agenda/"+id, g); r.StatusCode != 303 {
		t.Fatalf("editar a si mesmo = %d", r.StatusCode)
	}
	// Mover para o horário da ligação: conflito também na edição.
	if r, _ := adv.post("/agenda", f("diligencia", "Diligência 9h", "09:00", advID)); r.StatusCode != 303 {
		t.Fatalf("diligência = %d", r.StatusCode)
	}
	var did, dv string
	if err := e.pool.QueryRow(t.Context(), `SELECT id, version::text FROM agenda_eventos WHERE titulo='Diligência 9h'`).Scan(&did, &dv); err != nil {
		t.Fatal(err)
	}
	h := f("diligencia", "Diligência 9h", "08:00", advID)
	h.Set("version", dv)
	if r, body := adv.post("/agenda/"+did, h); r.StatusCode != 400 || !strings.Contains(body, "Já existe") {
		t.Fatalf("mover para horário ocupado = %d", r.StatusCode)
	}
}
