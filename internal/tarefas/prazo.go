package tarefas

import (
	"context"
	"time"
	"unicode/utf8"

	"painelteodoro/internal/platform/webx"
)

func descricaoDoPrazo(t *Tarefa) string {
	d := t.Codigo + " — " + t.Titulo
	if utf8.RuneCountInString(d) > 500 {
		d = string([]rune(d)[:500])
	}
	return d
}

// efeitoNoPrazo aplica à lei do domínio Prazos o que mudou na tarefa. antes é nil na criação.
//   - estado final (concluída, cancelada): o prazo é encerrado (cumprido ou cancelado);
//   - tarefa aberta com prazo fatal: sincroniza (o domínio Prazos não duplica; só advogado confirma);
//   - tarefa aberta sem prazo fatal, que antes tinha: o prazo é cancelado.
//
// O sistema nunca calcula a data: ela é a que a pessoa digitou.
func (s *Service) efeitoNoPrazo(ctx context.Context, ac webx.Actor, t, antes *Tarefa, fatal *time.Time, confirmar bool) error {
	fim := func(err error) error {
		if err != nil {
			return webx.Invalidf("A tarefa foi salva, mas o prazo fatal não foi registrado no domínio Prazos. Abra a tarefa e salve de novo.")
		}
		return nil
	}
	tinha := antes != nil && antes.PrazoFatal != nil
	switch {
	case !Aberto(t.Estado):
		if (tinha || antes == nil && fatal != nil) && (antes == nil || antes.Estado != t.Estado) {
			return fim(s.Prazos.Encerrar(ctx, ac, t.ID, motivoDeEncerramento(t.Estado)))
		}
	case fatal != nil:
		proc := ""
		if t.OrigemTipo == "processo" {
			proc = t.OrigemID
		}
		_, err := s.Prazos.Sincronizar(ctx, ac, t.ID, proc, descricaoDoPrazo(t), fatal, confirmar)
		return fim(err)
	case tinha:
		return fim(s.Prazos.Encerrar(ctx, ac, t.ID, "cancelada"))
	}
	return nil
}
