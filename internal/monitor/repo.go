package monitor

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"painelteodoro/internal/monitor/datajud"
	"painelteodoro/internal/platform/webx"
)

const insereMovimento = `INSERT INTO processo_movimentos (processo_id, fonte, grau, orgao_julgador, chave, data, data_hora, codigo, nome, complemento, complementos, base, intimacao_id)
	VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12,$13::uuid) ON CONFLICT (processo_id, fonte, chave) DO NOTHING`

// upsertEstado grava o resultado da consulta de um processo. Só "sucesso" avança o último sucesso.
func upsertEstado(ctx context.Context, exec func(context.Context, string, ...any) (pgconn.CommandTag, error), agora time.Time, id, alias, status, msg string) error {
	var sucesso *time.Time
	if status == "sucesso" {
		sucesso = &agora
	}
	_, err := exec(ctx, `INSERT INTO monitor_processos (processo_id, tribunal_alias, status, mensagem, ultima_consulta, ultimo_sucesso)
		VALUES ($1::uuid,$2,$3,$4,$5,$6)
		ON CONFLICT (processo_id) DO UPDATE SET tribunal_alias=CASE WHEN EXCLUDED.tribunal_alias='' THEN monitor_processos.tribunal_alias ELSE EXCLUDED.tribunal_alias END,
			status=EXCLUDED.status, mensagem=EXCLUDED.mensagem, ultima_consulta=EXCLUDED.ultima_consulta,
			ultimo_sucesso=COALESCE(EXCLUDED.ultimo_sucesso, monitor_processos.ultimo_sucesso)`, id, alias, status, msg, agora, sucesso)
	return err
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// dataCivil é a data do movimento no fuso de São Paulo (nunca o dia em UTC: o B errava a data ao fim da tarde).
func dataCivil(m datajud.Movimento) any {
	if m.DataHora.IsZero() {
		return nil
	}
	return webx.Hoje(m.DataHora)
}

func nullTempo(m datajud.Movimento) any {
	if m.DataHora.IsZero() {
		return nil
	}
	return m.DataHora
}

func complementosJSON(m datajud.Movimento) string {
	type c struct {
		Codigo    string `json:"codigo,omitempty"`
		Nome      string `json:"nome,omitempty"`
		Valor     string `json:"valor,omitempty"`
		Descricao string `json:"descricao,omitempty"`
	}
	out := make([]c, 0, len(m.Complementos))
	for _, x := range m.Complementos {
		out = append(out, c{x.Codigo, x.Nome, x.Valor, x.Descricao})
	}
	b, _ := json.Marshal(out)
	return string(b)
}
