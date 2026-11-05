package app

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// Regressões da auditoria de 05/11/2026 (docs/auditoria/AUDITORIA-2026-11-05.md).

// pdfDe devolve um PDF fictício com n bytes de enchimento.
func pdfDe(n int) []byte { return append(pdfBytes(), bytes.Repeat([]byte(" "), n)...) }

// Item 1.1: o PDF da tarefa aceita até 25 MiB (antes, o teto de 1 MiB do httpx barrava tudo acima de ~1 MiB
// com a mensagem falsa "o limite é 25 MiB").
func TestTarefaAceitaPDFGrande(t *testing.T) {
	c := novoCenario(t)
	id := c.adv.criarTarefa(t, tarefaForm("processo", c.pro, "Com PDF grande"))
	for _, n := range []int{2 << 20, 24 << 20} {
		if r, body := c.adv.enviarDocTarefa(id, "grande.pdf", pdfDe(n)); r.StatusCode != 200 || r.Header.Get("X-Painel-Ok") != "Documento guardado." {
			t.Fatalf("PDF de %d bytes = %d %q, esperado 200", n, r.StatusCode, body)
		}
	}
	if r, _ := c.adv.enviarDocTarefa(id, "enorme.pdf", pdfDe(27<<20)); r.StatusCode != 413 {
		t.Fatalf("PDF de 27 MiB = %d, esperado 413", r.StatusCode)
	}
	if n := c.e.count(`SELECT count(*) FROM documentos WHERE tarefa_id=$1`, id); n != 2 {
		t.Fatalf("documentos guardados = %d, esperado 2", n)
	}
}

// Item 1.1: rota comum continua com 1 MiB, mesmo em multipart (o formulário do drawer envia FormData).
func TestRotaComumMantemLimiteDe1MiB(t *testing.T) {
	c := novoCenario(t)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("titulo", strings.Repeat("x", 2<<20))
	_ = mw.Close()
	req, _ := http.NewRequest("POST", c.e.srv.URL+"/tarefas", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-CSRF-Token", c.adv.token())
	req.AddCookie(&http.Cookie{Name: "__Host-sid", Value: c.adv.sid})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 413 {
		t.Fatalf("multipart de 2 MiB numa rota comum = %d, esperado 413", resp.StatusCode)
	}
}

func situacaoDoPrazo(t *testing.T, c *cenario, tarefaID string) string {
	t.Helper()
	var s string
	if err := c.e.pool.QueryRow(context.Background(), `SELECT situacao FROM prazos WHERE tarefa_id=$1 AND excluido_em IS NULL`, tarefaID).Scan(&s); err != nil {
		t.Fatalf("prazo da tarefa: %v", err)
	}
	return s
}

// Item 1.2 (ADR 0012, item 3): o prazo vindo de tarefa nasce e permanece "a conferir"; só o ato explícito de confirmar,
// feito por advogado, o confirma — e esse ato deixa o evento "prazo.confirmado" na auditoria.
func TestPrazoDaTarefaSoConfirmaPorAtoExplicito(t *testing.T) {
	c := novoCenario(t)
	f := tarefaForm("processo", c.pro, "Do assistente")
	f.Set("prazo_fatal", "2026-12-02")
	ida := c.ass.criarTarefa(t, f)
	if s := situacaoDoPrazo(t, c, ida); s != "a_conferir" {
		t.Fatalf("prazo do assistente nasce %s", s)
	}
	ed := tarefaForm("processo", c.pro, "Título corrigido pelo advogado")
	ed.Set("prazo_fatal", "2026-12-02")
	ed.Set("version", "1")
	if r, _ := c.adv.post("/tarefas/"+ida, ed); r.StatusCode != 200 {
		t.Fatalf("advogado editando = %d", r.StatusCode)
	}
	if s := situacaoDoPrazo(t, c, ida); s != "a_conferir" {
		t.Fatalf("salvar a tarefa não pode confirmar o prazo; ficou %s", s)
	}
	fv := tarefaForm("processo", c.pro, "Do advogado")
	fv.Set("prazo_fatal", "2026-12-03")
	idv := c.adv.criarTarefa(t, fv)
	if s := situacaoDoPrazo(t, c, idv); s != "a_conferir" {
		t.Fatalf("prazo digitado pelo advogado na tarefa nasce %s, esperado a_conferir", s)
	}
	if r, _ := c.adv.post("/tarefas/"+idv+"/confirmar-prazo", url.Values{}); r.StatusCode != 200 {
		t.Fatalf("confirmar = %d", r.StatusCode)
	}
	if s := situacaoDoPrazo(t, c, idv); s != "confirmado" {
		t.Fatalf("depois de confirmar = %s", s)
	}
	if n := c.e.count(`SELECT count(*) FROM audit_events WHERE action='prazo.confirmado'`); n != 1 {
		t.Fatalf("eventos prazo.confirmado = %d, esperado 1", n)
	}
}

// Item 3.4: a contagem de partes do processo chega à auditoria (antes, meta = {}).
func TestMetaDoProcessoNaAuditoria(t *testing.T) {
	c := novoCenario(t)
	var meta string
	_ = c.e.pool.QueryRow(context.Background(), `SELECT meta->>'partes' FROM audit_events WHERE action='processo.criado' ORDER BY seq DESC LIMIT 1`).Scan(&meta)
	if meta != "1" {
		t.Fatalf("meta partes = %q, esperado 1", meta)
	}
}

// Item 1.4: com mais processos que o limite da tela, a lista avisa, a busca vai ao banco, o seletor de vínculo oferece
// todos e o rótulo do vínculo de uma tarefa de processo antigo continua aparecendo.
func TestListaDeProcessosCortadaAvisaEBusca(t *testing.T) {
	c := novoCenario(t)
	var codigo string
	_ = c.e.pool.QueryRow(context.Background(), `SELECT codigo FROM processos WHERE id=$1`, c.pro).Scan(&codigo)
	idTarefa := c.adv.criarTarefa(t, tarefaForm("processo", c.pro, "Tarefa do processo antigo"))
	_, err := c.e.pool.Exec(context.Background(), `INSERT INTO processos
		SELECT (jsonb_populate_record(NULL::processos, to_jsonb(p) || jsonb_build_object('id', gen_random_uuid(),
			'codigo', 'PRO. T' || lpad(g::text, 6, '0'), 'numero_cnj', '', 'titulo', 'Cópia ' || g,
			'created_at', p.created_at + make_interval(secs => g)))).*
		FROM processos p, generate_series(1, 2001) g WHERE p.id = $1`, c.pro)
	if err != nil {
		t.Fatal(err)
	}
	_, lista := c.adv.get("/processos")
	if !strings.Contains(lista, "Lista limitada") || strings.Contains(lista, ">"+codigo+"<") {
		t.Fatalf("a lista deveria avisar o corte e não trazer o processo mais antigo (%s)", codigo)
	}
	if _, achado := c.adv.get("/processos?busca=" + url.QueryEscape(codigo)); !strings.Contains(achado, ">"+codigo+"<") {
		t.Fatalf("a busca no banco deveria achar %s", codigo)
	}
	if novo := c.adv.frag("/tarefas/nova"); !strings.Contains(novo, `value="`+c.pro+`"`) {
		t.Fatal("o seletor de vínculo deveria oferecer o processo antigo")
	}
	if _, tar := c.adv.get("/tarefas"); !strings.Contains(tar, codigo) {
		t.Fatalf("a tarefa %s deveria mostrar o vínculo %s", idTarefa, codigo)
	}
}
