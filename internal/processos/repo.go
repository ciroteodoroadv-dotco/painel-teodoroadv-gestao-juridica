package processos

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/pessoas"
	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/webx"
)

const cols = `id, codigo, titulo, numero_cnj, situacao, polo, sistema, instancia, classe_processual, materia_direito,
	comarca, juizo, tipo_contingencia, coalesce(valor_causa::text,''),
	coalesce(to_char(distribuicao,'YYYY-MM-DD'),''), coalesce(to_char(citacao,'YYYY-MM-DD'),''),
	coalesce(to_char(audiencia,'YYYY-MM-DD'),''), coalesce(to_char(sentenca,'YYYY-MM-DD'),''),
	coalesce(to_char(recursos,'YYYY-MM-DD'),''), coalesce(to_char(fase_execucao,'YYYY-MM-DD'),''),
	coalesce(atendimento_origem_id::text,''), projeto_vinculado, observacoes, prox_o_que,
	coalesce(to_char(prox_ate_quando,'YYYY-MM-DD'),''), prox_quem, version, created_at, updated_at`

// limiteLista protege a tela: o acordeão do B mostra tudo, sem paginação.
const limiteLista = 2000

func scan(row pgx.Row) (*Processo, error) {
	var p Processo
	err := row.Scan(&p.ID, &p.Codigo, &p.Titulo, &p.Numero, &p.Estado, &p.Polo, &p.Sistema, &p.Instancia, &p.Classe,
		&p.Materia, &p.Comarca, &p.Juizo, &p.Contingencia, &p.Valor, &p.Distribuicao, &p.Citacao, &p.Audiencia,
		&p.Sentenca, &p.Recursos, &p.FaseExecucao, &p.AtendimentoOrigem, &p.Projeto, &p.Observacoes, &p.ProxOQue,
		&p.ProxAte, &p.ProxQuem, &p.Version, &p.CriadoEm, &p.AtualizadoEm)
	return &p, err
}

