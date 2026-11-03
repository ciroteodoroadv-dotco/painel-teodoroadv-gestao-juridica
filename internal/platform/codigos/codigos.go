// Package codigos atribui os códigos de ficha ("PES. 0000001", "PRO. 0000012"…).
// O número é dado UMA vez, na criação, dentro da mesma transação da ficha: se a transação
// desfaz, o número volta; duas transações concorrentes nunca recebem o mesmo.
package codigos

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"
)

var prefixoValido = regexp.MustCompile(`^[A-Z]{3}$`)

// Proximo devolve o próximo código do prefixo (ex.: "PES" → "PES. 0000001") dentro da transação tx.
// O contador é por prefixo. O UPSERT trava a linha do prefixo até o commit, o que serializa quem concorre.
func Proximo(ctx context.Context, tx pgx.Tx, prefixo string) (string, error) {
	if !prefixoValido.MatchString(prefixo) {
		return "", fmt.Errorf("prefixo de código inválido: %q", prefixo)
	}
	var n int64
	err := tx.QueryRow(ctx, `INSERT INTO codigos_contador (prefixo, ultimo) VALUES ($1, 1)
		ON CONFLICT (prefixo) DO UPDATE SET ultimo = codigos_contador.ultimo + 1
		RETURNING ultimo`, prefixo).Scan(&n)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s. %07d", prefixo, n), nil
}
