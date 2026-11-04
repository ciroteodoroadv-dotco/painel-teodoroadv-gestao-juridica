package app

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
)

// viaCEPFalso responde como o ViaCEP, sem rede, e guarda o que foi pedido.
type viaCEPFalso struct {
	urls  []string
	corpo string
	porte int
}

func (v *viaCEPFalso) RoundTrip(r *http.Request) (*http.Response, error) {
	v.urls = append(v.urls, r.URL.String())
	st := v.porte
	if st == 0 {
		st = 200
	}
	return &http.Response{StatusCode: st, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(v.corpo)), Request: r}, nil
}

const enderecoSe = `{"cep":"01001-000","logradouro":"Praça da Sé","bairro":"Sé","localidade":"São Paulo","uf":"sp"}`

type respCEP struct {
	Logradouro, Bairro, Municipio, UF, Fonte, Erro string
}

func consultaCEP(t *testing.T, b *browser, cep string) (int, respCEP) {
	t.Helper()
	r, corpo := b.post("/pessoas/cep", url.Values{"cep": {cep}})
	var out respCEP
	if err := json.Unmarshal([]byte(corpo), &out); err != nil {
		t.Fatalf("resposta não é JSON (%d): %s", r.StatusCode, corpo)
	}
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q", ct)
	}
	return r.StatusCode, out
}

func TestCEPDesligadoNaoUsaRede(t *testing.T) {
	e := newEnvOpts(t, WithCEP(CEP{Transport: semRede{t}, PermitirHTTP: true, BaseURL: "http://viacep.invalid"}))
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	st, r := consultaCEP(t, adv, "01001-000")
	if st != 503 || r.Erro != "Consulta de CEP desligada neste ambiente." || r.Logradouro != "" {
		t.Errorf("desligado: %d %+v", st, r)
	}
}

func TestCEPFuncionaSoComOitoDigitosECache(t *testing.T) {
	f := &viaCEPFalso{corpo: enderecoSe}
	e := newEnvOpts(t, WithCEP(CEP{Ativo: true, Transport: f, PermitirHTTP: true, BaseURL: "http://viacep.test"}))
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	for _, ruim := range []string{"", "123", "0100100", "010010000", "abcdefgh"} {
		st, r := consultaCEP(t, adv, ruim)
		if st != 400 || r.Erro != "Informe um CEP com oito dígitos." {
			t.Errorf("%q: %d %+v", ruim, st, r)
		}
	}
	if len(f.urls) != 0 {
		t.Fatalf("CEP inválido não pode sair para a rede: %v", f.urls)
	}
	st, r := consultaCEP(t, adv, "01001-000")
	if st != 200 || r.Logradouro != "Praça da Sé" || r.Bairro != "Sé" || r.Municipio != "São Paulo" || r.UF != "SP" || r.Fonte != "ViaCEP (público, não oficial)" {
		t.Fatalf("consulta: %d %+v", st, r)
	}
	if len(f.urls) != 1 || f.urls[0] != "http://viacep.test/ws/01001000/json/" {
		t.Errorf("URL pedida: %v", f.urls)
	}
	consultaCEP(t, adv, "01001000")
	if len(f.urls) != 1 {
		t.Errorf("a segunda consulta do mesmo CEP deveria vir do cache: %v", f.urls)
	}
}

func TestCEPErrosNaoPreenchemNada(t *testing.T) {
	f := &viaCEPFalso{corpo: `{"erro": "true"}`}
	e := newEnvOpts(t, WithCEP(CEP{Ativo: true, Transport: f, PermitirHTTP: true, BaseURL: "http://viacep.test"}))
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	if st, r := consultaCEP(t, adv, "99999999"); st != 404 || r.Erro != "CEP não encontrado." || r.Logradouro != "" {
		t.Errorf("inexistente: %d %+v", st, r)
	}
	f.corpo, f.porte = `<html>`, 503
	st, r := consultaCEP(t, adv, "88888888")
	if st != 502 || r.Erro == "" || r.Logradouro != "" || r.UF != "" {
		t.Errorf("indisponível: %d %+v", st, r)
	}
}

func TestCEPExigeSessaoPermissaoCSRFELimite(t *testing.T) {
	f := &viaCEPFalso{corpo: enderecoSe}
	var logs bytes.Buffer
	e := newEnvOpts(t, WithCEP(CEP{Ativo: true, Transport: f, PermitirHTTP: true, BaseURL: "http://viacep.test", LimitePorMinuto: 3}))
	e.a.Log = slog.New(slog.NewJSONHandler(&logs, nil))
	if r, err := http.Post(e.srv.URL+"/pessoas/cep", "application/x-www-form-urlencoded", strings.NewReader("cep=01001000")); err == nil {
		r.Body.Close()
		if r.StatusCode == 200 {
			t.Error("sem sessão não pode consultar")
		}
	}
	adm, _ := e.user("adm@exemplo.test", auth.Administrador)
	if r, _ := adm.post("/pessoas/cep", url.Values{"cep": {"01001000"}}); r.StatusCode != 403 {
		t.Errorf("administrador e assistente não escrevem pessoas: %d", r.StatusCode)
	}
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)
	if r, _ := ass.post("/pessoas/cep", url.Values{"cep": {"01001000"}}); r.StatusCode != 403 {
		t.Errorf("assistente não escreve pessoas: %d", r.StatusCode)
	}
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	if c := adv.semCSRF("POST", "/pessoas/cep", url.Values{"cep": {"01001000"}}); c != 403 {
		t.Errorf("sem CSRF: %d", c)
	}
	if len(f.urls) != 0 {
		t.Fatalf("nada deveria ter saído: %v", f.urls)
	}
	for i, cep := range []string{"11111111", "22222222", "33333333"} {
		if st, _ := consultaCEP(t, adv, cep); st != 200 {
			t.Fatalf("consulta %d = %d", i, st)
		}
	}
	if st, r := consultaCEP(t, adv, "44444444"); st != 429 || r.Erro == "" {
		t.Errorf("limite por usuário: %d %+v", st, r)
	}
	adv2, _ := e.user("adv2@exemplo.test", auth.Advogado)
	if st, _ := consultaCEP(t, adv2, "55555555"); st != 200 {
		t.Errorf("o limite é por usuário: %d", st)
	}
	for _, c := range []string{"11111111", "22222222", "01001000", "01001-000"} {
		if strings.Contains(logs.String(), c) {
			t.Errorf("CEP %s apareceu no log", c)
		}
	}
}

func TestFormularioDePessoaTemBotaoBuscarCEP(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	_, corpo := adv.get("/pessoas/nova")
	if !strings.Contains(corpo, `data-buscar-cep`) || !strings.Contains(corpo, `data-cep-fonte`) || !strings.Contains(corpo, `data-cep-url="/pessoas/cep"`) {
		t.Errorf("formulário sem o botão do CEP, a linha da fonte ou a URL da consulta")
	}
}
