// Package djen é o cliente isolado do DJEN (Diário de Justiça Eletrônico Nacional, API pública "Comunica").
// Só lê. Não segura estado, não grava em banco e não conhece o resto do painel: recebe a OAB e a janela de datas e devolve
// as comunicações já normalizadas. Nenhum teste fala com o servidor real; só com httptest.
package djen

import (
	"crypto/sha256"
	"encoding/hex"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

// MaxResumo é o limite do resumo, em caracteres (não em bytes).
const MaxResumo = 2000

var (
	tagHTML  = regexp.MustCompile(`<[^>]*>`)
	espacoAn = regexp.MustCompile(`\s+([,.;:!?])`)
)

// NormalizarResumo limpa o texto da comunicação (tira marcação HTML, junta espaços) e o corta em MaxResumo caracteres.
// O corte é por runes: nunca no meio de um caractere de vários bytes. Bytes que não são UTF-8 válido são removidos.
func NormalizarResumo(s string) string {
	s = strings.ToValidUTF8(s, "")
	s = html.UnescapeString(tagHTML.ReplaceAllString(s, " "))
	s = strings.Join(strings.Fields(s), " ")
	s = espacoAn.ReplaceAllString(s, "$1")
	if utf8.RuneCountInString(s) > MaxResumo {
		s = string([]rune(s)[:MaxResumo])
	}
	return s
}

// Hash identifica a comunicação para a deduplicação. Com id do DJEN, é o SHA-256 do id; sem id, o SHA-256 dos campos
// separados por \x1f (separador que não ocorre em texto), para ("ab","c") e ("a","bc") não darem o mesmo hash.
func Hash(c Comunicacao) string {
	chave := "id\x1f" + c.ID
	if c.ID == "" {
		data := ""
		if !c.DataPublicacao.IsZero() {
			data = c.DataPublicacao.Format("2006-01-02")
		}
		chave = strings.Join([]string{"campos", c.Tribunal, c.Orgao, c.Processo, data, c.Tipo, c.Link, c.Texto}, "\x1f")
	}
	h := sha256.Sum256([]byte(chave))
	return hex.EncodeToString(h[:])
}
