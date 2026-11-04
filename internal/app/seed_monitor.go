package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/intimacoes"
	"painelteodoro/internal/monitor"
	"painelteodoro/internal/monitor/datajud"
	"painelteodoro/internal/platform/coletas"
	"painelteodoro/internal/platform/webx"
	"painelteodoro/internal/processos"
)

// seedDataJud é o "DataJud" da demonstração: só o processo ativo (PRO. 0000001) tem registro. Primeira rodada = linha de base;
// segunda = duas movimentações novas, uma em cada grau. Nunca fala com a rede.
type seedDataJud struct{ fase int }

func (seedDataJud) TemChave() bool { return true }

func (s *seedDataJud) Consultar(_ context.Context, _, numero string) (datajud.Resultado, error) {
	if !strings.HasPrefix(numero, "0000101-") {
		return datajud.Resultado{SemRegistro: true}, nil
	}
	m := func(grau, cod, nome string, h int, dia int) datajud.Movimento {
		return datajud.Movimento{Grau: grau, Codigo: cod, Nome: nome, OrgaoJulgador: "1ª Vara de Exemplo",
			DataHora: time.Date(2026, 11, dia, h, 0, 0, 0, time.UTC)}
	}
	g1 := []datajud.Movimento{m("G1", "26", "Distribuição", 12, 1)}
	g2 := []datajud.Movimento{m("G2", "132", "Recebimento", 12, 1)}
	if s.fase >= 2 {
		g1 = append(g1, m("G1", "123", "Intimação", 9, 2))
		g2 = append(g2, m("G2", "11", "Despacho", 14, 2))
	}
	return datajud.Resultado{Processos: []datajud.Processo{
		{Grau: "G1", Tribunal: "TJSP", OrgaoJulgador: "1ª Vara de Exemplo", Movimentos: g1},
		{Grau: "G2", Tribunal: "TJSP", OrgaoJulgador: "Câmara de Exemplo", Movimentos: g2}}}, nil
}

// semearMonitor roda o monitor duas vezes sobre os processos da demonstração (base e novidades) pelo fluxo REAL e deixa o
// rodapé com o estado real do monitor. A MESMA massa existe em tools/refb/intimacoes.py.
func semearMonitor(ctx context.Context, pool *pgxpool.Pool, ac webx.Actor, proc *processos.Service, intim *intimacoes.Service, agora time.Time) error {
	cli := &seedDataJud{}
	svc := &monitor.Service{Pool: pool, Coletas: &coletas.Store{Pool: pool}, Cliente: cli, Processos: monProcessos{p: proc},
		OABs: monOABs{s: intim}, Registro: monRegistro{s: intim}, Cfg: monitor.Config{Ativo: true}, Now: func() time.Time { return agora }}
	for cli.fase = 1; cli.fase <= 2; cli.fase++ {
		if _, err := svc.ColetarAgora(ctx, ac); err != nil {
			return fmt.Errorf("demonstração do monitor (rodada %d): %w", cli.fase, err)
		}
	}
	if e, ok, err := svc.Coletas.Obter(ctx, monitor.Modulo); err != nil || !ok || e.Status != "parcial" {
		return errors.New("demonstração do monitor: estado inesperado")
	}
	return nil
}
