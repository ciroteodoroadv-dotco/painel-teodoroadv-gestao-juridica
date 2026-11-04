package processos

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/webx"
)

// ParteIn é uma linha de "Partes" do formulário.
type ParteIn struct {
	PessoaID, Papel string
	Cliente         bool
}

// Input vem do formulário. Datas em AAAA-MM-DD; sistema/instância já com o "Outro" resolvido pelo handler.
type Input struct {
	Partes                                               []ParteIn
	Responsaveis                                         []string
	Numero, ValorCausa, Sistema, Instancia               string
	Classe, Materia, Comarca, Juizo, Contingencia        string
	Distribuicao, Citacao, Audiencia, Sentenca, Recursos string
	FaseExecucao, Observacoes                            string
	Titulo, Polo, Projeto                                string // edição (na criação o título é derivado das partes)
	AtendimentoOrigem                                    string // id do atendimento (opcional)
	DuplicadoJustificativa                               string // exigida quando o número já existe em outro processo
}

type normalized struct {
	partes                              []ParteIn
	resp                                []string
	numero, valor                       string
	sistema, instancia, classe, materia string
	comarca, juizo, contingencia        string
	datas                               [6]*time.Time // distribuição, citação, audiência, sentença, recursos, execução
	obs, titulo, polo, projeto          string
	atendimento                         string
	duplicado                           bool
	dupJust, dupDe                      string
}

var rotulosData = [6]string{"Distribuição", "Citação", "Audiência", "Sentença", "Recursos", "Fase de execução"}

func texto(campo, v string, max int) (string, error) {
	v = strings.TrimSpace(v)
	if utf8.RuneCountInString(v) > max {
		return "", webx.Invalidf(campo + " com texto longo demais.")
	}
	return v, nil
}

func naLista(lista []string, v string) bool {
	for _, e := range lista {
		if e == v {
			return true
		}
	}
	return false
}

// normalize valida tudo o que independe do banco e confere as pessoas pela interface de pessoas.
func (s *Service) normalize(ctx context.Context, in Input) (n normalized, err error) {
	if v := strings.TrimSpace(in.Numero); v != "" {
		if n.numero, err = br.NormalizeCNJ(v); err != nil {
			m := err.Error()
			return n, webx.Invalidf(strings.ToUpper(m[:1]) + m[1:] + ".")
		}
	}
	var ok bool
	if n.valor, ok = numericDeBRL(in.ValorCausa); !ok {
		return n, webx.Invalidf("Valor da causa fora do limite aceito.")
	}
	for _, c := range []struct {
		nome string
		src  string
		dst  *string
		max  int
	}{{"Sistema", in.Sistema, &n.sistema, 80}, {"Instância", in.Instancia, &n.instancia, 80}, {"Comarca", in.Comarca, &n.comarca, 120},
		{"Juízo", in.Juizo, &n.juizo, 160}, {"Observações", in.Observacoes, &n.obs, 4000}, {"Título da ficha", in.Titulo, &n.titulo, 200},
		{"Polo", in.Polo, &n.polo, 80}, {"Projeto vinculado", in.Projeto, &n.projeto, 200}} {
		if *c.dst, err = texto(c.nome, c.src, c.max); err != nil {
			return n, err
		}
	}
	if n.classe = strings.TrimSpace(in.Classe); n.classe != "" && !naLista(Classes, n.classe) {
		return n, webx.Invalidf("Classe processual inválida.")
	}
	if n.materia = strings.TrimSpace(in.Materia); n.materia != "" && !naLista(Materias, n.materia) {
		return n, webx.Invalidf("Matéria / ramo do direito inválida.")
	}
	if n.contingencia = in.Contingencia; n.contingencia != "" && n.contingencia != "Ativa" && n.contingencia != "Passiva" {
		return n, webx.Invalidf("Contingência inválida.")
	}
	for i, s := range []string{in.Distribuicao, in.Citacao, in.Audiencia, in.Sentenca, in.Recursos, in.FaseExecucao} {
		if s = strings.TrimSpace(s); s != "" {
			t, err := time.Parse("2006-01-02", s)
			if err != nil {
				return n, webx.Invalidf("Data inválida em " + rotulosData[i] + ".")
			}
			n.datas[i] = &t
		}
	}
	if err := s.conferePessoas(ctx, in, &n); err != nil {
		return n, err
	}
	if n.atendimento = strings.TrimSpace(in.AtendimentoOrigem); n.atendimento != "" {
		ok := false
		if s.Atendimentos != nil && len(n.atendimento) == 36 {
			var err error
			if ok, err = s.Atendimentos.Existe(ctx, n.atendimento); err != nil {
				return n, err
			}
		}
		if !ok {
			return n, webx.Invalidf("Atendimento de origem não encontrado.")
		}
	}
	return n, nil
}

// conferePessoas elimina linhas em branco e repetidas, exige que cada pessoa exista e fixa o papel padrão do B.
func (s *Service) conferePessoas(ctx context.Context, in Input, n *normalized) error {
	var ids []string
	visto := map[string]bool{}
	for _, p := range in.Partes {
		id := strings.TrimSpace(p.PessoaID)
		if id == "" {
			continue // linha em branco deixada pelo usuário
		}
		papel := strings.TrimSpace(p.Papel)
		if papel == "" {
			papel = "parte" // o B também grava "parte" quando o papel não é escolhido
		}
		if utf8.RuneCountInString(papel) > 80 {
			return webx.Invalidf("Papel processual com texto longo demais.")
		}
		if chave := id + "|" + papel; !visto[chave] {
			visto[chave] = true
			n.partes = append(n.partes, ParteIn{PessoaID: id, Papel: papel, Cliente: p.Cliente})
			ids = append(ids, id)
		}
	}
	vistoR := map[string]bool{}
	for _, r := range in.Responsaveis {
		if r = strings.TrimSpace(r); r != "" && !vistoR[r] {
			vistoR[r] = true
			n.resp = append(n.resp, r)
			ids = append(ids, r)
		}
	}
	if len(n.resp) > 20 || len(n.partes) > 50 {
		return webx.Invalidf("Partes ou responsáveis em excesso.")
	}
	if len(ids) == 0 {
		return nil
	}
	existe, err := s.Pessoas.Exist(ctx, ids)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if !existe[id] {
			return webx.Invalidf("Pessoa não encontrada: cadastre-a em Pessoas antes de usá-la no processo.")
		}
	}
	return nil
}
