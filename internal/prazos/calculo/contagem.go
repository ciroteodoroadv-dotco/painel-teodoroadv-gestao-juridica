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
	c := &contador{cal: e.Cal}
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

// corridos trata CPP, prazo material cível e prazo material penal.
func (c *contador) corridos(e Entrada, r *Resultado) {
	type cand struct {
		ancora time.Time
		rotulo string
	}
	cands := []cand{{e.Evento, "da data do evento"}}
	switch e.Regra {
	case CPPCorridos:
		if e.Modo == DJe {
			pub := c.proximoUtil(e.Evento)
			r.Publicacao = &pub
			cands = []cand{{pub, "da publicação (dia útil seguinte à disponibilização)"}, {e.Evento, "da própria disponibilização"}}
			r.Duvidas = append(r.Duvidas, "[VERIFICAR] termo inicial pelo Diário no processo penal: adotada a publicação no dia útil seguinte à disponibilização (Lei 11.419/2006, art. 4º, §§3º e 4º; texto não consultado nesta rodada); mostrado o vencimento mais curto.")
		}
	case MaterialCivel, MaterialPenal:
		r.Premissas = append(r.Premissas, "Prazo material: conta da data do evento informada; o modo de contagem (Diário ou ciência) não altera o termo.")
	}
	var venc []time.Time
	for _, k := range cands {
		venc = append(venc, c.corridoDe(e.Regra, k.ancora, e.Dias, r, len(cands) == 1))
	}
	best := 0
	for i := range venc {
		if venc[i].Before(venc[best]) {
			best = i
		}
	}
	for i, k := range cands {
		if i != best && !venc[i].Equal(venc[best]) {
			r.Alternativas = append(r.Alternativas, fmt.Sprintf("Se o prazo correr %s, o vencimento seria %s.", k.rotulo, fmtD(venc[i])))
		}
	}
	v := venc[best]
	r.Vencimento = &v
	ini := cands[best].ancora
	if e.Regra != MaterialPenal {
		ini = ini.AddDate(0, 0, 1)
	}
	r.Inicio = &ini
	r.Premissas = append(r.Premissas, regraTexto(e.Regra, e.Dias, cands[best].rotulo))
	r.Duvidas = append(r.Duvidas, duvidasCorridos(e.Regra)...)
}

// corridoDe devolve o vencimento contando dias corridos a partir da âncora. registra=true grava os feriados pulados.
func (c *contador) corridoDe(regra Regra, ancora time.Time, dias int, r *Resultado, registra bool) time.Time {
	if regra == MaterialPenal { // CP art. 10: inclui o dia do começo; sem prorrogação
		return ancora.AddDate(0, 0, dias-1)
	}
	d := ancora.AddDate(0, 0, dias)
	moveu := false // depois da 1ª prorrogação, o dia final tem de ser útil (sábado também é pulado)
	for {
		x, f := c.feriado(d, true)
		if (!moveu && d.Weekday() != time.Sunday && !f) || (moveu && !fimDeSemana(d) && !f) {
			break
		}
		moveu = true
		if f && registra {
			c.registra(x)
		}
		d = d.AddDate(0, 0, 1)
	}
	if d.Weekday() == time.Saturday {
		alt := d
		for !c.util(alt) {
			alt = alt.AddDate(0, 0, 1)
		}
		r.Alternativas = append(r.Alternativas, fmt.Sprintf("Se o tribunal não tiver expediente no sábado, o vencimento seria %s (dia útil seguinte).", fmtD(alt)))
		r.Duvidas = append(r.Duvidas, "[VERIFICAR] vencimento em sábado: o texto legal prorroga domingo e feriado; o sistema NÃO prorrogou o sábado.")
	}
	return d
}

func regraTexto(regra Regra, dias int, rotulo string) string {
	switch regra {
	case CPPCorridos:
		return fmt.Sprintf("Regra: CPP art. 798, %d dia(s) corrido(s) a contar %s; não se computa o dia do começo; computa-se o do vencimento; prorroga se vencer em domingo ou feriado.", dias, rotulo)
	case MaterialCivel:
		return fmt.Sprintf("Regra: prazo material cível, Código Civil art. 132, %d dia(s) corrido(s); exclui o dia do começo, inclui o do vencimento; prorroga se vencer em domingo ou feriado.", dias)
	default:
		return fmt.Sprintf("Regra: prazo material penal, Código Penal art. 10, %d dia(s); inclui o dia do começo; sem prorrogação.", dias)
	}
}

func duvidasCorridos(regra Regra) []string {
	if regra == MaterialPenal {
		return []string{"[VERIFICAR] suspensão de 20/12 a 20/01 e suspensões por portaria NÃO aplicadas (prazo material penal)."}
	}
	return []string{"[VERIFICAR] suspensão de 20/12 a 20/01 NÃO aplicada (a regra do CPC art. 220 não foi estendida a este prazo); suspensões por portaria também não."}
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
