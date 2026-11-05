package tarefas

import (
	"context"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/platform/webx"
)

func descricaoDoPrazo(t *Tarefa) string {
	d := t.Codigo + " — " + t.Titulo
	if utf8.RuneCountInString(d) > 500 {
		d = string([]rune(d)[:500])
	}
	return d
}

// efeitoNoPrazo aplica à lei do domínio Prazos o que mudou na tarefa, DENTRO da transação da tarefa. antes é nil na criação.
//   - estado final (concluída, cancelada): o prazo é encerrado (cumprido ou cancelado);
//   - tarefa aberta com prazo fatal: sincroniza (o domínio Prazos não duplica; nasce/permanece "a conferir");
//   - tarefa aberta sem prazo fatal, que antes tinha: o prazo é cancelado.
//
// O sistema nunca calcula a data: ela é a que a pessoa digitou. E nunca confirma: confirmar é ato explícito do advogado.
func (s *Service) efeitoNoPrazo(ctx context.Context, tx pgx.Tx, ac webx.Actor, t, antes *Tarefa, fatal *time.Time) error {
	tinha := antes != nil && antes.PrazoFatal != nil
	switch {
	case !Aberto(t.Estado):
		if (tinha || antes == nil && fatal != nil) && (antes == nil || antes.Estado != t.Estado) {
			return s.Prazos.EncerrarTx(ctx, tx, ac, t.ID, motivoDeEncerramento(t.Estado))
		}
	case fatal != nil:
		proc := ""
		if t.OrigemTipo == "processo" {
			proc = t.OrigemID
		}
		_, err := s.Prazos.SincronizarTx(ctx, tx, ac, t.ID, proc, descricaoDoPrazo(t), fatal)
		return err
	case tinha:
		return s.Prazos.EncerrarTx(ctx, tx, ac, t.ID, "cancelada")
	}
	return nil
}
