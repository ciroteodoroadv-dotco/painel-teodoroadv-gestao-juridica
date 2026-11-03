package app

import (
	"context"
	"io"
	"time"

	"painelteodoro/internal/atendimentos"
	"painelteodoro/internal/auth"
	"painelteodoro/internal/documentos"
	"painelteodoro/internal/platform/webx"
	"painelteodoro/internal/prazos"
	"painelteodoro/internal/processos"
	"painelteodoro/internal/tarefas"
)

// Adaptadores: o único lugar que liga as interfaces de Tarefas aos serviços dos outros domínios.

type prazoSync struct{ p *prazos.Service }

func (a prazoSync) Sincronizar(ctx context.Context, ac webx.Actor, tarefaID, processoID, descricao string, data *time.Time, confirmar bool) (string, error) {
	return a.p.SincronizarDaTarefa(ctx, ac, tarefaID, processoID, descricao, data, confirmar)
}
func (a prazoSync) Situacoes(ctx context.Context, ids []string) (map[string]string, error) {
	return a.p.SituacoesDeTarefas(ctx, ids)
}
func (a prazoSync) Encerrar(ctx context.Context, ac webx.Actor, tarefaID, motivo string) error {
	return a.p.EncerrarDaTarefa(ctx, ac, tarefaID, motivo)
}
func (a prazoSync) Confirmar(ctx context.Context, ac webx.Actor, tarefaID string) error {
	return a.p.ConfirmarDaTarefa(ctx, ac, tarefaID)
}

type vinculos struct {
	ate  *atendimentos.Service
	proc *processos.Service
}

func (v vinculos) Resumos(ctx context.Context, tipo string, ids []string) (map[string][2]string, error) {
	if tipo == "atendimento" {
		return v.ate.Resumos(ctx, ids)
	}
	return v.proc.ResumosVinculo(ctx, ids)
}
func (v vinculos) Opcoes(ctx context.Context, tipo string) ([]tarefas.Opcao, error) {
	if tipo == "atendimento" {
		return v.ate.Opcoes(ctx)
	}
	return v.proc.Opcoes(ctx)
}

// docSink guarda o PDF da tarefa no domínio Documentos (cifrado, tipo conferido pelo conteúdo, auditado).
type docSink struct{ d *documentos.Service }

func (a docSink) Anexar(ctx context.Context, ac webx.Actor, tarefaID, origemTipo, origemID, nome string, c io.Reader) error {
	b, err := io.ReadAll(io.LimitReader(c, documentos.MaxArquivo+1))
	if err != nil {
		return err
	}
	u := documentos.Upload{Nome: nome, Data: b, TarefaID: tarefaID}
	if origemTipo == "atendimento" {
		u.AtendimentoID = origemID
	} else {
		u.ProcessoID = origemID
	}
	_, err = a.d.Enviar(ctx, ac, u)
	return err
}

// reauth confere a senha do usuário logado (Argon2id) — não existe "senha de administrador".
type reauth struct{ a *auth.Auth }

func (r reauth) Confirmar(ctx context.Context, ac webx.Actor, senha string) error {
	return r.a.Reautenticar(ctx, ac.UserID, senha)
}
