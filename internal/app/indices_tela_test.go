package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
)

// sgsFalso responde no formato do SGS sem sair para a rede.
type sgsFalso struct{ chamadas *int }

func (s sgsFalso) RoundTrip(r *http.Request) (*http.Response, error) {
	*s.chamadas++
	corpo := `[{"data":"01/10/2026","valor":"0.52"}]`
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(corpo)), Request: r}, nil
}

func TestFragmentoDeIndicesExigeSessao(t *testing.T) {
	e := newEnv(t)
	r, err := http.Get(e.srv.URL + "/fragmentos/indices")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Body.Close() }()
	if r.Request.URL.Path != "/login" && r.StatusCode != 401 && r.StatusCode != 403 {
		t.Errorf("sem sessão: %d %s", r.StatusCode, r.Request.URL.Path)
	}
}

func TestIndicesDesligadosMostramBloqueadoEZeroRede(t *testing.T) {
	e := newEnvOpts(t, WithIndices(Indices{Transport: semRede{t}, PermitirHTTP: true, BaseURL: "http://bcb.invalid"}))
	if err := e.svc.Indices.IniciarEstado(context.Background()); err != nil {
		t.Fatal(err)
	}
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	r, corpo := adv.get("/fragmentos/indices")
	if r.StatusCode != 200 || r.Header.Get("X-Painel-Fragmento") != "indices" || r.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("fragmento: %d %v", r.StatusCode, r.Header)
	}
	if !strings.Contains(corpo, "Índices econômicos: bloqueado — Coleta desligada neste ambiente.") || strings.Contains(corpo, "<html") {
		t.Errorf("corpo = %s", corpo)
	}
	_, pag := adv.get("/pessoas")
	if !strings.Contains(pag, `id="idxTicker"`) || !strings.Contains(pag, `data-refresh="/fragmentos/indices"`) || !strings.Contains(pag, `data-intervalo="30"`) ||
		!strings.Contains(pag, "Índices econômicos: bloqueado") {
		t.Errorf("rodapé da página sem o ticker do serviço")
	}
	adm, _ := e.user("adm@exemplo.test", auth.Administrador)
	if r, corpo := adm.post("/admin/indices/atualizar", nil); r.StatusCode != 400 || !strings.Contains(corpo, "desligada") {
		t.Errorf("atualizar desligado: %d %s", r.StatusCode, corpo)
	}
}

func TestIndicesLigadosComTransporteFalso(t *testing.T) {
	n := 0
	e := newEnvOpts(t, WithIndices(Indices{Ativo: true, Transport: sgsFalso{&n}, PermitirHTTP: true, BaseURL: "http://bcb.test", Pausa: 1}))
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	adm, _ := e.user("adm@exemplo.test", auth.Administrador)
	if r, _ := adv.post("/admin/indices/atualizar", nil); r.StatusCode != 403 {
		t.Errorf("advogado não atualiza: %d", r.StatusCode)
	}
	if r, _ := adv.get("/admin/indices"); r.StatusCode != 403 {
		t.Errorf("advogado não abre a página: %d", r.StatusCode)
	}
	if n != 0 {
		t.Fatalf("nada deveria sair antes da ação do administrador: %d", n)
	}
	r, _ := adm.post("/admin/indices/atualizar", nil)
	if r.StatusCode != 200 || r.Header.Get("X-Painel-Ok") == "" || n != 6 {
		t.Fatalf("atualizar: %d chamadas=%d", r.StatusCode, n)
	}
	_, corpo := adv.get("/fragmentos/indices")
	for _, trecho := range []string{"Selic (meta Copom)", "<b>0,52 %</b>", "Ref.: 01/10/2026 · sincronizado", "Salário mínimo"} {
		if !strings.Contains(corpo, trecho) {
			t.Errorf("faltou %q em %s", trecho, corpo)
		}
	}
	// ler o fragmento nunca dispara coleta
	adv.get("/fragmentos/indices")
	if n != 6 {
		t.Errorf("a leitura do fragmento disparou coleta: %d", n)
	}
	_, pg := adm.get("/admin/indices")
	if !strings.Contains(pg, "Atualizar índices agora") || !strings.Contains(pg, "Selic") {
		t.Errorf("página do administrador: %s", pg)
	}
}
