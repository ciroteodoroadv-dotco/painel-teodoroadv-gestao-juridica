package intimacoes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/intimacoes/djen"
	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/coletas"
	"painelteodoro/internal/platform/webx"
)

// MsgDesligada é o texto do estado "bloqueado" quando a integração está desligada.
const MsgDesligada = "Coleta desligada neste ambiente."

const (
	lockColeta  = 7002 // pg_try_advisory_lock: uma coleta por vez, mesmo com mais de um processo do painel
	diasPadrao  = 7
	OrigemDJEN  = "DJEN"
	atorSistema = "sistema"
)

func coletasEstado(status, msg string, tentativa *time.Time) coletas.Estado {
	return coletas.Estado{Modulo: Modulo, Origem: OrigemDJEN, Status: status, Mensagem: msg, UltimaTentativa: tentativa}
}

var errParcial = webxErro("O DJEN devolveu o teto de resultados; coleta não confirmada.")

// ErrOcupada: já existe uma coleta em andamento.
var ErrOcupada = webx.Invalidf("Já existe uma coleta em andamento. Aguarde terminar.")

// ColetarAgora roda uma rodada a pedido do administrador (a rota exige a permissão) e audita quem pediu.
func (s *Service) ColetarAgora(ctx context.Context, ac webx.Actor) (Resumo, error) {
	_ = audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ac.UserID, Action: "intimacao.coleta_pedida", TargetType: "coleta", TargetID: Modulo,
		Result: audit.OK, RequestID: ac.RequestID})
	return s.rodada(ctx, ac.UserID, ac.RequestID)
}

// Rodada é a coleta agendada (auditada como "sistema").
func (s *Service) Rodada(ctx context.Context) (Resumo, error) { return s.rodada(ctx, atorSistema, "") }

func (s *Service) rodada(ctx context.Context, ator, reqID string) (Resumo, error) {
	agora := s.now()
	if !s.Cfg.Ativo || s.Coletor == nil { // desligada: ZERO rede
		if err := s.Coletas.Registrar(ctx, coletasEstado("bloqueado", MsgDesligada, &agora)); err != nil {
			return Resumo{}, err
		}
		return Resumo{Status: "bloqueado", Mensagem: MsgDesligada}, nil
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

	oabs, err := s.oabs(ctx, true)
	if err != nil {
		return Resumo{}, err
	}
	r := Resumo{OABs: len(oabs)}
	if len(oabs) == 0 {
		r.Status, r.Mensagem = "bloqueado", "Nenhuma OAB ativa cadastrada para a coleta."
		return r, s.Coletas.Registrar(ctx, coletasEstado(r.Status, r.Mensagem, &agora))
	}
	var desdeMin *time.Time
	sucessos := 0
	for _, o := range oabs {
		if o.AutorizacaoBase == "" || o.AutorizacaoEm.After(webx.Hoje(agora)) {
			r.Recusadas++
			_ = audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ator, Action: "intimacao.coleta_recusada", TargetType: "oab", TargetID: o.ID,
				Result: audit.Negado, RequestID: reqID, Meta: map[string]string{"motivo": "autorizacao"}})
			continue
		}
		desde, ate := djen.JanelaDesde(agora, o.UltimoSucesso, diasPadrao)
		novas, rep, falhas, parcial, err := s.coletaOAB(ctx, o, desde, ate, ator, reqID)
		r.Novas, r.Repetidas, r.FalhasItem = r.Novas+novas, r.Repetidas+rep, r.FalhasItem+falhas
		switch {
		case err != nil:
			r.FalhasOAB++
		case falhas > 0 || parcial:
			r.FalhasOAB++ // a janela não avança: a próxima rodada refaz o que faltou
		default:
			sucessos++
			if desdeMin == nil || desde.Before(*desdeMin) {
				d := desde
				desdeMin = &d
			}
			_, _ = s.Pool.Exec(ctx, `UPDATE oab_monitoradas SET ultimo_sucesso=$2 WHERE id=$1::uuid`, o.ID, agora)
		}
	}
	r.Status = "sucesso"
	switch {
	case sucessos == 0:
		r.Status = "falha"
	case r.FalhasOAB > 0 || r.Recusadas > 0:
		r.Status = "parcial"
	}
	r.Mensagem = fmt.Sprintf("OABs: %d. Comunicações novas: %d. Já registradas: %d. OABs com falha: %d. OABs recusadas: %d. Itens não registrados: %d.",
		r.OABs, r.Novas, r.Repetidas, r.FalhasOAB, r.Recusadas, r.FalhasItem)
	est := coletasEstado(r.Status, r.Mensagem, &agora)
	est.JanelaDesde = desdeMin
	if err := s.Coletas.Registrar(ctx, est); err != nil {
		return r, err
	}
	res := audit.OK
	if r.Status != "sucesso" {
		res = audit.Falha
	}
	_ = audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ator, Action: "intimacao.coleta", TargetType: "coleta", TargetID: Modulo, Result: res,
		RequestID: reqID, Meta: map[string]string{"motivo": strings.ReplaceAll(r.Status+" novas="+fmt.Sprint(r.Novas), "\n", " ")}})
	return r, nil
}

