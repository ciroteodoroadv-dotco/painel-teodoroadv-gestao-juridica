package cnj

import (
	"strings"
	"testing"
)

// Lista INDEPENDENTE, escrita à mão a partir da Res. CNJ 65/2008 (Anexo VI) e dos nomes de endpoint do wiki do DataJud
// conhecidos em 04/11/2026. O wiki (datajud-wiki.cnj.jus.br/api-publica/endpoints/) devolveu HTTP 403 ao WebFetch nesta
// data e NÃO foi contornado: o alias do TRE do Distrito Federal ("tre-dft") e os demais ficam como PENDÊNCIA DE CONFERÊNCIA
// no teste manual orientado (ver NOTAS-PLANO-6.md). Esta lista NÃO é gerada pela tabela do pacote.
var esperado = map[string]string{
	// J=8 Justiça Estadual (TR = código do tribunal)
	"8.01": "tjac", "8.02": "tjal", "8.03": "tjap", "8.04": "tjam", "8.05": "tjba", "8.06": "tjce", "8.07": "tjdft",
	"8.08": "tjes", "8.09": "tjgo", "8.10": "tjma", "8.11": "tjmt", "8.12": "tjms", "8.13": "tjmg", "8.14": "tjpa",
	"8.15": "tjpb", "8.16": "tjpr", "8.17": "tjpe", "8.18": "tjpi", "8.19": "tjrj", "8.20": "tjrn", "8.21": "tjrs",
	"8.22": "tjro", "8.23": "tjrr", "8.24": "tjsc", "8.25": "tjse", "8.26": "tjsp", "8.27": "tjto",
	// J=4 Justiça Federal
	"4.01": "trf1", "4.02": "trf2", "4.03": "trf3", "4.04": "trf4", "4.05": "trf5", "4.06": "trf6",
	// J=5 Justiça do Trabalho
	"5.00": "tst",
	"5.01": "trt1", "5.02": "trt2", "5.03": "trt3", "5.04": "trt4", "5.05": "trt5", "5.06": "trt6", "5.07": "trt7", "5.08": "trt8",
	"5.09": "trt9", "5.10": "trt10", "5.11": "trt11", "5.12": "trt12", "5.13": "trt13", "5.14": "trt14", "5.15": "trt15",
	"5.16": "trt16", "5.17": "trt17", "5.18": "trt18", "5.19": "trt19", "5.20": "trt20", "5.21": "trt21", "5.22": "trt22",
	"5.23": "trt23", "5.24": "trt24",
	// J=6 Justiça Eleitoral
	"6.00": "tse",
	"6.01": "tre-ac", "6.02": "tre-al", "6.03": "tre-ap", "6.04": "tre-am", "6.05": "tre-ba", "6.06": "tre-ce", "6.07": "tre-dft",
	"6.08": "tre-es", "6.09": "tre-go", "6.10": "tre-ma", "6.11": "tre-mt", "6.12": "tre-ms", "6.13": "tre-mg", "6.14": "tre-pa",
	"6.15": "tre-pb", "6.16": "tre-pr", "6.17": "tre-pe", "6.18": "tre-pi", "6.19": "tre-rj", "6.20": "tre-rn", "6.21": "tre-rs",
	"6.22": "tre-ro", "6.23": "tre-rr", "6.24": "tre-sc", "6.25": "tre-se", "6.26": "tre-sp", "6.27": "tre-to",
	// J=9 Justiça Militar estadual (MG, RS, SP); J=7 STM; J=3 STJ
	"9.13": "tjmmg", "9.21": "tjmrs", "9.26": "tjmsp",
	"7.00": "stm", "3.00": "stj",
}

func numeroCom(j, tr string) string { return "0000001-00.2025." + j + "." + tr + ".0001" }

func pad2(n int) string { return string([]byte{byte('0' + n/10), byte('0' + n%10)}) }

func TestAliasContraListaIndependente(t *testing.T) {
	for k, alias := range esperado {
		p := strings.Split(k, ".")
		got, err := aliasPorJTR(Digitos(numeroCom(p[0], p[1])))
		if err != nil || got != alias {
			t.Errorf("J.TR %s: alias = %q, %v; esperado %q", k, got, err, alias)
		}
	}
	if len(esperado) != 27+6+25+28+3+2 {
		t.Errorf("a lista à mão deveria ter 91 entradas, tem %d", len(esperado))
	}
	// Tudo o que a tabela do pacote aceita tem de estar na lista à mão (nada a mais).
	for j := 1; j <= 9; j++ {
		for tr := 0; tr <= 99; tr++ {
			js := string(rune('0' + j))
			k := js + "." + pad2(tr)
			if a, err := aliasPorJTR(Digitos(numeroCom(js, pad2(tr)))); err == nil && esperado[k] != a {
				t.Errorf("a tabela aceita %s -> %s, que não está na lista oficial à mão", k, a)
			}
		}
	}
}
