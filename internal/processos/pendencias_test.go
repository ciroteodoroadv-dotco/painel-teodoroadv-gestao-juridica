package processos

import (
	"reflect"
	"testing"
)

func TestPendenciasProcessoVazioListaTodasNaOrdemFixa(t *testing.T) {
	quer := []string{"Número do processo (CNJ)", "Sistema", "Instância", "Classe processual", "Matéria / ramo do direito",
		"Comarca", "Juízo", "Distribuição", "Valor da causa", "Pelo menos uma parte marcada como cliente"}
	if got := Pendencias(Processo{}); !reflect.DeepEqual(got, quer) {
		t.Errorf("processo vazio:\n got %q\nquer %q", got, quer)
	}
}

func completo() Processo {
	return Processo{Numero: "x", Sistema: "PJe", Instancia: "1º grau", Classe: "Execução Fiscal", Materia: "Tributário",
		Comarca: "SP", Juizo: "1ª Vara", Distribuicao: "2026-01-01", Valor: "10.00",
		Partes: []Parte{{PessoaID: "a", Papel: "Autor", Cliente: true}}}
}

func TestPendenciasProcessoCompletoNaoListaNada(t *testing.T) {
	if got := Pendencias(completo()); len(got) != 0 {
		t.Errorf("processo completo: %q", got)
	}
}

func TestPendenciasParteClienteConta(t *testing.T) {
	p := completo()
	p.Partes = []Parte{{PessoaID: "a", Papel: "Réu"}}
	if got := Pendencias(p); !reflect.DeepEqual(got, []string{"Pelo menos uma parte marcada como cliente"}) {
		t.Errorf("sem parte cliente: %q", got)
	}
}

func TestFormatacaoDoValor(t *testing.T) {
	for in, quer := range map[string]string{"": "", "10000.00": "R$ 10.000,00", "5.5": "R$ 5,50", "1234567.89": "R$ 1.234.567,89", "0.00": "R$ 0,00"} {
		if got := formatBRL(in); got != quer {
			t.Errorf("formatBRL(%q) = %q, quer %q", in, got, quer)
		}
	}
	for in, quer := range map[string]string{"R$ 10.000,00": "10000.00", "1500": "15.00", "R$ 0,05": "0.05", "": ""} {
		if got, ok := numericDeBRL(in); !ok || got != quer {
			t.Errorf("numericDeBRL(%q) = %q %v, quer %q", in, got, ok, quer)
		}
	}
	if _, ok := numericDeBRL("9999999999999999"); ok {
		t.Error("valor gigante deveria ser recusado")
	}
}

func TestClienteEParteContrariaDerivadas(t *testing.T) {
	p := Processo{Partes: []Parte{{Nome: "RÉU", Papel: "Réu"}, {Nome: "AUTORA", Papel: "Autor", Cliente: true}, {Nome: "OUTRA CLIENTE", Cliente: true}}}
	if p.ClienteNome() != "AUTORA" {
		t.Errorf("cliente = %q", p.ClienteNome())
	}
	if got := p.ParteContraria(); !reflect.DeepEqual(got, []string{"RÉU", "OUTRA CLIENTE"}) {
		t.Errorf("parte contrária = %q", got)
	}
}
