package crypt

import (
	"crypto/hmac"
	"crypto/sha256"
)

// Derive gera uma subchave de 32 bytes, própria de cada finalidade, a partir da chave mestra.
// Assim a chave de cifra e a do índice cego nunca são a mesma.
func Derive(master []byte, purpose string) []byte {
	m := hmac.New(sha256.New, master)
	m.Write([]byte("painel|" + purpose))
	return m.Sum(nil)
}

// BlindIndex permite buscar por igualdade um campo cifrado (ex.: CPF) sem guardá-lo em claro.
func BlindIndex(master []byte, purpose, value string) []byte {
	m := hmac.New(sha256.New, Derive(master, purpose+"|idx"))
	m.Write([]byte(value))
	return m.Sum(nil)
}