// Get devolve o processo com partes e responsáveis (nomes pela interface de pessoas).
func (s *Service) Get(ctx context.Context, id string) (*Processo, error) {
	p, err := scan(s.Pool.QueryRow(ctx, `SELECT `+cols+` FROM processos WHERE id=$1 AND excluido_em IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, webx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.carregarVinculos(ctx, []*Processo{p}); err != nil {
		return nil, err
	}
	return p, nil
}

// Listar devolve todos os processos não excluídos (mais novos primeiro), com as partes.
func (s *Service) Listar(ctx context.Context) ([]*Processo, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+cols+` FROM processos WHERE excluido_em IS NULL
		ORDER BY created_at DESC, id LIMIT $1`, limiteLista)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Processo
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, s.carregarVinculos(ctx, out)
}

// carregarVinculos preenche partes (com nomes) e responsáveis de uma lista de processos, em poucas consultas.
func (s *Service) carregarVinculos(ctx context.Context, ps []*Processo) error {
	if len(ps) == 0 {
		return nil
	}
	ids := make([]string, len(ps))
	por := map[string]*Processo{}
	for i, p := range ps {
		ids[i], por[p.ID] = p.ID, p
	}
	var pessoaIDs []string
	pr, err := s.Pool.Query(ctx, `SELECT processo_id, pessoa_id, papel, cliente FROM processo_partes
		WHERE processo_id = ANY($1::uuid[]) ORDER BY processo_id, ordem, papel`, ids)
	if err != nil {
		return err
	}
	for pr.Next() {
		var proc string
		var pt Parte
		if err := pr.Scan(&proc, &pt.PessoaID, &pt.Papel, &pt.Cliente); err != nil {
			pr.Close()
			return err
		}
		por[proc].Partes = append(por[proc].Partes, pt)
		pessoaIDs = append(pessoaIDs, pt.PessoaID)
	}
	pr.Close()
	if err := pr.Err(); err != nil {
		return err
	}
	rr, err := s.Pool.Query(ctx, `SELECT processo_id, pessoa_id FROM processo_responsaveis
		WHERE processo_id = ANY($1::uuid[]) ORDER BY processo_id, ordem`, ids)
	if err != nil {
		return err
	}
	for rr.Next() {
		var proc, pes string
		if err := rr.Scan(&proc, &pes); err != nil {
			rr.Close()
			return err
		}
		por[proc].Responsaveis = append(por[proc].Responsaveis, pes)
	}
	rr.Close()
	if err := rr.Err(); err != nil {
		return err
	}
	nomes, err := s.Pessoas.Names(ctx, pessoaIDs)
	if err != nil {
		return err
	}
	for _, p := range ps {
		for i := range p.Partes {
			p.Partes[i].Nome = nomes[p.Partes[i].PessoaID]
		}
	}
	return nil
}

// PapeisPorPessoa atende ao domínio de pessoas (pessoas.PapeisLookup): em que processos e papéis cada uma aparece.
func (s *Service) PapeisPorPessoa(ctx context.Context, ids []string) (map[string][]pessoas.PapelEmProcesso, error) {
	out := map[string][]pessoas.PapelEmProcesso{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT x.pessoa_id, p.id, p.numero_cnj, p.titulo, x.papel, x.cliente, x.responsavel FROM (
			SELECT processo_id, pessoa_id, papel, cliente, false AS responsavel, 0 AS grupo, ordem FROM processo_partes
			UNION ALL
			SELECT processo_id, pessoa_id, $2, false, true, 1, ordem FROM processo_responsaveis
		) x JOIN processos p ON p.id = x.processo_id
		WHERE x.pessoa_id = ANY($1::uuid[]) AND p.excluido_em IS NULL
		ORDER BY p.created_at, p.id, x.grupo, x.ordem`, ids, pessoas.PapelResponsavel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pid string
		var r pessoas.PapelEmProcesso
		if err := rows.Scan(&pid, &r.ProcessoID, &r.ProcessoNumero, &r.ProcessoTitulo, &r.Papel, &r.Cliente, &r.Responsavel); err != nil {
			return nil, err
		}
		out[pid] = append(out[pid], r)
	}
	return out, rows.Err()
}

// rotuloSQL é o texto curto que outros domínios mostram do processo: número, senão título, senão código.
const rotuloSQL = `coalesce(nullif(numero_cnj,''), nullif(titulo,''), codigo)`

// Resumos devolve id → texto curto, para outros domínios (tarefas, prazos…) exibirem o processo.
func (s *Service) Resumos(ctx context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, `+rotuloSQL+` FROM processos WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, n string
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

// Existe informa se o processo existe e não está excluído.
func (s *Service) Existe(ctx context.Context, id string) (bool, error) {
	var one int
	err := s.Pool.QueryRow(ctx, `SELECT 1 FROM processos WHERE id=$1 AND excluido_em IS NULL`, id).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// Buscar devolve opções (id → texto) para seletores de outros domínios; busca por número (dígitos), título ou código.
func (s *Service) Buscar(ctx context.Context, q string, limit int) ([]Opcao, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, `+rotuloSQL+` FROM processos
		WHERE excluido_em IS NULL AND ($1 = '' OR titulo ILIKE $2 OR codigo ILIKE $2
		  OR ($3 <> '' AND regexp_replace(numero_cnj,'\D','','g') LIKE '%'||$3||'%'))
		ORDER BY created_at DESC, id LIMIT $4`, strings.TrimSpace(q), webx.Like(q), br.Digits(q), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Opcao
	for rows.Next() {
		var o Opcao
		if err := rows.Scan(&o.Valor, &o.Rotulo); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// Opcoes monta o seletor de processo do Plano 3 (Tarefas): "PRO. 0000012 · NOME DO CLIENTE".
func (s *Service) Opcoes(ctx context.Context) ([]Opcao, error) {
	ps, err := s.Listar(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Opcao, len(ps))
	for i, p := range ps {
		r := p.Codigo
		if c := p.ClienteNome(); c != "" {
			r += " · " + c
		}
		out[i] = Opcao{Valor: p.ID, Rotulo: r}
	}
	return out, nil
}
