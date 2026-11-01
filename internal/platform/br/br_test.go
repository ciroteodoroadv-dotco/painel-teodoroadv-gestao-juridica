package br

import "testing"

func TestCPF(t *testing.T) {
	for _, ok := range []string{"529.982.247-25", "52998224725", "111.444.777-35"} {
		if _, err := NormalizeCPF(ok); err != nil {
			t.Errorf("CPF %q deveria ser válido: %v", ok, err)
		}
	}
	for _, bad := range []string{"529.982.247-24", "111.111.111-11", "123", "", "abc", "000.000.000-00"} {
		if _, err := NormalizeCPF(bad); err == nil {
			t.Errorf("CPF %q deveria ser inválido", bad)
		}
	}
}

func TestCNPJ(t *testing.T) {
	for _, ok := range []string{"11.222.333/0001-81", "11222333000181"} {
		if _, err := NormalizeCNPJ(ok); err != nil {
			t.Errorf("CNPJ %q deveria ser válido: %v", ok, err)
		}
	}
	for _, bad := range []string{"11.222.333/0001-80", "00000000000000", "123"} {
		if _, err := NormalizeCNPJ(bad); err == nil {
			t.Errorf("CNPJ %q deveria ser inválido", bad)
		}
	}
}

func TestCNJ(t *testing.T) {
	// Exemplo oficial citado na Resolução CNJ 65/2008 (formato) e números montados pelo algoritmo.
	n := MakeCNJ("0000001", "2020", "8", "26", "0100")
	got, err := NormalizeCNJ(n)
	if err != nil || got != n {
		t.Fatalf("número montado deveria validar: %q %v", got, err)
	}
	// Verificação independente (ISO 7064 MOD 97-10, citada na Res. CNJ 65): o número com o DV no
	// fim, lido como inteiro, deixa resto 1 na divisão por 97.
	d := Digits(n) // NNNNNNN DD AAAA J TR OOOO
	rearranjado := d[0:7] + d[9:] + d[7:9]
	var resto int64
	for _, c := range rearranjado {
		resto = (resto*10 + int64(c-'0')) % 97
	}
	if resto != 1 {
		t.Errorf("DV gerado não satisfaz MOD 97-10: resto %d", resto)
	}
	semPontos := Digits(n)
	if g, err := NormalizeCNJ(semPontos); err != nil || g != n {
		t.Errorf("sem pontuação deveria normalizar para %q, veio %q %v", n, g, err)
	}
	// Troca de um dígito invalida.
	errado := []byte(n)
	errado[1] = '9'
	if _, err := NormalizeCNJ(string(errado)); err == nil {
		t.Error("dígito alterado deveria invalidar")
	}
	for _, bad := range []string{"", "123", "0000001-00.2020.8.26.0100x1"} {
		if _, err := NormalizeCNJ(bad); err == nil {
			t.Errorf("%q deveria ser inválido", bad)
		}
	}
}

func TestMascaras(t *testing.T) {
	if m := MaskDocumento("52998224725"); m != "***.***.***-25" {
		t.Errorf("máscara CPF = %q", m)
	}
	if m := MaskDocumento("11222333000181"); m != "**.***.***/****-81" {
		t.Errorf("máscara CNPJ = %q", m)
	}
	if f := FormatDocumento("52998224725"); f != "529.982.247-25" {
		t.Errorf("formato CPF = %q", f)
	}
}
