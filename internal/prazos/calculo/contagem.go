package calculo

import (
	"fmt"
	"time"
)

func dia(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }

func fmtD(t time.Time) string { return t.Format("02/01/2006") }

func fimDeSemana(d time.Time) bool { return d.Weekday() == time.Saturday || d.Weekday() == time.Sunday }

// recesso: suspensão de 20/12 a 20/01, inclusive (CPC art. 220).
func recesso(d time.Time) bool {
	return (d.Month() == time.December && d.Day() >= 20) || (d.Month() == time.January && d.Day() <= 20)
}

type contador struct {
	cal       Calendario
	excluidos []DiaNaoUtil
	// sextaSanta=true trata a Sexta-Feira Santa como dia sem expediente (só na leitura alternativa; ver Calcular).
	sextaSanta bool
}

// feriado devolve o dia não útil (nacional, depois o do tribunal). Nos prazos que não são do CPC só vale
// o tipo "feriado" do tribunal: suspensão por portaria não é aplicada a eles (marcado como dúvida).
func (c *contador) feriado(d time.Time, soFeriado bool) (DiaNaoUtil, bool) {
	if x, ok := (Nacional{}).NaoUtil(d); ok {
		return x, true
	}
	if c.cal != nil {
		if x, ok := c.cal.NaoUtil(d); ok && (!soFeriado || x.Tipo == "feriado") {
			return x, true
		}
	}
	if c.sextaSanta && ehSextaSanta(d) {
		return DiaNaoUtil{Data: d, Tipo: "feriado", Descricao: "Sexta-Feira Santa (Paixão de Cristo)", Fonte: FonteSextaSanta}, true
	}
	return DiaNaoUtil{}, false
}

func (c *contador) util(d time.Time) bool {
	_, f := c.feriado(d, false)
	return !fimDeSemana(d) && !f
}

func (c *contador) registra(x DiaNaoUtil) {
	for _, e := range c.excluidos {
		if e.Data.Equal(x.Data) {
			return
		}
	}
	c.excluidos = append(c.excluidos, x)
}

// proximoUtil é o primeiro dia útil estritamente depois de d (ignora a suspensão de fim de ano).
func (c *contador) proximoUtil(d time.Time) time.Time {
	for {
		d = d.AddDate(0, 0, 1)
		if c.util(d) {
			return d
		}
	}
}

// Calcular aplica a regra. Nunca escolhe a data mais longa na dúvida: mostra a mais curta e lista a outra.
func Calcular(e Entrada) (Resultado, error) {
	if e.Dias <= 0 || e.Dias > MaxDias {
		return Resultado{}, ErrDias
	}
	if e.Evento.IsZero() || e.Evento.Year() < 1990 || e.Evento.Year() > 2100 {
		return Resultado{}, ErrEvento
	}
	if e.Modo != DJe && e.Modo != Ciencia {
		return Resultado{}, ErrModo
	}
	e.Evento = dia(e.Evento)
	r, err := calcula(e, false)
	if err != nil {
		return r, err
	}
	// Sexta-Feira Santa: a Lei 9.093/1995 art. 2º a trata como feriado religioso de lei municipal; no calendário
	// nacional ela não é excluída (vencimento mais curto). A leitura em que o tribunal a adota vai como alternativa.
	if alt, _ := calcula(e, true); !alt.Vencimento.Equal(*r.Vencimento) {
		r.Alternativas = append(r.Alternativas, fmt.Sprintf("Se o tribunal adotar a Sexta-Feira Santa como dia sem expediente, o vencimento seria %s.", fmtD(*alt.Vencimento)))
		r.Duvidas = append(r.Duvidas, "[VERIFICAR] Sexta-Feira Santa: a Lei 9.093/1995, art. 2º, trata os feriados religiosos por lei municipal; o sistema NÃO a excluiu (vencimento mais curto). Confira no calendário do tribunal.")
	}
	return r, nil
}

