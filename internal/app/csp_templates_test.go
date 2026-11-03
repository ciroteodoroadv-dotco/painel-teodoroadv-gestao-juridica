package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	reHandler = regexp.MustCompile(`(?i)\son[a-z]+\s*=`)
	reStyle   = regexp.MustCompile(`(?i)\sstyle\s*=`)
	reScript  = regexp.MustCompile(`(?i)<script(\s[^>]*)?>`)
)

// Nenhum template pode ter handler inline, style= ou <script> sem src (a CSP bloquearia).
func TestTemplatesSemInline(t *testing.T) {
	raiz := filepath.Join("..", "..")
	n := 0
	err := filepath.WalkDir(raiz, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".html") || !strings.Contains(filepath.ToSlash(p), "/templates/") {
			return err
		}
		n++
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		s := string(b)
		if reHandler.MatchString(s) {
			t.Errorf("%s: handler inline (on*=)", p)
		}
		if reStyle.MatchString(s) {
			t.Errorf("%s: atributo style=", p)
		}
		for _, m := range reScript.FindAllString(s, -1) {
			if !strings.Contains(strings.ToLower(m), "src=") {
				t.Errorf("%s: <script> inline: %s", p, m)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n < 10 {
		t.Fatalf("poucos templates encontrados (%d): caminho errado?", n)
	}
}
