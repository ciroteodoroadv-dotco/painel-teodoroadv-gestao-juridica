package coletas

import (
	"testing"
	"time"
)

// A linha do rodapé tem o texto de 16-sync-status.js do painel B, palavra por palavra.
func TestLinhaComoNoB(t *testing.T) {
	tent := time.Date(2026, 11, 3, 15, 4, 5, 0, time.UTC) // 12:04:05 em Brasília
	suc := time.Date(2026, 11, 2, 23, 0, 0, 0, time.UTC)  // 20:00:00 em Brasília
	casos := []struct {
		e    Estado
		want string
		erro bool
	}{
		{Estado{Modulo: "intimacoes", Status: "sucesso", UltimaTentativa: &tent, UltimoSucesso: &suc, Mensagem: "OABs: 1."},
			"DJEN: última coleta concluída. Tentativa: 03/11/2026 12:04:05 (Brasília). Sucesso: 02/11/2026 20:00:00 (Brasília). OABs: 1.", false},
		{Estado{Modulo: "intimacoes", Status: "parcial", UltimaTentativa: &tent}, "DJEN: coleta parcial. Tentativa: 03/11/2026 12:04:05 (Brasília). Sucesso: não registrado. ", true},
		{Estado{Modulo: "intimacoes", Status: "falha", Mensagem: "djen: HTTP 500"}, "DJEN: erro. Tentativa: não registrado. Sucesso: não registrado. djen: HTTP 500", true},
		{Estado{Modulo: "intimacoes", Status: "bloqueado", Mensagem: "Coleta desligada neste ambiente."}, "DJEN: bloqueado. Tentativa: não registrado. Sucesso: não registrado. Coleta desligada neste ambiente.", true},
		{Estado{Modulo: "intimacoes", Status: "nao_sincronizado"}, "DJEN: sem tentativa registrada. Tentativa: não registrado. Sucesso: não registrado. ", true},
		{Estado{Modulo: "monitor", Status: "sucesso"}, "DataJud: última coleta concluída. Tentativa: não registrado. Sucesso: não registrado. ", false},
		{Estado{Modulo: "outro", Status: "xyz"}, "outro: estado desconhecido. Tentativa: não registrado. Sucesso: não registrado. ", true},
	}
	for _, c := range casos {
		got, erro := c.e.Linha()
		if got != c.want || erro != c.erro {
			t.Errorf("Linha(%+v)\n got  %q erro=%v\n want %q erro=%v", c.e, got, erro, c.want, c.erro)
		}
	}
}
