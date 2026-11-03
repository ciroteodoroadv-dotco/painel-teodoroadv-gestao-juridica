package tarefas

import (
	"strings"
	"testing"
)

func entrada() Input { return Input{Titulo: "Tarefa válida", Status: "aberta"} }

func TestInputSubtituloEPrioridade(t *testing.T) {
	in := entrada()
	in.Subtitulo, in.Prioridade = "  Ligação ", "alta"
	n, err := in.normalize()
	if err != nil || n.subtitulo != "Ligação" || n.prioridade != "alta" {
		t.Fatalf("normalize = %+v, %v", n, err)
	}
	if n, err := entrada().normalize(); err != nil || n.prioridade != "normal" {
		t.Fatalf("prioridade vazia deveria virar normal: %+v, %v", n, err)
	}
	in = entrada()
	in.Prioridade = "urgente"
	if _, err := in.normalize(); err == nil {
		t.Error("prioridade inválida deveria ser recusada")
	}
	in = entrada()
	in.Subtitulo = strings.Repeat("ã", 81)
	if _, err := in.normalize(); err == nil {
		t.Error("subtítulo de 81 caracteres deveria ser recusado")
	}
	in.Subtitulo = strings.Repeat("ã", 80)
	if _, err := in.normalize(); err != nil {
		t.Errorf("subtítulo de 80 caracteres (runes) deveria passar: %v", err)
	}
}

func TestEmRevisaoContaComoAberta(t *testing.T) {
	for _, s := range []string{"aberta", "em_andamento", "em_revisao"} {
		if !(Tarefa{Status: s}).Aberta() {
			t.Errorf("%s deveria contar como aberta", s)
		}
	}
	for _, s := range []string{"concluida", "cancelada"} {
		if (Tarefa{Status: s}).Aberta() {
			t.Errorf("%s não é aberta", s)
		}
	}
	if in := (Input{Titulo: "Tarefa válida", Status: "em_revisao"}); func() bool { _, e := in.normalize(); return e != nil }() {
		t.Error("em_revisao deveria ser status válido")
	}
}

func TestPrioridadeRotuloEClasse(t *testing.T) {
	cases := map[string][2]string{"alta": {"Alta", "prioridade-alta"}, "media": {"Média", "prioridade-media"},
		"normal": {"Normal", "prioridade-normal"}, "baixa": {"Baixa", "prioridade-baixa"}, "": {"Normal", "prioridade-normal"}}
	for p, want := range cases {
		tr := Tarefa{Prioridade: p}
		if tr.PrioridadeRotulo() != want[0] || tr.PrioridadeClasse() != want[1] {
			t.Errorf("%q: %s/%s, esperado %v", p, tr.PrioridadeRotulo(), tr.PrioridadeClasse(), want)
		}
	}
}
