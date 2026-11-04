package processos

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/codigos"
	"painelteodoro/internal/platform/webx"
)

// Service concentra as regras de processos. Só toca nas próprias tabelas.
type Service struct {
	Pool         *pgxpool.Pool
	Pessoas      PessoaLookup
	Atendimentos AtendimentoLookup // opcional: seletor e conferência do "Atendimento de origem"
	Vinculos     []VinculosLookup  // tarefas, prazos… (opcional): quem tem trabalho em aberto ligado ao processo
	Monitor      Monitoramento     // opcional: tarefa de monitoramento criada e baixada pelo ciclo de vida (Plano 6)
}

func (s *Service) record(ctx context.Context, tx pgx.Tx, ac webx.Actor, action, id string, meta map[string]string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: action, TargetType: "processo",
		TargetID: id, Result: audit.OK, RequestID: ac.RequestID, Meta: meta})
}

func (s *Service) whyNoRows(ctx context.Context, id string) error {
	var one int
	err := s.Pool.QueryRow(ctx, `SELECT 1 FROM processos WHERE id=$1 AND excluido_em IS NULL`, id).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return webx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return webx.ErrConflict
}

// tituloDe deriva o título da ficha ("CLIENTE x PARTE") quando ninguém o informou.
func (s *Service) tituloDe(ctx context.Context, n normalized) string {
	if n.titulo != "" {
		return n.titulo
	}
	var ids []string
	for _, p := range n.partes {
		ids = append(ids, p.PessoaID)
	}
	nomes, err := s.Pessoas.Names(ctx, ids)
	if err != nil {
		return ""
	}
	var cli, outra string
	for _, p := range n.partes {
		if p.Cliente && cli == "" {
			cli = nomes[p.PessoaID]
		} else if outra == "" {
			outra = nomes[p.PessoaID]
		}
	}
	if cli == "" {
		cli, outra = outra, ""
	}
	if outra != "" {
		return cli + " x " + outra
	}
	return cli
}

func escreverPartes(ctx context.Context, tx pgx.Tx, id string, n normalized) error {
	for i, p := range n.partes {
		if _, err := tx.Exec(ctx, `INSERT INTO processo_partes (processo_id, pessoa_id, papel, cliente, ordem)
			VALUES ($1,$2,$3,$4,$5)`, id, p.PessoaID, p.Papel, p.Cliente, i); err != nil {
			return err
		}
	}
	for i, r := range n.resp {
		if _, err := tx.Exec(ctx, `INSERT INTO processo_responsaveis (processo_id, pessoa_id, ordem) VALUES ($1,$2,$3)`, id, r, i); err != nil {
			return err
		}
	}
	return nil
}

// valores devolve os parâmetros comuns de INSERT e UPDATE, na ordem: numero, titulo, polo, sistema, instancia,
// classe, materia, comarca, juizo, contingencia, valor, 6 datas, projeto, observacoes, atendimento de origem.
func (n normalized) valores(titulo string) []any {
	return []any{n.numero, titulo, n.polo, n.sistema, n.instancia, n.classe, n.materia, n.comarca, n.juizo,
		n.contingencia, nullableNumeric(n.valor), n.datas[0], n.datas[1], n.datas[2], n.datas[3], n.datas[4], n.datas[5],
		n.projeto, n.obs, webx.Nullable(n.atendimento)}
}

func nullableNumeric(v string) any {
	if v == "" {
		return nil
	}
	return v
}

