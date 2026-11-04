package calendario

import (
	"context"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/prazos/calculo"
)

// SiglaNacional é o calendário semeado a partir da lei.
const SiglaNacional = "NACIONAL"

// Semear grava o calendário Nacional e os feriados nacionais de data FIXA de anoBase e dos 2 anos seguintes,
// cada um com a norma em fonte_citacao. Os móveis (Sexta-feira Santa) são calculados no cálculo, não gravados.
// Idempotente: rodar de novo não duplica. Os registros nascem sem verificador ([VERIFICAR] até o advogado validar).
func (s *Service) Semear(ctx context.Context, anoBase int) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO calendarios (sigla, nome, ambito) VALUES ($1,'Nacional','nacional')
		ON CONFLICT (sigla) DO UPDATE SET sigla = EXCLUDED.sigla RETURNING id::text`, SiglaNacional).Scan(&id); err != nil {
		return err
	}
	novos := 0
	for ano := anoBase; ano <= anoBase+2; ano++ {
		for _, d := range calculo.FixosDoAno(ano) {
			tag, err := tx.Exec(ctx, `INSERT INTO calendario_dias (calendario_id, data, tipo, conta_como_nao_util, descricao, fonte_citacao)
				VALUES ($1,$2,'feriado',true,$3,$4) ON CONFLICT (calendario_id, data, tipo) WHERE ativo DO NOTHING`,
				id, d.Data, d.Descricao, d.Fonte)
			if err != nil {
				return err
			}
			novos += int(tag.RowsAffected())
		}
	}
	if novos > 0 {
		if err := audit.Record(ctx, tx, audit.Event{Actor: "sistema", Action: "calendario.semeado", TargetType: "calendario",
			TargetID: id, Result: audit.OK, Meta: map[string]string{"calendario": SiglaNacional}}); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.invalida()
	return nil
}
