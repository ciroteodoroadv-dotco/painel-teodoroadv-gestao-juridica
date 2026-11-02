package documentos

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"regexp"

	"painelteodoro/internal/platform/crypt"
)

var nameRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Store guarda arquivos cifrados num diretório privado. Quem lê o volume vê só bytes cifrados.
type Store struct {
	Dir string
	key []byte
}

// NewStore cria o diretório (0700) e deriva a chave de cifra dos arquivos a partir da chave mestra.
func NewStore(dir string, master []byte) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Store{Dir: dir, key: crypt.Derive(master, "documentos.arquivo")}, nil
}

// Put cifra e grava o conteúdo com nome aleatório (escrita atômica: arquivo temporário + rename).
func (s *Store) Put(data []byte) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	name := hex.EncodeToString(b)
	sealed, err := crypt.Seal(s.key, data)
	if err != nil {
		return "", err
	}
	tmp := filepath.Join(s.Dir, name+".tmp")
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	if _, err = f.Write(sealed); err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, filepath.Join(s.Dir, name))
	}
	if err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return name, nil
}

func (s *Store) path(name string) (string, error) {
	if !nameRe.MatchString(name) { // impede "../" e qualquer nome fora do padrão
		return "", errors.New("nome de arquivo inválido")
	}
	return filepath.Join(s.Dir, name), nil
}

// Get lê e decifra. Falha se o arquivo foi adulterado ou a chave mudou.
func (s *Store) Get(name string) ([]byte, error) {
	p, err := s.path(name)
	if err != nil {
		return nil, err
	}
	sealed, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return crypt.Open(s.key, sealed)
}

// Remove apaga o arquivo (usado só para desfazer um envio que falhou no banco).
func (s *Store) Remove(name string) {
	if p, err := s.path(name); err == nil {
		_ = os.Remove(p)
	}
}
