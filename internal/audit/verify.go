package audit

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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

// Ancora devolve o último elo da cadeia no formato "SEQ:HASH_HEX" (vazio se a cadeia está vazia). O backup grava
// esse elo no MANIFEST: como o backup fica fora do servidor, ele é uma âncora externa da cadeia (auditoria 05/11,
// item 2.6). Sem âncora, quem tem poder de dono do banco reescreve a cadeia inteira e recalcula os hashes.
func Ancora(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	var seq int64
	var hash []byte
	err := pool.QueryRow(ctx, `SELECT seq, hash FROM audit_events ORDER BY seq DESC LIMIT 1`).Scan(&seq, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d:%s", seq, hex.EncodeToString(hash)), nil
}

// ConfereAncora confere que o evento da âncora "SEQ:HASH_HEX" (tirada de um backup antigo) ainda existe com o MESMO
// hash e que a cadeia inteira está íntegra. Se alguém reescreveu a história depois do backup, o hash não confere.
func ConfereAncora(ctx context.Context, pool *pgxpool.Pool, ancora string) error {
	seqTxt, hashTxt, ok := strings.Cut(strings.TrimSpace(ancora), ":")
	seq, err := strconv.ParseInt(seqTxt, 10, 64)
	want, err2 := hex.DecodeString(hashTxt)
	if !ok || err != nil || err2 != nil || seq <= 0 || len(want) != 32 {
		return errors.New("âncora inválida: use SEQ:HASH (como no MANIFEST do backup)")
	}
	var got []byte
	err = pool.QueryRow(ctx, `SELECT hash FROM audit_events WHERE seq=$1`, seq).Scan(&got)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("o evento %d da âncora NÃO existe mais: a cadeia foi alterada depois do backup", seq)
	}
	if err != nil {
		return err
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("o evento %d tem hash diferente do backup: a cadeia foi reescrita depois do backup", seq)
	}
	if _, err := Verify(ctx, pool); err != nil {
		return err
	}
	return nil
}
