package logs

import (
	"bytes"
	"strings"
	"testing"
)

func TestRedigeChavesSensiveis(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "info")
	l.Info("teste", "cpf", "000.000.000-00", "user_email", "a@b.c", "Authorization", "Bearer x",
		"request_id", "abc123", "status", 200)
	out := buf.String()
	for _, vazou := range []string{"000.000.000-00", "a@b.c", "Bearer"} {
		if strings.Contains(out, vazou) {
			t.Errorf("log vazou %q: %s", vazou, out)
		}
	}
	if !strings.Contains(out, "abc123") || !strings.Contains(out, `"status":200`) {
		t.Errorf("log perdeu campo permitido: %s", out)
	}
}

func TestNivelInvalidoCaiParaInfo(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "xyz")
	l.Debug("não deve aparecer")
	if buf.Len() != 0 {
		t.Fatalf("debug apareceu com nível inválido: %s", buf.String())
	}
	l.Info("aparece")
	if buf.Len() == 0 {
		t.Fatal("info deveria aparecer")
	}
}
