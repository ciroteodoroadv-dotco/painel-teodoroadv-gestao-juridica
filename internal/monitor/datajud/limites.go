package datajud

import (
	"strings"
	"unicode/utf8"
)

// Limites de texto, sempre em caracteres (nunca em bytes: o corte não pode partir um caractere de vários bytes).
const (
	MaxComplemento = 500  // complemento resumido (entra na chave e na coluna processo_movimentos.complemento)
	MaxCampo       = 300  // grau, código, nome do movimento, órgão julgador
	MaxComplTab    = 1000 // cada campo de um complemento tabelado
	maxMovimentos  = 5000 // por grau: proteção contra resposta absurda
)

// Cortar limita s a max caracteres, tirando bytes que não são UTF-8 válido e espaços das pontas.
func Cortar(s string, max int) string {
	s = strings.TrimSpace(strings.ToValidUTF8(s, ""))
	if utf8.RuneCountInString(s) > max {
		return string([]rune(s)[:max])
	}
	return s
}
