package auth

import (
	"strings"
	"testing"
)

func TestHashEVerify(t *testing.T) {
	h, err := HashPassword("uma-senha-bem-longa")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("parâmetros não gravados no hash: %s", h)
	}
	if !VerifyPassword("uma-senha-bem-longa", h) {
		t.Error("senha correta recusada")
	}
	if VerifyPassword("outra-senha-longa", h) {
		t.Error("senha errada aceita")
	}
	if VerifyPassword("x", "lixo") || VerifyPassword("x", "$argon2id$v=19$m=1,t=1,p=1$%%$%%") {
		t.Error("hash malformado deveria falhar sem pânico")
	}
	h2, _ := HashPassword("uma-senha-bem-longa")
	if h == h2 {
		t.Error("salt deveria ser aleatório")
	}
}

func TestPolitica(t *testing.T) {
	ruins := []string{"curta", "password1234", "aaaaaaaaaaaaaa", "xx-maria.silva-xx"}
	for _, p := range ruins {
		if CheckPolicy(p, "maria.silva@exemplo.test") == nil {
			t.Errorf("senha %q deveria ser recusada", p)
		}
	}
	primeira, _, _ := strings.Cut(senhasComuns, "\n")
	if len(primeira) < 12 || CheckPolicy(primeira, "x@exemplo.test") == nil {
		t.Errorf("entrada da lista de senhas vazadas (%q) deveria ser recusada", primeira)
	}
	if err := CheckPolicy("cavalo-bateria-grampo-7", "maria.silva@exemplo.test"); err != nil {
		t.Errorf("senha boa recusada: %v", err)
	}
}