func calcula(e Entrada, sextaSanta bool) (Resultado, error) {
	c := &contador{cal: e.Cal, sextaSanta: sextaSanta}
	var r Resultado
	switch e.Regra {
	case CPCUteis:
		c.cpc(e, &r)
	case CPPCorridos, MaterialCivel, MaterialPenal:
		c.corridos(e, &r)
	default:
		return Resultado{}, ErrRegra
	}
	c.fecha(e, &r)
	return r, nil
}

func (c *contador) cpc(e Entrada, r *Resultado) {
	ancora := e.Evento
	if e.Modo == DJe {
		pub := c.proximoUtil(e.Evento)
		r.Publicacao, ancora = &pub, pub
		r.Premissas = append(r.Premissas, fmt.Sprintf("Disponibilização em %s; publicação em %s, o primeiro dia útil seguinte (CPC art. 224, §2º).", fmtD(e.Evento), fmtD(pub)))
	} else {
		r.Premissas = append(r.Premissas, fmt.Sprintf("Ciência em %s.", fmtD(e.Evento)))
	}
	n := 0
	for d := ancora.AddDate(0, 0, 1); ; d = d.AddDate(0, 0, 1) {
		if recesso(d) {
			continue
		}
		if fimDeSemana(d) {
			continue
		}
		if x, f := c.feriado(d, false); f {
			c.registra(x)
			continue
		}
		n++
		if n == 1 {
			ini := d
			r.Inicio = &ini
		}
		if n == e.Dias {
			v := d
			r.Vencimento = &v
			break
		}
	}
	r.Premissas = append(r.Premissas,
		fmt.Sprintf("Início da contagem em %s, o primeiro dia útil depois do termo inicial; o dia do começo não conta e o do vencimento conta (CPC arts. 219, 224 e 224, §3º).", fmtD(*r.Inicio)),
		fmt.Sprintf("Regra: CPC, %d dia(s) útil(eis), sem sábados, domingos e feriados (CPC art. 216).", e.Dias),
		"Suspensão de 20/12 a 20/01 aplicada: nesses dias o prazo não corre (CPC art. 220).")
}

// fecha acrescenta calendário, ordenação, lista fixa e a contagem de dias úteis até hoje.
func (c *contador) fecha(e Entrada, r *Resultado) {
	r.Excluidos = c.excluidos
	for i := 1; i < len(r.Excluidos); i++ { // ordem cronológica
		for j := i; j > 0 && r.Excluidos[j].Data.Before(r.Excluidos[j-1].Data); j-- {
			r.Excluidos[j], r.Excluidos[j-1] = r.Excluidos[j-1], r.Excluidos[j]
		}
	}
	if !c.cobre(e.Evento.Year(), r.Vencimento.Year()) {
		r.Premissas = append(r.Premissas, "Sem calendário verificado deste tribunal: considerados só os feriados nacionais.")
		r.Duvidas = append(r.Duvidas, "Feriados, suspensões e pontos facultativos do tribunal não estão cadastrados: confira no calendário do tribunal.")
	} else {
		r.Premissas = append(r.Premissas, "Calendário do tribunal: considerados os dias cadastrados com fonte, além dos feriados nacionais.")
	}
	r.NaoConsiderado = append([]string(nil), NaoConsideradoFixo...)
	if e.Hoje.IsZero() {
		return
	}
	hoje, v := dia(e.Hoje), *r.Vencimento
	switch {
	case v.After(hoje):
		for d := hoje.AddDate(0, 0, 1); !d.After(v); d = d.AddDate(0, 0, 1) {
			if c.util(d) {
				r.DiasUteisAteVencimento++
			}
		}
	case v.Before(hoje):
		for d := v.AddDate(0, 0, 1); !d.After(hoje); d = d.AddDate(0, 0, 1) {
			if c.util(d) {
				r.DiasUteisAteVencimento--
			}
		}
	}
}

func (c *contador) cobre(anos ...int) bool {
	if c.cal == nil {
		return false
	}
	cb, ok := c.cal.(Cobertor)
	if !ok {
		return true
	}
	for _, a := range anos {
		if !cb.Cobre(a) {
			return false
		}
	}
	return true
}
