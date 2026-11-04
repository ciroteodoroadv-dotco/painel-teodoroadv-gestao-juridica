package app

import (
	"context"
	"time"

	"painelteodoro/internal/agenda"
	"painelteodoro/internal/atendimentos"
	"painelteodoro/internal/audiencias"
	"painelteodoro/internal/diligencias"
	"painelteodoro/internal/platform/webx"
	"painelteodoro/internal/processos"
	"painelteodoro/internal/tarefas"
)

// Adaptadores da Agenda, Audiências e Diligências: o único lugar que liga as interfaces desses domínios aos serviços
// dos outros (os domínios só se falam por interfaces).

// agendaVinculos entrega à Agenda os seletores e os códigos de atendimentos e processos.
type agendaVinculos struct {
	ate  *atendimentos.Service
	proc *processos.Service
}

func (v agendaVinculos) Opcoes(ctx context.Context, tipo string) ([]agenda.Opcao, error) {
	if tipo == "atendimento" {
		return v.ate.OpcoesCompromisso(ctx)
	}
	return v.proc.OpcoesCompromisso(ctx)
}

func (v agendaVinculos) Resumos(ctx context.Context, tipo string, ids []string) (map[string]string, error) {
	out := map[string]string{}
	var m map[string][2]string
	var err error
	if tipo == "atendimento" {
		m, err = v.ate.Resumos(ctx, ids)
	} else {
		m, err = v.proc.ResumosVinculo(ctx, ids)
	}
	for id, r := range m {
		out[id] = r[0]
	}
	return out, err
}

func (v agendaVinculos) Existe(ctx context.Context, tipo, id string) (bool, error) {
	if tipo == "atendimento" {
		return v.ate.Existe(ctx, id)
	}
	return v.proc.Existe(ctx, id)
}

// agendaAudiencias entrega à Agenda as audiências em aberto, o total e a regra de conflito de horário.
type agendaAudiencias struct{ s *audiencias.Service }

func (a agendaAudiencias) EmAberto(ctx context.Context, hoje time.Time) ([]agenda.AudienciaLinha, error) {
	ls, err := a.s.EmAberto(ctx, hoje)
	out := make([]agenda.AudienciaLinha, len(ls))
	for i, l := range ls {
		out[i] = agenda.AudienciaLinha{ID: l.ID, TipoRotulo: l.TipoRotulo, Codigo: l.Codigo, Titulo: l.Titulo, Data: l.Data, Hora: l.Hora,
			Local: l.Local, Processo: l.Processo, Cancelada: l.Cancelada, Busca: l.Busca}
	}
	return out, err
}

func (a agendaAudiencias) Total(ctx context.Context) (int, error) { return a.s.Total(ctx) }

func (a agendaAudiencias) NoHorario(ctx context.Context, data time.Time, hora string) ([]agenda.Conflito, error) {
	cs, err := a.s.NoHorario(ctx, data, hora)
	out := make([]agenda.Conflito, len(cs))
	for i, c := range cs {
		out[i] = agenda.Conflito{Titulo: c.Titulo, Data: c.Data, Hora: c.Hora}
	}
	return out, err
}

// diligenciaConflitos entrega às Diligências a regra de conflito de horário de Audiências.
type diligenciaConflitos struct{ s *audiencias.Service }

func (a diligenciaConflitos) NoHorario(ctx context.Context, data time.Time, hora string) ([]diligencias.Conflito, error) {
	cs, err := a.s.NoHorario(ctx, data, hora)
	out := make([]diligencias.Conflito, len(cs))
	for i, c := range cs {
		out[i] = diligencias.Conflito{Titulo: c.Titulo, Data: c.Data, Hora: c.Hora}
	}
	return out, err
}

// agendaDiligencias entrega à Agenda as diligências em aberto, o total e a criação a partir do "Novo compromisso".
type agendaDiligencias struct{ s *diligencias.Service }

