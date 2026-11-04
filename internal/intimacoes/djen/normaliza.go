// Package djen é o cliente isolado do DJEN (Diário de Justiça Eletrônico Nacional, API pública "Comunica").
// Só lê. Não segura estado, não grava em banco e não conhece o resto do painel: recebe a OAB e a janela de datas e devolve
// as comunicações já normalizadas. Nenhum teste fala com o servidor real; só com httptest.
package djen

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"
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

func texto(m itemJSON, chaves ...string) string {
	for _, k := range chaves {
		if v, ok := m[k]; ok && v != nil {
			var s string
			switch x := v.(type) {
			case string:
				s = x
			case float64:
				s = strconv.FormatFloat(x, 'f', -1, 64)
			default:
				continue
			}
			if s = strings.TrimSpace(s); s != "" {
				return s
			}
		}
	}
	return ""
}

func normalizar(m itemJSON) (Comunicacao, error) {
	c := Comunicacao{ID: texto(m, "id", "identificador"), Tribunal: texto(m, "siglaTribunal", "tribunal"),
		Orgao: texto(m, "nomeOrgao", "orgao"), Processo: texto(m, "numeroprocessocommascara", "numero_processo", "processo"),
		Tipo: texto(m, "tipoComunicacao", "tipo"), Texto: NormalizarTexto(texto(m, "texto", "resumo"))}
	if l := texto(m, "link", "url"); strings.HasPrefix(l, "https://") {
		c.Link = l
	}
	d := texto(m, "data_disponibilizacao", "dataPublicacao", "data_publicacao")
	for _, f := range []string{"2006-01-02", "02/01/2006", time.RFC3339} {
		if t, err := time.Parse(f, d); err == nil {
			c.DataPublicacao = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			return c, nil
		}
	}
	return c, errors.New("djen: comunicação com data inválida; não descartada")
}

// MaxTexto é o teto de segurança do texto integral da comunicação, em caracteres; MarcaCorte é acrescentada se ele for atingido.
const (
	MaxTexto   = 100000
	MarcaCorte = "[texto cortado]"
)

// NormalizarTexto limpa o texto integral (tira marcação HTML, junta espaços) SEM o corte de 2000 caracteres do resumo.
// Só corta no teto de segurança, por caracteres, e então marca "[texto cortado]".
func NormalizarTexto(s string) string {
	s = strings.ToValidUTF8(s, "")
	s = html.UnescapeString(tagHTML.ReplaceAllString(s, " "))
	s = strings.Join(strings.Fields(s), " ")
	s = espacoAn.ReplaceAllString(s, "$1")
	if utf8.RuneCountInString(s) > MaxTexto {
		s = string([]rune(s)[:MaxTexto]) + MarcaCorte
	}
	return s
}
