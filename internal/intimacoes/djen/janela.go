package djen

import "time"

func saoPaulo() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return time.FixedZone("BRT", -3*3600)
	}
	return loc
}

// dataCivil devolve a data (sem hora) que t tem em São Paulo, como meia-noite UTC daquele dia: assim a comparação e a
// formatação não dependem do fuso da máquina. Às 21h de 02/11 em São Paulo (00h UTC de 03/11) a data é 02/11.
func dataCivil(t time.Time) time.Time {
	x := t.In(saoPaulo())
	return time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, time.UTC)
}

// JanelaDesde calcula o intervalo de datas de disponibilização a coletar: de "diasPadrao dias atrás" (ou, se a coleta
// ficou parada, de 7 dias antes do último sucesso, o que for mais antigo) até hoje, sempre pela data civil de São Paulo.
// O recuo de 7 dias sobre o último sucesso é a sobreposição que pega comunicação publicada com atraso; a deduplicação
// por hash impede que isso duplique algo.
func JanelaDesde(agora time.Time, ultimoSucesso *time.Time, diasPadrao int) (desde, ate time.Time) {
	if diasPadrao <= 0 {
		diasPadrao = 7
	}
	ate = dataCivil(agora)
	desde = ate.AddDate(0, 0, -diasPadrao)
	if ultimoSucesso != nil && !ultimoSucesso.IsZero() {
		if d := dataCivil(*ultimoSucesso).AddDate(0, 0, -7); d.Before(desde) {
			desde = d
		}
	}
	return desde, ate
}
