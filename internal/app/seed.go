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
	proc := &processos.Service{Pool: pool, Pessoas: pes, Users: users}
	pes.Processos = proc
	tar := &tarefas.Service{Pool: pool, Processos: proc, Users: users}
	ac := webx.Actor{UserID: actorID, RequestID: "seed-demo"}

	ids := map[string]string{}
	for _, p := range []struct {
		chave string
		in    pessoas.Input
	}{
		{"cli1", pessoas.Input{Tipo: "PF", Nome: "Cliente Exemplo Um", Documento: "529.982.247-25", Email: "um@exemplo.test", Telefone: "11 90000-0001", Papel: "cliente"}},
		{"cli2", pessoas.Input{Tipo: "PJ", Nome: "Empresa Exemplo Ltda", Documento: "11.222.333/0001-81", Email: "contato@exemplo.test", Papel: "cliente"}},
		{"contra", pessoas.Input{Tipo: "PF", Nome: "Parte Contrária Exemplo", Documento: "123.456.789-09", Papel: "parte_contraria"}},
		{"test", pessoas.Input{Tipo: "PF", Nome: "Testemunha Exemplo", Papel: "testemunha"}},
	} {
		id, err := pes.Create(ctx, ac, p.in)
		if err != nil {
			return err
		}
		ids[p.chave] = id
	}

	p1, err := proc.Create(ctx, ac, processos.Input{Numero: br.MakeCNJ("0000101", "2025", "8", "26", "0100"),
		Orgao: "1ª Vara Cível (exemplo)", Situacao: "ativo", ResponsavelID: actorID})
	if err != nil {
		return err
	}
	p2, err := proc.Create(ctx, ac, processos.Input{Numero: br.MakeCNJ("0000202", "2024", "5", "02", "0001"),
		Orgao: "2ª Vara do Trabalho (exemplo)", Situacao: "suspenso", ResponsavelID: actorID})
	if err != nil {
		return err
	}
	for _, v := range [][3]string{{p1, ids["cli1"], "cliente"}, {p1, ids["contra"], "reu"}, {p2, ids["cli2"], "cliente"}} {
		if err := proc.AddParte(ctx, ac, v[0], v[1], v[2]); err != nil {
			return err
		}
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
