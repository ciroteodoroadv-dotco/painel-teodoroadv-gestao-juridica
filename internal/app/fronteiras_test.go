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
	"pessoas":      {"processos", "atendimentos", "tarefas", "agenda", "audiencias", "diligencias", "documentos", "prazos", "intimacoes"},
	"atendimentos": {"pessoas", "processos", "tarefas", "agenda", "audiencias", "diligencias", "documentos", "prazos", "intimacoes"},
	"processos":    {"atendimentos", "tarefas", "agenda", "audiencias", "diligencias", "documentos", "prazos", "intimacoes"},
	"tarefas":      {"pessoas", "processos", "atendimentos", "agenda", "audiencias", "diligencias", "documentos", "prazos", "intimacoes"},
	"agenda":       {"pessoas", "processos", "atendimentos", "tarefas", "audiencias", "diligencias", "documentos", "prazos", "intimacoes"},
	"audiencias":   {"pessoas", "processos", "atendimentos", "tarefas", "agenda", "diligencias", "documentos", "prazos", "intimacoes"},
	"diligencias":  {"pessoas", "processos", "atendimentos", "tarefas", "agenda", "audiencias", "documentos", "prazos", "intimacoes"},
	"documentos":   {"pessoas", "processos", "atendimentos", "tarefas", "agenda", "audiencias", "diligencias", "prazos", "intimacoes"},
	"prazos":       {"pessoas", "processos", "atendimentos", "tarefas", "agenda", "audiencias", "diligencias", "documentos", "intimacoes"},
	"audit":        {"auth", "intimacoes", "monitor", "pessoas", "processos", "atendimentos", "tarefas", "agenda", "audiencias", "diligencias", "documentos", "prazos", "app"},
	"intimacoes":   {"pessoas", "processos", "atendimentos", "tarefas", "agenda", "audiencias", "diligencias", "documentos", "prazos"},
	"monitor":      {"pessoas", "processos", "atendimentos", "tarefas", "agenda", "audiencias", "diligencias", "documentos", "prazos", "intimacoes"},
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
	outrosDominios := func(minha ...string) []string {
		todas := []string{"pessoas", "processos", "processo_partes", "atendimentos", "atendimento_pessoas", "tarefas", "tarefa_partes",
			"tarefa_responsaveis", "agenda_compromissos", "audiencias", "diligencias", "documentos", "prazos", "intimacoes", "oab_monitoradas", "monitor_processos", "processo_movimentos", "monitor_sigilosos"}
		var out []string
		for _, t := range todas {
			proprio := false
			for _, m := range minha {
				proprio = proprio || t == m
			}
			if !proprio {
				out = append(out, t)
			}
		}
		return out
	}
	tabelas := map[string][]string{
		"pessoas":      outrosDominios("pessoas"),
		"processos":    outrosDominios("processos", "processo_partes"),
		"tarefas":      outrosDominios("tarefas", "tarefa_partes", "tarefa_responsaveis"),
		"atendimentos": outrosDominios("atendimentos", "atendimento_pessoas"),
		"agenda":       outrosDominios("agenda_compromissos"),
		"audiencias":   outrosDominios("audiencias"),
		"diligencias":  outrosDominios("diligencias"),
		"documentos":   outrosDominios("documentos"),
		"prazos":       outrosDominios("prazos"),
		"intimacoes":   outrosDominios("intimacoes", "oab_monitoradas", "coletas"),
		"monitor":      outrosDominios("monitor_processos", "processo_movimentos", "monitor_sigilosos"),
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
