package documentos

import (
	"context"
	"crypto/sha256"
	"fmt"
)

// Verificacao resume a conferência de todos os arquivos contra o banco.
type Verificacao struct {
	Total     int
	OK        int
	Problemas []string // um item por arquivo com falha (ausente, adulterado ou chave errada)
}

// Verificar decifra CADA arquivo registrado e confere o SHA-256 gravado no banco.
// É a prova de que um backup restaurado está inteiro e utilizável (não só "existe").
func (s *Service) Verificar(ctx context.Context) (Verificacao, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, arquivo, sha256 FROM documentos ORDER BY created_at, id`)
	if err != nil {
		return Verificacao{}, err
	}
	defer rows.Close()
	var v Verificacao
	for rows.Next() {
		var id, arq string
		var want []byte
		if err := rows.Scan(&id, &arq, &want); err != nil {
			return v, err
		}
		v.Total++
		data, err := s.Store.Get(arq)
		switch {
		case err != nil:
			v.Problemas = append(v.Problemas, fmt.Sprintf("documento %s: arquivo ausente ou não decifra", id))
		case func() bool { h := sha256.Sum256(data); return string(h[:]) != string(want) }():
			v.Problemas = append(v.Problemas, fmt.Sprintf("documento %s: conteúdo não confere com o registro", id))
		default:
			v.OK++
		}
	}
	return v, rows.Err()
}
