package prazos

import (
"time"
)

var feriadosFixos = map[string]bool{
"01-01": true,
"04-21": true,
"05-01": true,
"09-07": true,
"10-12": true,
"11-02": true,
"11-15": true,
"11-20": true,
"12-25": true,
}

func IsFinalDeSemana(t time.Time) bool {
w := t.Weekday()
return w == time.Saturday || w == time.Sunday
}

func IsRecessoForense(t time.Time) bool {
month := t.Month()
day := t.Day()

if month == time.December && day >= 20 {
return true
}
if month == time.January && day <= 20 {
return true
}
return false
}

func IsDiaUtil(t time.Time, feriadosCustomizados []time.Time) bool {
if IsFinalDeSemana(t) || IsRecessoForense(t) {
return false
}

chave := t.Format("01-02")
if feriadosFixos[chave] {
return false
}

dataStr := t.Format("2006-01-02")
for _, f := range feriadosCustomizados {
if f.Format("2006-01-02") == dataStr {
return false
}
}

return true
}
