// Package bcb é o cliente isolado do SGS (Sistema Gerenciador de Séries Temporais) do Banco Central. Só lê o último
// ponto de cada série. Não grava em banco nem conhece o painel. Nenhum teste fala com o servidor real.
package bcb

import "time"

// Serie é uma série do SGS exibida no rodapé. A ordem de Series é a de ORDEM_INDICES do painel B.
type Serie struct {
	Codigo  int
	Rotulo  string
	Unidade string // "% a.a.", "% mensal" ou "R$"
}

// Series: Selic meta (432), IPCA (433), INPC (188), IGP-M (189), PTAX venda (1), salário mínimo (1619).
var Series = []Serie{
	{432, "Selic (meta Copom)", "% a.a."},
	{433, "IPCA", "% mensal"},
	{188, "INPC", "% mensal"},
	{189, "IGP-M", "% mensal"},
	{1, "Dólar (PTAX)", "R$"},
	{1619, "Salário mínimo", "R$"},
}

// Ponto é o último valor de uma série. Valor é decimal em texto ("0.52"): nunca passa por float.
type Ponto struct {
	Data  time.Time
	Valor string
}

func conhecida(codigo int) bool {
	for _, s := range Series {
		if s.Codigo == codigo {
			return true
		}
	}
	return false
}
