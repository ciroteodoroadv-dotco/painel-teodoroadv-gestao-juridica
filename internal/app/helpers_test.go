package app

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// pessoaForm monta o formulário do drawer "Nova pessoa" (pessoa física) com dados fictícios.
func pessoaForm(nome, doc string) url.Values {
	return url.Values{"tipo_pessoa": {"Física"}, "nome_completo": {nome}, "cpf": {doc}, "nacionalidade": {"Brasileiro(a)"},
		"email": {"p@exemplo.test"}, "telefone": {"(11) 91234-5678"}}
}

// criarPessoa cadastra uma pessoa física pelo formulário e devolve o id (cabeçalho X-Painel-Id).
func (b *browser) criarPessoa(t *testing.T, nome, doc string) string {
	t.Helper()
	r, body := b.post("/pessoas", pessoaForm(nome, doc))
	if r.StatusCode != 200 || r.Header.Get("X-Painel-Id") == "" {
		t.Fatalf("criar pessoa %q = %d %s", nome, r.StatusCode, body)
	}
	return r.Header.Get("X-Painel-Id")
}

// processoForm monta o formulário do drawer "Novo processo" com o número informado e uma parte por pessoa.
// As linhas de Partes viajam em listas paralelas: pessoa, papel, papel "Outro" e marca de cliente.
func processoForm(numero string, partes ...[3]string) url.Values { // cada parte: {pessoaID, papel, "1" se cliente}
	f := url.Values{"numero_processo": {numero}}
	for _, p := range partes {
		f.Add("parte_pessoa", p[0])
		f.Add("parte_papel", p[1])
		f.Add("parte_papel_outro", "")
		f.Add("parte_cliente", p[2])
	}
	return f
}

// criarProcesso cadastra um processo mínimo (só o número) e devolve o id.
func (b *browser) criarProcesso(t *testing.T, numero string, partes ...[3]string) string {
	t.Helper()
	r, body := b.post("/processos", processoForm(numero, partes...))
	if r.StatusCode != 200 || r.Header.Get("X-Painel-Id") == "" {
		t.Fatalf("criar processo %q = %d %s", numero, r.StatusCode, body)
	}
	return r.Header.Get("X-Painel-Id")
}

// semCSRF envia o formulário SEM o token CSRF e devolve só o status (a escrita precisa ser negada).
func (b *browser) semCSRF(method, path string, form url.Values) int {
	b.e.t.Helper()
	req, _ := http.NewRequest(method, b.e.srv.URL+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "__Host-sid", Value: b.sid})
	c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Do(req)
	if err != nil {
		b.e.t.Fatal(err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}
