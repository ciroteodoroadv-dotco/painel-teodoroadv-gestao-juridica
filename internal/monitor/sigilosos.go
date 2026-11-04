package monitor

import (
	"context"
	"fmt"
	"time"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/monitor/cnj"
	"painelteodoro/internal/platform/webx"
)

// janelaSigilosos: quantos dias para trás o DJEN é consultado, por processo. A deduplicação pelo hash evita repetir.
const janelaSigilosos = 30

// Sigiloso é um processo em segredo de justiça monitorado pelo DJEN.
type Sigiloso struct {
	ID, ProcessoID, OabID string
	Ativo                 bool
}

// CadastrarSigiloso liga um processo cadastrado a uma OAB ativa, com a base da autorização. A consulta usa SEMPRE o número.
func (s *Service) CadastrarSigiloso(ctx context.Context, ac webx.Actor, processoID, oabID string) error {
	if !webx.UUID(processoID) || !webx.UUID(oabID) {
		return webx.Invalidf("Escolha o processo e a OAB.")
	}
	p, ok, err := s.Processos.Ref(ctx, processoID)
	if err != nil {
		return err
	}
	if !ok || !cnj.Valido(p.Numero) {
		return webx.Invalidf("O processo precisa estar cadastrado e ter número CNJ válido.")
	}
	o, err := s.OABs.Obter(ctx, oabID)
	if err != nil {
		return webx.Invalidf("OAB não encontrada.")
	}
	if !o.Ativa || o.AutorizacaoBase == "" || o.AutorizacaoEm.After(webx.Hoje(s.now())) {
		return webx.Invalidf("A OAB precisa estar ativa e ter a base da autorização registrada.")
	}
	var id string
	err = s.Pool.QueryRow(ctx, `INSERT INTO monitor_sigilosos (processo_id, oab_id, created_by, updated_by) VALUES ($1::uuid,$2::uuid,$3::uuid,$3::uuid)
		ON CONFLICT (processo_id) DO UPDATE SET oab_id=EXCLUDED.oab_id, ativo=true, updated_at=now(), updated_by=EXCLUDED.updated_by, version=monitor_sigilosos.version+1
		RETURNING id::text`, processoID, oabID, ac.UserID).Scan(&id)
	if err != nil {
		return err
	}
	return audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ac.UserID, Action: "monitor.sigiloso_cadastrado", TargetType: "monitor_sigiloso",
		TargetID: id, Result: audit.OK, RequestID: ac.RequestID})
}

