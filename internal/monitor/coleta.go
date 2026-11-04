package monitor

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/monitor/cnj"
	"painelteodoro/internal/monitor/datajud"
	"painelteodoro/internal/platform/coletas"
	"painelteodoro/internal/platform/webx"
)

const (
	lockColeta  = 7003 // pg_try_advisory_lock: uma rodada por vez, mesmo com mais de um processo do painel
	atorSistema = "sistema"
	origemDJ    = "DataJud"
)

// ErrOcupada: já existe uma rodada em andamento.
var ErrOcupada = webx.Invalidf("Já existe uma coleta em andamento. Aguarde terminar.")

func estadoColeta(modulo, origem, status, msg string, tentativa *time.Time) coletas.Estado {
	return coletas.Estado{Modulo: modulo, Origem: origem, Status: status, Mensagem: msg, UltimaTentativa: tentativa}
}

// ColetarAgora roda uma rodada a pedido do administrador (a rota exige a permissão) e audita quem pediu.
func (s *Service) ColetarAgora(ctx context.Context, ac webx.Actor) (Resumo, error) {
	_ = audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ac.UserID, Action: "monitor.coleta_pedida", TargetType: "coleta", TargetID: Modulo,
		Result: audit.OK, RequestID: ac.RequestID})
	return s.rodada(ctx, ac.UserID, ac.RequestID)
}

// Rodada é a coleta agendada (auditada como "sistema").
func (s *Service) Rodada(ctx context.Context) (Resumo, error) { return s.rodada(ctx, atorSistema, "") }

func (s *Service) bloqueado(ctx context.Context, msg string) (Resumo, error) {
	agora := s.now()
	for _, m := range []string{Modulo, ModuloSigilosos} {
		if err := s.Coletas.Registrar(ctx, estadoColeta(m, origemDJ, "bloqueado", msg, &agora)); err != nil {
			return Resumo{}, err
		}
	}
	return Resumo{Status: "bloqueado", Mensagem: msg}, nil
}

func (s *Service) rodada(ctx context.Context, ator, reqID string) (Resumo, error) {
	if !s.Cfg.Ativo || s.Cliente == nil { // desligada: ZERO rede
		return s.bloqueado(ctx, MsgDesligada)
	}
	if !s.Cliente.TemChave() {
		return s.bloqueado(ctx, MsgSemChave)
	}
	conn, err := s.Pool.Acquire(ctx)
	if err != nil {
		return Resumo{}, err
	}
	defer conn.Release()
	var obteve bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockColeta).Scan(&obteve); err != nil {
		return Resumo{}, err
	}
	if !obteve {
		return Resumo{}, ErrOcupada
	}
	defer func() { _, _ = conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, lockColeta) }()

	procs, err := s.Processos.Monitoraveis(ctx)
	if err != nil {
		return Resumo{}, err
	}
	r := Resumo{Processos: len(procs)}
	for _, p := range procs {
		res := s.consultar(ctx, p, ator, reqID)
		r.Novas, r.Bases = r.Novas+res.novas, r.Bases+res.base
		switch res.status {
		case "sucesso":
			r.Sucesso++
		case "falha":
			r.Falhas++
		case "bloqueado":
			r.SemNumero++
		default:
			r.SemRegistro++
		}
	}
	agora := s.now()
	r.Status = "sucesso"
	switch {
	case r.Processos == 0:
		r.Status, r.Mensagem = "bloqueado", "Nenhum processo cadastrado para monitorar."
	case r.Sucesso == 0 && r.Falhas > 0:
		r.Status = "falha"
	case r.Falhas > 0 || r.SemRegistro > 0 || r.SemNumero > 0:
		r.Status = "parcial"
	}
	if r.Mensagem == "" {
		r.Mensagem = fmt.Sprintf("Processos: %d. Consultados com sucesso: %d. Sem registro ou parciais: %d. Sem número CNJ válido: %d. Falhas: %d. Movimentações novas: %d. Linhas de base registradas: %d.",
			r.Processos, r.Sucesso, r.SemRegistro, r.SemNumero, r.Falhas, r.Novas, r.Bases)
	}
	if err := s.Coletas.Registrar(ctx, estadoColeta(Modulo, origemDJ, r.Status, r.Mensagem, &agora)); err != nil {
		return r, err
	}
	s.sigilosos(ctx, ator, reqID)
	res := audit.OK
	if r.Status != "sucesso" {
		res = audit.Falha
	}
	_ = audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ator, Action: "monitor.coleta", TargetType: "coleta", TargetID: Modulo, Result: res,
		RequestID: reqID, Meta: map[string]string{"motivo": r.Status + " novas=" + fmt.Sprint(r.Novas)}})
	return r, nil
}

type resProc struct {
	status      string
	novas, base int
}

func (s *Service) estadoProcesso(ctx context.Context, id, alias, status, msg string) error {
	return upsertEstado(ctx, s.Pool.Exec, s.now(), id, alias, status, msg)
}

