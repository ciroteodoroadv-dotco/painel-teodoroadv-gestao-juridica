// Package indices guarda e mostra no rodapé o último valor de cada série do Banco Central (SGS). Nunca presume valor:
// só aparece número que veio da fonte, com a data de referência e o estado da coleta. Não substitui consulta à fonte.
package indices

import (
	"strings"
	"time"
)

// Modulo é o nome da coleta em "coletas" (rodapé "Estado das coletas").
const Modulo = "indices"

// Origem da coleta.
const Origem = "BCB SGS"

// MsgDesligada é o texto do estado quando a integração está desligada.
const MsgDesligada = "Coleta desligada neste ambiente."

// Valor é o último valor conhecido de uma série. Valor é decimal em texto (nunca float).
type Valor struct {
	Codigo          int
	Rotulo, Unidade string
	Valor           string
	Referencia      time.Time
	ObtidoEm        time.Time
	Falhou          bool
}

// Formatar escreve o valor em pt-BR com a unidade: "0,52 %", "R$ 5,43", "R$ 1.518,00". Mínimo de 2 casas; as casas a
// mais que a fonte entrega são mantidas (ex.: PTAX com 4); zeros sobrando além de 2 são tirados.
func Formatar(valor, unidade string) string {
	neg := strings.HasPrefix(valor, "-")
	valor = strings.TrimPrefix(valor, "-")
	inteiro, frac, _ := strings.Cut(valor, ".")
	frac = strings.TrimRight(frac, "0")
	for len(frac) < 2 {
		frac += "0"
	}
	var b strings.Builder
	for i, c := range inteiro {
		if i > 0 && (len(inteiro)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	n := b.String() + "," + frac
	if neg {
		n = "-" + n
	}
	if unidade == "R$" {
		return "R$ " + n
	}
	return n + " %"
}
