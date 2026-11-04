package agenda

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/webx"
)

// Service concentra as regras dos compromissos e monta a página. Só toca na própria tabela; os outros domínios
// entram por interfaces (ligadas em internal/app).
type Service struct {
	Pool        *pgxpool.Pool
	Pessoas     PessoaLookup
	Vinculos    VinculoLookup
	Audiencias  AudienciaLookup
	Diligencias DiligenciaLookup
	Tarefas     TarefaLookup
	Intimacoes  IntimacaoLookup
	Now         func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) record(ctx context.Context, tx pgx.Tx, ac webx.Actor, action, id, tipo string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: action, TargetType: "compromisso", TargetID: id,
		Result: audit.OK, RequestID: ac.RequestID, Meta: map[string]string{"tipo": tipo}})
}

// checarRefs confere que cliente, atendimento, processo e "criado por" existem (vínculo inexistente é recusado).
func (s *Service) checarRefs(ctx context.Context, n normalized) error {
	var pessoas []string
	for _, id := range []string{n.cliente, n.criadoPor} {
		if id != "" {
			if !webx.UUID(id) {
				return webx.Invalidf("Pessoa não encontrada.")
			}
			pessoas = append(pessoas, id)
		}
	}
	if len(pessoas) > 0 {
		ok, err := s.Pessoas.Exist(ctx, pessoas)
		if err != nil {
			return err
		}
		for _, id := range pessoas {
			if !ok[id] {
				return webx.Invalidf("Pessoa não encontrada.")
			}
		}
	}
	for tipo, id := range map[string]string{"atendimento": n.atend, "processo": n.proc} {
		if id == "" {
			continue
		}
		existe := webx.UUID(id)
		if existe {
			var err error
			if existe, err = s.Vinculos.Existe(ctx, tipo, id); err != nil {
				return err
			}
		}
		if !existe {
			return webx.Invalidf(map[string]string{"atendimento": "Atendimento não encontrado.", "processo": "Processo vinculado não encontrado."}[tipo])
		}
	}
	return nil
}

// bloqueio é a regra de conflito de horário do B: compromisso (flexível) é recusado quando cai no mesmo dia e na
// mesma hora exata de uma audiência (item crítico). A audiência nunca é bloqueada.
func (s *Service) bloqueio(ctx context.Context, data time.Time, hora string) error {
	cs, err := s.Audiencias.NoHorario(ctx, data, hora)
	if err != nil || len(cs) == 0 {
		return err
	}
	c := cs[0]
	return webx.Invalidf(fmt.Sprintf("Bloqueado: conflito de horário com Audiência \"%s\" em %s às %s.", c.Titulo, c.Data, c.Hora))
}

// aviso é o aviso não bloqueante do B: há tarefa com prazo de entrega no mesmo dia.
func (s *Service) aviso(ctx context.Context, data time.Time) string {
	ts, err := s.Tarefas.TitulosComPrazoEm(ctx, data)
	if err != nil || len(ts) == 0 {
		return ""
	}
	return fmt.Sprintf("Atenção: há uma tarefa com prazo no mesmo dia (\"%s\").", ts[0])
}

