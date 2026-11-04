package monitor

import (
	"context"
	"sort"
	"time"

	"painelteodoro/internal/monitor/cnj"
	"painelteodoro/internal/platform/webx"
)

// LinhaProcesso é um processo na tabela "Monitor de processos (DataJud)".
type LinhaProcesso struct {
	ProcessoID, Numero, Tribunal, Resultado, Classe, Mensagem, UltimaConsulta, Base string
}

// LinhaSigiloso é um processo em segredo de justiça na tabela.
type LinhaSigiloso struct {
	ID, Numero, OAB, Situacao, Classe string
	Ativo                             bool
}

// Opcoes para o drawer de cadastro de sigiloso.
type Opcoes struct {
	Processos []struct{ ID, Rotulo string }
	OABs      []struct{ ID, Rotulo string }
}

var resultados = map[string][2]string{
	"sucesso":          {"Concluída", "badge-ok"},
	"parcial":          {"Parcial", "badge-warn"},
	"falha":            {"Erro", "badge-danger"},
	"bloqueado":        {"Bloqueado", "badge-neutral"},
	"nao_sincronizado": {"Sem tentativa registrada", "badge-neutral"},
}

type estadoProc struct {
	alias, status, msg string
	consulta, base     *time.Time
}

func (s *Service) estados(ctx context.Context) (map[string]estadoProc, error) {
	rows, err := s.Pool.Query(ctx, `SELECT processo_id::text, tribunal_alias, status, mensagem, ultima_consulta, base_registrada_em FROM monitor_processos`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]estadoProc{}
	for rows.Next() {
		var id string
		var e estadoProc
		if err := rows.Scan(&id, &e.alias, &e.status, &e.msg, &e.consulta, &e.base); err != nil {
			return nil, err
		}
		out[id] = e
	}
	return out, rows.Err()
}

func dataHora(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return t.In(webx.SP()).Format("02/01/2006 15:04")
}

// Linhas devolve, por processo cadastrado, o estado da última consulta (processos sem número válido dizem isso).
func (s *Service) Linhas(ctx context.Context) ([]LinhaProcesso, error) {
	procs, err := s.Processos.Monitoraveis(ctx)
	if err != nil {
		return nil, err
	}
	est, err := s.estados(ctx)
	if err != nil {
		return nil, err
	}
	var out []LinhaProcesso
	for _, p := range procs {
		l := LinhaProcesso{ProcessoID: p.ID, Numero: p.Numero, Tribunal: "—", Resultado: "Sem tentativa registrada", Classe: "badge-neutral", UltimaConsulta: "—", Base: "—"}
		if l.Numero == "" {
			l.Numero = "(sem número)"
		}
		if !cnj.Valido(p.Numero) {
			l.Resultado, l.Mensagem = "Bloqueado", MsgSemNumero
		} else if a, err := cnj.AliasDataJud(p.Numero); err == nil {
			l.Tribunal = a
		}
		if e, ok := est[p.ID]; ok {
			r := resultados[e.status]
			l.Resultado, l.Classe, l.Mensagem, l.UltimaConsulta = r[0], r[1], e.msg, dataHora(e.consulta)
			if e.alias != "" {
				l.Tribunal = e.alias
			}
			if e.base != nil {
				l.Base = "Linha de base registrada em " + webx.DataBR(webx.Hoje(*e.base))
			}
		}
		out = append(out, l)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Numero < out[j].Numero })
	return out, nil
}

// LinhasSigilosos devolve os processos em segredo de justiça cadastrados.
func (s *Service) LinhasSigilosos(ctx context.Context) ([]LinhaSigiloso, error) {
	sig, err := s.ListarSigilosos(ctx)
	if err != nil {
		return nil, err
	}
	var out []LinhaSigiloso
	for _, x := range sig {
		l := LinhaSigiloso{ID: x.ID, Numero: "—", OAB: "—", Ativo: x.Ativo, Situacao: "Ativo", Classe: "badge-ok"}
		if !x.Ativo {
			l.Situacao, l.Classe = "Inativo", "badge-neutral"
		}
		if p, ok, _ := s.Processos.Ref(ctx, x.ProcessoID); ok {
			l.Numero = cnj.Formatar(p.Numero)
		}
		if o, err := s.OABs.Obter(ctx, x.OabID); err == nil {
			l.OAB = o.Numero + "/" + o.UF
		}
		out = append(out, l)
	}
	return out, nil
}

// OpcoesSigiloso lista os processos com número válido e as OABs ativas com autorização.
func (s *Service) OpcoesSigiloso(ctx context.Context) (Opcoes, error) {
	var o Opcoes
	procs, err := s.Processos.Monitoraveis(ctx)
	if err != nil {
		return o, err
	}
	for _, p := range procs {
		if cnj.Valido(p.Numero) {
			o.Processos = append(o.Processos, struct{ ID, Rotulo string }{p.ID, cnj.Formatar(p.Numero)})
		}
	}
	oabs, err := s.OABs.Listar(ctx)
	if err != nil {
		return o, err
	}
	for _, b := range oabs {
		if b.Ativa && b.AutorizacaoBase != "" {
			o.OABs = append(o.OABs, struct{ ID, Rotulo string }{b.ID, b.Numero + "/" + b.UF})
		}
	}
	return o, nil
}
