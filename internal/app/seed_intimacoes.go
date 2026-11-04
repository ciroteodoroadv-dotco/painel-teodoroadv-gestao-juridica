package app

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/intimacoes"
	"painelteodoro/internal/intimacoes/djen"
	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/coletas"
	"painelteodoro/internal/platform/webx"
	"painelteodoro/internal/processos"
	"painelteodoro/internal/tarefas"
)

// seedDJEN é o "DJEN" da demonstração: devolve três comunicações INVENTADAS. Nunca fala com a rede.
type seedDJEN struct{}

func (seedDJEN) Coletar(context.Context, string, string, time.Time, time.Time) (djen.Resultado, error) {
	dia := func(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }
	c := func(id, tipo, num string, data time.Time) djen.Comunicacao {
		return djen.Comunicacao{ID: id, Tribunal: "TJXX", Orgao: "1ª Vara de Exemplo", Processo: num, Tipo: tipo, DataPublicacao: data,
			Texto: tipo + " fictícia (exemplo).", Link: "https://exemplo.test/djen/" + id}
	}
	return djen.Resultado{Itens: []djen.Comunicacao{
		c("exemplo-1", "Sentença", br.MakeCNJ("0000301", "2026", "8", "26", "0100"), dia(30)),
		c("exemplo-2", "Despacho", br.MakeCNJ("0000302", "2026", "8", "26", "0100"), time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)),
		c("exemplo-3", "Intimação", br.MakeCNJ("0000303", "2026", "8", "26", "0100"), time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)),
	}}, nil
}

// semearIntimacoes cadastra uma OAB fictícia, recolhe as três comunicações de exemplo pelo fluxo REAL (intimação + tarefa de
// análise numa transação), confirma uma e descarta outra, e deixa o rodapé com um sucesso e uma falha de exemplo.
// A MESMA massa existe em tools/refb/intimacoes.py. Os horários são fixos para a prova de paridade.
func semearIntimacoes(ctx context.Context, pool *pgxpool.Pool, ac webx.Actor, proc *processos.Service, tar *tarefas.Service) error {
	agora := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	svc := &intimacoes.Service{Pool: pool, Coletas: &coletas.Store{Pool: pool}, Coletor: seedDJEN{},
		Tarefas: intimTarefas{s: tar}, TarefasDe: intimTarefas{s: tar}, Processos: intimProcessos{p: proc},
		Cfg: intimacoes.Config{Ativo: true}, Now: func() time.Time { return agora }}
	if _, err := svc.CadastrarOAB(ctx, ac, intimacoes.OABIn{Nome: "Advogado Exemplo", Numero: "123456", UF: "SP",
		AutorizacaoBase: intimacoes.BaseAutorizacaoPadrao, AutorizacaoEm: "2026-11-01", Ativa: true}); err != nil {
		return err
	}
	if _, err := svc.ColetarAgora(ctx, ac); err != nil {
		return err
	}
	linhas, _, _, err := svc.Lista(ctx) // mais recente primeiro: Intimação (11/02), Despacho (11/02), Sentença (10/30)
	if err != nil {
		return err
	}
	if len(linhas) != 3 {
		return errors.New("demonstração de intimações: esperava 3 comunicações")
	}
	if err := svc.Decidir(ctx, ac, linhas[1].ID, "confirmada"); err != nil {
		return err
	}
	if err := svc.Decidir(ctx, ac, linhas[2].ID, "descartada"); err != nil {
		return err
	}
	return (&coletas.Store{Pool: pool}).Registrar(ctx, coletas.Estado{Modulo: "monitor", Origem: "DataJud", Status: "falha",
		Mensagem: "Dado fictício de exemplo.", UltimaTentativa: &agora})
}
