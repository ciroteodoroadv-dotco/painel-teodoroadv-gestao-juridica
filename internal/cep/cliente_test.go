package cep

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func novoTeste(t *testing.T, h http.HandlerFunc) *Cliente {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := NovoCliente(Config{BaseURL: srv.URL, PermitirHTTP: true})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestBuscarSoEnviaOsOitoDigitos(t *testing.T) {
	c := novoTeste(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() != "/ws/01001000/json/" {
			t.Errorf("URL = %s", r.URL.RequestURI())
		}
		for k := range r.Header {
			switch k {
			case "User-Agent", "Accept", "Accept-Encoding":
			default:
				t.Errorf("cabeçalho inesperado %s", k)
			}
		}
		io.WriteString(w, `{"cep":"01001-000","logradouro":"Praça da Sé","complemento":"lado ímpar","bairro":"Sé","localidade":"São Paulo","uf":"sp"}`)
	})
	e, err := c.Buscar(context.Background(), "01001000")
	if err != nil || e.Logradouro != "Praça da Sé" || e.Bairro != "Sé" || e.Municipio != "São Paulo" || e.UF != "SP" {
		t.Fatalf("%+v %v", e, err)
	}
}

func TestBuscarCEPInexistente(t *testing.T) {
	for _, corpo := range []string{`{"erro": true}`, `{"erro": "true"}`} {
		c := novoTeste(t, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, corpo) })
		if _, err := c.Buscar(context.Background(), "99999999"); !errors.Is(err, ErrNaoEncontrado) {
			t.Errorf("%s: err = %v", corpo, err)
		}
	}
}

func TestBuscarFalhasNaoPreenchemNada(t *testing.T) {
	casos := map[string]http.HandlerFunc{
		"500":       func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) },
		"400":       func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(400) },
		"429":       func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(429) },
		"json ruim": func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `<html>`) },
		"vazio":     func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{}`) },
		"gigante": func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, `{"logradouro":"`+strings.Repeat("a", 100000)+`"}`)
		},
		"outro host": func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://outro-host.invalid/x", http.StatusFound)
		},
	}
	for nome, h := range casos {
		c := novoTeste(t, h)
		e, err := c.Buscar(context.Background(), "01001000")
		if err == nil || errors.Is(err, ErrNaoEncontrado) || (e != Endereco{}) {
			t.Errorf("%s: %+v %v", nome, e, err)
		}
	}
}

func TestBuscarRecusaCEPInvalidoSemRede(t *testing.T) {
	c := novoTeste(t, func(w http.ResponseWriter, r *http.Request) { t.Error("não deveria sair para a rede") })
	for _, v := range []string{"", "1234567", "123456789", "0100100a", "../etc/pw"} {
		if _, err := c.Buscar(context.Background(), v); !errors.Is(err, ErrCEPInvalido) {
			t.Errorf("%q: %v", v, err)
		}
	}
}

func TestURLPrecisaSerHTTPS(t *testing.T) {
	if _, err := NovoCliente(Config{BaseURL: "http://viacep.com.br"}); err == nil {
		t.Error("http deveria ser recusado")
	}
}
