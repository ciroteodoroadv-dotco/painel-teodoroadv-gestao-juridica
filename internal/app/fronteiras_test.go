package app

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const mod = "painelteodoro/internal/"

// Fronteiras entre domínios (especificação §4): um domínio nunca importa o outro por cima da
// interface combinada. Só processos pode conhecer os TIPOS de pessoas (contrato pessoas.ProcessosLookup).
var proibido = map[string][]string{
	"pessoas":      {"processos", "atendimentos", "tarefas", "agenda", "documentos", "prazos"},
	"atendimentos": {"pessoas", "processos", "tarefas", "agenda", "documentos", "prazos"},
	"processos":    {"atendimentos", "tarefas", "agenda", "documentos", "prazos"},
	"tarefas":      {"pessoas", "processos", "atendimentos", "agenda", "documentos", "prazos"},
	"agenda":       {"pessoas", "processos", "atendimentos", "tarefas", "documentos", "prazos"},
	"documentos":   {"pessoas", "processos", "atendimentos", "tarefas", "agenda", "prazos"},
	"prazos":       {"pessoas", "processos", "atendimentos", "tarefas", "agenda", "documentos"},
	"audit":        {"auth", "pessoas", "processos", "atendimentos", "tarefas", "agenda", "documentos", "prazos", "app"},
}

func TestFronteirasEntreDominios(t *testing.T) {
	for dom, bans := range proibido {
		dir := filepath.Join("..", dom)
		files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
		if len(files) == 0 {
			t.Fatalf("domínio %s sem arquivos em %s", dom, dir)
		}
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			af, err := parser.ParseFile(token.NewFileSet(), f, src, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, imp := range af.Imports {
				p := strings.Trim(imp.Path.Value, `"`)
				for _, ban := range bans {
					if p == mod+ban || strings.HasPrefix(p, mod+ban+"/") {
						t.Errorf("%s importa %s: fronteira de domínio violada (use uma interface)", f, p)
					}
				}
			}
		}
	}
}

// Domínios só tocam as próprias tabelas: nenhum SQL de um domínio cita tabela de outro.
func TestSQLNaoCruzaDominios(t *testing.T) {
	tabelas := map[string][]string{
		"pessoas":      {"processos", "processo_partes", "atendimentos", "atendimento_pessoas", "tarefas", "tarefa_partes", "tarefa_responsaveis", "agenda_eventos", "documentos"},
		"processos":    {"pessoas", "atendimentos", "atendimento_pessoas", "tarefas", "tarefa_partes", "tarefa_responsaveis", "agenda_eventos", "documentos"},
		"tarefas":      {"pessoas", "processos", "processo_partes", "atendimentos", "atendimento_pessoas", "prazos", "agenda_eventos", "documentos"},
		"atendimentos": {"pessoas", "processos", "tarefas", "tarefa_partes", "tarefa_responsaveis", "agenda_eventos", "documentos", "prazos"},
		"agenda":       {"pessoas", "processos", "atendimentos", "tarefas", "documentos"},
		"documentos":   {"pessoas", "processos", "atendimentos", "tarefas", "agenda_eventos", "prazos"},
		"prazos":       {"pessoas", "processos", "atendimentos", "tarefas", "tarefa_partes", "tarefa_responsaveis", "agenda_eventos", "documentos"},
	}
	for dom, outras := range tabelas {
		files, _ := filepath.Glob(filepath.Join("..", dom, "*.go"))
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			b, _ := os.ReadFile(f)
			s := strings.ToLower(string(b))
			for _, o := range outras {
				for _, kw := range []string{"from " + o + " ", "join " + o + " ", "into " + o + " ", "update " + o + " "} {
					if strings.Contains(s, kw) {
						t.Errorf("%s: SQL toca a tabela %q de outro domínio (%q)", f, o, kw)
					}
				}
			}
		}
	}
}
