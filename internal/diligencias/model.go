// Package diligencias cuida das diligências (cartório, perícia, vistoria, protocolo presencial…), no desenho do painel B:
// código DIL. 0000001, status Pendente/Concluída/Cancelada que não se edita pela tela, responsável em texto livre e
// bloqueio quando coincide com uma audiência. O B vale na forma; o v2 vale na regra.
package diligencias

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

// Status do B (ficha/diligencia.go e diligencias/http.go): o padrão é Pendente.
const (
	StatusPendente  = "Pendente"
	StatusConcluida = "Concluída"
	StatusCancelada = "Cancelada"
)

// Diligencia é a visão de leitura.
type Diligencia struct {
	ID, Codigo, Titulo, ProcessoNumero, ProcessoID, ClienteID, Local, Responsavel, Status string
	DataLimite                                                                            time.Time
	Hora                                                                                  string
	Version                                                                               int
}

// DataISO preenche o campo <input type="date">.
func (d Diligencia) DataISO() string {
	if d.DataLimite.IsZero() {
		return ""
	}
	return d.DataLimite.Format("2006-01-02")
}

// Input vem do formulário de edição (e da Agenda, que traz ProcessoID em vez do número digitado).
type Input struct {
	Titulo, Processo, ProcessoID, ClienteID, Local, DataLimite, Hora, Responsavel string
}

type normalized struct {
	titulo, processo, procID, cliente, local, responsavel string
	data                                                  time.Time
	hora                                                  string
}

func (in Input) normalize() (n normalized, err error) {
	n.titulo = strings.TrimSpace(in.Titulo)
	d, derr := webx.DataISO(in.DataLimite)
	if n.titulo == "" || derr != nil {
		return n, webx.Invalidf("Título e data-limite são obrigatórios.")
	}
	n.data = d
	if h := strings.TrimSpace(in.Hora); h != "" {
		var ok bool
		if n.hora, ok = webx.HoraHM(h); !ok {
			return n, webx.Invalidf("Hora inválida.")
		}
	}
	n.processo = strings.TrimSpace(in.Processo)
	if norm, e := br.NormalizeCNJ(n.processo); e == nil {
		n.processo = norm
	}
	n.procID, n.cliente = strings.TrimSpace(in.ProcessoID), strings.TrimSpace(in.ClienteID)
	n.local, n.responsavel = strings.TrimSpace(in.Local), strings.TrimSpace(in.Responsavel)
	for _, t := range []string{n.titulo, n.processo, n.local, n.responsavel} {
		if utf8.RuneCountInString(t) > 1000 {
			return n, webx.Invalidf("Texto longo demais.")
		}
	}
	return n, nil
}

// PessoaLookup é o que este domínio precisa do cadastro de pessoas.
type PessoaLookup interface {
	Refs(ctx context.Context, ids []string) ([]Ref, error)
	Exist(ctx context.Context, ids []string) (map[string]bool, error)
	Opcoes(ctx context.Context, ac webx.Actor, comDocumento bool) ([]Opcao, error)
}

// ProcessoLookup liga o processo ao texto da diligência.
type ProcessoLookup interface {
	// PorNumero devolve o id do processo de número CNJ já normalizado, ou "" se não houver.
	PorNumero(ctx context.Context, numero string) (string, error)
	// Rotulo devolve o texto do processo (número, senão título, senão código); ok=false se não existir.
	Rotulo(ctx context.Context, id string) (rotulo string, ok bool, err error)
}

// Conflito é uma audiência que ocupa o mesmo dia e a mesma hora (Data em AAAA-MM-DD, como o B imprime).
type Conflito struct{ Titulo, Data, Hora string }

// ConflitoLookup é o que este domínio precisa de Audiências para a regra de conflito de horário.
type ConflitoLookup interface {
	NoHorario(ctx context.Context, data time.Time, hora string) ([]Conflito, error)
}

// Linha é uma diligência nas tabelas (data já em dd/mm/aaaa).
type Linha struct {
	ID, Codigo, Titulo, DataLimite, Hora, Local, Responsavel, Status, Processo string
	Busca                                                                      string
	Cancelada                                                                  bool
}
