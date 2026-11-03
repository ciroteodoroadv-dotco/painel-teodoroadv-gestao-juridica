package pessoas

import (
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/webx"
)

// Input é o que vem do formulário (texto cru). Os nomes dos campos seguem o painel B.
type Input struct {
	TipoPessoa                                              string
	NomeCompleto, CPF, RG, EstadoCivil, Profissao           string
	Nacionalidade, DataNascimento                           string
	RazaoSocial, NomeFantasia, CNPJ                         string
	RepresentanteNome, RepresentanteCPF, RepresentanteCargo string
	Logradouro, Numero, Complemento, Bairro, CEP            string
	Cidade, UF, Email, Telefone, Observacoes                string
}

type normalized struct {
	tipo                                    string
	nomeCompleto, estadoCivil, profissao    string
	nacionalidade                           string
	nascimento                              *time.Time
	razao, fantasia                         string
	repNome, repCargo                       string
	doc, rg, repCPF                         string // dígitos / texto a cifrar
	logradouro, numero, complemento, bairro string
	cep, cidade                             string
	uf                                      *string
	email, tel, obs                         string
}

// normalize valida e padroniza. O tipo vem de fora (na edição é o gravado: não se troca).
// As mensagens são as do painel B quando existem lá.
func (in Input) normalize(tipo string) (normalized, error) {
	n := normalized{tipo: tipo}
	if tipo != Fisica && tipo != Juridica {
		return n, webx.Invalidf("Escolha se é pessoa física ou jurídica.")
	}
	n.estadoCivil = strings.TrimSpace(in.EstadoCivil)
	if n.estadoCivil != "" && !contem(EstadosCivis, n.estadoCivil) {
		return n, webx.Invalidf("Estado civil inválido.")
	}
	var err error
	if tipo == Fisica {
		if n.nomeCompleto = upper(in.NomeCompleto); n.nomeCompleto == "" {
			return n, webx.Invalidf("Nome completo é obrigatório.")
		}
		if d := strings.TrimSpace(in.CPF); d != "" {
			if n.doc, err = br.NormalizeCPF(d); err != nil {
				return n, webx.Invalidf("CPF inválido — confira os dígitos informados.")
			}
		}
		n.rg = strings.TrimSpace(in.RG)
		n.profissao = strings.TrimSpace(in.Profissao)
		n.nacionalidade = strings.TrimSpace(in.Nacionalidade)
		if s := strings.TrimSpace(in.DataNascimento); s != "" {
			t, err := time.Parse("2006-01-02", s)
			if err != nil || t.After(time.Now().AddDate(0, 0, 1)) {
				return n, webx.Invalidf("Data de nascimento inválida.")
			}
			n.nascimento = &t
		}
	} else {
		if n.razao = upper(in.RazaoSocial); n.razao == "" {
			return n, webx.Invalidf("Razão social é obrigatória.")
		}
		n.fantasia = upper(in.NomeFantasia)
		if d := strings.TrimSpace(in.CNPJ); d != "" {
			if n.doc, err = br.NormalizeCNPJ(d); err != nil {
				return n, webx.Invalidf("CNPJ inválido — confira os dígitos informados.")
			}
		}
		n.repNome = strings.TrimSpace(in.RepresentanteNome)
		n.repCargo = strings.TrimSpace(in.RepresentanteCargo)
		if d := strings.TrimSpace(in.RepresentanteCPF); d != "" {
			if n.repCPF, err = br.NormalizeCPF(d); err != nil {
				return n, webx.Invalidf("CPF do representante inválido — confira os dígitos informados.")
			}
		}
	}
	n.logradouro, n.numero = strings.TrimSpace(in.Logradouro), strings.TrimSpace(in.Numero)
	n.complemento, n.bairro = strings.TrimSpace(in.Complemento), strings.TrimSpace(in.Bairro)
	n.cidade, n.obs = strings.TrimSpace(in.Cidade), strings.TrimSpace(in.Observacoes)
	if c := br.Digits(in.CEP); c != "" {
		if len(c) != 8 {
			return n, webx.Invalidf("CEP inválido — informe oito dígitos.")
		}
		n.cep = c
	}
	if u := strings.ToUpper(strings.TrimSpace(in.UF)); u != "" {
		if utf8.RuneCountInString(u) != 2 {
			return n, webx.Invalidf("UF inválida — use duas letras.")
		}
		n.uf = &u
	}
	if e := strings.TrimSpace(in.Email); e != "" {
		a, err := mail.ParseAddress(e)
		if err != nil || a.Address != e {
			return n, webx.Invalidf("E-mail inválido.")
		}
		n.email = e
	}
	if t := strings.TrimSpace(in.Telefone); t != "" {
		d := br.Digits(t)
		if len(d) < 8 || len(d) > 13 {
			return n, webx.Invalidf("Telefone inválido (use DDD + número).")
		}
		n.tel = d
	}
	return n, n.limites()
}

func contem(lista []string, v string) bool {
	for _, e := range lista {
		if e == v {
			return true
		}
	}
	return false
}

// limites impede texto desmedido em qualquer campo livre.
func (n normalized) limites() error {
	for _, c := range []struct {
		nome, v string
		max     int
	}{{"Nome", n.nomeCompleto + n.razao, 200}, {"Nome fantasia", n.fantasia, 200}, {"Profissão", n.profissao, 120},
		{"Nacionalidade", n.nacionalidade, 80}, {"RG", n.rg, 60}, {"Representante legal", n.repNome, 200},
		{"Cargo", n.repCargo, 120}, {"Logradouro", n.logradouro, 200}, {"Número", n.numero, 30},
		{"Complemento", n.complemento, 120}, {"Bairro", n.bairro, 120}, {"Cidade", n.cidade, 120},
		{"E-mail", n.email, 200}, {"Observações", n.obs, 4000}} {
		if utf8.RuneCountInString(c.v) > c.max {
			return webx.Invalidf(c.nome + " com mais de " + strconv.Itoa(c.max) + " caracteres.")
		}
	}
	return nil
}
