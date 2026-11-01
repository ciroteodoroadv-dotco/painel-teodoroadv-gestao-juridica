// Package processos cuida do cadastro de processos e de suas partes.
// O número é validado pelo dígito verificador (Res. CNJ 65). O sistema NÃO calcula prazo.
package processos

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"painelteodoro/internal/pessoas"
	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/webx"
)

// Opcao é valor/rótulo de lista suspensa.
type Opcao = webx.Opcao

// Situacoes possíveis.
var Situacoes = []Opcao{
	{Valor: "ativo", Rotulo: "Ativo"}, {Valor: "suspenso", Rotulo: "Suspenso"},
	{Valor: "arquivado", Rotulo: "Arquivado"}, {Valor: "encerrado", Rotulo: "Encerrado"},
}

// Qualidades das partes.
var Qualidades = []Opcao{
	{Valor: "cliente", Rotulo: "Cliente"}, {Valor: "autor", Rotulo: "Autor"},
	{Valor: "reu", Rotulo: "Réu"}, {Valor: "terceiro", Rotulo: "Terceiro"},
}

func rotulo(list []Opcao, v string) string {
	for _, o := range list {
		if o.Valor == v {
			return o.Rotulo
		}
	}
	return v
}

// PessoaLookup é o que este domínio precisa do domínio de pessoas (interface explícita, sem SQL cruzado).
type PessoaLookup interface {
	Names(ctx context.Context, ids []string) (map[string]string, error)
	Exist(ctx context.Context, ids []string) (map[string]bool, error)
	Buscar(ctx context.Context, q string, limit int) ([]pessoas.Opcao, error)
}

// UserLookup é o que este domínio precisa do cadastro de usuários.
type UserLookup interface {
	Options(ctx context.Context) ([]Opcao, error)
	Emails(ctx context.Context, ids []string) (map[string]string, error)
}

// Parte liga uma pessoa ao processo.
type Parte struct{ PessoaID, Nome, Qualidade string }

// QualidadeRotulo devolve o rótulo da qualidade.
func (p Parte) QualidadeRotulo() string { return rotulo(Qualidades, p.Qualidade) }

// Processo é a visão de leitura.
type Processo struct {
	ID, Numero, Orgao, Situacao string
	ResponsavelID               string
	ResponsavelEmail            string
	Version                     int
	CriadoEm, AtualizadoEm      time.Time
	Partes                      []Parte
}

// SituacaoRotulo devolve o rótulo da situação.
func (p Processo) SituacaoRotulo() string { return rotulo(Situacoes, p.Situacao) }

// Input vem do formulário.
type Input struct{ Numero, Orgao, Situacao, ResponsavelID string }

type normalized struct{ numero, orgao, situacao, responsavel string }

func (in Input) normalize() (normalized, error) {
	num, err := br.NormalizeCNJ(in.Numero)
	if err != nil {
		return normalized{}, webx.Invalidf(strings.ToUpper(err.Error()[:1]) + err.Error()[1:] + ".")
	}
	orgao := strings.TrimSpace(in.Orgao)
	if utf8.RuneCountInString(orgao) > 200 {
		return normalized{}, webx.Invalidf("Órgão/vara com mais de 200 caracteres.")
	}
	if rotulo(Situacoes, in.Situacao) == in.Situacao {
		return normalized{}, webx.Invalidf("Escolha a situação do processo.")
	}
	return normalized{numero: num, orgao: orgao, situacao: in.Situacao, responsavel: strings.TrimSpace(in.ResponsavelID)}, nil
}
