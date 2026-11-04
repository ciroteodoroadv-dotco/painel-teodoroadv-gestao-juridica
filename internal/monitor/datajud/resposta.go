package datajud

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Complemento é um complemento tabelado do movimento (nome, valor e descrição, como o DataJud devolve).
type Complemento struct{ Codigo, Nome, Valor, Descricao string }

// Movimento é uma movimentação do processo, com tudo o que a API devolve dela.
type Movimento struct {
	Grau, OrgaoJulgador, Codigo, Nome string
	Complemento                       string // resumo (primeiro complemento), até MaxComplemento caracteres
	Complementos                      []Complemento
	DataHora                          time.Time
	DataBruta                         string // texto original da data, usado na chave quando não foi possível interpretá-la
}

// Processo é o que um grau (um "hit") devolve.
type Processo struct {
	Grau, Tribunal, OrgaoJulgador string
	Sigilo                        bool
	Movimentos                    []Movimento
}

// Resultado da consulta: um Processo por grau. SemRegistro: nenhum hit (NÃO significa "sem movimentação").
type Resultado struct {
	Processos   []Processo
	SemRegistro bool
}

// escalar lê número ou texto de um campo JSON solto.
type escalar string

func (e *escalar) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch x := v.(type) {
	case string:
		*e = escalar(x)
	case float64:
		*e = escalar(strconv.FormatFloat(x, 'f', -1, 64))
	case nil:
	default:
		*e = escalar(fmt.Sprint(x))
	}
	return nil
}

type orgaoJSON struct {
	Nome      escalar `json:"nome"`
	NomeOrgao escalar `json:"nomeOrgao"`
}

func (o orgaoJSON) nome() string {
	if o.Nome != "" {
		return string(o.Nome)
	}
	return string(o.NomeOrgao)
}

type complJSON struct {
	Codigo    escalar `json:"codigo"`
	Nome      escalar `json:"nome"`
	Valor     escalar `json:"valor"`
	Descricao escalar `json:"descricao"`
}

type movJSON struct {
	Codigo   escalar     `json:"codigo"`
	Nome     escalar     `json:"nome"`
	DataHora escalar     `json:"dataHora"`
	Orgao    orgaoJSON   `json:"orgaoJulgador"`
	Compl    []complJSON `json:"complementosTabelados"`
}

type respostaJSON struct {
	Hits *struct {
		Hits []struct {
			Source struct {
				Tribunal    escalar   `json:"tribunal"`
				Grau        escalar   `json:"grau"`
				NivelSigilo int       `json:"nivelSigilo"`
				Orgao       orgaoJSON `json:"orgaoJulgador"`
				Movimentos  []movJSON `json:"movimentos"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

var formatosData = []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z", "2006-01-02T15:04:05", "20060102150405", "2006-01-02"}

func lerData(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, f := range formatosData {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func movimento(m movJSON, grau, orgaoProc string) Movimento {
	mv := Movimento{Grau: grau, Codigo: Cortar(string(m.Codigo), MaxCampo), Nome: Cortar(string(m.Nome), MaxCampo),
		OrgaoJulgador: Cortar(m.Orgao.nome(), MaxCampo), DataBruta: Cortar(string(m.DataHora), MaxCampo)}
	if mv.OrgaoJulgador == "" {
		mv.OrgaoJulgador = orgaoProc
	}
	mv.DataHora = lerData(mv.DataBruta)
	for _, c := range m.Compl {
		mv.Complementos = append(mv.Complementos, Complemento{Cortar(string(c.Codigo), MaxComplTab), Cortar(string(c.Nome), MaxComplTab),
			Cortar(string(c.Valor), MaxComplTab), Cortar(string(c.Descricao), MaxComplTab)})
	}
	if len(mv.Complementos) > 0 {
		c := mv.Complementos[0]
		mv.Complemento = Cortar(c.Nome+" "+c.Valor+" "+c.Descricao, MaxComplemento)
	}
	return mv
}

// interpretar lê TODOS os hits (um por grau/instância).
func interpretar(b []byte) (Resultado, error) {
	var r respostaJSON
	if err := json.Unmarshal(b, &r); err != nil {
		return Resultado{}, errors.New("datajud: JSON inválido")
	}
	if r.Hits == nil {
		return Resultado{}, errors.New("datajud: resposta sem o campo hits; consulta não confirmada")
	}
	out := Resultado{SemRegistro: len(r.Hits.Hits) == 0}
	for _, h := range r.Hits.Hits {
		s := h.Source
		p := Processo{Grau: Cortar(string(s.Grau), MaxCampo), Tribunal: Cortar(string(s.Tribunal), MaxCampo),
			OrgaoJulgador: Cortar(s.Orgao.nome(), MaxCampo), Sigilo: s.NivelSigilo > 0}
		if len(s.Movimentos) > maxMovimentos {
			return Resultado{}, errors.New("datajud: movimentos em excesso; consulta não confirmada")
		}
		for _, m := range s.Movimentos {
			p.Movimentos = append(p.Movimentos, movimento(m, p.Grau, p.OrgaoJulgador))
		}
		out.Processos = append(out.Processos, p)
	}
	return out, nil
}

// Chave identifica o movimento entre coletas: grau|data|código|nome|complemento resumido. Os campos são escapados
// (barra e barra vertical), então dois movimentos diferentes nunca dão a mesma chave por causa de um separador no texto.
func Chave(m Movimento) string {
	data := m.DataBruta
	if !m.DataHora.IsZero() {
		data = m.DataHora.UTC().Format("2006-01-02T15:04:05")
	}
	esc := strings.NewReplacer(`\`, `\\`, `|`, `\|`)
	return strings.Join([]string{esc.Replace(m.Grau), esc.Replace(data), esc.Replace(m.Codigo), esc.Replace(m.Nome), esc.Replace(m.Complemento)}, "|")
}
