// Package pessoas cuida do cadastro de pessoas (clientes, partes adversas, terceiros interessados).
// A ficha segue o painel B (art. 319 do CPC). CPF, CNPJ, RG e CPF do representante são sensíveis:
// guardados cifrados, mascarados nas telas de quem só lê. O papel da pessoa NÃO é um campo: deriva dos processos.
package pessoas

import (
	"strings"
	"time"

	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/webx"
)

// Opcao é um valor/rótulo de lista suspensa.
type Opcao = webx.Opcao

// EstadosCivis é a lista do painel B (OPCOES.estadoCivil), na mesma ordem.
var EstadosCivis = []string{"Solteiro(a)", "Casado(a)", "Divorciado(a)", "Viúvo(a)", "União Estável", "Separado(a) Judicialmente"}

// Tipos de pessoa (os mesmos valores do painel B).
const (
	Fisica   = "Física"
	Juridica = "Jurídica"
)

// Pessoa é a visão de leitura.
type Pessoa struct {
	ID, Codigo, TipoPessoa, Nome                        string // Nome = nome de exibição (nome completo ou razão social)
	NomeCompleto, EstadoCivil, Profissao, Nacionalidade string
	DataNascimento                                      *time.Time
	RazaoSocial, NomeFantasia                           string
	RepresentanteNome, RepresentanteCargo               string
	Logradouro, Numero, Complemento, Bairro             string
	CEP, Cidade, UF                                     string
	Email, Telefone, Observacoes                        string
	Documento                                           string // CPF/CNPJ: completo e formatado, ou mascarado
	RG, RepresentanteCPF                                string // só preenchidos por GetCompleto (auditado)
	Version                                             int
	CriadoEm, AtualizadoEm                              time.Time

	encDoc, encRG, encRep []byte
}

// Juridica diz se é pessoa jurídica.
func (p Pessoa) Juridica() bool { return p.TipoPessoa == Juridica }

// NomeLista é o texto da coluna Nome (pessoaNome do B).
func (p Pessoa) NomeLista() string {
	if p.Juridica() {
		if p.RazaoSocial == "" {
			return "(sem razão social)"
		}
		return p.RazaoSocial
	}
	if p.NomeCompleto == "" {
		return "(sem nome)"
	}
	return p.NomeCompleto
}

// TipoLista é o texto da coluna Tipo.
func (p Pessoa) TipoLista() string {
	if p.TipoPessoa == "" {
		return "—"
	}
	return p.TipoPessoa
}

// DocLista é o texto da coluna Documento (pessoaDoc do B).
func (p Pessoa) DocLista() string {
	if p.Documento == "" {
		return "—"
	}
	return p.Documento
}

// CodigoLista é o texto da coluna Código.
func (p Pessoa) CodigoLista() string {
	if p.Codigo == "" {
		return "—"
	}
	return p.Codigo
}

// Nascimento devolve a data no formato do campo date (AAAA-MM-DD) ou vazio.
func (p Pessoa) Nascimento() string {
	if p.DataNascimento == nil {
		return ""
	}
	return p.DataNascimento.Format("2006-01-02")
}

// CPFMask devolve o CPF formatado para preencher o formulário (vazio para pessoa jurídica).
func (p Pessoa) CPFMask() string {
	if p.Juridica() {
		return ""
	}
	return p.Documento
}

// CNPJMask devolve o CNPJ formatado (vazio para pessoa física).
func (p Pessoa) CNPJMask() string {
	if !p.Juridica() {
		return ""
	}
	return p.Documento
}

// RepCPFFormatado devolve o CPF do representante formatado.
func (p Pessoa) RepCPFFormatado() string { return br.FormatDocumento(p.RepresentanteCPF) }

// CEPMask devolve o CEP como #####-###.
func (p Pessoa) CEPMask() string { return MaskCEP(p.CEP) }

// TelefoneMask devolve o telefone como (##) #####-####.
func (p Pessoa) TelefoneMask() string { return MaskTelefone(p.Telefone) }

// MaskCEP aplica #####-### (igual a maskCEP do B).
func MaskCEP(v string) string {
	d := br.Digits(v)
	if len(d) > 8 {
		d = d[:8]
	}
	if len(d) <= 5 {
		return d
	}
	return d[:5] + "-" + d[5:]
}

// MaskTelefone aplica (##) ####-#### ou (##) #####-#### (igual a maskPhone do B).
func MaskTelefone(v string) string {
	d := br.Digits(v)
	if len(d) > 11 {
		d = d[:11]
	}
	switch {
	case len(d) == 0:
		return ""
	case len(d) <= 2:
		return "(" + d
	case len(d) <= 6:
		return "(" + d[:2] + ") " + d[2:]
	case len(d) <= 10:
		return "(" + d[:2] + ") " + d[2:6] + "-" + d[6:]
	}
	return "(" + d[:2] + ") " + d[2:7] + "-" + d[7:]
}

func upper(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }
