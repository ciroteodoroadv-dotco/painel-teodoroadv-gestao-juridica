package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"painelteodoro/internal/platform/coletas"
	"painelteodoro/internal/platform/webx"
)

// Iniciar liga o agendador em segundo plano. Só existe com a coleta ATIVA: desligada, não nasce goroutine e não há rede.
// Uma rodada com erro só vira estado da coleta (rodapé e faixa de alerta); o agendador segue.
func (s *Service) Iniciar(ctx context.Context, log *slog.Logger, primeira time.Duration) (iniciou bool) {
	if !s.Cfg.Ativo || s.Cliente == nil {
		return false
	}
	intervalo := s.Cfg.Intervalo
	if intervalo <= 0 {
		intervalo = 24 * time.Hour
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
			if r, err := coletas.Protegida(func() (Resumo, error) { return s.Rodada(ctx) }); err != nil {
				if log != nil && ctx.Err() == nil {
					log.Error("monitor DataJud não executado", "motivo", err.Error())
				}
			} else if log != nil {
				log.Info("monitor DataJud", "status", r.Status, "novas", r.Novas, "falhas", r.Falhas)
			}
			espera = intervalo
		}
	}()
	return true
}

// Alerta diz se o monitor está ativo e sem sucesso há mais de AlertaHoras (padrão 36 h: o DataJud não é em tempo real).
func (s *Service) Alerta(ctx context.Context) (strong, texto string, ativo bool, err error) {
	if !s.Cfg.Ativo {
		return "", "", false, nil
	}
	e, existe, err := s.Coletas.Obter(ctx, Modulo)
	if err != nil {
		return "", "", false, err
	}
	horas := s.Cfg.AlertaHoras
	if horas <= 0 {
		horas = 36
	}
	const aviso = "— confira o andamento nas fontes oficiais."
	if !existe || e.UltimoSucesso == nil {
		return "Monitor do DataJud sem nenhum sucesso registrado.", aviso, true, nil
	}
	if s.now().Sub(*e.UltimoSucesso) <= time.Duration(horas)*time.Hour {
		return "", "", false, nil
	}
	return fmt.Sprintf("Monitor do DataJud sem sucesso desde %s.", e.UltimoSucesso.In(webx.SP()).Format("02/01/2006 15:04")), aviso, true, nil
}

// IniciarEstado deixa o rodapé com a situação real logo na subida (sem nenhuma requisição).
func (s *Service) IniciarEstado(ctx context.Context) error {
	for _, m := range []string{Modulo, ModuloSigilosos} {
		e, existe, err := s.Coletas.Obter(ctx, m)
		if err != nil {
			return err
		}
		agora := s.now()
		switch {
		case !s.Cfg.Ativo:
			err = s.Coletas.Registrar(ctx, estadoColeta(m, origemDJ, "bloqueado", MsgDesligada, &agora))
		case !existe:
			err = s.Coletas.Registrar(ctx, estadoColeta(m, origemDJ, "nao_sincronizado", "", nil))
		case e.Status == "bloqueado":
			err = s.Coletas.Registrar(ctx, estadoColeta(m, origemDJ, "nao_sincronizado", "Coleta ligada; aguardando a primeira rodada.", nil))
		}
		if err != nil {
			return err
		}
	}
	return nil
}
