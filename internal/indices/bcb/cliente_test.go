package bcb

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func novoTeste(t *testing.T, h http.HandlerFunc) *Cliente {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := Novo(Config{BaseURL: srv.URL, PermitirHTTP: true, Pausa: time.Nanosecond})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestUltimoLeAsSeisSeriesNoFormatoDoSGS(t *testing.T) {
	var caminhos []string
	c := novoTeste(t, func(w http.ResponseWriter, r *http.Request) {
		caminhos = append(caminhos, r.URL.RequestURI())
		if r.Header.Get("User-Agent") == "" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Errorf("cabeçalhos inesperados: %v", r.Header)
		}
		_, _ = io.WriteString(w, `[{"data":"01/09/2026","valor":"0.52"}]`)
	})
	if len(Series) != 6 {
		t.Fatalf("séries = %d", len(Series))
	}
	for _, s := range Series {
		p, err := c.Ultimo(context.Background(), s.Codigo)
		if err != nil {
			t.Fatalf("série %d: %v", s.Codigo, err)
		}
		if p.Valor != "0.52" || p.Data.Format("2006-01-02") != "2026-09-01" {
			t.Errorf("ponto = %+v", p)
		}
	}
	if len(caminhos) != 6 || caminhos[0] != "/dados/serie/bcdata.sgs.432/dados/ultimos/1?formato=json" {
		t.Errorf("caminhos = %v", caminhos)
	}
	ordem := []string{"Selic (meta Copom)", "IPCA", "INPC", "IGP-M", "Dólar (PTAX)", "Salário mínimo"}
	for i, s := range Series {
		if s.Rotulo != ordem[i] {
			t.Errorf("ordem[%d] = %q", i, s.Rotulo)
		}
	}
}

func TestVirgulaEPontoDecimal(t *testing.T) {
	for entrada, esperado := range map[string]string{"0.52": "0.52", "5,4321": "5.4321", "1.518,00": "1518.00", "1518.00": "1518.00", "-0.10": "-0.10"} {
		c := novoTeste(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, `[{"data":"02/11/2026","valor":"`+entrada+`"}]`)
		})
		p, err := c.Ultimo(context.Background(), 1)
		if err != nil || p.Valor != esperado {
			t.Errorf("%q -> %+v %v (esperado %q)", entrada, p, err, esperado)
		}
	}
}

func TestRespostasRuinsDaoErro(t *testing.T) {
	casos := map[string]http.HandlerFunc{
		"404":       func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) },
		"429":       func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(429) },
		"500":       func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) },
		"json ruim": func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, `{nao`) },
		"vazia":     func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, `[]`) },
		"valor ruim": func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, `[{"data":"01/09/2026","valor":"abc"}]`)
		},
		"data ruim": func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, `[{"data":"2026-09-01","valor":"1.0"}]`)
		},
		"gigante": func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, `[{"data":"01/09/2026","valor":"1.0","x":"`+strings.Repeat("a", 200000)+`"}]`)
		},
		"redireciona a outro host": func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://outro-host.invalid/x", http.StatusFound)
		},
	}
	for nome, h := range casos {
		c := novoTeste(t, h)
		if p, err := c.Ultimo(context.Background(), 433); err == nil {
			t.Errorf("%s: esperava erro, veio %+v", nome, p)
		}
	}
}

func TestURLPrecisaSerHTTPS(t *testing.T) {
	if _, err := Novo(Config{BaseURL: "http://api.bcb.gov.br"}); err == nil {
		t.Error("http deveria ser recusado")
	}
	if _, err := Novo(Config{}); err != nil {
		t.Errorf("padrão deveria valer: %v", err)
	}
}

func TestCodigoForaDaListaNaoVaiParaARede(t *testing.T) {
	c := novoTeste(t, func(w http.ResponseWriter, r *http.Request) { t.Error("não deveria chamar") })
	if _, err := c.Ultimo(context.Background(), 99999); err == nil {
		t.Error("série desconhecida deveria dar erro")
	}
}
