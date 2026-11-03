package pessoas

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/migrate"
	"painelteodoro/internal/platform/webx"
)

var chave = []byte("0123456789abcdef0123456789abcdef")

type falsoPapeis map[string][]PapelEmProcesso

func (f falsoPapeis) PapeisPorPessoa(_ context.Context, ids []string) (map[string][]PapelEmProcesso, error) {
	out := map[string][]PapelEmProcesso{}
	for _, id := range ids {
		out[id] = f[id]
	}
	return out, nil
}

func novoServico(t *testing.T) (*Service, webx.Actor, *pgxpool.Pool) {
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
	uid, err := st.CreateUser(ctx, tx, "svc@exemplo.test", auth.Advogado, h, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `TRUNCATE pessoas, processos CASCADE`) })
	return &Service{Pool: pool, FieldKey: chave}, webx.Actor{UserID: uid, RequestID: "t"}, pool
}

func pf(nome, cpf string) Input {
	return Input{TipoPessoa: Fisica, NomeCompleto: nome, CPF: cpf, Nacionalidade: "Brasileiro(a)"}
}

func TestCategoriaDeOsQuatroCasosDoB(t *testing.T) {
	casos := []struct {
		nome string
		in   []PapelEmProcesso
		quer Categoria
	}{
		{"sem processo", nil, CatSemProcesso},
		{"cliente em um e adversa em outro: cliente vence", []PapelEmProcesso{{Papel: "Réu"}, {Papel: "Autor", Cliente: true}}, CatCliente},
		{"terceiro interessado", []PapelEmProcesso{{Papel: "Réu"}, {Papel: "Terceiro interessado"}}, CatTerceiro},
		{"qualquer outro papel é parte adversa", []PapelEmProcesso{{Papel: "Réu"}}, CatAdversa},
		{"só responsável pelo cadastro cai em adversa (regra literal do B)", []PapelEmProcesso{{Papel: PapelResponsavel, Responsavel: true}}, CatAdversa},
	}
	for _, c := range casos {
		if got := CategoriaDe(c.in); got != c.quer {
			t.Errorf("%s: %q, esperado %q", c.nome, got, c.quer)
		}
	}
}

func TestCriarFisicaEJuridicaComCodigoSequencial(t *testing.T) {
	s, ac, pool := novoServico(t)
	ctx := context.Background()
	id1, c1, err := s.Create(ctx, ac, pf("  maria exemplo ", "529.982.247-25"))
	if err != nil || c1 != "PES. 0000001" {
		t.Fatalf("PF: %q %v", c1, err)
	}
	id2, c2, err := s.Create(ctx, ac, Input{TipoPessoa: Juridica, RazaoSocial: "empresa exemplo ltda", NomeFantasia: "modelo",
		CNPJ: "11.222.333/0001-81", RepresentanteNome: "Ana", RepresentanteCPF: "111.444.777-35"})
	if err != nil || c2 != "PES. 0000002" {
		t.Fatalf("PJ: %q %v", c2, err)
	}
	p1, _ := s.Get(ctx, id1)
	if p1.Nome != "MARIA EXEMPLO" || p1.NomeCompleto != "MARIA EXEMPLO" {
		t.Errorf("nome deveria sair em maiúsculas: %q", p1.Nome)
	}
	p2, _ := s.Get(ctx, id2)
	if p2.Nome != "EMPRESA EXEMPLO LTDA" || p2.NomeFantasia != "MODELO" || p2.Documento != "**.***.***/****-81" {
		t.Errorf("PJ gravada errado: %+v", p2)
	}
	// LGPD: nada em claro no banco.
	var enc, rep []byte
	var raw string
	if err := pool.QueryRow(ctx, `SELECT documento_enc, representante_cpf_enc, observacoes||nome_completo||razao_social FROM pessoas WHERE id=$1`, id2).Scan(&enc, &rep, &raw); err != nil {
		t.Fatal(err)
	}
	if len(enc) == 0 || len(rep) == 0 || strings.Contains(string(enc), "11222333") || strings.Contains(string(rep), "11144477735") || strings.Contains(raw, "11222333") {
		t.Error("CNPJ e CPF do representante deveriam estar cifrados")
	}
}

func TestValidacoesDoServidor(t *testing.T) {
	s, ac, _ := novoServico(t)
	ctx := context.Background()
	casos := map[string]struct {
		in  Input
		msg string
	}{
		"cpf inválido":      {pf("Ana", "529.982.247-24"), "CPF inválido — confira os dígitos informados."},
		"nome vazio":        {pf("  ", ""), "Nome completo é obrigatório."},
		"razão vazia":       {Input{TipoPessoa: Juridica}, "Razão social é obrigatória."},
		"cnpj inválido":     {Input{TipoPessoa: Juridica, RazaoSocial: "X", CNPJ: "11.222.333/0001-82"}, "CNPJ inválido — confira os dígitos informados."},
		"tipo desconhecido": {Input{TipoPessoa: "PF", NomeCompleto: "X"}, "Escolha se é pessoa física ou jurídica."},
		"cep curto":         {Input{TipoPessoa: Fisica, NomeCompleto: "X", CEP: "123"}, "CEP inválido — informe oito dígitos."},
		"email ruim":        {Input{TipoPessoa: Fisica, NomeCompleto: "X", Email: "x@"}, "E-mail inválido."},
		"estado civil":      {Input{TipoPessoa: Fisica, NomeCompleto: "X", EstadoCivil: "Complicado"}, "Estado civil inválido."},
	}
	for nome, c := range casos {
		_, _, err := s.Create(ctx, ac, c.in)
		var inv webx.Invalid
		if !errors.As(err, &inv) || inv.Msg != c.msg {
			t.Errorf("%s: %v, esperado %q", nome, err, c.msg)
		}
	}
}

