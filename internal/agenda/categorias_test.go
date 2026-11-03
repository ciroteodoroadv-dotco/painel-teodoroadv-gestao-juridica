package agenda

import (
	"context"
	"strings"
	"testing"
	"time"

	"painelteodoro/internal/platform/webx"
)

// falsoTarefas é um TarefaLookup de teste: devolve só tarefas abertas.
type falsoTarefas struct{ lista []TarefaResumo }

func (f falsoTarefas) AbertasPorSubtitulo(context.Context, string, int) ([]TarefaResumo, error) {
	return f.lista, nil
}

var agora = time.Date(2026, 11, 3, 10, 0, 0, 0, webx.SP())

func ev(tipo, titulo string, dias int) Evento {
	return Evento{ID: titulo, Tipo: tipo, Titulo: titulo, Inicio: agora.AddDate(0, 0, dias)}
}

func TestNormalizarSubtitulo(t *testing.T) {
	for in, want := range map[string]string{"  Ligação ": "ligacao", "REUNIÕES": "reunioes", "Aula": "aula", "": ""} {
		if got := NormalizarSubtitulo(in); got != want {
			t.Errorf("NormalizarSubtitulo(%q) = %q, esperado %q", in, got, want)
		}
	}
}

func TestSeteCategoriasNaOrdem(t *testing.T) {
	want := []string{"Tarefas", "Intimações", "Audiências", "Diligências", "Reuniões", "Ligações", "Aulas"}
	if len(Categorias) != len(want) {
		t.Fatalf("%d categorias, esperado %d", len(Categorias), len(want))
	}
	for i, c := range Categorias {
		if c.Rotulo != want[i] {
			t.Errorf("categoria %d = %q, esperado %q", i, c.Rotulo, want[i])
		}
	}
}

func painelDe(t *testing.T, cat, q string, eventos []Evento, tarefas []TarefaResumo) PainelView {
	t.Helper()
	tl, err := falsoTarefas{tarefas}.AbertasPorSubtitulo(context.Background(), "", 500)
	if err != nil {
		t.Fatal(err)
	}
	return montarPainel(cat, q, agora, eventos, tl)
}

func quantidade(v PainelView, chave string) int {
	for _, r := range v.Resumos {
		if r.Chave == chave {
			return r.Quantidade
		}
	}
	return -1
}

func TestContagemAulas(t *testing.T) {
	eventos := []Evento{ev("aula", "Aula 1", 1), ev("aula", "Aula 2", 2), ev("aula", "Aula passada", -1)}
	tarefas := []TarefaResumo{{ID: "1", Titulo: "Preparar aula", Subtitulo: " Aula "}, {ID: "2", Titulo: "Outra", Subtitulo: "Cobrança"}}
	v := painelDe(t, "", "", eventos, tarefas)
	if got := quantidade(v, "aulas"); got != 3 {
		t.Errorf("Aulas = %d, esperado 3 (2 futuras + 1 tarefa)", got)
	}
	if got := quantidade(v, "tarefas"); got != 2 {
		t.Errorf("Tarefas = %d, esperado 2", got)
	}
	for _, r := range v.Resumos {
		if r.Aberta {
			t.Errorf("%s não deveria vir aberta por padrão", r.Chave)
		}
	}
}

func TestCategoriaAbertaEFiltro(t *testing.T) {
	eventos := []Evento{ev("aula", "Aula 1", 1), ev("reuniao", "Reunião 1", 1), ev("aula", "Aula passada", -1)}
	tarefas := []TarefaResumo{{Titulo: "Preparar aula", Subtitulo: "aula"}, {Titulo: "Ligar", Subtitulo: "Ligação"}}
	v := painelDe(t, "aulas", "", eventos, tarefas)
	if len(v.Eventos) != 1 || v.Eventos[0].Titulo != "Aula 1" || len(v.Tarefas) != 1 || v.Tarefas[0].Titulo != "Preparar aula" {
		t.Errorf("cat=aulas: %+v / %+v", v.Eventos, v.Tarefas)
	}
	if v.Resumos[6].Chave != "aulas" || !v.Resumos[6].Aberta {
		t.Error("a pílula Aulas deveria estar aberta")
	}
	v = painelDe(t, "xyz", "", eventos, tarefas)
	for _, r := range v.Resumos {
		if r.Aberta {
			t.Error("cat inválida: nenhuma deveria estar aberta")
		}
	}
	if len(v.Eventos)+len(v.Tarefas) != 0 {
		t.Error("cat inválida não lista nada")
	}
	v = painelDe(t, "tarefas", "", eventos, tarefas)
	if len(v.Tarefas) != 2 || len(v.Eventos) != 0 {
		t.Errorf("cat=tarefas deveria listar todas as tarefas abertas: %d eventos, %d tarefas", len(v.Eventos), len(v.Tarefas))
	}
}

func TestBuscaSemAcento(t *testing.T) {
	eventos := []Evento{ev("reuniao", "Reunião com Conceição", 1), ev("reuniao", "Reunião interna", 1)}
	v := painelDe(t, "reunioes", "conceicao", eventos, nil)
	if len(v.Eventos) != 1 || quantidade(v, "reunioes") != 1 {
		t.Errorf("busca sem acento: %d eventos, contagem %d", len(v.Eventos), quantidade(v, "reunioes"))
	}
}

func TestIntimacoesZeroComAviso(t *testing.T) {
	v := painelDe(t, "intimacoes", "", nil, nil)
	if quantidade(v, "intimacoes") != 0 || !strings.Contains(v.Aviso, "Plano 5") {
		t.Errorf("intimações = %d, aviso = %q", quantidade(v, "intimacoes"), v.Aviso)
	}
	if painelDe(t, "aulas", "", nil, nil).Aviso != "" {
		t.Error("só intimações traz o aviso")
	}
}

func TestDataPorExtenso(t *testing.T) {
	v := painelDe(t, "", "", nil, nil)
	if v.Hoje != "terça-feira, 3 de novembro de 2026" || v.HojeCurto != "03/11/2026" {
		t.Errorf("Hoje = %q / %q", v.Hoje, v.HojeCurto)
	}
}
