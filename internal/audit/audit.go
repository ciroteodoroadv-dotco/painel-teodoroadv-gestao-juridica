// Package audit grava eventos append-only encadeados por hash (ADR 0005).
// Nunca grava senha, token, CPF nem texto integral: metadados só pela lista permitida.
package audit

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const lockKey = 7001 // serializa a escrita da cadeia

// Resultados padronizados.
const (
	OK     = "ok"
	Falha  = "falha"
	Negado = "negado"
)

// allowedMeta é a lista permitida de chaves de metadado. Qualquer outra é descartada.
var allowedMeta = map[string]bool{
	"motivo": true, "de": true, "para": true, "papel": true, "email_hash": true, "etapa": true, "rota": true, "metodo": true, "suprimidos": true,
	"regra": true, "modo": true, "dias": true, "calendario": true, "tipo": true, "data": true,
	// Contagens (nunca dado pessoal): antes eram descartadas em silêncio (auditoria 05/11, item 3.4).
	"partes": true, "abertos_vinculados": true, "quantidade": true,
}

// Event é um fato auditável.
type Event struct {
	Actor      string // id do usuário, "sistema" ou "anonimo"
	Action     string
	TargetType string
	TargetID   string
	Result     string
	RequestID  string
	Meta       map[string]string
}

// Record grava o evento dentro da transação tx, encadeando ao anterior.
func Record(ctx context.Context, tx pgx.Tx, ev Event) error {
	if ev.Actor == "" || ev.Action == "" || ev.Result == "" {
		return errors.New("auditoria: ator, ação e resultado são obrigatórios")
	}
	meta := filterMeta(ev.Meta)
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockKey); err != nil {
		return err
	}
	prev := make([]byte, sha256.Size)
	err := tx.QueryRow(ctx, "SELECT hash FROM audit_events ORDER BY seq DESC LIMIT 1").Scan(&prev)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	at := time.Now().UTC().Truncate(time.Microsecond)
	h, err := compute(prev, at, ev, meta)
	if err != nil {
		return err
	}
	metaJSON, _ := json.Marshal(meta)
	_, err = tx.Exec(ctx, `INSERT INTO audit_events
		(at, actor, action, target_type, target_id, result, request_id, meta, prev_hash, hash)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		at, ev.Actor, ev.Action, ev.TargetType, ev.TargetID, ev.Result, ev.RequestID, metaJSON, prev, h)
	return err
}

// RecordPool grava o evento em transação própria (para eventos sem outra escrita junto).
func RecordPool(ctx context.Context, pool *pgxpool.Pool, ev Event) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := Record(ctx, tx, ev); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func filterMeta(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		if allowedMeta[k] {
			out[k] = v
		}
	}
	return out
}

// compute devolve SHA-256(prev || campos). json.Marshal ordena as chaves do mapa, então é estável.
func compute(prev []byte, at time.Time, ev Event, meta map[string]string) ([]byte, error) {
	m, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	s := sha256.New()
	s.Write(prev)
	fmt.Fprintf(s, "|%d|%s|%s|%s|%s|%s|%s|", at.UnixMicro(), ev.Actor, ev.Action, ev.TargetType, ev.TargetID, ev.Result, ev.RequestID)
	s.Write(m)
	return s.Sum(nil), nil
}
