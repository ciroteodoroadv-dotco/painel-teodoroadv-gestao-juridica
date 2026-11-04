package cnj

import "testing"

// Dígitos verificadores calculados à mão (bc: 98 - (N7 AAAA J TR OOOO 00 mod 97)), fora do código.
func TestValido(t *testing.T) {
	ok := []string{"0000101-57.2025.8.26.0100", "1234567-37.2024.5.02.0001", "0001234-25.2023.4.03.6100", "0000001-30.2026.6.26.0005",
		" 00001015720258260100 ", "0000101-57.2025.8.26.0100 "}
	for _, n := range ok {
		if !Valido(n) {
			t.Errorf("%q deveria ser válido", n)
		}
	}
	for _, n := range []string{"", "0000101-58.2025.8.26.0100", "123", "0000101-57.2025.8.26.01000", "abc"} {
		if Valido(n) {
			t.Errorf("%q deveria ser inválido", n)
		}
	}
}

func TestFormatarEDigitos(t *testing.T) {
	if got := Formatar("00001015720258260100"); got != "0000101-57.2025.8.26.0100" {
		t.Errorf("Formatar = %q", got)
	}
	if got := Formatar("curto"); got != "curto" {
		t.Errorf("entrada fora do padrão fica como veio: %q", got)
	}
	if Digitos(" 0000101-57.2025.8.26.0100 ") != "00001015720258260100" {
		t.Error("Digitos")
	}
}

func TestAliasDataJud(t *testing.T) {
	a, err := AliasDataJud("0000101-57.2025.8.26.0100")
	if err != nil || a != "tjsp" {
		t.Errorf("alias = %q, %v", a, err)
	}
	if a, err = AliasDataJud("1234567-37.2024.5.02.0001"); err != nil || a != "trt2" {
		t.Errorf("trt2: %q %v", a, err)
	}
	for _, n := range []string{"0000101-58.2025.8.26.0100", "", "abc"} {
		if _, err := AliasDataJud(n); err == nil {
			t.Errorf("%q deveria dar erro", n)
		}
	}
	for _, j := range []string{"1", "2", "0"} {
		if _, err := aliasPorJTR(Digitos(numeroCom(j, "00"))); err == nil {
			t.Errorf("J=%s não tem endpoint público", j)
		}
	}
	if _, err := aliasPorJTR(Digitos(numeroCom("8", "28"))); err == nil {
		t.Error("TR 28 de TJ fora de faixa")
	}
	if _, err := aliasPorJTR(Digitos(numeroCom("5", "25"))); err == nil {
		t.Error("TR 25 de TRT fora de faixa")
	}
}
