package calculo

import (
	"fmt"
	"time"
)

// variante é uma leitura possível da prorrogação do vencimento em prazo corrido. O texto legal só prorroga o que
// diz (CPP 798 §3º: domingo e feriado; CC 132 §1º: feriado). Como não se decide a questão jurídica, o sistema calcula
// cada leitura, exibe a MAIS CURTA (sempre a primeira) e lista as outras como alternativa.
type variante struct {
	domingo, sabado bool // prorroga também o domingo / o sábado
	chave           string
}

// variantesDe devolve as leituras da mais curta para a mais longa. O prazo material penal não prorroga.
func variantesDe(regra Regra) []variante {
	switch regra {
	case CPPCorridos:
		return []variante{{true, false, "base"}, {true, true, "sabado"}}
	case MaterialCivel:
		return []variante{{false, false, "base"}, {true, false, "domingo"}, {true, true, "sabado"}}
	}
	return []variante{{false, false, "base"}}
}

type vencVar struct {
	data  time.Time
	chave string
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
	var vars [][]vencVar
	for _, k := range cands {
		v, vs := c.corridoDe(e.Regra, k.ancora, e.Dias, len(cands) == 1)
		venc, vars = append(venc, v), append(vars, vs)
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
	alternativasDeProrrogacao(vars[best], r)
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

// alternativasDeProrrogacao lista as leituras mais longas que mudam a data (sem repetir data já listada).
func alternativasDeProrrogacao(vs []vencVar, r *Resultado) {
	for i := 1; i < len(vs); i++ {
		if vs[i].data.Equal(vs[i-1].data) {
			continue
		}
		switch vs[i].chave {
		case "domingo":
			r.Alternativas = append(r.Alternativas, fmt.Sprintf("Se o domingo for tido como feriado (prorrogação do CC art. 132, §1º), o vencimento seria %s.", fmtD(vs[i].data)))
			r.Duvidas = append(r.Duvidas, "[VERIFICAR] prorrogação por domingo no prazo material cível: o CC art. 132, §1º só cita \"feriado\"; o sistema NÃO prorrogou o domingo (vencimento mais curto).")
		case "sabado":
			r.Alternativas = append(r.Alternativas, fmt.Sprintf("Se o tribunal não tiver expediente no sábado, o vencimento seria %s (dia útil seguinte).", fmtD(vs[i].data)))
			r.Duvidas = append(r.Duvidas, "[VERIFICAR] vencimento em sábado: o texto legal não prorroga o sábado; o sistema NÃO o prorrogou, nem depois de prorrogação por feriado (vencimento mais curto).")
		}
	}
}

// corridoDe devolve o vencimento (leitura mais curta) contando dias corridos a partir da âncora, e todas as leituras.
// registra=true grava os feriados pulados na leitura exibida.
func (c *contador) corridoDe(regra Regra, ancora time.Time, dias int, registra bool) (time.Time, []vencVar) {
	if regra == MaterialPenal { // CP art. 10: inclui o dia do começo; sem prorrogação
		d := ancora.AddDate(0, 0, dias-1)
		return d, []vencVar{{d, "base"}}
	}
	var out []vencVar
	for i, v := range variantesDe(regra) {
		d := ancora.AddDate(0, 0, dias)
		for {
			x, f := c.feriado(d, true)
			if !f && !(v.domingo && d.Weekday() == time.Sunday) && !(v.sabado && d.Weekday() == time.Saturday) {
				break
			}
			if f && registra && i == 0 {
				c.registra(x)
			}
			d = d.AddDate(0, 0, 1)
		}
		out = append(out, vencVar{d, v.chave})
	}
	return out[0].data, out
}

func regraTexto(regra Regra, dias int, rotulo string) string {
	switch regra {
	case CPPCorridos:
		return fmt.Sprintf("Regra: CPP art. 798, %d dia(s) corrido(s) a contar %s; não se computa o dia do começo; computa-se o do vencimento; prorroga se vencer em domingo ou feriado (o sábado não é prorrogado).", dias, rotulo)
	case MaterialCivel:
		return fmt.Sprintf("Regra: prazo material cível, Código Civil art. 132, %d dia(s) corrido(s); exclui o dia do começo, inclui o do vencimento; prorroga se vencer em feriado (o texto do §1º não cita domingo nem sábado: não prorrogados).", dias)
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
