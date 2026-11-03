package djen

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Todos os testes falam só com httptest.Server: nenhuma requisição sai da máquina.

type espiao struct {
	mu        sync.Mutex
	urls      []string
	cabecalho []http.Header
}

func (e *espiao) anota(r *http.Request) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.urls = append(e.urls, r.URL.String())
	e.cabecalho = append(e.cabecalho, r.Header.Clone())
}

func item(i int) map[string]any {
	return map[string]any{"id": 1000 + i, "siglaTribunal": "TJXX", "nomeOrgao": "1ª Vara de Exemplo",
		"numero_processo": "00001012520258260100", "numeroprocessocommascara": "0000101-25.2025.8.26.0100",
		"tipoComunicacao": "Intimação", "data_disponibilizacao": "2026-11-02", "texto": "<p>Fica intimada a parte (exemplo).</p>",
		"link": "https://exemplo.test/c/" + strconv.Itoa(i)}
}

// paginado responde com total itens fictícios, respeitando pagina e itensPorPagina.
func paginado(total int, e *espiao) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if e != nil {
			e.anota(r)
		}
		pag, _ := strconv.Atoi(r.URL.Query().Get("pagina"))
		por, _ := strconv.Atoi(r.URL.Query().Get("itensPorPagina"))
		var itens []map[string]any
		for i := (pag - 1) * por; i < pag*por && i < total; i++ {
			itens = append(itens, item(i))
		}
		if itens == nil {
			itens = []map[string]any{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "count": total, "items": itens})
	}
}

type dorminhoco struct {
	mu sync.Mutex
	ds []time.Duration
}

func (d *dorminhoco) dorme(x time.Duration) { d.mu.Lock(); d.ds = append(d.ds, x); d.mu.Unlock() }
func (d *dorminhoco) conta(x time.Duration) (n int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, v := range d.ds {
		if v == x {
			n++
		}
	}
	return n
}

func novo(t *testing.T, h http.Handler) (*Cliente, *httptest.Server, *dorminhoco) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	d := &dorminhoco{}
	c, err := Novo(Config{BaseURL: srv.URL + "/api/v1/comunicacao", PermitirHTTP: true, Timeout: 2 * time.Second,
		IntervaloMinimo: time.Second, EsperaTaxa: 60 * time.Second, MaxBytesResposta: 1 << 20, MaxPaginas: 100, MaxResultados: 10000,
		Dormir: d.dorme})
	if err != nil {
		t.Fatal(err)
	}
	return c, srv, d
}

var (
	d1 = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	d3 = time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
)

func TestTresPaginasTodosOsItens(t *testing.T) {
	c, _, d := novo(t, paginado(250, nil))
	r, err := c.Coletar(t.Context(), "123456", "SP", d1, d1)
	if err != nil || len(r.Itens) != 250 || r.Parcial {
		t.Fatalf("itens=%d parcial=%v err=%v", len(r.Itens), r.Parcial, err)
	}
	if r.Itens[0].Processo != "0000101-25.2025.8.26.0100" || r.Itens[0].Tribunal != "TJXX" || r.Itens[0].Tipo != "Intimação" ||
		r.Itens[0].ID != "1000" || r.Itens[0].DataPublicacao.Format("2006-01-02") != "2026-11-02" || !strings.HasPrefix(r.Itens[0].Link, "https://") {
		t.Errorf("normalização: %+v", r.Itens[0])
	}
	if strings.Contains(r.Itens[0].Texto, "<p>") {
		t.Errorf("o HTML do texto deveria ser limpo: %q", r.Itens[0].Texto)
	}
	if d.conta(time.Second) != 2 {
		t.Errorf("intervalo mínimo entre requisições (3 páginas = 2 esperas de 1 s): %v", d.ds)
	}
}

func TestLimite429EsperaEMesmaPagina(t *testing.T) {
	var mu sync.Mutex
	vistas := map[string]int{}
	base := paginado(150, nil)
	c, _, d := novo(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		p := r.URL.Query().Get("pagina")
		vistas[p]++
		n := vistas[p]
		mu.Unlock()
		if p == "2" && n == 1 {
			http.Error(w, "muitas requisições", http.StatusTooManyRequests)
			return
		}
		base(w, r)
	}))
	r, err := c.Coletar(t.Context(), "123456", "SP", d1, d1)
	if err != nil || len(r.Itens) != 150 {
		t.Fatalf("itens=%d err=%v", len(r.Itens), err)
	}
	if d.conta(60*time.Second) != 1 || vistas["2"] != 2 {
		t.Errorf("deveria dormir 60 s uma vez e repetir a mesma página: dormidos=%v vistas=%v", d.ds, vistas)
	}
}

