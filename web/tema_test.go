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

var scriptsDoPainel = []string{"app.js", "drawer.js", "forms.js", "mascaras.js"}

func TestScriptsSemEvalNemInline(t *testing.T) {
	for _, nome := range scriptsDoPainel {
		js := lerStatic(t, nome)
		for _, proibido := range []string{"innerHTML", "eval(", "new Function", "document.write", ".onclick =", `setAttribute("on`, "insertAdjacentHTML"} {
			if strings.Contains(js, proibido) {
				t.Errorf("%s contém %q", nome, proibido)
			}
		}
		if n := strings.Count(js, "\n"); n > 300 {
			t.Errorf("%s tem %d linhas (limite 300)", nome, n)
		}
	}
}

func TestMascarasPortadasDoB(t *testing.T) {
	js := lerStatic(t, "mascaras.js")
	for _, f := range []string{"maskCPF", "maskCNPJ", "maskCEP", "maskPhone", "maskCNJ", "maskCurrencyInput", "isValidCPF", "isValidCNPJ"} {
		if !strings.Contains(js, "function "+f+"(") {
			t.Errorf("mascaras.js sem a função %s", f)
		}
	}
}

func TestDrawerNaoFechaPorEscNemPorCliqueFora(t *testing.T) {
	js := lerStatic(t, "drawer.js")
	for _, proibido := range []string{"Escape", "keydown", "keyup", "keypress"} {
		if strings.Contains(js, proibido) {
			t.Errorf("drawer.js contém %q: o drawer só fecha pelos botões dele", proibido)
		}
	}
	if !strings.Contains(js, "data-drawer-fechar") || !strings.Contains(js, "data-drawer-sobrepor") {
		t.Error("drawer.js sem o contrato data-drawer-fechar/data-drawer-sobrepor")
	}
}