// CriarCompromisso cadastra Reunião, Ligação ou Aula e devolve o aviso do B (vazio se não houver). Diligência não passa
// por aqui: tem ficha própria (CriarDiligencia).
func (s *Service) CriarCompromisso(ctx context.Context, ac webx.Actor, in Input) (aviso string, err error) {
	n, err := in.normalize()
	if err != nil {
		return "", err
	}
	if n.tipo == TipoDiligencia {
		return "", webx.Invalidf("Diligência tem ficha própria — use Novo compromisso.")
	}
	if err = s.checarRefs(ctx, n); err != nil {
		return "", err
	}
	if err = s.bloqueio(ctx, n.data, n.hora); err != nil {
		return "", err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO agenda_compromissos (tipo, descricao, data, hora, local, cliente_pessoa_id, atendimento_id,
		processo_id, criado_por_pessoa_id, created_by, updated_by)
		VALUES ($1,$2,$3,$4::time,$5,$6::uuid,$7::uuid,$8::uuid,$9::uuid,$10,$10) RETURNING id::text`,
		n.tipo, n.descricao, n.data, n.hora, n.local, webx.Nullable(n.cliente), webx.Nullable(n.atend), webx.Nullable(n.proc),
		webx.Nullable(n.criadoPor), ac.UserID).Scan(&id)
	if err == nil {
		err = s.record(ctx, tx, ac, "compromisso.criado", id, n.tipo)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return "", err
	}
	return s.aviso(ctx, n.data), nil
}

// AtualizarCompromisso altera com concorrência otimista (version).
func (s *Service) AtualizarCompromisso(ctx context.Context, ac webx.Actor, id string, version int, in Input) error {
	atual, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if atual.Cancelado {
		return webx.Invalidf("Compromisso cancelado não pode ser editado.")
	}
	n, err := in.normalize()
	if err != nil {
		return err
	}
	if n.tipo == TipoDiligencia {
		return webx.Invalidf("Diligência tem ficha própria — use Novo compromisso.")
	}
	if err = s.checarRefs(ctx, n); err != nil {
		return err
	}
	if err = s.bloqueio(ctx, n.data, n.hora); err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE agenda_compromissos SET tipo=$3, descricao=$4, data=$5, hora=$6::time, local=$7,
		cliente_pessoa_id=$8::uuid, atendimento_id=$9::uuid, processo_id=$10::uuid, criado_por_pessoa_id=$11::uuid,
		updated_at=now(), updated_by=$12, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL AND NOT cancelado`, atual.ID, version, n.tipo, n.descricao, n.data, n.hora, n.local,
		webx.Nullable(n.cliente), webx.Nullable(n.atend), webx.Nullable(n.proc), webx.Nullable(n.criadoPor), ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return webx.ErrConflict
	}
	if err = s.record(ctx, tx, ac, "compromisso.atualizado", atual.ID, n.tipo); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CriarDiligencia cria a diligência a partir do "Novo compromisso" com tipo Diligência (Descrição vira Título, Data vira
// Data-limite). Devolve o código DIL. 0000001. Atendimento e "Criado por" não valem para diligência, como no B.
func (s *Service) CriarDiligencia(ctx context.Context, ac webx.Actor, in Input) (string, error) {
	n, err := in.normalize()
	if err != nil {
		return "", err
	}
	n.atend, n.criadoPor = "", ""
	if err = s.checarRefs(ctx, n); err != nil {
		return "", err
	}
	return s.Diligencias.Criar(ctx, ac, DiligenciaNova{Titulo: n.descricao, ProcessoID: n.proc, ClienteID: n.cliente, Local: n.local,
		DataLimite: n.data.Format("2006-01-02"), Hora: n.hora, Responsavel: in.Responsavel})
}

// CancelarCompromisso preserva o compromisso com o motivo, quem cancelou e quando (nada se apaga). Cancelado, ele
// continua listado, com selo, e não pode mais ser editado.
func (s *Service) CancelarCompromisso(ctx context.Context, ac webx.Actor, id string, version int, motivo string) error {
	if motivo = strings.TrimSpace(motivo); motivo == "" {
		return webx.Invalidf("Informe o motivo do cancelamento.")
	}
	if utf8.RuneCountInString(motivo) > 1000 {
		return webx.Invalidf("Texto longo demais.")
	}
	atual, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if atual.Cancelado {
		return webx.Invalidf("Este compromisso já está cancelado.")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE agenda_compromissos SET cancelado=true, cancelamento_motivo=$3, cancelado_em=now(), cancelado_por=$4::uuid,
		updated_at=now(), updated_by=$4::uuid, version=version+1 WHERE id=$1::uuid AND version=$2 AND excluido_em IS NULL AND NOT cancelado`,
		atual.ID, version, motivo, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return webx.ErrConflict
	}
	if err = s.record(ctx, tx, ac, "compromisso.cancelado", atual.ID, atual.Tipo); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
