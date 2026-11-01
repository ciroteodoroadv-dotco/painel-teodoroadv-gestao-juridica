// Package pessoas cuida do cadastro de pessoas (clientes, partes, testemunhas).
// Documento (CPF/CNPJ) é sensível: guardado cifrado, buscável por índice cego, mascarado nas telas.
package pessoas

import (
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/webx"
)

// Papéis aceitos e seus rótulos.
var Papeis = []Opcao{
	{Valor: "cliente", Rotulo: "Cliente"}, {Valor: "parte_contraria", Rotulo: "Parte contrária"},
	{Valor: "testemunha", Rotulo: "Testemunha"}, {Valor: "outro", Rotulo: "Outro"},
}

// Opcao é um valor/rótulo de lista suspensa.
type Opcao = webx.Opcao

// Pessoa é a visão de leitura.
type Pessoa struct {
	ID, Tipo, Nome, Papel  string
	Email, Telefone        string
	DocMask                string
	RetencaoAte            *time.Time
	Version                int
	CriadoEm, AtualizadoEm time.Time
}

// PapelRotulo devolve o rótulo do papel.
func (p Pessoa) PapelRotulo() string { return rotulo(p.Papel) }

func rotulo(v string) string {
	for _, o := range Papeis {
		if o.Valor == v {
			return o.Rotulo
		}
	}
	return v
}

// Input é o que vem do formulário (texto cru).
type Input struct {
	Tipo, Nome, Documento, Email, Telefone, Papel, RetencaoAte string
}

type normalized struct {
	tipo, nome, doc, email, tel, papel string
	retencao                           *time.Time
}

func (in Input) normalize() (normalized, error) {
	n := normalized{tipo: in.Tipo, papel: in.Papel, nome: strings.TrimSpace(in.Nome)}
	if n.tipo != "PF" && n.tipo != "PJ" {
		return n, webx.Invalidf("Escolha se é pessoa física ou jurídica.")
	}
	if c := utf8.RuneCountInString(n.nome); c < 2 || c > 200 {
		return n, webx.Invalidf("Informe o nome (de 2 a 200 caracteres).")
	}
	if rotulo(n.papel) == n.papel {
		return n, webx.Invalidf("Escolha o papel da pessoa.")
	}
	if d := strings.TrimSpace(in.Documento); d != "" {
		doc, err := br.NormalizeDocumento(n.tipo, d)
		if err != nil {
			return n, webx.Invalidf(err.Error() + " (confira os números e se o tipo PF/PJ está certo).")
		}
		n.doc = doc
	}
	if e := strings.TrimSpace(in.Email); e != "" {
		a, err := mail.ParseAddress(e)
		if err != nil || a.Address != e || len(e) > 200 {
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
	if r := strings.TrimSpace(in.RetencaoAte); r != "" {
		t, err := time.Parse("2006-01-02", r)
		if err != nil {
			return n, webx.Invalidf("Data de retenção inválida.")
		}
		n.retencao = &t
	}
	return n, nil
}