// coletaOAB coleta uma OAB. Cada comunicação nova é gravada com a sua tarefa de análise numa transação só.
func (s *Service) coletaOAB(ctx context.Context, o OAB, desde, ate time.Time, ator, reqID string) (novas, repetidas, falhas int, parcial bool, err error) {
	res, err := s.Coletor.Coletar(ctx, o.Numero, o.UF, desde, ate)
	if err != nil {
		return 0, 0, 0, false, err
	}
	for _, c := range res.Itens {
		ok, repetida, err := s.registra(ctx, o, c, ator, reqID)
		switch {
		case err != nil:
			falhas++
		case repetida:
			repetidas++
		case ok:
			novas++
		}
	}
	return novas, repetidas, falhas, res.Parcial, nil
}

// registra grava a intimação nova, resolve o processo e abre as tarefas de análise — tudo ou nada.
func (s *Service) registra(ctx context.Context, o OAB, c djen.Comunicacao, ator, reqID string) (novo, repetida bool, err error) {
	hash := djen.Hash(c)
	var um int
	if s.Pool.QueryRow(ctx, `SELECT 1 FROM intimacoes WHERE hash=$1`, hash).Scan(&um) == nil {
		return false, true, nil
	}
	numero, procID := c.Processo, ""
	if n, e := br.NormalizeCNJ(c.Processo); e == nil {
		numero = n
		if s.Processos != nil {
			if id, achou, e := s.Processos.IDPorNumero(ctx, n); e != nil {
				return false, false, e
			} else if achou {
				procID = id
			}
		}
	}
	return s.registraNova(ctx, Nova{Hash: hash, Origem: OrigemDJEN, OabID: o.ID, Tribunal: c.Tribunal, Orgao: c.Orgao, ProcessoNumero: numero,
		ProcessoID: procID, Tipo: c.Tipo, Data: &c.DataPublicacao, Texto: c.Texto, Resumo: djen.NormalizarResumo(c.Texto), Link: c.Link,
		TituloAnalise: tituloAnalise(c.Tipo, numero), DescricaoAnalise: descricaoAnalise(c.Link),
		LinhaConsolidada: "DJEN: " + strings.TrimSpace(c.Tipo+" "+c.Orgao), CriadoPor: o.CriadoPor, Ator: ator, RequestID: reqID})
}

// registraNova abre a transação própria, registra e confirma.
func (s *Service) registraNova(ctx context.Context, n Nova) (novo, repetida bool, err error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return false, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, rep, err := s.RegistrarNaTx(ctx, tx, n)
	if err != nil || rep {
		return false, rep, err
	}
	return true, false, tx.Commit(ctx)
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// tituloAnalise: "Analisar intimação — {tipo} — {processo}" (sem os pedaços vazios).
func tituloAnalise(tipo, processo string) string {
	t := "Analisar intimação"
	for _, p := range []string{tipo, processo} {
		if p = strings.TrimSpace(p); p != "" {
			t += " — " + p
		}
	}
	if len([]rune(t)) > 300 {
		t = string([]rune(t)[:300])
	}
	return t
}

// descricaoAnalise: o sistema não calculou prazo; diz isso na própria tarefa.
func descricaoAnalise(link string) string {
	d := "Prazo a apurar pelo advogado. O sistema não calculou prazo."
	if link != "" {
		d += "\nLink: " + link
	}
	return d
}
