package calculo

import "time"

// Nacional é o calendário de feriados nacionais por lei, SEMPRE aplicado. Só traz o que tem norma citada:
// Quarta-feira de Cinzas, Corpus Christi, Carnaval e Quarta/Quinta-feira Santa dependem do tribunal e só
// entram pelo calendário dele (cadastrado com fonte). A Sexta-Feira Santa também NÃO é nacional por padrão (Lei 9.093/1995,
// art. 2º: feriado religioso por lei municipal): o cálculo a trata como alternativa (ver ehSextaSanta). As normas citadas aqui estão em REGRAS-DE-PRAZO.md.
type Nacional struct{}

type fixo struct {
	mes   time.Month
	dia   int
	nome  string
	fonte string
}

var feriadosFixos = []fixo{
	{time.January, 1, "Confraternização Universal", "Lei 662/1949 (com alterações)"},
	{time.April, 21, "Tiradentes", "Lei 662/1949 (com alterações)"},
	{time.May, 1, "Dia do Trabalho", "Lei 662/1949 (com alterações)"},
	{time.September, 7, "Independência do Brasil", "Lei 662/1949 (com alterações)"},
	{time.October, 12, "Nossa Senhora Aparecida", "Lei 6.802/1980, art. 1º"},
	{time.November, 2, "Finados", "Lei 662/1949 (com alterações)"},
	{time.November, 15, "Proclamação da República", "Lei 662/1949 (com alterações)"},
	{time.December, 25, "Natal", "Lei 662/1949 (com alterações)"},
}

// PrimeiroAno20Nov é o primeiro ano em que 20/11 é feriado nacional (Lei 14.759/2023, publicada em 22/12/2023).
const PrimeiroAno20Nov = 2024

// NaoUtil diz se o dia é feriado nacional, com a fonte.
func (Nacional) NaoUtil(d time.Time) (DiaNaoUtil, bool) {
	for _, f := range feriadosFixos {
		if d.Month() == f.mes && d.Day() == f.dia {
			return DiaNaoUtil{Data: d, Tipo: "feriado", Descricao: f.nome, Fonte: f.fonte}, true
		}
	}
	if d.Month() == time.November && d.Day() == 20 && d.Year() >= PrimeiroAno20Nov {
		return DiaNaoUtil{Data: d, Tipo: "feriado", Descricao: "Dia Nacional de Zumbi e da Consciência Negra",
			Fonte: "Lei 14.759/2023 (a partir de 2024)"}, true
	}
	return DiaNaoUtil{}, false
}

// FonteSextaSanta é a norma que trata a Sexta-Feira Santa (sem inciso: o art. 2º é um só dispositivo).
const FonteSextaSanta = "Lei 9.093/1995, art. 2º (feriado religioso por lei municipal; adoção pelo tribunal a verificar)"

// ehSextaSanta diz se d é a Sexta-Feira da Paixão (Páscoa menos 2 dias).
func ehSextaSanta(d time.Time) bool { return d.Equal(Pascoa(d.Year()).AddDate(0, 0, -2)) }

// FixosDoAno devolve os feriados nacionais de data fixa do ano (para a semente do cadastro).
func FixosDoAno(ano int) []DiaNaoUtil {
	var out []DiaNaoUtil
	for m := time.January; m <= time.December; m++ {
		for dia := 1; dia <= 31; dia++ {
			d := time.Date(ano, m, dia, 0, 0, 0, 0, time.UTC)
			if d.Month() != m {
				continue
			}
			if x, ok := (Nacional{}).NaoUtil(d); ok {
				out = append(out, x)
			}
		}
	}
	return out
}
