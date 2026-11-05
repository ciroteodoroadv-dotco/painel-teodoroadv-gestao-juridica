package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Verify recalcula toda a cadeia. Devolve o total de eventos e o primeiro defeito encontrado.
func Verify(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	n, _, err := VerifyFrom(ctx, pool, Ponto{})
	return n, err
}

// Ponto é um elo já conferido da cadeia: o seq e o hash do último evento verificado (zero = início da cadeia).
type Ponto struct {
	Seq  int64
	Hash []byte
}

// VerifyFrom confere só os eventos DEPOIS do ponto dado, exigindo que o primeiro deles encadeie no hash do ponto.
// Devolve quantos conferiu e o novo ponto. Uso: a tela de auditoria confere de forma incremental (auditoria 05/11,
// item 4.2); a conferência COMPLETA continua no comando "painel audit verify" e no teste mensal de restauração.
func VerifyFrom(ctx context.Context, pool *pgxpool.Pool, desde Ponto) (int, Ponto, error) {
	rows, err := pool.Query(ctx, `SELECT seq, at, actor, action, target_type, target_id, result,
		request_id, meta, prev_hash, hash FROM audit_events WHERE seq > $1 ORDER BY seq`, desde.Seq)
	if err != nil {
		return 0, desde, err
	}
	defer rows.Close()

	expectedPrev := make([]byte, 32)
	if desde.Seq > 0 {
		expectedPrev = desde.Hash
	}
	ultimo := desde
	n := 0
	for rows.Next() {
		var (
			seq        int64
			at         time.Time
			ev         Event
			metaRaw    []byte
			prev, hash []byte
		)
		if err := rows.Scan(&seq, &at, &ev.Actor, &ev.Action, &ev.TargetType, &ev.TargetID,
			&ev.Result, &ev.RequestID, &metaRaw, &prev, &hash); err != nil {
			return n, ultimo, err
		}
		meta := map[string]string{}
		if err := json.Unmarshal(metaRaw, &meta); err != nil {
			return n, ultimo, fmt.Errorf("evento %d: metadado ilegível", seq)
		}
		if !bytes.Equal(prev, expectedPrev) {
			return n, ultimo, fmt.Errorf("evento %d: encadeamento quebrado (elo anterior não confere)", seq)
		}
		want, err := compute(prev, at.UTC().Truncate(time.Microsecond), ev, meta)
		if err != nil {
			return n, ultimo, err
		}
		if !bytes.Equal(want, hash) {
			return n, ultimo, fmt.Errorf("evento %d: conteúdo alterado (hash não confere)", seq)
		}
		expectedPrev = hash
		ultimo = Ponto{Seq: seq, Hash: hash}
		n++
	}
	return n, ultimo, rows.Err()
}
