package intimacoes

import (
	"context"
	"time"

	"painelteodoro/internal/intimacoes/djen"
)

// ComunicacaoDoProcesso é uma comunicação do DJEN trazida pela consulta FILTRADA por número de processo (segredo de justiça).
type ComunicacaoDoProcesso struct {
	ID, Tribunal, Orgao, Processo, Tipo, Texto, Link string
	Data                                             time.Time
}

// Hash é o mesmo hash da coleta por OAB: a mesma comunicação nunca vira duas linhas.
func (c ComunicacaoDoProcesso) Hash() string {
	return djen.Hash(djen.Comunicacao{ID: c.ID, Tribunal: c.Tribunal, Orgao: c.Orgao, Processo: c.Processo, Tipo: c.Tipo,
		DataPublicacao: c.Data, Texto: c.Texto, Link: c.Link})
}

// Resumo é o texto cortado em 2000 caracteres (a coluna resumo).
func (c ComunicacaoDoProcesso) Resumo() string { return djen.NormalizarResumo(c.Texto) }

// ColetorPorNumero consulta o DJEN pela OAB filtrando pelo número do processo.
type ColetorPorNumero interface {
	ColetarPorNumero(ctx context.Context, oab, uf, numero string, desde, ate time.Time) ([]ComunicacaoDoProcesso, error)
}

// ClienteSigiloso liga o cliente do DJEN à consulta por número.
type ClienteSigiloso struct{ C *djen.Cliente }

// ColetarPorNumero implementa ColetorPorNumero.
func (k ClienteSigiloso) ColetarPorNumero(ctx context.Context, oab, uf, numero string, desde, ate time.Time) ([]ComunicacaoDoProcesso, error) {
	r, err := k.C.ColetarPorNumero(ctx, oab, uf, numero, desde, ate)
	if err != nil {
		return nil, err
	}
	if r.Parcial {
		return nil, errParcial
	}
	out := make([]ComunicacaoDoProcesso, 0, len(r.Itens))
	for _, c := range r.Itens {
		out = append(out, ComunicacaoDoProcesso{ID: c.ID, Tribunal: c.Tribunal, Orgao: c.Orgao, Processo: c.Processo, Tipo: c.Tipo,
			Texto: c.Texto, Link: c.Link, Data: c.DataPublicacao})
	}
	return out, nil
}

type webxErro string

func (e webxErro) Error() string { return string(e) }