func (s *Service) consultar(ctx context.Context, p ProcessoRef, ator, reqID string) resProc {
	if !cnj.Valido(p.Numero) {
		_ = s.estadoProcesso(ctx, p.ID, "", "bloqueado", MsgSemNumero)
		return resProc{status: "bloqueado"}
	}
	alias, err := cnj.AliasDataJud(p.Numero)
	if err != nil {
		_ = s.estadoProcesso(ctx, p.ID, "", "bloqueado", "Tribunal sem consulta pública no DataJud: "+err.Error()+".")
		return resProc{status: "bloqueado"}
	}
	res, err := s.Cliente.Consultar(ctx, alias, p.Numero)
	if err != nil {
		_ = s.estadoProcesso(ctx, p.ID, alias, "falha", "Falha na consulta ao DataJud: "+strings.TrimPrefix(err.Error(), "datajud: ")+".")
		return resProc{status: "falha"}
	}
	if res.SemRegistro {
		_ = s.estadoProcesso(ctx, p.ID, alias, "parcial", MsgSemRegistro)
		return resProc{status: "parcial"}
	}
	var movs []datajud.Movimento
	sigilo := false
	for _, pr := range res.Processos {
		sigilo = sigilo || pr.Sigilo
		movs = append(movs, pr.Movimentos...)
	}
	if sigilo && len(movs) == 0 {
		_ = s.estadoProcesso(ctx, p.ID, alias, "parcial", MsgSigilo)
		return resProc{status: "parcial"}
	}
	sort.SliceStable(movs, func(i, j int) bool { return movs[i].DataHora.Before(movs[j].DataHora) })
	return s.gravar(ctx, p, alias, tribunalDe(res, alias), movs, ator, reqID)
}

func tribunalDe(r datajud.Resultado, alias string) string {
	for _, p := range r.Processos {
		if p.Tribunal != "" {
			return p.Tribunal
		}
	}
	return strings.ToUpper(alias)
}

// gravar separa base de novidade. Primeira leitura: tudo vira base (sem linha nem tarefa). Depois: cada movimento novo é um item
// atômico (linha de Intimações + tarefas + movimento visto, na mesma transação).
func (s *Service) gravar(ctx context.Context, p ProcessoRef, alias, tribunal string, movs []datajud.Movimento, ator, reqID string) resProc {
	var baseEm *time.Time
	_ = s.Pool.QueryRow(ctx, `SELECT base_registrada_em FROM monitor_processos WHERE processo_id=$1::uuid`, p.ID).Scan(&baseEm)
	if baseEm == nil {
		return s.registrarBase(ctx, p, alias, movs)
	}
	vistas, err := s.chavesVistas(ctx, p.ID)
	if err != nil {
		_ = s.estadoProcesso(ctx, p.ID, alias, "falha", "Falha ao ler os movimentos já vistos.")
		return resProc{status: "falha"}
	}
	novas, falhas := 0, 0
	for _, m := range movs {
		k := datajud.Chave(m)
		if vistas[k] {
			continue
		}
		vistas[k] = true
		if err := s.registrarNovo(ctx, p, alias, tribunal, m, ator, reqID); err != nil {
			falhas++
			break // a próxima rodada refaz a partir daqui: o movimento só é "visto" na transação que o registra
		}
		novas++
	}
	if falhas > 0 {
		_ = s.estadoProcesso(ctx, p.ID, alias, "falha", fmt.Sprintf("Movimentação não registrada (falha ao gravar); a próxima rodada tenta de novo. Registradas nesta rodada: %d.", novas))
		return resProc{status: "falha", novas: novas}
	}
	_ = s.estadoProcesso(ctx, p.ID, alias, "sucesso", fmt.Sprintf("Consulta concluída. Movimentações novas: %d.", novas))
	return resProc{status: "sucesso", novas: novas}
}

func (s *Service) chavesVistas(ctx context.Context, id string) (map[string]bool, error) {
	rows, err := s.Pool.Query(ctx, `SELECT chave FROM processo_movimentos WHERE processo_id=$1::uuid AND fonte='DataJud'`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out[k] = true
	}
	return out, rows.Err()
}

// registrarBase grava os movimentos existentes como base (não aparecem como novos nem geram tarefa).
func (s *Service) registrarBase(ctx context.Context, p ProcessoRef, alias string, movs []datajud.Movimento) resProc {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return resProc{status: "falha"}
	}
	defer func() { _ = tx.Rollback(ctx) }()
	n := 0
	for _, m := range movs {
		tag, err := tx.Exec(ctx, insereMovimento, p.ID, origemDJ, m.Grau, m.OrgaoJulgador, datajud.Chave(m), dataCivil(m), nullTempo(m), m.Codigo, m.Nome,
			m.Complemento, complementosJSON(m), true, nil)
		if err != nil {
			_ = s.estadoProcesso(ctx, p.ID, alias, "falha", "Falha ao registrar a linha de base.")
			return resProc{status: "falha"}
		}
		n += int(tag.RowsAffected())
	}
	agora := s.now()
	if err = upsertEstado(ctx, tx.Exec, agora, p.ID, alias, "sucesso", fmt.Sprintf("Linha de base registrada em %s (%d movimentações existentes; não geram alerta).", agora.In(webx.SP()).Format("02/01/2006"), n)); err == nil {
		_, err = tx.Exec(ctx, `UPDATE monitor_processos SET base_registrada_em=$2 WHERE processo_id=$1::uuid`, p.ID, agora)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		_ = s.estadoProcesso(ctx, p.ID, alias, "falha", "Falha ao registrar a linha de base.")
		return resProc{status: "falha"}
	}
	return resProc{status: "sucesso", base: n}
}

func (s *Service) registrarNovo(ctx context.Context, p ProcessoRef, alias, tribunal string, m datajud.Movimento, ator, reqID string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	n := novaDoMovimento(p, tribunal, m, ator, reqID)
	id, _, err := s.Registro.RegistrarNaTx(ctx, tx, n)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, insereMovimento, p.ID, origemDJ, m.Grau, m.OrgaoJulgador, datajud.Chave(m), dataCivil(m), nullTempo(m), m.Codigo, m.Nome,
		m.Complemento, complementosJSON(m), false, nullable(id)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
