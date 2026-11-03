package app

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/agenda"
	"painelteodoro/internal/auth"
	"painelteodoro/internal/pessoas"
	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/webx"
	"painelteodoro/internal/processos"
	"painelteodoro/internal/tarefas"
)

// SeedDemo cria dados 100% FICTÍCIOS para conhecer o sistema (nomes "Exemplo", documentos de teste
// públicos, e-mails @exemplo.test). Recusa rodar se já houver pessoas cadastradas.
// Nunca use em produção: o comando de linha exige PAINEL_ENV=dev.
// Os mesmos dados (nomes, códigos, datas fixas) existem em tools/refb/fixtures.py, para a prova de paridade com o painel B.
func SeedDemo(ctx context.Context, pool *pgxpool.Pool, fieldKey []byte, store *auth.Store, actorID string) error {
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pessoas`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return errors.New("já existem pessoas cadastradas; a demonstração só roda em banco vazio")
	}
	users := userLookup{s: store}
	pes := &pessoas.Service{Pool: pool, FieldKey: fieldKey}
	proc := &processos.Service{Pool: pool, Pessoas: pes}
	pes.Papeis = proc
	tar := &tarefas.Service{Pool: pool, Processos: proc, Users: users}
	ac := webx.Actor{UserID: actorID, RequestID: "seed-demo"}

	ids := map[string]string{}
	for _, p := range []struct {
		chave string
		in    pessoas.Input
	}{
		{"maria", pessoas.Input{TipoPessoa: pessoas.Fisica, NomeCompleto: "Maria Exemplo", CPF: "529.982.247-25", RG: "12.345.678-9 SSP/SP",
			EstadoCivil: "Solteiro(a)", Profissao: "Professora", Nacionalidade: "Brasileiro(a)", DataNascimento: "1985-04-12",
			Logradouro: "Rua das Flores", Numero: "100", Complemento: "Apto 12", Bairro: "Centro", CEP: "01001-000",
			Cidade: "São Paulo", UF: "SP", Email: "maria@exemplo.test", Telefone: "(11) 91234-5678"}},
		{"joao", pessoas.Input{TipoPessoa: pessoas.Fisica, NomeCompleto: "João Fictício", CPF: "111.444.777-35",
			Nacionalidade: "Brasileiro(a)", Cidade: "Rio de Janeiro", UF: "RJ", Email: "joao@exemplo.test", Telefone: "(21) 98888-7777"}},
		{"empresa", pessoas.Input{TipoPessoa: pessoas.Juridica, RazaoSocial: "Empresa Modelo Ltda", NomeFantasia: "Modelo",
			CNPJ: "11.222.333/0001-81", RepresentanteNome: "Ana Representante", RepresentanteCPF: "123.456.789-09",
			RepresentanteCargo: "Sócia-administradora", Cidade: "Campinas", UF: "SP", Email: "contato@modelo.test", Telefone: "(11) 3333-4444"}},
		{"banco", pessoas.Input{TipoPessoa: pessoas.Juridica, RazaoSocial: "Banco de Teste S.A.", CNPJ: "22.233.344/0001-83",
			Cidade: "Brasília", UF: "DF"}},
	} {
		id, _, err := pes.Create(ctx, ac, p.in)
		if err != nil {
			return err
		}
		ids[p.chave] = id
	}

	novo := func(in processos.Input) (string, error) { id, _, err := proc.Create(ctx, ac, in); return id, err }
	p1, err := novo(processos.Input{Numero: br.MakeCNJ("0000101", "2025", "8", "26", "0100"), ValorCausa: "R$ 10.000,00",
		Partes:       []processos.ParteIn{{PessoaID: ids["maria"], Papel: "Autor", Cliente: true}, {PessoaID: ids["banco"], Papel: "Réu"}},
		Responsaveis: []string{ids["joao"]}, Sistema: "PJe", Instancia: "1º grau", Classe: "Procedimento Comum Cível",
		Materia: "Consumidor", Comarca: "São Paulo", Juizo: "1ª Vara Cível de Exemplo", Contingencia: "Ativa",
		Distribuicao: "2026-09-15", Citacao: "2026-09-30", Polo: "Ativo"})
	if err != nil {
		return err
	}
	p2, err := novo(processos.Input{Numero: br.MakeCNJ("0000202", "2024", "5", "02", "0001"), ValorCausa: "R$ 50.000,00",
		Partes:  []processos.ParteIn{{PessoaID: ids["empresa"], Papel: "Exequente", Cliente: true}, {PessoaID: ids["joao"], Papel: "Executado"}},
		Sistema: "e-SAJ", Instancia: "1º grau", Classe: "Execução de Título Extrajudicial", Materia: "Empresarial",
		Comarca: "Campinas", Juizo: "2ª Vara de Exemplo", Distribuicao: "2026-10-20"})
	if err != nil {
		return err
	}
	p3, err := novo(processos.Input{ // propositalmente incompleto: mostra o selo de pendências
		Partes: []processos.ParteIn{{PessoaID: ids["maria"], Papel: "Autor", Cliente: true}, {PessoaID: ids["joao"], Papel: "Terceiro interessado"}},
		Classe: "Mandado de Segurança"})
	if err != nil {
		return err
	}
	if err := proc.MoverEstado(ctx, ac, p2, 1, "sobrestado", "Suspensão por acordo de exemplo"); err != nil {
		return err
	}
	if err := proc.MoverEstado(ctx, ac, p3, 1, "extinto", "Extinção de exemplo"); err != nil {
		return err
	}
	dia := func(s string) time.Time { t, _ := time.Parse("2006-01-02", s); return t }
	for _, a := range []struct{ d, t, doc string }{
		{"2026-09-16", "Distribuição da inicial (exemplo)", ""}, {"2026-10-02", "Juntada de contestação (exemplo)", "Contestação"},
	} {
		if err := proc.RegistrarAndamento(ctx, ac, p1, dia(a.d), a.t, a.doc); err != nil {
			return err
		}
	}
	if err := proc.AtualizarFinanceiro(ctx, ac, p1, "Honorários", "R$ 1.500,00", "Pendente"); err != nil {
		return err
	}
	if err := proc.AtualizarFinanceiro(ctx, ac, p1, "Custas", "R$ 300,00", "Pago"); err != nil {
		return err
	}
	ag := &agenda.Service{Pool: pool, Processos: proc, Pessoas: pes, Users: users}
	amanha := time.Now().AddDate(0, 0, 1).In(webx.SP()).Format("2006-01-02") + "T14:00"
	if _, err := ag.Create(ctx, ac, agenda.Input{Tipo: "audiencia", Titulo: "Exemplo: audiência de instrução", Inicio: amanha,
		Local: "Fórum (exemplo), sala 3", ProcessoID: p1, ResponsavelID: actorID}); err != nil {
		return err
	}
	d := func(dias int) string { return time.Now().AddDate(0, 0, dias).Format("2006-01-02") }
	for _, t := range []tarefas.Input{
		{Titulo: "Exemplo: revisar contestação", ProcessoID: p1, ResponsavelID: actorID, Vencimento: d(-1), Status: "aberta"},
		{Titulo: "Exemplo: reunir documentos do cliente", ProcessoID: p1, ResponsavelID: actorID, Vencimento: d(2), Status: "em_andamento"},
		{Titulo: "Exemplo: ligar para a Empresa Exemplo", ProcessoID: p2, ResponsavelID: actorID, Vencimento: d(5), Status: "aberta"},
		{Titulo: "Exemplo: tarefa sem data", ResponsavelID: actorID, Status: "aberta"},
	} {
		if _, err := tar.Create(ctx, ac, t); err != nil {
			return err
		}
	}
	return nil
}
