package pessoas

import "context"

// Categoria é o grupo do acordeão de Pessoas. Vem dos processos, não da ficha da pessoa
// (decisão do painel B de 13/09/2026: o mesmo sujeito pode ser cliente num caso e parte adversa em outro).
type Categoria string

// Categorias, na ordem do acordeão do B.
const (
	CatCliente     Categoria = "cliente"
	CatAdversa     Categoria = "parte_contraria"
	CatTerceiro    Categoria = "terceiro_interessado"
	CatSemProcesso Categoria = "sem_processo"
)

// PapelResponsavel é o papel mostrado quando a pessoa só consta como responsável pelo cadastro do processo.
const PapelResponsavel = "Responsável pelo cadastro"

// PapelEmProcesso diz em que processo e com que papel a pessoa aparece.
type PapelEmProcesso struct {
	ProcessoID, ProcessoNumero, ProcessoTitulo, Papel string
	Cliente, Responsavel                              bool
}

// Rotulo é o texto do papel para o usuário (ROTULO_PAPEL_PESSOA do B, com o papel livre como reserva).
func (p PapelEmProcesso) Rotulo() string {
	switch p.Papel {
	case "cliente":
		return "cliente"
	case "parte_contraria":
		return "parte contrária"
	case "terceiro_interessado":
		return "terceiro interessado"
	}
	return p.Papel
}

// NumeroOuTitulo é o texto do botão "Escolha o processo" (numeroOuTitulo do B).
func (p PapelEmProcesso) NumeroOuTitulo() string {
	switch {
	case p.ProcessoNumero != "":
		return p.ProcessoNumero
	case p.ProcessoTitulo != "":
		return p.ProcessoTitulo
	}
	return "(sem número/título)"
}

// PapeisLookup é o que este domínio precisa do domínio de processos (interface; nunca SQL cruzado).
type PapeisLookup interface {
	PapeisPorPessoa(ctx context.Context, ids []string) (map[string][]PapelEmProcesso, error)
}

// CategoriaDe aplica a regra categoriaPapelPessoa do B: quem é cliente em algum processo é "cliente";
// senão, quem é "Terceiro interessado" em algum é terceiro; senão, quem aparece em algum é parte adversa;
// quem nunca apareceu fica "sem processo".
func CategoriaDe(p []PapelEmProcesso) Categoria {
	for _, x := range p {
		if x.Cliente {
			return CatCliente
		}
	}
	for _, x := range p {
		if x.Papel == "Terceiro interessado" {
			return CatTerceiro
		}
	}
	if len(p) > 0 {
		return CatAdversa
	}
	return CatSemProcesso
}

// Grupo é um painel do acordeão.
type Grupo struct {
	Chave   Categoria
	Rotulo  string
	Pessoas []Pessoa
}

// Quantidade é o número que aparece no botão do grupo.
func (g Grupo) Quantidade() int { return len(g.Pessoas) }

// Agrupar distribui as pessoas nas quatro categorias, na ordem e com os rótulos do B.
func Agrupar(pessoas []Pessoa, papeis map[string][]PapelEmProcesso) []Grupo {
	gs := []Grupo{{Chave: CatCliente, Rotulo: "Cliente"}, {Chave: CatAdversa, Rotulo: "Parte adversa"},
		{Chave: CatTerceiro, Rotulo: "Terceiro interessado"}, {Chave: CatSemProcesso, Rotulo: "Sem processo vinculado"}}
	for _, p := range pessoas {
		c := CategoriaDe(papeis[p.ID])
		for i := range gs {
			if gs[i].Chave == c {
				gs[i].Pessoas = append(gs[i].Pessoas, p)
			}
		}
	}
	return gs
}
