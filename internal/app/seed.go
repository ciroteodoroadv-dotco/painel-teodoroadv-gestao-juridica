package app

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/agenda"
	"painelteodoro/internal/atendimentos"
	"painelteodoro/internal/audiencias"
	"painelteodoro/internal/auth"
	"painelteodoro/internal/diligencias"
	"painelteodoro/internal/pessoas"
	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/webx"
	"painelteodoro/internal/prazos"
	"painelteodoro/internal/prazos/calendario"
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
	ate := &atendimentos.Service{Pool: pool, Pessoas: pes}
	praz := &prazos.Service{Pool: pool, Processos: proc, Users: users}
	tar := &tarefas.Service{Pool: pool, Pessoas: pes, Vinculos: vinculos{ate: ate, proc: proc}, Prazos: prazoSync{p: praz}}
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
	// Atendimentos e tarefas fictícios (a mesma massa existe em tools/refb/atendimentos_tarefas.py).
	var a1, a2 string
	for _, a := range []struct {
		in   atendimentos.Input
		dest *string
	}{
		{atendimentos.Input{Clientes: []string{ids["maria"]}, PartesAdversas: []string{ids["banco"]}, CriadoPorID: ids["joao"],
			Descricao: "Revisão de contrato bancário (exemplo)", DataAbertura: "2026-10-05"}, &a1},
		{atendimentos.Input{Clientes: []string{ids["empresa"]}, Descricao: "Consulta sobre cobrança de duplicatas (exemplo)",
			DataAbertura: "2026-10-20", Observacoes: "Cliente pediu retorno por telefone."}, &a2},
		{atendimentos.Input{Clientes: []string{ids["joao"]}, PartesAdversas: []string{ids["maria"]},
			Descricao: "Triagem de caso trabalhista (exemplo)", DataAbertura: "2026-11-02"}, new(string)},
	} {
		id, _, err := ate.Create(ctx, ac, a.in)
		if err != nil {
			return err
		}
		*a.dest = id
	}
	partes := []tarefas.ParteIn{{PessoaID: ids["maria"], Papel: "Autor"}, {PessoaID: ids["banco"], Papel: "Réu"}}
	t1, _, err := tar.Create(ctx, ac, tarefas.Input{OrigemTipo: "processo", OrigemID: p1, Titulo: "Protocolar embargos à execução (exemplo)",
		Estado: "pendente", Prioridade: "Urgente", CriadoPorID: ids["joao"], DataBase: "2026-11-03", Prazo: "2026-11-10", PrazoFatal: "2026-11-12",
		Partes: partes, Responsaveis: []string{ids["joao"]}, Descricao: "Conferir os cálculos antes do protocolo."}, true)
	if err != nil {
		return err
	}
	for _, t := range []struct {
		in        tarefas.Input
		confirmar bool
	}{
		{tarefas.Input{OrigemTipo: "atendimento", OrigemID: a1, Titulo: "Ligar para a cliente (exemplo)", Subtitulo: "Ligação",
			Estado: "em-execucao", Prioridade: "Alta", Prazo: "2026-11-05", Responsaveis: []string{ids["joao"]}}, true},
		{tarefas.Input{OrigemTipo: "atendimento", OrigemID: a2, Titulo: "Aula sobre execução (exemplo)", Subtitulo: "Aula",
			Estado: "revisao", Prioridade: "Normal", Prazo: "2026-11-20"}, true},
		{tarefas.Input{OrigemTipo: "processo", OrigemID: p2, Titulo: "Conferir intimação (exemplo)", Estado: "a-confirmar",
			Prioridade: "Normal", PrazoFatal: "2026-11-15", Responsaveis: []string{ids["joao"]}}, false},
		{tarefas.Input{OrigemTipo: "atendimento", OrigemID: a1, Titulo: "Enviar minuta ao cliente (exemplo)",
			Estado: "concluidas-com-sucesso", Prioridade: "Baixa"}, true},
		{tarefas.Input{OrigemTipo: "processo", OrigemID: p1, TarefaPai: t1, Titulo: "Juntar procuração (exemplo)", Estado: "pendente",
			Prioridade: "Normal", PrazoFatal: "2026-11-11", Partes: partes, Responsaveis: []string{ids["joao"]}}, true},
	} {
		if _, _, err := tar.Create(ctx, ac, t.in, t.confirmar); err != nil {
			return err
		}
	}
	if err := semearAgenda(ctx, pool, ac, pes, proc, ate, tar, ids, p1, p2, a2); err != nil {
		return err
	}
	if err := semearIntimacoes(ctx, pool, ac, proc, tar); err != nil {
		return err
	}
	cal := &calendario.Service{Pool: pool}
	if err := cal.Semear(ctx, webx.Hoje(time.Now()).Year()); err != nil {
		return err
	}
	return semearCalendarios(ctx, ac, cal)
}