func TestLimite429PersistenteDesiste(t *testing.T) {
	c, _, d := novo(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "x", 429) }))
	if _, err := c.Coletar(t.Context(), "123456", "SP", d1, d1); err == nil {
		t.Fatal("429 sem fim deveria virar erro")
	}
	if d.conta(60*time.Second) > 5 {
		t.Errorf("tentativas demais: %v", d.ds)
	}
}

func TestPaginaRepetidaNaoEntraEmLaco(t *testing.T) {
	chamadas := 0
	c, _, _ := novo(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamadas++
		// sempre a mesma página de 100 itens, anunciando 250
		var itens []map[string]any
		for i := 0; i < 100; i++ {
			itens = append(itens, item(i))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"count": 250, "items": itens})
	}))
	r, err := c.Coletar(t.Context(), "123456", "SP", d1, d1)
	if err == nil {
		t.Fatalf("página repetida com contagem incompleta deveria ser erro, não sucesso (%d itens)", len(r.Itens))
	}
	if chamadas > 3 {
		t.Errorf("laço: %d chamadas", chamadas)
	}
	if len(r.Itens) != 0 {
		t.Error("em erro nenhum item parcial pode ser devolvido")
	}
}

func TestTetoDividePorDia(t *testing.T) {
	e := &espiao{}
	c, _, _ := novo(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e.anota(r)
		q := r.URL.Query()
		if q.Get("dataDisponibilizacaoInicio") != q.Get("dataDisponibilizacaoFim") { // janela de vários dias: estoura o teto
			_ = json.NewEncoder(w).Encode(map[string]any{"count": 10000, "items": []map[string]any{item(1)}})
			return
		}
		dia, _ := time.Parse("2006-01-02", q.Get("dataDisponibilizacaoInicio"))
		base := dia.Day() * 100
		_ = json.NewEncoder(w).Encode(map[string]any{"count": 2, "items": []map[string]any{item(base), item(base + 1)}})
	}))
	r, err := c.Coletar(t.Context(), "123456", "SP", d1, d3)
	if err != nil || r.Parcial || len(r.Itens) != 6 {
		t.Fatalf("itens=%d parcial=%v err=%v (esperado 6 itens, um dia de cada vez)", len(r.Itens), r.Parcial, err)
	}
}

func TestTetoNumDiaSoMarcaParcial(t *testing.T) {
	c, _, _ := novo(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"count": 10000, "items": []map[string]any{item(1)}})
	}))
	r, err := c.Coletar(t.Context(), "123456", "SP", d1, d1)
	if err != nil || !r.Parcial || r.Motivo == "" {
		t.Fatalf("um dia que ainda estoura o teto deve marcar Parcial com motivo: %+v err=%v", r, err)
	}
}

func TestErro500NoMeioNaoDevolveParcial(t *testing.T) {
	base := paginado(250, nil)
	c, _, _ := novo(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("pagina") == "2" {
			http.Error(w, "falha", http.StatusInternalServerError)
			return
		}
		base(w, r)
	}))
	r, err := c.Coletar(t.Context(), "123456", "SP", d1, d1)
	if err == nil || len(r.Itens) != 0 || r.Parcial {
		t.Fatalf("500 no meio deve ser erro sem itens: itens=%d parcial=%v err=%v", len(r.Itens), r.Parcial, err)
	}
	if strings.Contains(err.Error(), "Fica intimada") {
		t.Error("o erro não pode carregar texto de intimação")
	}
}

func TestJSONInvalidoETimeout(t *testing.T) {
	c, _, _ := novo(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("{não é json")) }))
	if _, err := c.Coletar(t.Context(), "123456", "SP", d1, d1); err == nil {
		t.Error("JSON inválido deveria ser erro")
	}
	c, _, _ = novo(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"count":0}`)) }))
	if _, err := c.Coletar(t.Context(), "123456", "SP", d1, d1); err == nil {
		t.Error("resposta sem 'items' não pode ser tomada como 'nenhuma intimação'")
	}
	lento := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-lento }))
	defer srv.Close()
	defer close(lento)
	c2, _ := Novo(Config{BaseURL: srv.URL, PermitirHTTP: true, Timeout: 100 * time.Millisecond, Dormir: func(time.Duration) {}})
	if _, err := c2.Coletar(t.Context(), "123456", "SP", d1, d1); err == nil {
		t.Error("tempo limite deveria ser erro")
	}
}

func TestRespostaGigante(t *testing.T) {
	c, _, _ := novo(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"count":1,"items":[{"texto":"` + strings.Repeat("a", 2<<20) + `"}]}`))
	}))
	if _, err := c.Coletar(t.Context(), "123456", "SP", d1, d1); err == nil {
		t.Fatal("resposta acima do limite de tamanho deveria ser erro")
	}
}

