package crypt

import (
	"bytes"
	"testing"
)

func TestSealOpen(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	sealed, err := Seal(key, []byte("segredo-totp"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("segredo")) {
		t.Fatal("texto claro apareceu no cifrado")
	}
	got, err := Open(key, sealed)
	if err != nil || string(got) != "segredo-totp" {
		t.Fatalf("Open = %q, %v", got, err)
	}
}

func TestChaveErradaOuAdulteracao(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	sealed, _ := Seal(key, []byte("x"))
	if _, err := Open(bytes.Repeat([]byte{8}, 32), sealed); err == nil {
		t.Error("chave errada deveria falhar")
	}
	sealed[len(sealed)-1] ^= 1
	if _, err := Open(key, sealed); err == nil {
		t.Error("dado adulterado deveria falhar")
	}
	if _, err := Seal([]byte("curta"), []byte("x")); err == nil {
		t.Error("chave curta deveria falhar")
	}
}
