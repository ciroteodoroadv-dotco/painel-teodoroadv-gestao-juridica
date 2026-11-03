package djen

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestJanelaDesdeUsaDataCivilDeSaoPaulo(t *testing.T) {
	// 00h UTC de 03/11 = 21h de 02/11 em São Paulo: a data civil ainda é 02/11.
	agora := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	desde, ate := JanelaDesde(agora, nil, 7)
	if ate.Format("2006-01-02") != "2026-11-02" || desde.Format("2006-01-02") != "2026-10-26" {
		t.Errorf("sem último sucesso: desde=%s ate=%s", desde.Format("2006-01-02"), ate.Format("2006-01-02"))
	}
	// Último sucesso às 09h de SP em 02/11: recua 7 dias sobre ele (26/10), igual ao padrão.
	u := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	if desde, _ := JanelaDesde(agora, &u, 7); desde.Format("2006-01-02") != "2026-10-26" {
		t.Errorf("recuo de 7 dias sobre o último sucesso: %s", desde.Format("2006-01-02"))
	}
	// Coleta parada há dias: a janela cobre desde o último sucesso (menos 7 dias), não só os últimos 7.
	u2 := time.Date(2026, 10, 20, 20, 0, 0, 0, time.UTC)
	if desde, _ := JanelaDesde(agora, &u2, 7); desde.Format("2006-01-02") != "2026-10-13" {
		t.Errorf("janela longa: %s", desde.Format("2006-01-02"))
	}
	// 01h UTC de 03/11 de um sucesso em 01/11 22h SP não vira 02/11 por UTC.
	u3 := time.Date(2026, 11, 2, 1, 0, 0, 0, time.UTC) // 22h de 01/11 em SP
	if desde, _ := JanelaDesde(time.Date(2026, 11, 20, 15, 0, 0, 0, time.UTC), &u3, 7); desde.Format("2006-01-02") != "2026-10-25" {
		t.Errorf("data civil de SP no último sucesso: %s", desde.Format("2006-01-02"))
	}
}

func TestNormalizarResumoCortaPorRunes(t *testing.T) {
	s := NormalizarResumo(strings.Repeat("€", 2500)) // "€" tem 3 bytes
	if utf8.RuneCountInString(s) != 2000 || !utf8.ValidString(s) {
		t.Errorf("runes=%d válido=%v", utf8.RuneCountInString(s), utf8.ValidString(s))
	}
	if got := NormalizarResumo("  <b>Olá</b>   mundo &amp; cia.  "); got != "Olá mundo & cia." {
		t.Errorf("limpeza = %q", got)
	}
	if got := NormalizarResumo("ok\xff\xfeok"); !utf8.ValidString(got) {
		t.Errorf("bytes inválidos devem ser removidos/substituídos: %q", got)
	}
	if NormalizarResumo("") != "" {
		t.Error("vazio")
	}
}

func TestHashEstavelESemColisao(t *testing.T) {
	a := Comunicacao{ID: "42", Tribunal: "TJSP"}
	if Hash(a) != Hash(Comunicacao{ID: "42", Tribunal: "OUTRO"}) {
		t.Error("com id do DJEN, o hash depende só do id")
	}
	if Hash(a) == Hash(Comunicacao{ID: "43"}) {
		t.Error("ids diferentes, hashes diferentes")
	}
	if len(Hash(a)) != 64 {
		t.Errorf("hash = %q", Hash(a))
	}
	// Sem id: o separador evita que ("ab","c") e ("a","bc") colidam.
	x := Comunicacao{Tribunal: "ab", Orgao: "c", Processo: "p", Tipo: "t", Texto: "r"}
	y := Comunicacao{Tribunal: "a", Orgao: "bc", Processo: "p", Tipo: "t", Texto: "r"}
	if Hash(x) == Hash(y) {
		t.Error("campos deslocados não podem colidir")
	}
	if Hash(x) != Hash(x) {
		t.Error("estável")
	}
	d := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	x2 := x
	x2.DataPublicacao = d
	if Hash(x) == Hash(x2) {
		t.Error("a data entra no hash sem id")
	}
}
