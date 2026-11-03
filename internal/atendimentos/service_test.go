package atendimentos

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/pessoas"
	"painelteodoro/internal/platform/migrate"
	"painelteodoro/internal/platform/webx"
)

var chave = []byte("0123456789abcdef0123456789abcdef")

func novoServico(t *testing.T) (*Service, webx.Actor, *pgxpool.Pool, map[string]string) {
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
	_, _ = pool.Exec(ctx, `TRUNCATE pessoas, processos CASCADE`)
	if err := migrate.Reset(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Up(ctx, pool); err != nil {
		t.Fatal(err)
	}
	st := &auth.Store{Pool: pool}
	h, _ := auth.HashPassword("cavalo-bateria-grampo-7")
	tx, _ := pool.Begin(ctx)
	uid, err := st.CreateUser(ctx, tx, "ate@exemplo.test", auth.Advogado, h, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `TRUNCATE pessoas, processos CASCADE`) })
	ac := webx.Actor{UserID: uid, RequestID: "t"}
	pes := &pessoas.Service{Pool: pool, FieldKey: chave}
	ids := map[string]string{}
	for k, nome := range map[string]string{"maria": "Maria Exemplo", "joao": "João Fictício", "ana": "Ana Teste"} {
		id, _, err := pes.Create(ctx, ac, pessoas.Input{TipoPessoa: pessoas.Fisica, NomeCompleto: nome, Nacionalidade: "Brasileiro(a)"})
		if err != nil {
			t.Fatal(err)
		}
		ids[k] = id
	}
	return &Service{Pool: pool, Pessoas: pes}, ac, pool, ids
}

func TestCriarAtendimento(t *testing.T) {
	s, ac, pool, p := novoServico(t)
	ctx := context.Background()
	in := Input{Clientes: []string{p["maria"]}, PartesAdversas: []string{p["joao"]}, Descricao: "Revisão de contrato de ação", DataAbertura: "2026-11-03", CriadoPorID: p["ana"]}
	id, cod, err := s.Create(ctx, ac, in)
	if err != nil || cod != "ATE. 0000001" {
		t.Fatalf("criar = %q %v", cod, err)
	}
	_, cod2, _ := s.Create(ctx, ac, Input{Clientes: []string{p["joao"]}, Descricao: "Outro", DataAbertura: "2026-11-03"})
	if cod2 != "ATE. 0000002" {
		t.Errorf("segundo código = %q", cod2)
	}
	a, err := s.Get(ctx, id)
	if err != nil || a.Tipo != "Extrajudicial" || a.Status != "Em triagem" || len(a.Clientes) != 1 || a.Clientes[0].Nome != "MARIA EXEMPLO" ||
		len(a.PartesAdversas) != 1 || a.CriadoPor != "ANA TESTE" || a.DataAbertura.Format("2006-01-02") != "2026-11-03" {
		t.Fatalf("leitura = %+v %v", a, err)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='atendimento.criado'`).Scan(&n)
	if n != 2 {
		t.Errorf("auditoria atendimento.criado = %d", n)
	}
}

func TestValidacoesDoAtendimento(t *testing.T) {
	s, ac, _, p := novoServico(t)
	ctx := context.Background()
	casos := []struct {
		nome string
		in   Input
		msg  string
	}{
		{"sem cliente", Input{PartesAdversas: []string{p["joao"]}, Descricao: "x", DataAbertura: "2026-11-03"}, "Selecione ao menos um cliente."},
		{"cliente inexistente", Input{Clientes: []string{"00000000-0000-0000-0000-000000000000"}, Descricao: "x", DataAbertura: "2026-11-03"}, "Pessoa não encontrada."},
		{"parte adversa inexistente", Input{Clientes: []string{p["maria"]}, PartesAdversas: []string{"00000000-0000-0000-0000-000000000000"}, Descricao: "x", DataAbertura: "2026-11-03"}, "Pessoa não encontrada."},
		{"sem descrição", Input{Clientes: []string{p["maria"]}, Descricao: "  ", DataAbertura: "2026-11-03"}, "Descrição / objeto é obrigatória."},
		{"data inválida", Input{Clientes: []string{p["maria"]}, Descricao: "x", DataAbertura: "03/11/2026"}, "Data do cadastro inválida."},
		{"criado por inexistente", Input{Clientes: []string{p["maria"]}, Descricao: "x", DataAbertura: "2026-11-03", CriadoPorID: "00000000-0000-0000-0000-000000000000"}, "Pessoa não encontrada."},
	}
	for _, c := range casos {
		_, _, err := s.Create(ctx, ac, c.in)
		var inv webx.Invalid
		if !errors.As(err, &inv) || inv.Msg != c.msg {
			t.Errorf("%s: erro = %v, esperado %q", c.nome, err, c.msg)
		}
	}
}

func TestEditarAtendimento(t *testing.T) {
	s, ac, pool, p := novoServico(t)
	ctx := context.Background()
	id, _, _ := s.Create(ctx, ac, Input{Clientes: []string{p["maria"]}, PartesAdversas: []string{p["joao"]}, Descricao: "Antes", DataAbertura: "2026-11-03"})
	novo := Input{Clientes: []string{p["joao"], p["ana"]}, Descricao: "Depois", Observacoes: "nota", DataAbertura: "2026-11-04"}
	if err := s.Update(ctx, ac, id, 1, novo); err != nil {
		t.Fatal(err)
	}
	a, _ := s.Get(ctx, id)
	if len(a.Clientes) != 2 || a.Clientes[0].Nome != "JOÃO FICTÍCIO" || a.Clientes[1].Nome != "ANA TESTE" || len(a.PartesAdversas) != 0 ||
		a.Descricao != "Depois" || a.Observacoes != "nota" || a.Version != 2 {
		t.Fatalf("depois de editar = %+v", a)
	}
	if err := s.Update(ctx, ac, id, 1, novo); !errors.Is(err, webx.ErrConflict) {
		t.Errorf("versão antiga = %v, esperado conflito", err)
	}
	if err := s.Update(ctx, ac, "00000000-0000-0000-0000-000000000000", 1, novo); !errors.Is(err, webx.ErrNotFound) {
		t.Errorf("inexistente = %v", err)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='atendimento.atualizado'`).Scan(&n)
	if n != 1 {
		t.Errorf("auditoria atendimento.atualizado = %d", n)
	}
}

