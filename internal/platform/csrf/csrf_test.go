package csrf

import "testing"

func TestToken(t *testing.T) {
	k := []byte("0123456789abcdef0123456789abcdef")
	tok := Token(k, "sessao-1")
	if !Valid(k, "sessao-1", tok) {
		t.Error("token legítimo recusado")
	}
	if Valid(k, "sessao-2", tok) {
		t.Error("token de outra sessão aceito")
	}
	if Valid([]byte("outra-chave-outra-chave-outra-ch"), "sessao-1", tok) {
		t.Error("token com outra chave aceito")
	}
	if Valid(k, "", tok) || Valid(k, "sessao-1", "") {
		t.Error("vazio nunca é válido")
	}
}
