package datajud

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"
)

const chaveSecreta = "CHAVE-SECRETA-123"

type servidor struct {
	srv     *httptest.Server
	chamou  atomic.Int32
	ultimo  atomic.Value // *http.Request
	corpo   atomic.Value // string
	resp    func(n int32, w http.ResponseWriter)
	esperas []time.Duration
}

func novoServidor(t *testing.T, resp func(n int32, w http.ResponseWriter)) *servidor {
	s := &servidor{resp: resp}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := s.chamou.Add(1)
		b := make([]byte, 4096)
		k, _ := r.Body.Read(b)
		s.ultimo.Store(r)
		s.corpo.Store(string(b[:k]))
		s.resp(n, w)
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *servidor) cliente(t *testing.T, chave string) *Cliente {
	c, err := Novo(Config{BaseURL: s.srv.URL, PermitirHTTP: true, Chave: func() string { return chave },
		EsperaTaxa: time.Minute, Intervalo: time.Second, Dormir: func(d time.Duration) { s.esperas = append(s.esperas, d) }})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

const doisGraus = `{"hits":{"hits":[
 {"_source":{"numeroProcesso":"00001015720258260100","tribunal":"TJSP","grau":"G1","orgaoJulgador":{"codigo":1,"nome":"1ª Vara Cível"},
   "movimentos":[{"codigo":26,"nome":"Distribuição","dataHora":"2025-03-10T10:00:00.000Z"},
                 {"codigo":123,"nome":"Intimação","dataHora":"2025-03-12T09:30:00.000Z","complementosTabelados":[{"codigo":1,"valor":5,"nome":"prazo","descricao":"dias úteis"}]}]}},
 {"_source":{"numeroProcesso":"00001015720258260100","tribunal":"TJSP","grau":"G2","orgaoJulgador":{"codigo":2,"nome":"Câmara"},
   "movimentos":[{"codigo":200,"nome":"Recebimento","dataHora":"2025-06-01T00:00:00.000Z"}]}}]}}`

func TestLeTodosOsHitsEOsCamposDoMovimento(t *testing.T) {
	s := novoServidor(t, func(_ int32, w http.ResponseWriter) { _, _ = w.Write([]byte(doisGraus)) })
	r, err := s.cliente(t, chaveSecreta).Consultar(context.Background(), "tjsp", "0000101-57.2025.8.26.0100")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Processos) != 2 || r.Processos[0].Grau != "G1" || r.Processos[1].Grau != "G2" || r.SemRegistro {
		t.Fatalf("deveria ler G1 e G2 (regressão do Hits[0] do B): %+v", r)
	}
	m := r.Processos[0].Movimentos[1]
	if m.Grau != "G1" || m.OrgaoJulgador != "1ª Vara Cível" || m.Codigo != "123" || m.Nome != "Intimação" || m.DataHora.IsZero() {
		t.Errorf("campos do movimento: %+v", m)
	}
	if len(m.Complementos) != 1 || m.Complementos[0].Nome != "prazo" || m.Complementos[0].Valor != "5" || m.Complementos[0].Descricao != "dias úteis" {
		t.Errorf("complementos tabelados: %+v", m.Complementos)
	}
	req := s.ultimo.Load().(*http.Request)
	if req.Method != "POST" || req.URL.Path != "/api_publica_tjsp/_search" || req.Header.Get("Authorization") != "APIKey "+chaveSecreta {
		t.Errorf("requisição: %s %s auth=%q", req.Method, req.URL.Path, req.Header.Get("Authorization"))
	}
	if c := s.corpo.Load().(string); !strings.Contains(c, `"numeroProcesso":"00001015720258260100"`) || !strings.Contains(c, "match") {
		t.Errorf("corpo: %s", c)
	}
	if strings.Contains(req.URL.String(), chaveSecreta) || strings.Contains(req.UserAgent(), "@") {
		t.Error("chave na URL ou e-mail no User-Agent")
	}
}

func TestZeroHitsESemRegistro(t *testing.T) {
	s := novoServidor(t, func(_ int32, w http.ResponseWriter) { _, _ = w.Write([]byte(`{"hits":{"hits":[]}}`)) })
	r, err := s.cliente(t, chaveSecreta).Consultar(context.Background(), "tjsp", "0000101-57.2025.8.26.0100")
	if err != nil || !r.SemRegistro || len(r.Processos) != 0 {
		t.Errorf("zero hits = %+v, %v", r, err)
	}
}

func TestTaxa429EsperaERepete(t *testing.T) {
	s := novoServidor(t, func(n int32, w http.ResponseWriter) {
		if n == 1 {
			w.WriteHeader(429)
			return
		}
		_, _ = w.Write([]byte(doisGraus))
	})
	r, err := s.cliente(t, chaveSecreta).Consultar(context.Background(), "tjsp", "0000101-57.2025.8.26.0100")
	if err != nil || len(r.Processos) != 2 || s.chamou.Load() != 2 {
		t.Fatalf("429: %+v %v chamadas=%d", r, err, s.chamou.Load())
	}
	if len(s.esperas) != 1 || s.esperas[0] != time.Minute {
		t.Errorf("deveria esperar EsperaTaxa uma vez: %v", s.esperas)
	}
}

