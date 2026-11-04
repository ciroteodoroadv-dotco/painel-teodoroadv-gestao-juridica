package tarefas

import (
	"context"
	"strings"

	"painelteodoro/internal/platform/webx"
)

// enrich preenche vínculo (código e cliente), tarefa-pai, partes, responsáveis, "criado por" e a situação do prazo fatal.
func (s *Service) enrich(ctx context.Context, list []*Tarefa) error {
	if len(list) == 0 {
		return nil
	}
	ids, porTipo := make([]string, len(list)), map[string][]string{}
	idx := map[string]*Tarefa{}
	for i, t := range list {
		ids[i], idx[t.ID] = t.ID, t
		porTipo[t.OrigemTipo] = append(porTipo[t.OrigemTipo], t.OrigemID)
	}
	for tipo, oids := range porTipo {
		if tipo == OrigemIntimacao { // não há ficha para consultar: a origem é o próprio DJEN
			for _, t := range list {
				if t.OrigemTipo == tipo {
					t.OrigemCodigo = fonteOu(t.Fonte)
				}
			}
			continue
		}
		m, err := s.Vinculos.Resumos(ctx, tipo, oids)
		if err != nil {
			return err
		}
		for _, t := range list {
			if v, ok := m[t.OrigemID]; ok && t.OrigemTipo == tipo {
				t.OrigemCodigo, t.ClienteNome = v[0], v[1]
				if t.IntimacaoID != "" {
					t.OrigemCodigo += " · " + fonteOu(t.Fonte)
				}
			}
		}
	}
	if err := s.carregarPais(ctx, list); err != nil {
		return err
	}
	pessoaIDs := []string{}
	type lig struct {
		tarefa, pessoa, papel string
		resp                  bool
	}
	var ligs []lig
	for _, q := range []struct {
		sql  string
		resp bool
	}{{`SELECT tarefa_id::text, pessoa_id::text, papel FROM tarefa_partes WHERE tarefa_id = ANY($1::uuid[]) ORDER BY tarefa_id, ordem`, false},
		{`SELECT tarefa_id::text, pessoa_id::text, '' FROM tarefa_responsaveis WHERE tarefa_id = ANY($1::uuid[]) ORDER BY tarefa_id, ordem`, true}} {
		rows, err := s.Pool.Query(ctx, q.sql, ids)
		if err != nil {
			return err
		}
		for rows.Next() {
			l := lig{resp: q.resp}
			if err := rows.Scan(&l.tarefa, &l.pessoa, &l.papel); err != nil {
				rows.Close()
				return err
			}
			ligs = append(ligs, l)
			pessoaIDs = append(pessoaIDs, l.pessoa)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}
	for _, t := range list {
		if t.CriadoPorID != "" {
			pessoaIDs = append(pessoaIDs, t.CriadoPorID)
		}
	}
	refs, err := s.Pessoas.Refs(ctx, pessoaIDs)
	if err != nil {
		return err
	}
	nome := map[string]Ref{}
	for _, r := range refs {
		nome[r.ID] = r
	}
	for _, l := range ligs {
		t := idx[l.tarefa]
		if l.resp {
			t.Responsaveis = append(t.Responsaveis, nome[l.pessoa])
		} else {
			t.Partes = append(t.Partes, ParteTarefa{PessoaID: l.pessoa, Nome: nome[l.pessoa].Nome, Papel: l.papel})
		}
	}
	sit, err := s.Prazos.Situacoes(ctx, ids)
	if err != nil {
		return err
	}
	for _, t := range list {
		t.CriadoPor = nome[t.CriadoPorID].Nome
		t.PrazoFatalSituacao = sit[t.ID]
	}
	return nil
}

func (s *Service) carregarPais(ctx context.Context, list []*Tarefa) error {
	var pais []string
	for _, t := range list {
		if t.TarefaPaiID != "" {
			pais = append(pais, t.TarefaPaiID)
		}
	}
	if len(pais) == 0 {
		return nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT id::text, codigo FROM tarefas WHERE id = ANY($1::uuid[])`, pais)
	if err != nil {
		return err
	}
	defer rows.Close()
	cod := map[string]string{}
	for rows.Next() {
		var id, c string
		if err := rows.Scan(&id, &c); err != nil {
			return err
		}
		cod[id] = c
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, t := range list {
		t.TarefaPaiCodigo = cod[t.TarefaPaiID]
	}
	return nil
}

// busca é o texto que a busca da tela consulta (código, título, subtítulo, cliente, origem, responsáveis, partes,
// descrição e observações — a mesma lista da App.viewTarefas do B).
func (t *Tarefa) busca() string {
	p := make([]string, len(t.Partes))
	for i, x := range t.Partes {
		p[i] = x.Nome
	}
	return strings.Join([]string{t.Codigo, t.Titulo, t.Subtitulo, t.ClienteNome, t.OrigemCodigo, t.ResponsaveisTexto(),
		strings.Join(p, " "), t.Descricao, t.Observacoes}, " ")
}

// Listar devolve as tarefas não excluídas (por código). q filtra sem acento nem maiúsculas.
func (s *Service) Listar(ctx context.Context, q string) ([]*Tarefa, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+cols+` FROM tarefas WHERE excluido_em IS NULL ORDER BY codigo`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var all []*Tarefa
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		all = append(all, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.enrich(ctx, all); err != nil {
		return nil, err
	}
	if q = webx.SemAcento(strings.TrimSpace(q)); q == "" {
		return all, nil
	}
	var out []*Tarefa
	for _, t := range all {
		if strings.Contains(webx.SemAcento(t.busca()), q) {
			out = append(out, t)
		}
	}
	return out, nil
}

// AbertosDoProcesso conta as tarefas em aberto ligadas ao processo (pede confirmação extra ao excluí-lo). A tarefa automática de
// monitoramento não conta: ela existe por causa do processo e não é trabalho pendente do advogado.
func (s *Service) AbertosDoProcesso(ctx context.Context, processoID string) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM tarefas WHERE processo_id=$1::uuid AND excluido_em IS NULL
		AND NOT monitoramento AND estado NOT IN ('concluidas-com-sucesso','concluidas-sem-sucesso','canceladas')`, processoID).Scan(&n)
	return n, err
}