func TestDuplicidadeDeDocumento(t *testing.T) {
	s, ac, _ := novoServico(t)
	ctx := context.Background()
	if _, _, err := s.Create(ctx, ac, pf("Ana", "529.982.247-25")); err != nil {
		t.Fatal(err)
	}
	_, _, err := s.Create(ctx, ac, pf("Outra Ana", "52998224725"))
	var inv webx.Invalid
	if !errors.As(err, &inv) || inv.Msg != "Já existe uma pessoa com esse documento." {
		t.Errorf("duplicidade: %v", err)
	}
}

func TestEditarNaoTrocaTipoELevaVersao(t *testing.T) {
	s, ac, _ := novoServico(t)
	ctx := context.Background()
	id, _, _ := s.Create(ctx, ac, pf("Ana", ""))
	in := Input{TipoPessoa: Juridica, RazaoSocial: "Fingindo ser empresa", NomeCompleto: "Ana Souza", Observacoes: "nota"}
	if err := s.Update(ctx, ac, id, 1, in); err != nil {
		t.Fatal(err)
	}
	p, _ := s.Get(ctx, id)
	if p.TipoPessoa != Fisica || p.NomeCompleto != "ANA SOUZA" || p.RazaoSocial != "" || p.Observacoes != "nota" || p.Version != 2 {
		t.Errorf("edição: %+v", p)
	}
	if err := s.Update(ctx, ac, id, 1, in); !errors.Is(err, webx.ErrConflict) {
		t.Errorf("versão antiga deveria dar conflito: %v", err)
	}
}

func TestListagemMascaradaOuCompletaEAuditada(t *testing.T) {
	s, ac, pool := novoServico(t)
	ctx := context.Background()
	id, _, _ := s.Create(ctx, ac, pf("Ana", "529.982.247-25"))
	mask, _ := s.Listar(ctx, ac, false)
	if len(mask) != 1 || mask[0].Documento != "***.***.***-25" {
		t.Errorf("sem permissão: %+v", mask)
	}
	inteira, _ := s.Listar(ctx, ac, true)
	if inteira[0].Documento != "529.982.247-25" {
		t.Errorf("com permissão: %q", inteira[0].Documento)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='pessoa.documentos_listados'`).Scan(&n)
	if n != 1 {
		t.Errorf("a listagem completa deveria ser auditada uma vez, foi %d", n)
	}
	op, _ := s.Opcoes(ctx, ac, true)
	if len(op) != 1 || op[0].Valor != id || op[0].Rotulo != "ANA — Física (529.982.247-25)" {
		t.Errorf("rótulo da opção: %+v", op)
	}
	opm, _ := s.Opcoes(ctx, ac, false)
	if opm[0].Rotulo != "ANA — Física" {
		t.Errorf("rótulo sem documento: %+v", opm)
	}
}

func TestExcluirLogicoAuditadoERecusaSeEmProcesso(t *testing.T) {
	s, ac, pool := novoServico(t)
	ctx := context.Background()
	id, _, _ := s.Create(ctx, ac, pf("Ana", ""))
	s.Papeis = falsoPapeis{id: {{ProcessoID: "p", Papel: "Autor", Cliente: true}}}
	err := s.Delete(ctx, ac, id, 1)
	var inv webx.Invalid
	if !errors.As(err, &inv) || !strings.Contains(inv.Msg, "parte em processo") {
		t.Fatalf("deveria recusar: %v", err)
	}
	s.Papeis = falsoPapeis{}
	if err := s.Delete(ctx, ac, id, 1); err != nil {
		t.Fatal(err)
	}
	if rows, _ := s.Listar(ctx, ac, false); len(rows) != 0 {
		t.Error("pessoa excluída ainda aparece")
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM pessoas WHERE id=$1 AND excluido_em IS NOT NULL`, id).Scan(&n)
	if n != 1 {
		t.Error("exclusão deveria só marcar")
	}
	for _, a := range []string{"pessoa.criada", "pessoa.excluida"} {
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action=$1`, a).Scan(&n)
		if n == 0 {
			t.Errorf("faltou auditoria %s", a)
		}
	}
	if _, err := audit.Verify(ctx, pool); err != nil {
		t.Fatal(err)
	}
}

func TestGetCompletoRevelaEAudita(t *testing.T) {
	s, ac, pool := novoServico(t)
	ctx := context.Background()
	in := pf("Ana", "529.982.247-25")
	in.RG = "12.345.678-9 SSP/SP"
	id, _, _ := s.Create(ctx, ac, in)
	p, err := s.GetCompleto(ctx, ac, id)
	if err != nil || p.Documento != "529.982.247-25" || p.RG != "12.345.678-9 SSP/SP" {
		t.Fatalf("%+v %v", p, err)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='pessoa.documento_revelado' AND target_id=$1`, id).Scan(&n)
	if n != 1 {
		t.Error("revelar deveria ser auditado")
	}
}
