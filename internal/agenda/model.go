// Package agenda cuida de audiências, reuniões e diligências. Datas e horas são as que a pessoa digita
// (fuso America/Sao_Paulo); o sistema não calcula prazo nem sugere data.
package agenda

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"painelteodoro/internal/platform/webx"
)

// Opcao é valor/rótulo de lista suspensa.
type Opcao = webx.Opcao

// Tipos de evento.
var Tipos = []Opcao{
	{Valor: "audiencia", Rotulo: "Audiência"}, {Valor: "reuniao", Rotulo: "Reunião"}, {Valor: "ligacao", Rotulo: "Ligação"},
	{Valor: "diligencia", Rotulo: "Diligência"}, {Valor: "aula", Rotulo: "Aula"}, {Valor: "outro", Rotulo: "Outro"},
}

func rotulo(v string) string {
	for _, o := range Tipos {
		if o.Valor == v {
			return o.Rotulo
		}
	}
	return v
}

// ProcessoLookup, PessoaLookup e UserLookup são os contratos com os outros domínios.
type ProcessoLookup interface {
	Existe(ctx context.Context, id string) (bool, error)
	Resumos(ctx context.Context, ids []string) (map[string]string, error)
	Buscar(ctx context.Context, q string, limit int) ([]Opcao, error)
}

// PessoaLookup é o que a agenda precisa do cadastro de pessoas.
type PessoaLookup interface {
	Names(ctx context.Context, ids []string) (map[string]string, error)
	Exist(ctx context.Context, ids []string) (map[string]bool, error)
	Buscar(ctx context.Context, q string, limit int) ([]Opcao, error)
}

// UserLookup é o que a agenda precisa do cadastro de usuários.
type UserLookup interface {
	Options(ctx context.Context) ([]Opcao, error)
	Emails(ctx context.Context, ids []string) (map[string]string, error)
}

// Participante é uma pessoa convocada ao evento.
type Participante struct{ PessoaID, Nome string }

// Evento é a visão de leitura (horários já no fuso do escritório).
type Evento struct {
	ID, Tipo, Titulo, Local    string
	Inicio                     time.Time
	Fim                        *time.Time
	ProcessoID, ProcessoNumero string
	ResponsavelID, Responsavel string
	Version                    int
	Participantes              []Participante
}

// TipoRotulo devolve o rótulo do tipo.
func (e Evento) TipoRotulo() string { return rotulo(e.Tipo) }

// InicioFmt e FimFmt formatam no fuso do escritório.
func (e Evento) InicioFmt() string { return e.Inicio.In(webx.SP()).Format("02/01/2006 15:04") }

// FimFmt devolve só a hora de término (vazio se não houver).
func (e Evento) FimFmt() string {
	if e.Fim == nil {
		return ""
	}
	return e.Fim.In(webx.SP()).Format("15:04")
}

// Input vem do formulário.
type Input struct{ Tipo, Titulo, Inicio, Fim, Local, ProcessoID, ResponsavelID string }

type normalized struct {
	tipo, titulo, local, processo, responsavel string
	inicio                                     time.Time
	fim                                        *time.Time
}

func (in Input) normalize() (normalized, error) {
	n := normalized{tipo: in.Tipo, titulo: strings.TrimSpace(in.Titulo), local: strings.TrimSpace(in.Local),
		processo: strings.TrimSpace(in.ProcessoID), responsavel: strings.TrimSpace(in.ResponsavelID)}
	if rotulo(n.tipo) == n.tipo {
		return n, webx.Invalidf("Escolha o tipo do evento.")
	}
	if c := utf8.RuneCountInString(n.titulo); c < 2 || c > 300 {
		return n, webx.Invalidf("Informe o título (de 2 a 300 caracteres).")
	}
	if utf8.RuneCountInString(n.local) > 500 {
		return n, webx.Invalidf("Local com mais de 500 caracteres.")
	}
	t, err := webx.ParseLocal(in.Inicio)
	if err != nil {
		return n, webx.Invalidf("Informe a data e a hora de início.")
	}
	n.inicio = t
	if strings.TrimSpace(in.Fim) != "" {
		f, err := webx.ParseLocal(in.Fim)
		if err != nil {
			return n, webx.Invalidf("Data/hora de término inválida.")
		}
		if f.Before(t) {
			return n, webx.Invalidf("O término não pode ser antes do início.")
		}
		n.fim = &f
	}
	return n, nil
}

// ErrConflito indica outro compromisso (não audiência) no mesmo instante, para o mesmo responsável.
// Com é «título» às HH:MM do compromisso que já ocupa o horário. Para a tela é um erro de validação.
type ErrConflito struct{ Com string }

func (e ErrConflito) Error() string { return "Já existe " + e.Com + " para este responsável." }

// Unwrap faz o conflito aparecer ao handler como erro de validação (mensagem na tela, dados preservados).
func (e ErrConflito) Unwrap() error { return webx.Invalid{Msg: e.Error()} }

// TipoClasse: audiência em vermelho, demais tipos neutros (selo sempre com texto).
func (e Evento) TipoClasse() string {
	if e.Tipo == "audiencia" {
		return "badge-danger"
	}
	return "badge-neutral"
}

// DataFmt e HoraFmt formatam no fuso do escritório.
func (e Evento) DataFmt() string { return e.Inicio.In(webx.SP()).Format("02/01/2006") }

// HoraFmt devolve a hora de início.
func (e Evento) HoraFmt() string { return e.Inicio.In(webx.SP()).Format("15:04") }