// Create cadastra o processo (estado "ativo") e devolve o id e o código (PRO. 0000012).
func (s *Service) Create(ctx context.Context, ac webx.Actor, in Input) (id, codigo string, err error) {
	n, err := s.normalize(ctx, in)
	if err != nil {
		return "", "", err
	}
	justificou, err := s.decideDuplicidade(ctx, &n, in, "", "", false, "")
	if err != nil {
		return "", "", err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if codigo, err = codigos.Proximo(ctx, tx, "PRO"); err != nil {
		return "", "", err
	}
	args := append([]any{codigo, "ativo", ac.UserID}, n.valores(s.tituloDe(ctx, n))...)
	args = append(args, n.duplicado, n.dupJust, webx.Nullable(n.dupDe))
	err = tx.QueryRow(ctx, `INSERT INTO processos (codigo, situacao, created_by, updated_by, numero_cnj, titulo, polo, sistema,
		instancia, classe_processual, materia_direito, comarca, juizo, tipo_contingencia, valor_causa, distribuicao,
		citacao, audiencia, sentenca, recursos, fase_execucao, projeto_vinculado, observacoes, atendimento_origem_id,
		duplicado, duplicado_justificativa, duplicado_de)
		VALUES ($1,$2,$3,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26::uuid) RETURNING id`, args...).Scan(&id)
	if webx.IsUniqueViolation(err) {
		return "", "", webx.Invalidf("Já existe um processo cadastrado com este número.")
	}
	if err == nil {
		err = escreverPartes(ctx, tx, id, n)
	}
	if err == nil {
		err = s.record(ctx, tx, ac, "processo.criado", id, map[string]string{"partes": strconv.Itoa(len(n.partes))})
	}
	if err == nil {
		err = s.efeitoMonitoramento(ctx, tx, ac, id, "ativo")
	}
	if err == nil && justificou {
		err = s.record(ctx, tx, ac, "processo.duplicado_justificado", id, nil)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return "", "", err
	}
	return id, codigo, nil
}

// Update altera com concorrência otimista. O estado NÃO muda aqui (use MoverEstado).
func (s *Service) Update(ctx context.Context, ac webx.Actor, id string, version int, in Input) error {
	n, err := s.normalize(ctx, in)
	if err != nil {
		return err
	}
	var atualNum, atualJust string
	var atualDup bool
	if err = s.Pool.QueryRow(ctx, `SELECT numero_cnj, duplicado, duplicado_justificativa FROM processos WHERE id=$1 AND excluido_em IS NULL`, id).
		Scan(&atualNum, &atualDup, &atualJust); err != nil {
		return s.whyNoRows(ctx, id)
	}
	justificou, err := s.decideDuplicidade(ctx, &n, in, id, atualNum, atualDup, atualJust)
	if err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	args := append([]any{id, version, ac.UserID}, n.valores(s.tituloDe(ctx, n))...)
	args = append(args, n.duplicado, n.dupJust, webx.Nullable(n.dupDe))
	tag, err := tx.Exec(ctx, `UPDATE processos SET numero_cnj=$4, titulo=$5, polo=$6, sistema=$7, instancia=$8,
		classe_processual=$9, materia_direito=$10, comarca=$11, juizo=$12, tipo_contingencia=$13, valor_causa=$14,
		distribuicao=$15, citacao=$16, audiencia=$17, sentenca=$18, recursos=$19, fase_execucao=$20,
		projeto_vinculado=$21, observacoes=$22, atendimento_origem_id=$23, duplicado=$24, duplicado_justificativa=$25,
		duplicado_de=$26::uuid, updated_at=now(), updated_by=$3, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL`, args...)
	if webx.IsUniqueViolation(err) {
		return webx.Invalidf("Já existe um processo cadastrado com este número.")
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return s.whyNoRows(ctx, id)
	}
	for _, t := range []string{"processo_partes", "processo_responsaveis"} {
		if _, err = tx.Exec(ctx, `DELETE FROM `+t+` WHERE processo_id=$1`, id); err != nil { // #nosec G202 -- tabela constante
			return err
		}
	}
	if err = escreverPartes(ctx, tx, id, n); err != nil {
		return err
	}
	if err = s.record(ctx, tx, ac, "processo.alterado", id, map[string]string{"partes": strconv.Itoa(len(n.partes))}); err != nil {
		return err
	}
	if justificou {
		if err = s.record(ctx, tx, ac, "processo.duplicado_justificado", id, nil); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// AbertosVinculados soma o trabalho em aberto (tarefas, prazos…) ligado ao processo.
func (s *Service) AbertosVinculados(ctx context.Context, id string) (int, error) {
	total := 0
	for _, v := range s.Vinculos {
		n, err := v.AbertosDoProcesso(ctx, id)
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

// Delete marca como excluído (sem apagar). Com trabalho em aberto ligado, exige confirmacaoExtra.
func (s *Service) Delete(ctx context.Context, ac webx.Actor, id string, version int, confirmacaoExtra bool) error {
	abertos, err := s.AbertosVinculados(ctx, id)
	if err != nil {
		return err
	}
	if abertos > 0 && !confirmacaoExtra {
		return webx.Invalidf("Há " + strconv.Itoa(abertos) + " tarefa(s)/prazo(s) em aberto vinculados. Confirme para excluir mesmo assim.")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE processos SET excluido_em=now(), updated_at=now(), updated_by=$3, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL`, id, version, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return s.whyNoRows(ctx, id)
	}
	if err = s.record(ctx, tx, ac, "processo.excluido", id, map[string]string{"abertos_vinculados": strconv.Itoa(abertos)}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