func TestBuscaSemAcentoEOpcoes(t *testing.T) {
	s, ac, _, p := novoServico(t)
	ctx := context.Background()
	id, _, _ := s.Create(ctx, ac, Input{Clientes: []string{p["maria"]}, PartesAdversas: []string{p["joao"]}, Descricao: "Revisão de contrato", DataAbertura: "2026-11-03"})
	_, _, _ = s.Create(ctx, ac, Input{Clientes: []string{p["ana"]}, Descricao: "Outra demanda", DataAbertura: "2026-11-02"})
	for q, want := range map[string]int{"": 2, "revisao": 1, "REVISÃO": 1, "ficticio": 1, "ATE. 0000002": 1, "inexistente": 0, "maria": 1} {
		l, err := s.Listar(ctx, q)
		if err != nil || len(l) != want {
			t.Errorf("busca %q = %d resultados (%v), esperado %d", q, len(l), err, want)
		}
	}
	ops, err := s.Opcoes(ctx)
	if err != nil || len(ops) != 2 || ops[0].Valor != id && ops[1].Valor != id {
		t.Fatalf("opções = %v %v", ops, err)
	}
	if ops[0].Rotulo != "ATE. 0000001 · MARIA EXEMPLO" {
		t.Errorf("rótulo para Tarefas = %q", ops[0].Rotulo)
	}
	po, _ := s.OpcoesOrigem(ctx)
	if !strings.HasPrefix(po[0].Rotulo, "ATE. 0000001 · Extrajudicial · MARIA EXEMPLO · 03/11/2026") {
		t.Errorf("rótulo para Processos = %q", po[0].Rotulo)
	}
	if ok, _ := s.Existe(ctx, id); !ok {
		t.Error("Existe deveria ser verdadeiro")
	}
	if ok, _ := s.Existe(ctx, "00000000-0000-0000-0000-000000000000"); ok {
		t.Error("Existe deveria ser falso para id desconhecido")
	}
	if cod, cli, err := s.Resumo(ctx, id); err != nil || cod != "ATE. 0000001" || cli != "MARIA EXEMPLO" {
		t.Errorf("resumo = %q %q %v", cod, cli, err)
	}
}
