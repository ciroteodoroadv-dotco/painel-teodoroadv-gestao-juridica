// Package atendimentos cuida do registro das demandas do escritório (aba "Atendimentos" do painel B):
// cada atendimento tem cliente(s), parte(s) adversa(s), objeto e data. Não há exclusão (o B não tem).
package atendimentos

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"painelteodoro/internal/platform/webx"
)

// Opcao é valor/rótulo de lista suspensa; Ref é a referência curta a uma pessoa.
type (
	Opcao = webx.Opcao
	Ref   = webx.PessoaRef
)

// Valores padrão: o B guarda tipo e status só no mapeamento, sem campo na tela.
const (
	TipoPadrao   = "Extrajudicial"
	StatusPadrao = "Em triagem"
)

// PessoaLookup é o que este domínio precisa do cadastro de pessoas (interface; sem SQL cruzado).
// ids nulo = todas as pessoas não excluídas.
type PessoaLookup interface {
	Refs(ctx context.Context, ids []string) ([]Ref, error)
}

// Atendimento é a visão de leitura.
type Atendimento struct {
	ID, Codigo, Tipo, Status, Descricao, Observacoes string
	DataAbertura                                     time.Time
	CriadoPorID, CriadoPor                           string
	Clientes, PartesAdversas                         []Ref
	Version                                          int
}

// DataBR devolve a data no formato dd/mm/aaaa (fmtDateOrRaw do B).
func (a Atendimento) DataBR() string { return a.DataAbertura.Format("02/01/2006") }

// DataISO devolve a data para o campo <input type="date">.
func (a Atendimento) DataISO() string { return a.DataAbertura.Format("2006-01-02") }

func nomes(rs []Ref) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = strings.ToUpper(r.Nome)
	}
	return out
}

// NomesClientes lista os clientes em MAIÚSCULAS separados por "; " (nomesAtendimento do B).
func (a Atendimento) NomesClientes() string { return strings.Join(nomes(a.Clientes), "; ") }

// NomesPartes lista as partes adversas em MAIÚSCULAS separadas por "; ".
func (a Atendimento) NomesPartes() string { return strings.Join(nomes(a.PartesAdversas), "; ") }

// Busca é o texto que a busca consulta: código, clientes, partes adversas, objeto e observações.
func (a Atendimento) Busca() string {
	return strings.Join([]string{a.Codigo, a.NomesClientes(), a.NomesPartes(), a.Descricao, a.Observacoes}, " ")
}

// Input vem do formulário. Datas em AAAA-MM-DD; clientes e partes adversas são ids de pessoa.
type Input struct {
	CriadoPorID, Descricao, Observacoes, DataAbertura string
	Clientes, PartesAdversas                          []string
}

type normalized struct {
	criadoPor, descricao, obs string
	data                      time.Time
	clientes, partes          []string
}

func unicos(ids []string) []string {
	var out []string
	vistos := map[string]bool{}
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" && !vistos[id] {
			vistos[id] = true
			out = append(out, id)
		}
	}
	return out
}

func (in Input) normalize() (n normalized, err error) {
	n.clientes, n.partes = unicos(in.Clientes), unicos(in.PartesAdversas)
	n.criadoPor = strings.TrimSpace(in.CriadoPorID)
	if len(n.clientes) == 0 {
		return n, webx.Invalidf("Selecione ao menos um cliente.")
	}
	if n.descricao = strings.TrimSpace(in.Descricao); n.descricao == "" {
		return n, webx.Invalidf("Descrição / objeto é obrigatória.")
	}
	if n.obs = strings.TrimSpace(in.Observacoes); utf8.RuneCountInString(n.obs) > 4000 || utf8.RuneCountInString(n.descricao) > 4000 {
		return n, webx.Invalidf("Texto longo demais.")
	}
	if n.data, err = time.Parse("2006-01-02", strings.TrimSpace(in.DataAbertura)); err != nil {
		return n, webx.Invalidf("Data do cadastro inválida.")
	}
	return n, nil
}