func TestErrosNaoDevolvemParciais(t *testing.T) {
	casos := map[string]func(int32, http.ResponseWriter){
		"500":  func(_ int32, w http.ResponseWriter) { w.WriteHeader(500) },
		"json": func(_ int32, w http.ResponseWriter) { _, _ = w.Write([]byte(`{"hits":`)) },
		"gigante": func(_ int32, w http.ResponseWriter) {
			_, _ = w.Write([]byte(`{"x":"` + strings.Repeat("a", 3<<20) + `"}`))
		},
		"sem hits": func(_ int32, w http.ResponseWriter) { _, _ = w.Write([]byte(`{"foo":1}`)) },
	}
	for nome, h := range casos {
		s := novoServidor(t, h)
		c := s.cliente(t, chaveSecreta)
		c.cfg.MaxBytes = 1 << 20
		r, err := c.Consultar(context.Background(), "tjsp", "0000101-57.2025.8.26.0100")
		if err == nil || len(r.Processos) != 0 {
			t.Errorf("%s: deveria dar erro sem parciais: %+v %v", nome, r, err)
		}
		if err != nil && strings.Contains(err.Error(), chaveSecreta) {
			t.Errorf("%s: a chave vazou no erro", nome)
		}
	}
}

func TestChaveAusenteNaoChamaARede(t *testing.T) {
	s := novoServidor(t, func(_ int32, w http.ResponseWriter) { _, _ = w.Write([]byte(doisGraus)) })
	_, err := s.cliente(t, "  ").Consultar(context.Background(), "tjsp", "0000101-57.2025.8.26.0100")
	if err == nil || !strings.Contains(err.Error(), "Chave do DataJud não configurada") || s.chamou.Load() != 0 {
		t.Errorf("chave ausente: %v chamadas=%d", err, s.chamou.Load())
	}
}

func TestChaveNaoVazaNaConfig(t *testing.T) {
	cfg := Config{BaseURL: "https://x.test", Chave: func() string { return chaveSecreta }}
	for _, f := range []string{fmt.Sprint(cfg), fmt.Sprintf("%v %+v %s", cfg, cfg, cfg), cfg.String()} {
		if strings.Contains(f, chaveSecreta) {
			t.Error("a chave apareceu na representação da config")
		}
	}
}

func TestExigeHTTPSERecusaRedirecionamento(t *testing.T) {
	if _, err := Novo(Config{BaseURL: "http://x.test"}); err == nil {
		t.Error("http sem a opção de teste deveria ser recusado")
	}
	outro := novoServidor(t, func(_ int32, w http.ResponseWriter) { _, _ = w.Write([]byte(doisGraus)) })
	s := novoServidor(t, func(_ int32, w http.ResponseWriter) {
		w.Header().Set("Location", strings.Replace(outro.srv.URL, "127.0.0.1", "localhost", 1)+"/x")
		w.WriteHeader(307)
	})
	_, err := s.cliente(t, chaveSecreta).Consultar(context.Background(), "tjsp", "0000101-57.2025.8.26.0100")
	if err == nil || outro.chamou.Load() != 0 {
		t.Errorf("redirecionamento a outro host deveria ser recusado: %v (outro=%d)", err, outro.chamou.Load())
	}
}

func TestIntervaloEntreRequisicoes(t *testing.T) {
	s := novoServidor(t, func(_ int32, w http.ResponseWriter) { _, _ = w.Write([]byte(doisGraus)) })
	c := s.cliente(t, chaveSecreta)
	for range 2 {
		if _, err := c.Consultar(context.Background(), "tjsp", "0000101-57.2025.8.26.0100"); err != nil {
			t.Fatal(err)
		}
	}
	if len(s.esperas) != 1 || s.esperas[0] != time.Second {
		t.Errorf("a 1ª não espera, a 2ª espera o intervalo: %v", s.esperas)
	}
}

func TestCorteTextoPorCaracteres(t *testing.T) {
	longo := strings.Repeat("ç", 600)
	s := novoServidor(t, func(_ int32, w http.ResponseWriter) {
		_, _ = fmt.Fprintf(w, `{"hits":{"hits":[{"_source":{"grau":"G1","movimentos":[{"codigo":1,"nome":"N","dataHora":"2025-01-01T00:00:00Z","complementosTabelados":[{"nome":"%s","valor":"v"}]}]}}]}}`, longo)
	})
	r, err := s.cliente(t, chaveSecreta).Consultar(context.Background(), "tjsp", "0000101-57.2025.8.26.0100")
	if err != nil {
		t.Fatal(err)
	}
	comp := r.Processos[0].Movimentos[0].Complemento
	if !utf8.ValidString(comp) || utf8.RuneCountInString(comp) != MaxComplemento {
		t.Errorf("complemento resumido deveria ter %d caracteres (não bytes): %d", MaxComplemento, utf8.RuneCountInString(comp))
	}
}

func TestChaveDoMovimento(t *testing.T) {
	m := Movimento{Grau: "G1", Codigo: "26", Nome: "Distribuição", Complemento: "x", DataHora: time.Date(2025, 3, 10, 10, 0, 0, 0, time.UTC)}
	if c1, c2 := Chave(m), Chave(m); c1 != c2 || c1 == "" {
		t.Error("Chave instável")
	}
	a := Movimento{Grau: "G1", Nome: "a|b", Complemento: "c"}
	b := Movimento{Grau: "G1", Nome: "a", Complemento: "b|c"}
	if Chave(a) == Chave(b) {
		t.Error("colisão entre campos")
	}
	if Chave(m) == Chave(Movimento{Grau: "G2", Codigo: "26", Nome: "Distribuição", Complemento: "x", DataHora: m.DataHora}) {
		t.Error("o grau tem de entrar na chave")
	}
}
