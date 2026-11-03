package intimacoes

import (
	"context"
	"log/slog"
	"time"
)

// Iniciar liga o agendador em segundo plano. Só existe com a coleta ATIVA: desligada, não nasce goroutine nenhuma e não
// há rede. A primeira rodada espera "primeira" (para o painel subir antes); depois, uma a cada Cfg.Intervalo. Para no
// ctx.Done(). Uma rodada com erro só vira estado da coleta (o rodapé e a faixa de alerta mostram); o agendador segue.
func (s *Service) Iniciar(ctx context.Context, log *slog.Logger, primeira time.Duration) (iniciou bool) {
	if !s.Cfg.Ativo || s.Coletor == nil {
		return false
	}
	intervalo := s.Cfg.Intervalo
	if intervalo <= 0 {
		intervalo = 3 * time.Hour
	}
	go func() {
		espera := primeira
		for {
			t := time.NewTimer(espera)
			select {
			case <-ctx.Done():
				t.Stop()
				return
			case <-t.C:
			}
			if r, err := s.Rodada(ctx); err != nil {
				if log != nil && ctx.Err() == nil {
					log.Error("coleta de intimações não executada", "motivo", err.Error())
				}
			} else if log != nil {
				log.Info("coleta de intimações", "status", r.Status, "novas", r.Novas, "falhas", r.FalhasOAB)
			}
			espera = intervalo
		}
	}()
	return true
}
