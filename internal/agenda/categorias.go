package agenda

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"painelteodoro/internal/platform/webx"
)

// Categoria é uma pílula da Agenda. "Prazo fatal" NÃO é categoria: vive no módulo Prazos.
type Categoria struct {
	Chave, Rotulo string
	Tipo          string   // tipo de evento correspondente ("" para tarefas e intimações)
	Subtitulos    []string // formas normalizadas aceitas no subtítulo de tarefa
}

// Categorias são as sete pílulas, na ordem da tela.
var Categorias = []Categoria{
	{Chave: "tarefas", Rotulo: "Tarefas"},
	{Chave: "intimacoes", Rotulo: "Intimações"},
	{Chave: "audiencias", Rotulo: "Audiências", Tipo: "audiencia"},
	{Chave: "diligencias", Rotulo: "Diligências", Tipo: "diligencia"},
	{Chave: "reunioes", Rotulo: "Reuniões", Tipo: "reuniao", Subtitulos: []string{"reuniao", "reunioes"}},
	{Chave: "ligacoes", Rotulo: "Ligações", Tipo: "ligacao", Subtitulos: []string{"ligacao", "ligacoes"}},
	{Chave: "aulas", Rotulo: "Aulas", Tipo: "aula", Subtitulos: []string{"aula", "aulas"}},
}

// AvisoIntimacoes é a mensagem honesta enquanto o coletor de intimações não existe (não se inventa dado).
const AvisoIntimacoes = "As intimações do DJEN chegam com o coletor de intimações (Plano 5)."

// TarefaResumo é o que a Agenda mostra de uma tarefa. Status e PrioridadeRotulo já vêm em texto de tela.
type TarefaResumo struct {
	ID, Titulo, Subtitulo, Prioridade, PrioridadeRotulo, ProcessoNumero, Responsavel, Status string
	Vencimento                                                                               *time.Time
	AConferir                                                                                bool
}

// PrioridadeClasse devolve a classe CSS do selo de prioridade.
func (t TarefaResumo) PrioridadeClasse() string {
	if t.Prioridade == "" {
		return "prioridade-normal"
	}
	return "prioridade-" + t.Prioridade
}

// TarefaLookup é o que a Agenda precisa do domínio de tarefas (ligado no ponto de composição, em internal/app).
type TarefaLookup interface {
	AbertasPorSubtitulo(ctx context.Context, q string, limite int) ([]TarefaResumo, error)
}

var semAcento = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i", "ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u", "ç", "c", "ñ", "n")

// NormalizarSubtitulo põe em minúsculas, tira acentos e espaços das pontas ("  Ligação " -> "ligacao").
func NormalizarSubtitulo(s string) string {
	return semAcento.Replace(strings.ToLower(strings.TrimSpace(s)))
}

// Resumo é uma pílula com o contador.
type Resumo struct {
	Categoria
	Quantidade int
	Aberta     bool
	Href       string // clicar na aberta fecha; nas demais, abre
}

// PainelView é a tela principal da Agenda.
type PainelView struct {
	Hoje, HojeCurto string // "terça-feira, 3 de novembro de 2026" e "03/11/2026"
	Resumos         []Resumo
	Aberta          string // chave da categoria aberta ("" = nenhuma)
	AbertaRotulo    string
	Eventos         []Evento
	Tarefas         []TarefaResumo
	Q               string
	Aviso           string
	CanWrite        bool
}

var (
	diasSemana = [...]string{"domingo", "segunda-feira", "terça-feira", "quarta-feira", "quinta-feira", "sexta-feira", "sábado"}
	meses      = [...]string{"janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"}
)

func porExtenso(t time.Time) string {
	return diasSemana[t.Weekday()] + ", " + strconv.Itoa(t.Day()) + " de " + meses[t.Month()-1] + " de " + strconv.Itoa(t.Year())
}

func (c Categoria) aceitaSubtitulo(s string) bool {
	n := NormalizarSubtitulo(s)
	for _, x := range c.Subtitulos {
		if x == n {
			return true
		}
	}
	return false
}

func href(chave, q string, aberta bool) string {
	v := url.Values{}
	if !aberta {
		v.Set("cat", chave)
	}
	if q != "" {
		v.Set("q", q)
	}
	if len(v) == 0 {
		return "/agenda"
	}
	return "/agenda?" + v.Encode()
}

// montarPainel é a regra pura (sem banco): conta e seleciona o que cada categoria mostra.
// eventos devem vir ordenados por início; os anteriores a hoje 00:00 (São Paulo) são ignorados.
func montarPainel(cat, q string, agora time.Time, eventos []Evento, tarefas []TarefaResumo) PainelView {
	sp := agora.In(webx.SP())
	hoje := time.Date(sp.Year(), sp.Month(), sp.Day(), 0, 0, 0, 0, webx.SP())
	nq := NormalizarSubtitulo(q)
	v := PainelView{Hoje: porExtenso(sp), HojeCurto: sp.Format("02/01/2006"), Q: q}

	var evs []Evento
	for _, e := range eventos {
		if !e.Inicio.Before(hoje) && strings.Contains(NormalizarSubtitulo(e.Titulo), nq) {
			evs = append(evs, e)
		}
	}
	var tars []TarefaResumo
	for _, t := range tarefas {
		if strings.Contains(NormalizarSubtitulo(t.Titulo), nq) {
			tars = append(tars, t)
		}
	}
	for _, c := range Categorias {
		r := Resumo{Categoria: c, Aberta: c.Chave == cat}
		var seusEv []Evento
		var suasTar []TarefaResumo
		switch c.Chave {
		case "tarefas":
			suasTar = tars
		case "intimacoes":
		default:
			for _, e := range evs {
				if e.Tipo == c.Tipo {
					seusEv = append(seusEv, e)
				}
			}
			for _, t := range tars {
				if c.aceitaSubtitulo(t.Subtitulo) {
					suasTar = append(suasTar, t)
				}
			}
		}
		r.Quantidade = len(seusEv) + len(suasTar)
		r.Href = href(c.Chave, q, r.Aberta)
		if r.Aberta {
			v.Aberta, v.AbertaRotulo, v.Eventos, v.Tarefas = c.Chave, c.Rotulo, seusEv, suasTar
			if c.Chave == "intimacoes" {
				v.Aviso = AvisoIntimacoes
			}
		}
		v.Resumos = append(v.Resumos, r)
	}
	return v
}

// Painel monta a tela principal: eventos de hoje em diante e tarefas em aberto (pela interface TarefaLookup).
func (s *Service) Painel(ctx context.Context, cat, q string, agora time.Time) (PainelView, error) {
	sp := agora.In(webx.SP())
	hoje := time.Date(sp.Year(), sp.Month(), sp.Day(), 0, 0, 0, 0, webx.SP())
	eventos, err := s.List(ctx, Filtro{De: hoje.UTC()}, 1000, 0)
	if err != nil {
		return PainelView{}, err
	}
	var tarefas []TarefaResumo
	if s.Tarefas != nil {
		if tarefas, err = s.Tarefas.AbertasPorSubtitulo(ctx, "", 1000); err != nil {
			return PainelView{}, err
		}
	}
	return montarPainel(cat, q, agora, eventos, tarefas), nil
}
