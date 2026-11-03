package tarefas

import (
	"context"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"painelteodoro/internal/platform/webx"
)

type normalized struct {
	origemTipo, origemID, pai         string
	titulo, subtitulo, descricao, obs string
	criadoPor, estado, prioridade     string
	dataBase, prazo, prazoFatal       *time.Time
	partes                            []ParteIn
	resp                              []string
}

func dataOpcional(rotulo, v string) (*time.Time, error) {
	if v = strings.TrimSpace(v); v == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, webx.Invalidf("Data inválida em " + rotulo + ".")
	}
	return &t, nil
}

func tamanho(rotulo, v string, max int) (string, error) {
	v = strings.TrimSpace(v)
	if utf8.RuneCountInString(v) > max {
		return "", webx.Invalidf(rotulo + " com mais de " + strconv.Itoa(max) + " caracteres.")
	}
	return v, nil
}

// normalize confere o formulário na ordem em que o B confere (título, vínculo, vínculo específico) e depois o resto.
// Na edição o vínculo não muda (editando=true): o que veio do formulário é ignorado.
func (s *Service) normalize(ctx context.Context, in Input, editando bool) (n normalized, err error) {
	if n.titulo = strings.TrimSpace(in.Titulo); n.titulo == "" {
		return n, webx.Invalidf("Título é obrigatório.")
	}
	if utf8.RuneCountInString(n.titulo) > 300 {
		return n, webx.Invalidf("Título com mais de 300 caracteres.")
	}
	if !editando {
		n.origemTipo, n.origemID = in.OrigemTipo, strings.TrimSpace(in.OrigemID)
		if n.origemTipo != "atendimento" && n.origemTipo != "processo" {
			return n, webx.Invalidf("Selecione o vínculo (Atendimento ou Processo).")
		}
		if n.origemID == "" {
			return n, webx.Invalidf("Selecione o vínculo específico.")
		}
		if err := s.confereVinculo(ctx, n.origemTipo, n.origemID); err != nil {
			return n, err
		}
	}
	if n.subtitulo, err = tamanho("Subtítulo", in.Subtitulo, 80); err != nil {
		return n, err
	}
	if n.descricao, err = tamanho("Descrição", in.Descricao, 8000); err != nil {
		return n, err
	}
	if n.obs, err = tamanho("Observações", in.Observacoes, 8000); err != nil {
		return n, err
	}
	if n.estado = in.Estado; n.estado == "" {
		n.estado = "pendente"
	}
	if !EstadoValido(n.estado) {
		return n, webx.Invalidf("Estado inválido.")
	}
	if n.prioridade = in.Prioridade; n.prioridade == "" {
		n.prioridade = "Normal"
	}
	if !prioridadeValida(n.prioridade) {
		return n, webx.Invalidf("Prioridade inválida.")
	}
	for _, d := range []struct {
		rot string
		src string
		dst **time.Time
	}{{"Data base", in.DataBase, &n.dataBase}, {"Prazo de entrega", in.Prazo, &n.prazo}, {"Prazo fatal", in.PrazoFatal, &n.prazoFatal}} {
		if *d.dst, err = dataOpcional(d.rot, d.src); err != nil {
			return n, err
		}
	}
	if err := s.conferePessoas(ctx, in, &n); err != nil {
		return n, err
	}
	if n.pai = strings.TrimSpace(in.TarefaPai); n.pai != "" && !editando {
		if !webx.UUID(n.pai) || !s.existe(ctx, n.pai) {
			return n, webx.Invalidf("Tarefa de origem não encontrada.")
		}
	}
	return n, nil
}

func (s *Service) confereVinculo(ctx context.Context, tipo, id string) error {
	if !webx.UUID(id) {
		return webx.Invalidf("Vínculo não encontrado.")
	}
	m, err := s.Vinculos.Resumos(ctx, tipo, []string{id})
	if err != nil {
		return err
	}
	if _, ok := m[id]; !ok {
		return webx.Invalidf("Vínculo não encontrado.")
	}
	return nil
}

// conferePessoas elimina linhas em branco e repetidas e exige que cada pessoa exista.
func (s *Service) conferePessoas(ctx context.Context, in Input, n *normalized) error {
	var ids []string
	for _, p := range in.Partes {
		id := strings.TrimSpace(p.PessoaID)
		if id == "" {
			continue // linha em branco deixada pelo usuário
		}
		papel, err := tamanho("Papel", p.Papel, 80)
		if err != nil {
			return err
		}
		n.partes = append(n.partes, ParteIn{PessoaID: id, Papel: papel})
		ids = append(ids, id)
	}
	vistos := map[string]bool{}
	for _, r := range in.Responsaveis {
		if r = strings.TrimSpace(r); r != "" && !vistos[r] {
			vistos[r] = true
			n.resp = append(n.resp, r)
			ids = append(ids, r)
		}
	}
	if n.criadoPor = strings.TrimSpace(in.CriadoPorID); n.criadoPor != "" {
		ids = append(ids, n.criadoPor)
	}
	if len(n.partes) > 50 || len(n.resp) > 50 {
		return webx.Invalidf("Partes ou responsáveis em excesso.")
	}
	if len(ids) == 0 {
		return nil
	}
	for _, id := range ids {
		if !webx.UUID(id) {
			return webx.Invalidf(msgPessoa)
		}
	}
	refs, err := s.Pessoas.Refs(ctx, ids)
	if err != nil {
		return err
	}
	achou := map[string]bool{}
	for _, r := range refs {
		achou[r.ID] = true
	}
	for _, id := range ids {
		if !achou[id] {
			return webx.Invalidf(msgPessoa)
		}
	}
	return nil
}

const msgPessoa = "Pessoa não encontrada: cadastre-a em Pessoas antes de usá-la na tarefa."
