package calculo

import "time"

// Pascoa devolve o domingo de Páscoa (algoritmo gregoriano anônimo, "Meeus/Jones/Butcher").
func Pascoa(ano int) time.Time {
	a := ano % 19
	b, c := ano/100, ano%100
	d, e := b/4, b%4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i, k := c/4, c%4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	mes := (h + l - 7*m + 114) / 31
	dia := (h+l-7*m+114)%31 + 1
	return time.Date(ano, time.Month(mes), dia, 0, 0, 0, 0, time.UTC)
}
