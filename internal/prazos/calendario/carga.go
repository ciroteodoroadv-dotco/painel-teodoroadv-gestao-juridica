package calendario

import (
	"context"
	"time"

	"painelteodoro/internal/prazos/calculo"
)

// doBanco é o calendário entregue ao cálculo: só dias ATIVOS que contam como não úteis (e que têm fonte,
// por restrição do banco). Implementa calculo.Calendario e calculo.Cobertor.
type doBanco struct {
	dias map[time.Time]calculo.DiaNaoUtil
	anos map[int]bool
}

func (c doBanco) NaoUtil(d time.Time) (calculo.DiaNaoUtil, bool) { x, ok := c.dias[d]; return x, ok }
func (c doBanco) Cobre(ano int) bool                              { return c.anos[ano] }

// Carregar devolve o calendário para o cálculo, com cache curto (invalidado a cada gravação).
// id vazio = só os feriados nacionais (nil).
func (s *Service) Carregar(ctx context.Context, id string) (calculo.Calendario, error) {
	if id == "" {
		return nil, nil
	}
	s.mu.Lock()
	if e, ok := s.cache[id]; ok && time.Now().Before(e.ate) {
		s.mu.Unlock()
		return e.cal, nil
	}
	s.mu.Unlock()
	rows, err := s.Pool.Query(ctx, `SELECT data, tipo, descricao, fonte_citacao, conta_como_nao_util FROM calendario_dias
		WHERE calendario_id = $1 AND ativo`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	c := doBanco{dias: map[time.Time]calculo.DiaNaoUtil{}, anos: map[int]bool{}}
	for rows.Next() {
		var d calculo.DiaNaoUtil
		var conta bool
		if err := rows.Scan(&d.Data, &d.Tipo, &d.Descricao, &d.Fonte, &conta); err != nil {
			return nil, err
		}
		d.Data = time.Date(d.Data.Year(), d.Data.Month(), d.Data.Day(), 0, 0, 0, 0, time.UTC)
		c.anos[d.Data.Year()] = true // o ano está coberto mesmo que o dia não conte (ele foi avaliado)
		if conta {
			if _, dup := c.dias[d.Data]; !dup || d.Tipo == "feriado" {
				c.dias[d.Data] = d
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.cache == nil {
		s.cache = map[string]cacheEnt{}
	}
	s.cache[id] = cacheEnt{cal: c, ate: time.Now().Add(validadeCache)}
	s.mu.Unlock()
	return c, nil
}
