package app

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/br"
)

const marcador = "CONTEUDO-SECRETO-DO-CLIENTE"

func pdfBytes() []byte { return []byte("%PDF-1.4\n" + marcador + "\n%%EOF") }

func docxBytes(t *testing.T, comWord bool) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	names := []string{"[Content_Types].xml"}
	if comWord {
		names = append(names, "word/document.xml")
	}
	for _, n := range names {
		w, _ := zw.Create(n)
		_, _ = w.Write([]byte("<x>" + marcador + "</x>"))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// enviar faz um POST multipart em /documentos como o navegador faria.
func (b *browser) enviar(nome string, conteudo []byte, campos map[string]string) (*http.Response, string) {
	b.e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("csrf_token", b.token())
	for k, v := range campos {
		_ = mw.WriteField(k, v)
	}
	if nome != "" {
		fw, _ := mw.CreateFormFile("arquivo", nome)
		_, _ = fw.Write(conteudo)
	}
	_ = mw.Close()
	req, _ := http.NewRequest("POST", b.e.srv.URL+"/documentos", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "__Host-sid", Value: b.sid})
	resp, err := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(req)
	if err != nil {
		b.e.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)
	return resp, string(out)
}

func TestDocumentos(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)
	ctx := context.Background()
	proc := adv.criarProcesso(t, br.MakeCNJ("0000600", "2024", "8", "26", "0100"))
	vinc := map[string]string{"processo_id": proc, "descricao": "Contrato"}

	// Assistente lê, mas não envia.
	if r, _ := ass.enviar("a.pdf", pdfBytes(), vinc); r.StatusCode != 403 {
		t.Fatalf("assistente enviando = %d, esperado 403", r.StatusCode)
	}

	// Recusas: tipo falso, extensão que não confere, executável, zip comum, docx sem word/, sem vínculo, vazio.
	casos := []struct {
		nome string
		data []byte
		c    map[string]string
		msg  string
	}{
		{"contrato.pdf", []byte("MZ\x90\x00 programa"), vinc, "não aceito"},
		{"contrato.exe", pdfBytes(), vinc, "extensão"},
		{"foto.png", pdfBytes(), vinc, "extensão"},
		{"planilha.docx", docxBytes(t, false), vinc, "Word"},
		{"vazio.pdf", nil, vinc, "vazio"},
		{"a.pdf", pdfBytes(), map[string]string{}, "Vincule"},
		{"a.pdf", pdfBytes(), map[string]string{"processo_id": "00000000-0000-0000-0000-000000000000"}, "Processo não encontrado"},
	}
	for _, c := range casos {
		if r, body := adv.enviar(c.nome, c.data, c.c); r.StatusCode != 400 || !strings.Contains(body, c.msg) {
			t.Errorf("%s: status=%d, esperado 400 com %q", c.nome, r.StatusCode, c.msg)
		}
	}
	if e.count(`SELECT count(*) FROM documentos`) != 0 {
		t.Fatal("nenhum documento recusado pode ter sido gravado")
	}
	if entries, _ := os.ReadDir(e.docs); len(entries) != 0 {
		t.Fatalf("recusas não devem deixar arquivos no volume: %d", len(entries))
	}

	// Envio válido; nome com caminho é reduzido ao nome do arquivo.
	r, _ := adv.enviar("../../etc/Contrato Social.pdf", pdfBytes(), vinc)
	if r.StatusCode != 303 {
		t.Fatalf("envio válido = %d", r.StatusCode)
	}
	var id, nome, arq string
	if err := e.pool.QueryRow(ctx, `SELECT id, nome_original, arquivo FROM documentos`).Scan(&id, &nome, &arq); err != nil {
		t.Fatal(err)
	}
	if nome != "Contrato Social.pdf" {
		t.Errorf("nome deveria perder o caminho: %q", nome)
	}

	// Cifrado em repouso e com nome aleatório.
	raw, err := os.ReadFile(filepath.Join(e.docs, arq))
	if err != nil || bytes.Contains(raw, []byte(marcador)) || bytes.HasPrefix(raw, []byte("%PDF")) {
		t.Fatalf("o arquivo em disco deveria estar cifrado (err=%v)", err)
	}
	if len(arq) != 64 || strings.Contains(arq, ".") {
		t.Errorf("nome em disco deveria ser aleatório sem extensão: %q", arq)
	}

	// Download: conteúdo íntegro, tipo pelo conteúdo, sempre como anexo e sem execução; leitura auditada.
	for _, b := range []*browser{adv, ass} {
		resp, body := b.get("/documentos/" + id + "/baixar")
		if resp.StatusCode != 200 || body != string(pdfBytes()) {
			t.Fatalf("download = %d, conteúdo igual = %v", resp.StatusCode, body == string(pdfBytes()))
		}
		h := resp.Header
		if h.Get("Content-Type") != "application/pdf" || !strings.HasPrefix(h.Get("Content-Disposition"), "attachment") ||
			h.Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(h.Get("Content-Security-Policy"), "sandbox") {
			t.Errorf("cabeçalhos de download inseguros: %v", h)
		}
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='documento.lido' AND target_id=$1`, id) != 2 {
		t.Error("as duas leituras deveriam estar auditadas")
	}
	anon := &browser{e: e, sid: "x"}
	if r, _ := anon.get("/documentos/" + id + "/baixar"); r.StatusCode != 303 {
		t.Errorf("download sem login = %d, esperado redirecionamento", r.StatusCode)
	}

	// Lista, busca e vínculo (links nas fichas).
	if _, l := ass.get("/documentos?processo=" + proc); !strings.Contains(l, "Contrato Social.pdf") {
		t.Error("o documento deveria aparecer na lista do processo")
	}
	if _, l := adv.get("/documentos?q=nada-parecido"); strings.Contains(l, "Contrato Social.pdf") {
		t.Error("a busca sem correspondência trouxe resultado")
	}

	// DOCX válido e JPEG/PNG aceitos pelo conteúdo.
	for _, c := range []struct {
		n string
		d []byte
	}{{"a.docx", docxBytes(t, true)}, {"b.png", append([]byte("\x89PNG\r\n\x1a\n"), 1, 2, 3)}, {"c.JPG", []byte{0xFF, 0xD8, 0xFF, 0xE0, 1}}} {
		if r, _ := adv.enviar(c.n, c.d, vinc); r.StatusCode != 303 {
			t.Errorf("%s deveria ser aceito: %d", c.n, r.StatusCode)
		}
	}

	// Adulteração em disco é detectada (o GCM autentica) e nada é entregue.
	raw[len(raw)-1] ^= 1
	if err := os.WriteFile(filepath.Join(e.docs, arq), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if resp, body := adv.get("/documentos/" + id + "/baixar"); resp.StatusCode != 500 || strings.Contains(body, marcador) {
		t.Errorf("arquivo adulterado: status=%d", resp.StatusCode)
	}

	// Exclusão oculta (o arquivo permanece no volume).
	if r, _ := adv.post("/documentos/"+id+"/excluir", url.Values{"version": {"1"}}); r.StatusCode != 303 {
		t.Fatalf("exclusão = %d", r.StatusCode)
	}
	if resp, _ := adv.get("/documentos/" + id + "/baixar"); resp.StatusCode != 404 {
		t.Errorf("baixar excluído = %d, esperado 404", resp.StatusCode)
	}
}

func TestDocumentoGrandeDemais(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	proc := adv.criarProcesso(t, br.MakeCNJ("0000700", "2024", "8", "26", "0100"))
	grande := append([]byte("%PDF-"), make([]byte, 27<<20)...)
	if r, _ := adv.enviar("grande.pdf", grande, map[string]string{"processo_id": proc}); r.StatusCode != 413 {
		t.Fatalf("27 MiB = %d, esperado 413", r.StatusCode)
	}
	if e.count(`SELECT count(*) FROM documentos`) != 0 {
		t.Fatal("arquivo grande demais não pode ser gravado")
	}
	if entries, _ := os.ReadDir(e.docs); len(entries) != 0 {
		t.Fatal("arquivo grande demais não pode deixar resíduo no volume")
	}
}

// Falha no banco depois de gravar o arquivo: o arquivo é removido (nada de resíduo sem registro).
// Falha na auditoria da leitura: nada é entregue.
func TestDocumentosFalhasNoMeio(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ctx := context.Background()
	proc := adv.criarProcesso(t, br.MakeCNJ("0000800", "2024", "8", "26", "0100"))
	vinc := map[string]string{"processo_id": proc}

	if _, err := e.pool.Exec(ctx, `ALTER TABLE audit_events RENAME TO audit_events_off`); err != nil {
		t.Fatal(err)
	}
	r, _ := adv.enviar("a.pdf", pdfBytes(), vinc)
	_, _ = e.pool.Exec(ctx, `ALTER TABLE audit_events_off RENAME TO audit_events`)
	if r.StatusCode < 500 {
		t.Fatalf("falha na auditoria deveria virar erro interno: %d", r.StatusCode)
	}
	if e.count(`SELECT count(*) FROM documentos`) != 0 {
		t.Error("registro ficou gravado sem auditoria")
	}
	if entries, _ := os.ReadDir(e.docs); len(entries) != 0 {
		t.Errorf("arquivo ficou órfão no volume: %d", len(entries))
	}

	adv.enviar("a.pdf", pdfBytes(), vinc)
	var id string
	_ = e.pool.QueryRow(ctx, `SELECT id FROM documentos`).Scan(&id)
	_, _ = e.pool.Exec(ctx, `ALTER TABLE audit_events RENAME TO audit_events_off`)
	resp, body := adv.get("/documentos/" + id + "/baixar")
	_, _ = e.pool.Exec(ctx, `ALTER TABLE audit_events_off RENAME TO audit_events`)
	if resp.StatusCode < 500 || strings.Contains(body, marcador) {
		t.Errorf("sem auditoria da leitura não pode haver entrega: status=%d", resp.StatusCode)
	}
}