func (a agendaDiligencias) EmAberto(ctx context.Context, hoje time.Time) ([]agenda.DiligenciaLinha, error) {
	ls, err := a.s.EmAberto(ctx, hoje)
	out := make([]agenda.DiligenciaLinha, len(ls))
	for i, l := range ls {
		out[i] = agenda.DiligenciaLinha{ID: l.ID, Codigo: l.Codigo, Titulo: l.Titulo, DataLimite: l.DataLimite, Hora: l.Hora, Local: l.Local,
			Responsavel: l.Responsavel, Status: l.Status, Processo: l.Processo, Busca: l.Busca, Cancelada: l.Cancelada}
	}
	return out, err
}

func (a agendaDiligencias) Total(ctx context.Context) (int, error) { return a.s.Total(ctx) }

func (a agendaDiligencias) Criar(ctx context.Context, ac webx.Actor, n agenda.DiligenciaNova) (string, error) {
	return a.s.Criar(ctx, ac, diligencias.Input{Titulo: n.Titulo, ProcessoID: n.ProcessoID, ClienteID: n.ClienteID, Local: n.Local,
		DataLimite: n.DataLimite, Hora: n.Hora, Responsavel: n.Responsavel})
}

// agendaTarefas entrega à Agenda as tarefas em aberto (por categoria) e o aviso de prazo no mesmo dia.
type agendaTarefas struct{ s *tarefas.Service }

func convTarefas(ls []tarefas.AgendaLinha, err error) ([]agenda.TarefaLinha, error) {
	out := make([]agenda.TarefaLinha, len(ls))
	for i, l := range ls {
		out[i] = agenda.TarefaLinha{Codigo: l.Codigo, Titulo: l.Titulo, CriadoPor: l.CriadoPor, Prazo: l.Prazo, PrazoFatal: l.PrazoFatal,
			Clientes: l.Clientes, StatusRotulo: l.StatusRotulo, StatusClasse: l.StatusClasse, FatalAConferir: l.FatalAConferir, Busca: l.Busca}
	}
	return out, err
}

func (a agendaTarefas) EmAberto(ctx context.Context, hoje time.Time) ([]agenda.TarefaLinha, error) {
	return convTarefas(a.s.AgendaEmAberto(ctx, hoje))
}

func (a agendaTarefas) PorSubtitulo(ctx context.Context, hoje time.Time, normalizados []string) ([]agenda.TarefaLinha, error) {
	return convTarefas(a.s.AgendaPorSubtitulo(ctx, hoje, normalizados))
}

func (a agendaTarefas) TitulosComPrazoEm(ctx context.Context, data time.Time) ([]string, error) {
	return a.s.TitulosComPrazoEm(ctx, data)
}

// Clientes (de vinculos) entrega às Tarefas os clientes do atendimento ou do processo, com o código PES.
func (v vinculos) Clientes(ctx context.Context, tipo string, ids []string) (map[string][]tarefas.Ref, error) {
	out := map[string][]tarefas.Ref{}
	if len(ids) == 0 {
		return out, nil
	}
	quer := map[string]bool{}
	for _, id := range ids {
		quer[id] = true
	}
	if tipo == "atendimento" {
		l, err := v.ate.Listar(ctx, "")
		for _, a := range l {
			if quer[a.ID] {
				out[a.ID] = a.Clientes
			}
		}
		return out, err
	}
	ps, err := v.proc.Listar(ctx)
	if err != nil {
		return nil, err
	}
	var pessoaIDs []string
	for _, p := range ps {
		if quer[p.ID] {
			for _, pt := range p.Partes {
				if pt.Cliente {
					pessoaIDs = append(pessoaIDs, pt.PessoaID)
				}
			}
		}
	}
	refs, err := v.pes.Refs(ctx, pessoaIDs)
	if err != nil {
		return nil, err
	}
	porID := map[string]tarefas.Ref{}
	for _, r := range refs {
		porID[r.ID] = r
	}
	for _, p := range ps {
		if !quer[p.ID] {
			continue
		}
		for _, pt := range p.Partes {
			if r, ok := porID[pt.PessoaID]; ok && pt.Cliente {
				out[p.ID] = append(out[p.ID], r)
			}
		}
	}
	return out, nil
}
