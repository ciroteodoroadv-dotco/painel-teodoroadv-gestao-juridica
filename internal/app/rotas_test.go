package app

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
)

// Enumera TODAS as rotas registradas: nada se apaga e nenhuma escrita fica sem permissão.
func TestRotasNaoApagamEExigemPermissao(t *testing.T) {
	e := newEnv(t)
	rotas := e.rt.Routes()
	if len(rotas) < 60 {
		t.Fatalf("poucas rotas enumeradas (%d)", len(rotas))
	}
	apaga := regexp.MustCompile(`(?i)excluir|apagar|remover|delete`)
	registros := regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE) /(tarefas|audiencias|agenda|diligencias|intimacoes)(/|$)`)
	publicasDeEscrita := map[string]bool{"POST /login": true, "POST /logout": true}
	for _, r := range rotas {
		if registros.MatchString(r.Pattern) && apaga.MatchString(r.Pattern) {
			t.Errorf("rota de exclusão num registro que não se apaga: %s", r.Pattern)
		}
		metodo := strings.SplitN(r.Pattern, " ", 2)[0]
		if metodo == "DELETE" {
			t.Errorf("nenhuma rota usa DELETE: %s", r.Pattern)
		}
		if metodo != "GET" && r.Perm == auth.Public && !publicasDeEscrita[r.Pattern] {
			t.Errorf("rota de escrita sem permissão declarada: %s", r.Pattern)
		}
		if r.Perm == "" {
			t.Errorf("rota sem permissão: %s", r.Pattern)
		}
	}
}

// As mensagens do B para quando a ação deu certo mas a recarga da lista falhou existem na tabela de forms.js.
func TestFormsJSTemMensagensDeRecargaDoB(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "web", "static", "forms.js"))
	if err != nil {
		t.Fatal(err)
	}
	js := string(b)
	for _, m := range []string{
		"Ação registrada, mas houve falha ao recarregar a agenda.",
		"Ação registrada, mas houve falha ao recarregar as audiências.",
		"Ação registrada, mas houve falha ao recarregar as diligências.",
	} {
		if !strings.Contains(js, m) {
			t.Errorf("forms.js sem %q", m)
		}
	}
	for _, chave := range []string{"agenda:", "audiencias:", "diligencias:"} {
		if !strings.Contains(js, chave) {
			t.Errorf("tabela de mensagens sem a chave %q", chave)
		}
	}
}
