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
	"pessoas":    {"processos", "tarefas", "agenda", "documentos"},
	"processos":  {"tarefas", "agenda", "documentos"},
	"tarefas":    {"pessoas", "processos", "agenda", "documentos"},
	"agenda":     {"pessoas", "processos", "tarefas", "documentos"},
	"documentos": {"pessoas", "processos", "tarefas", "agenda"},
	"audit":      {"auth", "pessoas", "processos", "tarefas", "agenda", "documentos", "app"},
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
		"pessoas":    {"processos", "processo_partes", "tarefas", "agenda_eventos", "documentos"},
		"processos":  {"pessoas", "tarefas", "agenda_eventos", "documentos"},
		"tarefas":    {"pessoas", "processos", "processo_partes", "agenda_eventos", "documentos"},
		"agenda":     {"pessoas", "processos", "tarefas", "documentos"},
		"documentos": {"pessoas", "processos", "tarefas", "agenda_eventos"},
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
