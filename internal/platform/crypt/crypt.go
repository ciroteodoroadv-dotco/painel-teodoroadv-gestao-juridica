// Package crypt cifra campos sensíveis na aplicação (AES-256-GCM).
package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

// Seal cifra plaintext com a chave (usa os 32 primeiros bytes). Saída: nonce || texto cifrado.
func Seal(key, plaintext []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Open decifra o que Seal produziu. Falha se a chave estiver errada ou o dado for adulterado.
func Open(key, sealed []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(sealed) < gcm.NonceSize() {
		return nil, errors.New("dado cifrado curto demais")
	}
	n := gcm.NonceSize()
	return gcm.Open(nil, sealed[:n], sealed[n:], nil)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) < 32 {
		return nil, errors.New("chave precisa de 32 bytes")
	}
	block, err := aes.NewCipher(key[:32])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
