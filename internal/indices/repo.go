package indices

import (
	"context"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo lê e grava o último valor de cada série.
type Repo struct{ Pool *pgxpool.Pool }

// Gravar guarda o valor novo da série (e limpa a marca de falha). O valor viaja como texto: nunca float.
func (r Repo) Gravar(ctx context.Context, v Valor) error {
	_, err := r.Pool.Exec(ctx, `INSERT INTO indices_valores (serie_codigo, rotulo, valor, unidade, referencia, obtido_em, fonte, falhou)
		VALUES ($1,$2,$3::text::numeric,$4,$5,$6,$7,false)
		ON CONFLICT (serie_codigo) DO UPDATE SET rotulo=EXCLUDED.rotulo, valor=EXCLUDED.valor, unidade=EXCLUDED.unidade,
			referencia=EXCLUDED.referencia, obtido_em=EXCLUDED.obtido_em, fonte=EXCLUDED.fonte, falhou=false`,
		v.Codigo, v.Rotulo, v.Valor, v.Unidade, v.Referencia, v.ObtidoEm, Origem)
	return err
}

// MarcarFalha registra que a última coleta da série falhou: o valor e a data antigos ficam, marcados "desatualizado".
func (r Repo) MarcarFalha(ctx context.Context, codigo int) error {
	_, err := r.Pool.Exec(ctx, `UPDATE indices_valores SET falhou=true WHERE serie_codigo=$1`, codigo)
	return err
}

// Listar devolve os valores gravados, na ordem do rodapé.
func (r Repo) Listar(ctx context.Context) ([]Valor, error) {
	rows, err := r.Pool.Query(ctx, `SELECT serie_codigo, rotulo, valor::text, unidade, referencia, obtido_em, falhou FROM indices_valores`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Valor
	for rows.Next() {
		var v Valor
		var ref, obt time.Time
		if err := rows.Scan(&v.Codigo, &v.Rotulo, &v.Valor, &v.Unidade, &ref, &obt, &v.Falhou); err != nil {
			return nil, err
		}
		v.Referencia, v.ObtidoEm = ref, obt
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, j int) bool { return posicao(out[i].Codigo) < posicao(out[j].Codigo) })
	return out, rows.Err()
}
