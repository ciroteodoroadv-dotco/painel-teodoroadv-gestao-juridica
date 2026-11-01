package webx

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
)

// Actor identifica quem pratica uma escrita; vai para a auditoria.
type Actor struct {
	UserID    string
	RequestID string
}

// ActorOf monta o ator a partir da resposta (que já carrega o X-Request-Id) e do id do usuário.
func ActorOf(w http.ResponseWriter, userID string) Actor {
	return Actor{UserID: userID, RequestID: w.Header().Get("X-Request-Id")}
}

// IsUniqueViolation informa se o erro do PostgreSQL é de índice único (23505).
func IsUniqueViolation(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "23505"
}

// IsForeignKeyViolation informa se o erro é de chave estrangeira (23503).
func IsForeignKeyViolation(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "23503"
}