func TestRedirecionamentoParaOutroHostRecusado(t *testing.T) {
	var acertos int
	outro := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { acertos++; paginado(1, nil)(w, r) }))
	defer outro.Close()
	c, _, _ := novo(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, outro.URL+"/api", http.StatusFound)
	}))
	// 127.0.0.1:porta diferente = host diferente para o cliente
	if _, err := c.Coletar(t.Context(), "123456", "SP", d1, d1); err == nil {
		t.Fatal("redirecionamento a outro host deveria ser erro")
	}
	if acertos != 0 {
		t.Errorf("o outro host não pode ser acessado (%d acertos)", acertos)
	}
}

func TestNenhumCabecalhoNemURLTemArroba(t *testing.T) {
	e := &espiao{}
	c, _, _ := novo(t, paginado(3, e))
	if _, err := c.Coletar(t.Context(), "123456", "SP", d1, d1); err != nil {
		t.Fatal(err)
	}
	for _, u := range e.urls {
		if strings.Contains(u, "@") {
			t.Errorf("URL com @: %s", u)
		}
		if !strings.Contains(u, "numeroOab=123456") || !strings.Contains(u, "ufOab=SP") {
			t.Errorf("parâmetros da OAB ausentes: %s", u)
		}
	}
	for _, h := range e.cabecalho {
		if h.Get("User-Agent") != "painel-teodoro" {
			t.Errorf("User-Agent = %q", h.Get("User-Agent"))
		}
		for k, v := range h {
			if strings.Contains(strings.Join(v, ","), "@") {
				t.Errorf("cabeçalho %s com @", k)
			}
		}
	}
}

func TestEntradasInvalidasENaoHTTPS(t *testing.T) {
	if _, err := Novo(Config{BaseURL: "http://comunicaapi.pje.jus.br/api/v1/comunicacao"}); err == nil {
		t.Error("http:// sem a opção explícita de teste deveria ser recusado")
	}
	if _, err := Novo(Config{BaseURL: "https://comunicaapi.pje.jus.br/api/v1/comunicacao"}); err != nil {
		t.Errorf("https deveria ser aceito: %v", err)
	}
	c, _, _ := novo(t, paginado(1, nil))
	for _, oab := range [][2]string{{"12 34", "SP"}, {"123", "S"}, {"", "SP"}, {"123@x", "SP"}, {"123", "S1"}} {
		if _, err := c.Coletar(t.Context(), oab[0], oab[1], d1, d1); err == nil {
			t.Errorf("OAB %v deveria ser recusada antes de qualquer requisição", oab)
		}
	}
	if _, err := c.Coletar(t.Context(), "123", "SP", d3, d1); err == nil {
		t.Error("janela invertida deveria ser recusada")
	}
}

func TestDataInvalidaNaoDescartaEmSilencio(t *testing.T) {
	c, _, _ := novo(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		it := item(1)
		it["data_disponibilizacao"] = "ontem"
		_ = json.NewEncoder(w).Encode(map[string]any{"count": 1, "items": []map[string]any{it}})
	}))
	if _, err := c.Coletar(t.Context(), "123456", "SP", d1, d1); err == nil {
		t.Fatal("comunicação com data inválida deveria ser erro, nunca descartada em silêncio")
	}
}

func TestLinkHTTPEDataBR(t *testing.T) {
	c, _, _ := novo(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		it := item(1)
		it["link"] = "http://inseguro.test/x"
		it["data_disponibilizacao"] = "02/11/2026"
		_ = json.NewEncoder(w).Encode(map[string]any{"count": 1, "items": []map[string]any{it}})
	}))
	r, err := c.Coletar(t.Context(), "123456", "SP", d1, d1)
	if err != nil || len(r.Itens) != 1 {
		t.Fatalf("%v", err)
	}
	if r.Itens[0].Link != "" || r.Itens[0].DataPublicacao.Format("2006-01-02") != "2026-11-02" {
		t.Errorf("link http:// deve ser descartado e data dd/mm/aaaa aceita: %+v", r.Itens[0])
	}
}

var _ = fmt.Sprint
