package webx

import (
	"regexp"
	"strings"
	"time"
)

var horaHM = regexp.MustCompile(`^([01][0-9]|2[0-3]):([0-5][0-9])$`)

// DataISO lê uma data AAAA-MM-DD (campo <input type="date">) e a devolve à meia-noite UTC.
func DataISO(s string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(s))
}

// HoraHM confere uma hora HH:MM (campo <input type="time">) e a devolve como veio, sem espaços.
func HoraHM(s string) (string, bool) {
	s = strings.TrimSpace(s)
	return s, horaHM.MatchString(s)
}

// DataBR formata a data como dd/mm/aaaa (o fmtDate do B); data zerada vira "—".
func DataBR(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("02/01/2006")
}