// DesativarSigiloso para de consultar o processo pelo DJEN. Nada se apaga: a linha fica, inativa.
func (s *Service) DesativarSigiloso(ctx context.Context, ac webx.Actor, id string) error {
	if !webx.UUID(id) {
		return webx.ErrNotFound
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE monitor_sigilosos SET ativo=false, updated_at=now(), updated_by=$2::uuid, version=version+1 WHERE id=$1::uuid AND ativo`, id, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return webx.ErrNotFound
	}
	return audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ac.UserID, Action: "monitor.sigiloso_desativado", TargetType: "monitor_sigiloso",
		TargetID: id, Result: audit.OK, RequestID: ac.RequestID})
}

// ListarSigilosos devolve todos (ativos ou não).
func (s *Service) ListarSigilosos(ctx context.Context) ([]Sigiloso, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id::text, processo_id::text, oab_id::text, ativo FROM monitor_sigilosos ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Sigiloso
	for rows.Next() {
		var x Sigiloso
		if err := rows.Scan(&x.ID, &x.ProcessoID, &x.OabID, &x.Ativo); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// sigilosos roda a camada do DJEN por processo sigiloso e registra o módulo "monitor-segredos".
func (s *Service) sigilosos(ctx context.Context, ator, reqID string) {
	agora := s.now()
	reg := func(status, msg string) {
		_ = s.Coletas.Registrar(ctx, estadoColeta(ModuloSigilosos, "DJEN", status, msg, &agora))
	}
	todos, err := s.ListarSigilosos(ctx)
	if err != nil || s.Sigilosos == nil {
		reg("falha", "Não foi possível ler os processos em segredo de justiça.")
		return
	}
	var ativos []Sigiloso
	for _, x := range todos {
		if x.Ativo {
			ativos = append(ativos, x)
		}
	}
	if len(ativos) == 0 {
		reg("nao_sincronizado", "Nenhum processo em segredo de justiça cadastrado.")
		return
	}
	ok, falhas, novas := 0, 0, 0
	for _, x := range ativos {
		n, err := s.coletaSigiloso(ctx, x, ator, reqID)
		novas += n
		if err != nil {
			falhas++
			continue
		}
		ok++
	}
	status := "sucesso"
	switch {
	case ok == 0:
		status = "falha"
	case falhas > 0:
		status = "parcial"
	}
	reg(status, fmt.Sprintf("Processos em segredo de justiça: %d. Consultados com sucesso: %d. Falhas: %d. Comunicações novas: %d.", len(ativos), ok, falhas, novas))
}

func (s *Service) coletaSigiloso(ctx context.Context, x Sigiloso, ator, reqID string) (int, error) {
	p, existe, err := s.Processos.Ref(ctx, x.ProcessoID)
	if err != nil || !existe {
		return 0, fmt.Errorf("processo indisponível")
	}
	o, err := s.OABs.Obter(ctx, x.OabID)
	if err != nil || !o.Ativa || o.AutorizacaoBase == "" {
		return 0, fmt.Errorf("OAB indisponível ou sem autorização")
	}
	numero := cnj.Formatar(p.Numero)
	if !cnj.Valido(numero) {
		return 0, fmt.Errorf("sem número CNJ válido")
	}
	hoje := webx.Hoje(s.now())
	itens, err := s.Sigilosos.ColetarPorNumero(ctx, o.Numero, o.UF, numero, hoje.AddDate(0, 0, -janelaSigilosos), hoje)
	if err != nil {
		return 0, err
	}
	novas := 0
	for _, c := range itens {
		ok, err := s.registrarSigilosa(ctx, p, o, numero, c, ator, reqID)
		if err != nil {
			return novas, err
		}
		if ok {
			novas++
		}
	}
	return novas, nil
}

func (s *Service) registrarSigilosa(ctx context.Context, p ProcessoRef, o OABRef, numero string, c ComunicacaoSigilosa, ator, reqID string) (bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	d := c.Data
	n := NovaIntimacao{Hash: c.Hash, Origem: "DJEN (sigiloso)", OabID: o.ID, Tribunal: c.Tribunal, Orgao: c.Orgao, ProcessoNumero: numero, ProcessoID: p.ID,
		Tipo: c.Tipo, Texto: c.Texto, Resumo: c.Resumo, Link: c.Link, TituloAnalise: "Analisar intimação — " + corta(c.Tipo, 120) + " — " + numero,
		DescricaoAnalise: "Processo em segredo de justiça (DJEN consultado pelo número do processo).\nPrazo a apurar pelo advogado. O sistema não calculou prazo.",
		LinhaConsolidada: "DJEN (sigiloso): " + corta(c.Tipo+" "+c.Orgao, 120), CriadoPor: p.CriadoPor, Ator: ator, RequestID: reqID}
	if !d.IsZero() {
		n.Data = &d
	}
	id, rep, err := s.Registro.RegistrarNaTx(ctx, tx, n)
	if err != nil || rep {
		return false, err
	}
	if _, err = tx.Exec(ctx, insereMovimento, p.ID, "DJEN", "", c.Orgao, c.Hash, nullData(d), nil, "", nomeSigilosa(c), "", "[]", false, id); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func nullData(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func nomeSigilosa(c ComunicacaoSigilosa) string {
	if c.Tipo != "" {
		return c.Tipo
	}
	return "Comunicação do DJEN"
}
