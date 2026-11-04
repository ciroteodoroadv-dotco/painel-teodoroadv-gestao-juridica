package indices

import (
	"bytes"
	"context"
	"embed"
	"html/template"
	"sort"
	"time"

	"painelteodoro/internal/indices/bcb"
	"painelteodoro/internal/platform/coletas"
)

//go:embed templates/*.html
var tplFS embed.FS

var tpl = template.Must(template.ParseFS(tplFS, "templates/ticker.html"))

// ItemView é uma série no rodapé. Titulo sempre leva a data de referência (nunca valor sem data).
type ItemView struct {
	Rotulo, Valor, Titulo string
	Velho                 bool
}

// TickerView é o rodapé inteiro (forma do painel B: .idxt-item e .idxt-status).
type TickerView struct {
	Itens    []ItemView
	Ref      string
	Estado   string
	Mensagem string
	Velho    bool
}

var rotuloEstado = map[string]string{"sucesso": "sincronizado", "parcial": "parcial", "falha": "falha",
	"nao_sincronizado": "não sincronizado", "bloqueado": "bloqueado"}

func estadoRotulo(status string) string {
	if r, ok := rotuloEstado[status]; ok {
		return r
	}
	if status == "" {
		return "ausente"
	}
	return status
}

func posicao(codigo int) int {
	for i, s := range bcb.Series {
		if s.Codigo == codigo {
			return i
		}
	}
	return len(bcb.Series)
}

// Montar prepara o rodapé. Valor com referência mais antiga que validadeDias, ou cuja última coleta falhou, é marcado
// "desatualizado" (continua visível, com a data antiga). Sem valores, mostra o estado da coleta.
func Montar(vs []Valor, e coletas.Estado, existe bool, agora time.Time, validadeDias int) TickerView {
	status, msg := "nao_sincronizado", ""
	if existe {
		status, msg = e.Status, e.Mensagem
	}
	v := TickerView{Estado: estadoRotulo(status), Mensagem: msg}
	if len(vs) == 0 {
		return v
	}
	vs = append([]Valor(nil), vs...)
	sort.SliceStable(vs, func(i, j int) bool { return posicao(vs[i].Codigo) < posicao(vs[j].Codigo) })
	limite := time.Duration(validadeDias) * 24 * time.Hour
	for _, x := range vs {
		velho := x.Falhou || agora.Sub(x.Referencia) > limite
		titulo := "Ref. " + x.Referencia.Format("02/01/2006") + " · " + x.Unidade
		if velho {
			titulo += " · desatualizado"
			v.Velho = true
		}
		v.Itens = append(v.Itens, ItemView{Rotulo: x.Rotulo, Valor: Formatar(x.Valor, x.Unidade), Titulo: titulo, Velho: velho})
	}
	v.Ref = vs[0].Referencia.Format("02/01/2006")
	return v
}

// Ticker desenha o miolo do rodapé (html/template já escapa tudo).
func Ticker(v TickerView) template.HTML {
	var b bytes.Buffer
	if err := tpl.ExecuteTemplate(&b, "ticker", v); err != nil {
		return template.HTML(`<span class="idxt-status">Índices econômicos: indisponível</span>`)
	}
	return template.HTML(b.String()) // #nosec G203 -- saída de html/template, já escapada
}

// Fragmento lê o ÚLTIMO estado gravado (nunca dispara coleta) e devolve o miolo do rodapé.
func (s *Service) Fragmento(ctx context.Context) (template.HTML, error) {
	vs, err := s.Repo().Listar(ctx)
	if err != nil {
		return "", err
	}
	e, existe, err := s.Coletas.Obter(ctx, Modulo)
	if err != nil {
		return "", err
	}
	return Ticker(Montar(vs, e, existe, s.now(), s.validade())), nil
}
