package web

import (
	"io/fs"
	"strings"
	"testing"
)

func lerStatic(t *testing.T, nome string) string {
	t.Helper()
	b, err := fs.ReadFile(Static(), nome)
	if err != nil {
		t.Fatalf("ler %s: %v", nome, err)
	}
	return string(b)
}

func TestTemaPainelContrato(t *testing.T) {
	css := strings.Join(strings.Fields(lerStatic(t, "painel.css")), "")
	for _, tok := range []string{
		"--ink:#000000;", "--paper:#EEF0F0;", "--panel:#FFFFFF;",
		"--wine:#7C2D3B;", "--wine-dark:#5E2029;", "--brass:#A9822F;",
		"--radius:3px;", ".drawer{width:480px;",
	} {
		if !strings.Contains(css, tok) {
			t.Errorf("painel.css sem %q", tok)
		}
	}
	for _, nome := range []string{"painel.css", "compat.css"} {
		c := lerStatic(t, nome)
		for _, proibido := range []string{"@import", "url(http", "url(//", "prefers-color-scheme", "expression("} {
			if strings.Contains(c, proibido) {
				t.Errorf("%s contém %q", nome, proibido)
			}
		}
	}
}

func TestAppJSSemEvalNemInline(t *testing.T) {
	js := lerStatic(t, "app.js")
	for _, proibido := range []string{"eval(", "new Function", "innerHTML", "document.write"} {
		if strings.Contains(js, proibido) {
			t.Errorf("app.js contém %q", proibido)
		}
	}
}
