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
	rows, err := pool.Query(ctx, `SELECT seq, at, actor, action, target_type, target_id, result,
		request_id, meta, prev_hash, hash FROM audit_events ORDER BY seq`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	expectedPrev := make([]byte, 32)
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
			return n, err
		}
		meta := map[string]string{}
		if err := json.Unmarshal(metaRaw, &meta); err != nil {
			return n, fmt.Errorf("evento %d: metadado ilegível", seq)
		}
		if !bytes.Equal(prev, expectedPrev) {
			return n, fmt.Errorf("evento %d: encadeamento quebrado (elo anterior não confere)", seq)
		}
		want, err := compute(prev, at.UTC().Truncate(time.Microsecond), ev, meta)
		if err != nil {
			return n, err
		}
		if !bytes.Equal(want, hash) {
			return n, fmt.Errorf("evento %d: conteúdo alterado (hash não confere)", seq)
		}
		expectedPrev = hash
		n++
	}
	return n, rows.Err()
}
