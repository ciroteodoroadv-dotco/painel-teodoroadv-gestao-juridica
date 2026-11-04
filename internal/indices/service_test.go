package indices

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/indices/bcb"
	"painelteodoro/internal/platform/coletas"
	"painelteodoro/internal/platform/migrate"
)

type fonteFalsa struct {
	pontos map[int]bcb.Ponto
	erros  map[int]error
	chamou int
}

func (f *fonteFalsa) Ultimo(_ context.Context, codigo int) (bcb.Ponto, error) {
	f.chamou++
	if err := f.erros[codigo]; err != nil {
		return bcb.Ponto{}, err
	}
	return f.pontos[codigo], nil
}

func novoServico(t *testing.T, f Fonte, ativo bool) *Service {
	t.Helper()
	dsn := os.Getenv("PAINEL_TEST_DB")
	if dsn == "" {
		t.Skip("PAINEL_TEST_DB não definida: teste de integração pulado")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	limpa := `TRUNCATE pessoas, processos, atendimentos, agenda_compromissos, audiencias, diligencias, intimacoes, oab_monitoradas, coletas, calendario_dias, calendarios CASCADE`
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), limpa) })
	_, _ = pool.Exec(ctx, limpa)
	if err := migrate.Reset(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Up(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return &Service{Pool: pool, Coletas: &coletas.Store{Pool: pool}, Fonte: f, Ativo: ativo, ValidadeDias: 40, Now: func() time.Time { return agora }}
}

func todosOk() *fonteFalsa {
	f := &fonteFalsa{pontos: map[int]bcb.Ponto{}, erros: map[int]error{}}
	for i, s := range bcb.Series {
		f.pontos[s.Codigo] = bcb.Ponto{Data: data("2026-10-01"), Valor: []string{"14.90", "0.52", "0.40", "-0.10", "5.4321", "1518.00"}[i]}
	}
	return f
}

func TestColetaGravaAsSeisEMarcaSucesso(t *testing.T) {
	s := novoServico(t, todosOk(), true)
	r, err := s.Coletar(context.Background())
	if err != nil || r.Status != "sucesso" || r.Gravadas != 6 {
		t.Fatalf("coleta: %+v %v", r, err)
	}
	vs, _ := s.Repo().Listar(context.Background())
	if len(vs) != 6 || vs[0].Codigo != 432 || vs[5].Codigo != 1619 {
		t.Fatalf("valores = %+v", vs)
	}
	e, ok, _ := s.Coletas.Obter(context.Background(), Modulo)
	if !ok || e.Status != "sucesso" || e.UltimoSucesso == nil {
		t.Errorf("estado = %+v", e)
	}
}

func TestColetaParcialMantemOAntigoComMarca(t *testing.T) {
	f := todosOk()
	s := novoServico(t, f, true)
	if _, err := s.Coletar(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.erros[433] = errors.New("bcb: HTTP 500")
	f.pontos[432] = bcb.Ponto{Data: data("2026-10-30"), Valor: "15.00"}
	f.pontos[433] = bcb.Ponto{Data: data("2026-10-30"), Valor: "9.99"} // não pode entrar: a série falhou
	r, err := s.Coletar(context.Background())
	if err != nil || r.Status != "parcial" || r.Gravadas != 5 || r.Falhas != 1 {
		t.Fatalf("coleta parcial: %+v %v", r, err)
	}
	vs, _ := s.Repo().Listar(context.Background())
	for _, v := range vs {
		switch v.Codigo {
		case 432:
			if v.Valor != "15.000000" || v.Falhou {
				t.Errorf("Selic = %+v", v)
			}
		case 433:
			if v.Valor != "0.520000" || !v.Falhou || v.Referencia.Format("2006-01-02") != "2026-10-01" {
				t.Errorf("IPCA deveria manter o antigo e marcar falha: %+v", v)
			}
		}
	}
}

func TestColetaComTudoRuimNaoGravaNada(t *testing.T) {
	f := &fonteFalsa{erros: map[int]error{}}
	for _, s := range bcb.Series {
		f.erros[s.Codigo] = errors.New("bcb: HTTP 500")
	}
	s := novoServico(t, f, true)
	r, err := s.Coletar(context.Background())
	if err != nil || r.Status != "falha" || r.Gravadas != 0 {
		t.Fatalf("coleta: %+v %v", r, err)
	}
	if vs, _ := s.Repo().Listar(context.Background()); len(vs) != 0 {
		t.Errorf("não deveria gravar: %+v", vs)
	}
}

func TestDesligadoNaoChamaAFonteEFicaBloqueado(t *testing.T) {
	f := todosOk()
	s := novoServico(t, f, false)
	r, err := s.Coletar(context.Background())
	if err != nil || r.Status != "bloqueado" || f.chamou != 0 {
		t.Fatalf("desligado: %+v %v chamadas=%d", r, err, f.chamou)
	}
	if s.Iniciar(context.Background(), nil, time.Millisecond) {
		t.Error("desligado não pode criar agendador")
	}
	e, ok, _ := s.Coletas.Obter(context.Background(), Modulo)
	if !ok || e.Status != "bloqueado" || e.Mensagem != MsgDesligada {
		t.Errorf("estado = %+v", e)
	}
	if err := s.IniciarEstado(context.Background()); err != nil {
		t.Fatal(err)
	}
	h, err := s.Fragmento(context.Background())
	if err != nil || !contem(string(h), "Índices econômicos: bloqueado — "+MsgDesligada) {
		t.Errorf("fragmento = %s %v", h, err)
	}
}

func contem(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
