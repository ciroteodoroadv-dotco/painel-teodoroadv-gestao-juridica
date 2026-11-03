package intimacoes

import (
	"context"
	"fmt"
	"time"

	"painelteodoro/internal/platform/webx"
)

// Lista implementa a categoria Intimações da Agenda: TODAS as intimações (o histórico), da mais recente para a mais antiga.
// disponivel=false quando não há o que mostrar com honestidade: sem nenhuma intimação e sem nenhuma coleta concluída,
// a tela não diz "nenhuma intimação" (isso soaria como "nenhum prazo"): diz o estado real da última coleta.
func (s *Service) Lista(ctx context.Context) (linhas []Linha, disponivel bool, mensagem string, err error) {
	rows, err := s.Pool.Query(ctx, `SELECT id::text, origem, processo_numero, data_publicacao, status, tipo, orgao, tribunal
		FROM intimacoes ORDER BY data_publicacao DESC NULLS LAST, created_at DESC, id`)
	if err != nil {
		return nil, false, "", err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var l Linha
		var data *time.Time
		var tipo, orgao, tribunal string
		if err = rows.Scan(&l.ID, &l.Origem, &l.Processo, &data, &l.Status, &tipo, &orgao, &tribunal); err != nil {
			return nil, false, "", err
		}
		if data != nil {
			l.Data = webx.DataBR(*data)
		} else {
			l.Data = "—"
		}
		l.StatusRotulo, l.StatusClasse = statusRotulo(l.Status), statusClasse(l.Status)
		l.Busca = webx.SemAcento(l.Origem + " " + l.Processo + " " + l.Data + " " + l.StatusRotulo + " " + tipo + " " + orgao + " " + tribunal)
		ids = append(ids, l.ID)
		linhas = append(linhas, l)
	}
	if err = rows.Err(); err != nil {
		return nil, false, "", err
	}
	if len(ids) > 0 && s.TarefasDe != nil {
		m, err := s.TarefasDe.PorIntimacoes(ctx, ids)
		if err != nil {
			return nil, false, "", err
		}
		for i := range linhas {
			if t, ok := m[linhas[i].ID]; ok {
				linhas[i].Tarefa, linhas[i].TarefaEstado = t.Titulo, t.Estado
				linhas[i].Busca += " " + webx.SemAcento(t.Titulo)
			}
		}
	}
	e, existe, err := s.Coletas.Obter(ctx, Modulo)
	if err != nil {
		return nil, false, "", err
	}
	if len(linhas) > 0 || (existe && e.Status == "sucesso") {
		return linhas, true, "", nil
	}
	if existe && e.Mensagem != "" {
		return nil, false, e.Mensagem + " Confira o DJEN manualmente.", nil
	}
	return nil, false, "Nenhuma sincronização de intimações foi executada. Confira o DJEN manualmente.", nil
}

// Alerta diz se a coleta está ativa e sem sucesso há mais de AlertaHoras (heartbeat). Devolve o texto da faixa .notice.
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
		horas = 6
	}
	const aviso = "— confira o DJEN manualmente."
	if !existe || e.UltimoSucesso == nil {
		return "Coleta de intimações sem nenhum sucesso registrado.", aviso, true, nil
	}
	if s.now().Sub(*e.UltimoSucesso) <= time.Duration(horas)*time.Hour {
		return "", "", false, nil
	}
	desde := e.UltimoSucesso.In(webx.SP()).Format("02/01/2006 15:04")
	return fmt.Sprintf("Coleta de intimações sem sucesso desde %s.", desde), aviso, true, nil
}

// IniciarEstado deixa o rodapé com a situação real logo na subida: desligada → "bloqueado"; ligada e nunca tentada →
// "sem tentativa registrada". Não faz nenhuma requisição.
func (s *Service) IniciarEstado(ctx context.Context) error {
	e, existe, err := s.Coletas.Obter(ctx, Modulo)
	if err != nil {
		return err
	}
	switch {
	case !s.Cfg.Ativo:
		agora := s.now()
		return s.Coletas.Registrar(ctx, coletasEstado("bloqueado", MsgDesligada, &agora))
	case !existe:
		return s.Coletas.Registrar(ctx, coletasEstado("nao_sincronizado", "", nil))
	case e.Status == "bloqueado": // estava desligada e foi ligada: não deixa o aviso velho
		return s.Coletas.Registrar(ctx, coletasEstado("nao_sincronizado", "Coleta ligada; aguardando a primeira rodada.", nil))
	}
	return nil
}
