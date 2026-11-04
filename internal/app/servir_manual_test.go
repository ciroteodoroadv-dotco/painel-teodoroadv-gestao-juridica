package app

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"testing"
	"time"

	"painelteodoro/internal/auth"
)

// TestServirParaConferenciaManual NÃO roda no dia a dia: só com PAINEL_SERVIR_MANUAL=1 (e PAINEL_TEST_DB de um banco
// DESCARTÁVEL). Sobe o v2 inteiro (com a demonstração fictícia), imprime o endereço e o cookie de uma sessão de teste e
// espera até existir o arquivo /tmp/parar-servidor-v2 (ou 30 min). Serve para executar o JS no Chrome. Nada real.
func TestServirParaConferenciaManual(t *testing.T) {
	if os.Getenv("PAINEL_SERVIR_MANUAL") != "1" {
		t.Skip("conferência manual: defina PAINEL_SERVIR_MANUAL=1")
	}
	e := newEnv(t)
	_, id := e.user("adv@exemplo.test", auth.Advogado)
	if err := SeedDemo(context.Background(), e.pool, fieldKey, e.a.Store, id); err != nil {
		t.Fatal(err)
	}
	b, _ := e.user("adv2@exemplo.test", auth.Advogado)
	// O servidor de teste não serve /static: um proxy entrega os estáticos reais (web/static) e repassa o resto.
	alvo, _ := url.Parse(e.srv.URL)
	proxy := httputil.NewSingleHostReverseProxy(alvo)
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("../../web/static"))))
	mux.Handle("/", proxy)
	front := httptest.NewServer(mux)
	defer front.Close()
	fmt.Printf("ENDERECO=%s\nSID=%s\n", front.URL, b.sid)
	_ = os.Remove("/tmp/parar-servidor-v2")
	for i := 0; i < 1800; i++ {
		if _, err := os.Stat("/tmp/parar-servidor-v2"); err == nil {
			return
		}
		time.Sleep(time.Second)
	}
}
