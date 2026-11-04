// Package cnj trata o número único do processo (Res. CNJ 65/2008) e descobre o tribunal do DataJud a partir dele.
package cnj

import (
	"errors"
	"fmt"

	"painelteodoro/internal/platform/br"
)

// Digitos tira tudo o que não é dígito.
func Digitos(s string) string { return br.Digits(s) }

// Valido diz se o número tem 20 dígitos e o dígito verificador confere (módulo 97, ISO 7064; mesma regra do Plano 2).
func Valido(numero string) bool {
	_, err := br.NormalizeCNJ(numero)
	return err == nil
}

// Formatar devolve NNNNNNN-DD.AAAA.J.TR.OOOO; entrada que não tem 20 dígitos volta como veio.
func Formatar(numero string) string {
	d := Digitos(numero)
	if len(d) != 20 {
		return numero
	}
	return fmt.Sprintf("%s-%s.%s.%s.%s.%s", d[0:7], d[7:9], d[9:13], d[13:14], d[14:16], d[16:20])
}

// AliasDataJud devolve o alias do endpoint (api_publica_<alias>) do tribunal do número, ou um erro que explica por que
// o processo não pode ser consultado.
func AliasDataJud(numero string) (string, error) {
	if !Valido(numero) {
		return "", errors.New("sem número CNJ válido")
	}
	return aliasPorJTR(Digitos(numero))
}
