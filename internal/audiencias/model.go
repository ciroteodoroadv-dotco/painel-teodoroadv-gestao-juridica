// Package audiencias cuida das audiências, no desenho do painel B: seis tipos de ato, código AUD. 0000001, aviso (nunca
// bloqueio) quando coincide com outra audiência e cancelamento que preserva a ficha, confirmado com a senha do usuário.
// O B vale na forma; o v2 vale na regra (reautenticação, auditoria, versão).
package audiencias

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/webx"
)

// Opcao é valor/rótulo de lista suspensa; Ref é a referência curta a uma pessoa.
type (
	Opcao = webx.Opcao
	Ref   = webx.PessoaRef
)

// Ato é um dos 6 tipos de ato (TIPOS_AUDIENCIA do B): a chave é o valor gravado, o rótulo é o que se vê.
type Ato struct{ Chave, Rotulo string }

// Atos na ordem do B.
var Atos = []Ato{
	{"audiencia-ad-monitoria", "Audiência Ad Monitoria"},
	{"conciliacao-e-mediacao", "Conciliação e Mediação"},
	{"inicial-justica-do-trabalho", "Inicial — Justiça do Trabalho"},
	{"instrucao-e-julgamento", "Instrução e Julgamento"},
	{"sustentacao-oral", "Sustentação Oral"},
	{"una-justica-do-trabalho", "Una — Justiça do Trabalho"},
}

// RotuloAto devolve o rótulo do ato (a própria chave, ou "—", se desconhecida: labelTipoAudiencia do B).
func RotuloAto(chave string) string {
	for _, a := range Atos {
		if a.Chave == chave {
			return a.Rotulo
		}
	}
	if chave == "" {
		return "—"
	}
	return chave
}

func atoValido(chave string) bool {
	for _, a := range Atos {
		if a.Chave == chave {
			return true
		}
	}
	return false
}

// Audiencia é a visão de leitura.
type Audiencia struct {
	ID, Codigo, TipoAto, Titulo, ProcessoNumero, ProcessoID, ClienteID string
	Juizo, Local, Descricao, CancelamentoMotivo                        string
	Data                                                               time.Time
	Hora                                                               string
	Cancelada                                                          bool
	Version                                                            int
}

// RotuloTipo é o nome do tipo de ato.
func (a Audiencia) RotuloTipo() string { return RotuloAto(a.TipoAto) }

// DataISO preenche o campo <input type="date">.
func (a Audiencia) DataISO() string {
	if a.Data.IsZero() {
		return ""
	}
	return a.Data.Format("2006-01-02")
}

// Input vem do formulário. Data em AAAA-MM-DD, hora em HH:MM; Cliente é id de pessoa.
type Input struct {
	Tipo, Titulo, Processo, ClienteID, Data, Hora, Juizo, Local, Descricao string
}

type normalized struct {
	tipo, titulo, processo, cliente, juizo, local, descricao string
	data                                                     time.Time
	hora                                                     string
}

// normalize aplica as validações do B (saveAudiencia) pela ordem dele. Na edição o tipo do ato não conta (não se edita).
func (in Input) normalize(edicao bool) (n normalized, err error) {
	if !edicao && !atoValido(strings.TrimSpace(in.Tipo)) {
		return n, webx.Invalidf("Selecione o tipo do ato.")
	}
	n.tipo, n.titulo = strings.TrimSpace(in.Tipo), strings.TrimSpace(in.Titulo)
	d, derr := webx.DataISO(in.Data)
	h, hok := webx.HoraHM(in.Hora)
	if n.titulo == "" || derr != nil || !hok {
		return n, webx.Invalidf("Título, data e hora são obrigatórios.")
	}
	n.data, n.hora = d, h
	n.processo = strings.TrimSpace(in.Processo)
	if norm, e := br.NormalizeCNJ(n.processo); e == nil {
		n.processo = norm
	}
	n.cliente, n.juizo, n.local, n.descricao = strings.TrimSpace(in.ClienteID), strings.TrimSpace(in.Juizo), strings.TrimSpace(in.Local), strings.TrimSpace(in.Descricao)
	for _, t := range []string{n.titulo, n.processo, n.juizo, n.local} {
		if utf8.RuneCountInString(t) > 1000 {
			return n, webx.Invalidf("Texto longo demais.")
		}
	}
	if utf8.RuneCountInString(n.descricao) > 4000 {
		return n, webx.Invalidf("Texto longo demais.")
	}
	return n, nil
}

// PessoaLookup é o que este domínio precisa do cadastro de pessoas.
type PessoaLookup interface {
	Refs(ctx context.Context, ids []string) ([]Ref, error)
	Exist(ctx context.Context, ids []string) (map[string]bool, error)
	Opcoes(ctx context.Context, ac webx.Actor, comDocumento bool) ([]Opcao, error)
}

// ProcessoLookup liga o número digitado ao processo cadastrado (vínculo real).
type ProcessoLookup interface {
	// PorNumero devolve o id do processo de número CNJ já normalizado, ou "" se não houver.
	PorNumero(ctx context.Context, numero string) (string, error)
}

// Reauth confere a senha do usuário logado antes de uma ação destrutiva.
type Reauth interface {
	Confirmar(ctx context.Context, ac webx.Actor, senha string) error
}

// Linha é uma audiência nas tabelas (datas já em dd/mm/aaaa).
type Linha struct {
	ID, TipoAto, TipoRotulo, Codigo, Titulo, Data, Hora, Local, Processo string
	Cancelada                                                            bool
	Busca                                                                string
}
