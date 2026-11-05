package audit

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Auditoria 05/11, item 3.4: toda chave de metadado passada a um registro de auditoria precisa estar na lista
// permitida; senão filterMeta a descarta em silêncio e o fato some da trilha. Varre o código de produção.
func TestChavesDeMetaUsadasEstaoNaListaPermitida(t *testing.T) {
	reMapa := regexp.MustCompile(`map\[string\]string\{([^}]*)\}`)
	reChave := regexp.MustCompile(`"([a-z_]+)":`)
	err := filepath.WalkDir("..", func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		b, err := os.ReadFile(p) // #nosec G304 -- teste lendo o próprio código-fonte
		if err != nil {
			return err
		}
		for i, linha := range strings.Split(string(b), "\n") {
			if !strings.Contains(linha, "record(") && !strings.Contains(linha, "Meta:") && !strings.Contains(linha, "Record") {
				continue
			}
			for _, m := range reMapa.FindAllStringSubmatch(linha, -1) {
				for _, k := range reChave.FindAllStringSubmatch(m[1], -1) {
					if !allowedMeta[k[1]] {
						t.Errorf("%s:%d: chave de auditoria %q fora de allowedMeta (seria descartada)", p, i+1, k[1])
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
