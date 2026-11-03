package webx

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// cabecalho codifica o texto para um valor de cabeçalho HTTP: o navegador lê cabeçalhos byte a byte (Latin-1),
// então o que não é ASCII visível (e o próprio %) vai em %XX; o JS desfaz com decodeURIComponent.
func cabecalho(msg string) string {
	var b strings.Builder
	for _, c := range []byte(msg) {
		if c < 0x20 || c > 0x7e || c == '%' {
			fmt.Fprintf(&b, "%%%02X", c)
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// Ok responde sucesso ao formulário do drawer: o JS fecha o drawer, mostra o toast e recarrega o conteúdo.
func Ok(w http.ResponseWriter, msg string) {
	w.Header().Set("X-Painel-Ok", cabecalho(msg))
	w.WriteHeader(http.StatusOK)
}

// Falha traduz o erro em texto simples (o JS do drawer mostra como toast de erro, como o showToast(..., true) do B).
// Devolve true se havia erro. msgFalha, se vier, é o texto de falha genérica do painel B.
func Falha(w http.ResponseWriter, err error, naoEncontrado string, msgFalha ...string) bool {
	if err == nil {
		return false
	}
	var inv Invalid
	switch {
	case errors.As(err, &inv):
		http.Error(w, inv.Msg, http.StatusBadRequest)
	case errors.Is(err, ErrNotFound):
		http.Error(w, naoEncontrado, http.StatusNotFound)
	case errors.Is(err, ErrConflict):
		http.Error(w, "Este registro foi alterado por outra pessoa. Feche, recarregue a página e refaça a alteração.", http.StatusConflict)
	default:
		msg := "Erro interno."
		if len(msgFalha) > 0 {
			msg = msgFalha[0]
		}
		http.Error(w, msg, http.StatusInternalServerError)
	}
	return true
}
