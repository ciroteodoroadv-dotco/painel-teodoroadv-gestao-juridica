// Package coletas guarda o estado de cada coleta externa (DJEN e, nos planos seguintes, DataJud e índices):
// última tentativa, último sucesso, situação e uma mensagem curta. É o que alimenta o rodapé "Estado das coletas".
// A mensagem nunca leva texto de intimação: só contagens e códigos de erro.
package coletas

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Estado é a linha de uma coleta. Status: sucesso, parcial, falha, bloqueado ou nao_sincronizado.
type Estado struct {
	Modulo, Origem, Status, Mensagem string
	UltimaTentativa, UltimoSucesso   *time.Time
	JanelaDesde                      *time.Time
}

// Store lê e grava o estado das coletas.
type Store struct{ Pool *pgxpool.Pool }

var statusValidos = map[string]bool{"sucesso": true, "parcial": true, "falha": true, "bloqueado": true, "nao_sincronizado": true}

// Registrar atualiza a tentativa. Só o status "sucesso" avança o último sucesso e a janela: falha, coleta parcial
// ou bloqueada nunca fazem a janela andar (senão uma comunicação poderia ficar para sempre fora da coleta).
func (s *Store) Registrar(ctx context.Context, e Estado) error {
	if e.Modulo == "" || !statusValidos[e.Status] {
		return errors.New("coletas: módulo e status válidos são obrigatórios")
	}
	if utf8.RuneCountInString(e.Mensagem) > 500 {
		e.Mensagem = string([]rune(e.Mensagem)[:500])
	}
	if e.UltimaTentativa == nil {
		agora := time.Now()
		e.UltimaTentativa = &agora
	}
	var sucesso, janela *time.Time
	if e.Status == "sucesso" {
		sucesso, janela = e.UltimaTentativa, e.JanelaDesde
	}
	_, err := s.Pool.Exec(ctx, `INSERT INTO coletas (modulo, origem, ultima_tentativa, ultimo_sucesso, status, mensagem, janela_desde)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (modulo) DO UPDATE SET origem=EXCLUDED.origem, ultima_tentativa=EXCLUDED.ultima_tentativa,
			status=EXCLUDED.status, mensagem=EXCLUDED.mensagem,
			ultimo_sucesso=coalesce(EXCLUDED.ultimo_sucesso, coletas.ultimo_sucesso),
			janela_desde=coalesce(EXCLUDED.janela_desde, coletas.janela_desde)`,
		e.Modulo, e.Origem, e.UltimaTentativa, sucesso, e.Status, e.Mensagem, janela)
	return err
}

const colunas = `modulo, origem, ultima_tentativa, ultimo_sucesso, status, mensagem, janela_desde`

func varre(row pgx.Row) (Estado, error) {
	var e Estado
	err := row.Scan(&e.Modulo, &e.Origem, &e.UltimaTentativa, &e.UltimoSucesso, &e.Status, &e.Mensagem, &e.JanelaDesde)
	return e, err
}

// Listar devolve todas as coletas, por nome do módulo.
func (s *Store) Listar(ctx context.Context) ([]Estado, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+colunas+` FROM coletas ORDER BY modulo`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Estado
	for rows.Next() {
		e, err := varre(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Obter devolve a coleta do módulo; ok=false se nunca foi registrada.
func (s *Store) Obter(ctx context.Context, modulo string) (Estado, bool, error) {
	e, err := varre(s.Pool.QueryRow(ctx, `SELECT `+colunas+` FROM coletas WHERE modulo=$1`, modulo))
	if errors.Is(err, pgx.ErrNoRows) {
		return Estado{}, false, nil
	}
	return e, err == nil, err
}
