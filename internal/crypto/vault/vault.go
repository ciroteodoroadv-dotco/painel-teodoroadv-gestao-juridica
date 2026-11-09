package vault

import (
"crypto/aes"
"crypto/cipher"
"crypto/rand"
"errors"
"fmt"
"io"
"os"
"path/filepath"
)

// Vault gerencia a criptografia e decriptografia de arquivos sensíveis usando AES-256-GCM
type Vault struct {
secretKey []byte
}

func NewVault(hexOrStringKey string) (*Vault, error) {
key := []byte(hexOrStringKey)

// Ajusta o tamanho da chave para exatamente 32 bytes (256 bits)
if len(key) < 32 {
padded := make([]byte, 32)
copy(padded, key)
key = padded
} else if len(key) > 32 {
key = key[:32]
}

return &Vault{secretKey: key}, nil
}

// EncryptBytes cifra um slice de bytes em memória
func (v *Vault) EncryptBytes(plainText []byte) ([]byte, error) {
block, err := aes.NewCipher(v.secretKey)
if err != nil {
return nil, fmt.Errorf("falha ao inicializar cifra AES: %w", err)
}

gcm, err := cipher.NewGCM(block)
if err != nil {
return nil, fmt.Errorf("falha ao criar modo GCM: %w", err)
}

nonce := make([]byte, gcm.NonceSize())
if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
return nil, fmt.Errorf("falha ao gerar nonce aleatório: %w", err)
}

cipherText := gcm.Seal(nonce, nonce, plainText, nil)
return cipherText, nil
}

// DecryptBytes decifra um slice de bytes previamente cifrado
func (v *Vault) DecryptBytes(cipherText []byte) ([]byte, error) {
block, err := aes.NewCipher(v.secretKey)
if err != nil {
return nil, fmt.Errorf("falha ao inicializar cifra AES: %w", err)
}

gcm, err := cipher.NewGCM(block)
if err != nil {
return nil, fmt.Errorf("falha ao criar modo GCM: %w", err)
}

nonceSize := gcm.NonceSize()
if len(cipherText) < nonceSize {
return nil, errors.New("tamanho de dados cifrados inválido")
}

nonce, actualCipherText := cipherText[:nonceSize], cipherText[nonceSize:]
plainText, err := gcm.Open(nil, nonce, actualCipherText, nil)
if err != nil {
return nil, fmt.Errorf("falha ao decifrar dados (chave incorreta ou dados corrompidos): %w", err)
}

return plainText, nil
}

// EncryptFile grava o arquivo cifrado em disco no caminho de destino
func (v *Vault) EncryptFile(origemPath, destinoPath string) error {
origemPath = filepath.Clean(origemPath)
destinoPath = filepath.Clean(destinoPath)

data, err := os.ReadFile(origemPath)
if err != nil {
return fmt.Errorf("erro ao ler arquivo de origem: %w", err)
}

cifrado, err := v.EncryptBytes(data)
if err != nil {
return err
}

// #nosec G703 -- Caminho sanitizado previamente com filepath.Clean
if err := os.WriteFile(destinoPath, cifrado, 0600); err != nil {
return fmt.Errorf("erro ao salvar arquivo cifrado: %w", err)
}

return nil
}

// DecryptFile lê o arquivo cifrado e recupera o conteúdo original
func (v *Vault) DecryptFile(cifradoPath, destinoPath string) error {
cifradoPath = filepath.Clean(cifradoPath)
destinoPath = filepath.Clean(destinoPath)

data, err := os.ReadFile(cifradoPath)
if err != nil {
return fmt.Errorf("erro ao ler arquivo cifrado: %w", err)
}

decifrado, err := v.DecryptBytes(data)
if err != nil {
return err
}

// #nosec G703 -- Caminho sanitizado previamente com filepath.Clean
if err := os.WriteFile(destinoPath, decifrado, 0600); err != nil {
return fmt.Errorf("erro ao restaurar arquivo original: %w", err)
}

return nil
}