// semSenha aceita qualquer senha: só existe para o cancelamento de audiência da demonstração (SeedDemo roda só em dev).
type semSenha struct{}

func (semSenha) Confirmar(context.Context, webx.Actor, string) error { return nil }

// semearAgenda cria compromissos, audiências e diligências fictícios (a mesma massa existe em tools/refb/agenda.py).
func semearAgenda(ctx context.Context, pool *pgxpool.Pool, ac webx.Actor, pes *pessoas.Service, proc *processos.Service,
	ate *atendimentos.Service, tar *tarefas.Service, ids map[string]string, p1, p2, a2 string) error {
	aud := &audiencias.Service{Pool: pool, Pessoas: pes, Processos: proc, Reauth: semSenha{}}
	dil := &diligencias.Service{Pool: pool, Pessoas: pes, Processos: proc, Conflitos: diligenciaConflitos{s: aud}}
	ag := &agenda.Service{Pool: pool, Pessoas: pes, Vinculos: agendaVinculos{ate: ate, proc: proc}, Audiencias: agendaAudiencias{s: aud},
		Diligencias: agendaDiligencias{s: dil}, Tarefas: agendaTarefas{s: tar}}
	for _, c := range []agenda.Input{
		{Tipo: "Reunião", Descricao: "Reunião de alinhamento com a cliente (exemplo)", Data: "2026-11-10", Hora: "14:00", Local: "Escritório (exemplo)",
			ClienteID: ids["maria"], ProcessoID: p1, CriadoPorID: ids["joao"]},
		{Tipo: "Ligação", Descricao: "Ligação de retorno ao cliente (exemplo)", Data: "2026-11-11", Hora: "10:30", Local: "Telefone (exemplo)", AtendimentoID: a2},
		{Tipo: "Aula", Descricao: "Aula sobre processo civil (exemplo)", Data: "2026-11-12", Hora: "19:00", Local: "Sala virtual (exemplo)"},
	} {
		if _, err := ag.CriarCompromisso(ctx, ac, c); err != nil {
			return err
		}
	}
	num := br.MakeCNJ("0000101", "2025", "8", "26", "0100")
	if _, _, err := aud.Create(ctx, ac, audiencias.Input{Tipo: "instrucao-e-julgamento", Titulo: "Audiência de instrução — Maria Exemplo x Banco de Teste",
		Processo: num, ClienteID: ids["maria"], Data: "2026-11-13", Hora: "09:30", Juizo: "1ª Vara Cível de Exemplo", Local: "Fórum (exemplo), sala 3",
		Descricao: "Levar a procuração (exemplo)."}); err != nil {
		return err
	}
	if _, _, err := aud.Create(ctx, ac, audiencias.Input{Tipo: "sustentacao-oral", Titulo: "Sustentação oral no recurso (exemplo)",
		ClienteID: ids["empresa"], Data: "2026-11-18", Hora: "14:00", Juizo: "Câmara de Exemplo", Local: "Tribunal (exemplo)"}); err != nil {
		return err
	}
	var cancelada string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM audiencias WHERE codigo='AUD. 0000002'`).Scan(&cancelada); err != nil {
		return err
	}
	if err := aud.Cancelar(ctx, ac, cancelada, 1, "Remarcada pelo juízo (exemplo)", ""); err != nil {
		return err
	}
	for _, d := range []agenda.DiligenciaNova{
		{Titulo: "Protocolo presencial no fórum (exemplo)", ProcessoID: p1, ClienteID: ids["maria"], Local: "Fórum (exemplo)", DataLimite: "2026-11-14", Hora: "11:00", Responsavel: "Fulano Exemplo"},
		{Titulo: "Vistoria no imóvel (exemplo)", ProcessoID: p2, Local: "Imóvel (exemplo)", DataLimite: "2026-11-20", Hora: "15:00", Responsavel: "Beltrano Fictício"},
	} {
		if _, err := (agendaDiligencias{s: dil}).Criar(ctx, ac, d); err != nil {
			return err
		}
	}
	return nil
}
