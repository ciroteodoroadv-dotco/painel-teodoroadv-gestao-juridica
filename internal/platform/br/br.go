// Package br valida e formata documentos brasileiros: CPF, CNPJ e número único de processo (CNJ).
package br

import (
	"errors"
	"fmt"
	"strings"
)

// Digits mantém só os dígitos de s.
func Digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func allSame(d string) bool {
	for i := 1; i < len(d); i++ {
		if d[i] != d[0] {
			return false
		}
	}
	return true
}

// check calcula um dígito verificador módulo 11 com os pesos dados.
func check(d string, weights []int) int {
	sum := 0
	for i, w := range weights {
		sum += int(d[i]-'0') * w
	}
	r := sum % 11
	if r < 2 {
		return 0
	}
	return 11 - r
}

// NormalizeCPF devolve os 11 dígitos de um CPF válido.
func NormalizeCPF(s string) (string, error) {
	d := Digits(s)
	if len(d) != 11 || allSame(d) {
		return "", errors.New("CPF inválido")
	}
	w1 := []int{10, 9, 8, 7, 6, 5, 4, 3, 2}
	w2 := []int{11, 10, 9, 8, 7, 6, 5, 4, 3, 2}
	if check(d, w1) != int(d[9]-'0') || check(d, w2) != int(d[10]-'0') {
		return "", errors.New("CPF inválido")
	}
	return d, nil
}

// NormalizeCNPJ devolve os 14 dígitos de um CNPJ válido (numérico).
func NormalizeCNPJ(s string) (string, error) {
	d := Digits(s)
	if len(d) != 14 || allSame(d) {
		return "", errors.New("CNPJ inválido")
	}
	w1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	w2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	if check(d, w1) != int(d[12]-'0') || check(d, w2) != int(d[13]-'0') {
		return "", errors.New("CNPJ inválido")
	}
	return d, nil
}

// NormalizeDocumento valida CPF (PF) ou CNPJ (PJ) conforme o tipo.
func NormalizeDocumento(tipo, s string) (string, error) {
	switch tipo {
	case "PF":
		return NormalizeCPF(s)
	case "PJ":
		return NormalizeCNPJ(s)
	}
	return "", errors.New("tipo de pessoa inválido")
}

// MaskDocumento esconde o documento, deixando só os 2 últimos dígitos (ex.: ***.***.***-25).
func MaskDocumento(digits string) string {
	switch len(digits) {
	case 11:
		return "***.***.***-" + digits[9:]
	case 14:
		return "**.***.***/****-" + digits[12:]
	}
	return ""
}

// FormatDocumento aplica a máscara de exibição completa.
func FormatDocumento(d string) string {
	switch len(d) {
	case 11:
		return fmt.Sprintf("%s.%s.%s-%s", d[0:3], d[3:6], d[6:9], d[9:])
	case 14:
		return fmt.Sprintf("%s.%s.%s/%s-%s", d[0:2], d[2:5], d[5:8], d[8:12], d[12:])
	}
	return d
}

// cnjCheck calcula o DV do número único: 98 - ((NNNNNNN AAAA JTR OOOO 00) mod 97).
func cnjCheck(n7, aaaa, j, tr, oooo string) int {
	r := 0
	for _, c := range n7 + aaaa + j + tr + oooo + "00" {
		r = (r*10 + int(c-'0')) % 97
	}
	return 98 - r
}

// NormalizeCNJ valida o número único (Res. CNJ 65/2008) e devolve NNNNNNN-DD.AAAA.J.TR.OOOO.
func NormalizeCNJ(s string) (string, error) {
	d := Digits(s)
	if len(d) != 20 {
		return "", errors.New("número do processo deve ter 20 dígitos (NNNNNNN-DD.AAAA.J.TR.OOOO)")
	}
	n7, dv, aaaa, j, tr, oooo := d[0:7], d[7:9], d[9:13], d[13:14], d[14:16], d[16:20]
	if cnjCheck(n7, aaaa, j, tr, oooo) != int(dv[0]-'0')*10+int(dv[1]-'0') {
		return "", errors.New("número do processo inválido (dígito verificador não confere)")
	}
	return fmt.Sprintf("%s-%s.%s.%s.%s.%s", n7, dv, aaaa, j, tr, oooo), nil
}

// MakeCNJ monta um número válido a partir de seus componentes (usado em dados de demonstração e testes).
func MakeCNJ(n7, aaaa, j, tr, oooo string) string {
	return fmt.Sprintf("%s-%02d.%s.%s.%s.%s", n7, cnjCheck(n7, aaaa, j, tr, oooo), aaaa, j, tr, oooo)
}
