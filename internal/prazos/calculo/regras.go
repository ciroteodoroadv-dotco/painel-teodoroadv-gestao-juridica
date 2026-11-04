// Package calculo calcula o vencimento de prazos processuais e materiais. É PURO: não lê banco nem relógio;
// recebe a data de hoje e o calendário. O resultado é sempre apoio ("A CONFERIR"): a regra jurídica está em
// docs/juridico/REGRAS-DE-PRAZO.md e quem valida é o advogado.
package calculo

import "time"

// Regra é a forma de contar o prazo.
type Regra string

const (
	CPCUteis      Regra = "cpc-uteis"
	CPPCorridos   Regra = "cpp-corridos"
	MaterialCivel Regra = "material-civel"
	MaterialPenal Regra = "material-penal"
)

// Modo diz de onde nasce o termo inicial.
type Modo string

const (
	DJe     Modo = "dje"
	Ciencia Modo = "ciencia"
)

// DiaNaoUtil é um dia excluído da contagem, com a fonte que o justifica.
type DiaNaoUtil struct {
	Data                   time.Time
	Tipo, Descricao, Fonte string
}

// Calendario é o calendário de um tribunal. Sábado e domingo são tratados dentro do cálculo.
// Só devolve dia que tenha fonte registrada.
type Calendario interface {
	NaoUtil(d time.Time) (DiaNaoUtil, bool)
}

// Cobertor é opcional: diz se o calendário tem algum dia cadastrado para o ano.
type Cobertor interface{ Cobre(ano int) bool }

// Entrada reúne tudo de que o cálculo precisa. Hoje é informado pelo chamador (fuso America/Sao_Paulo).
type Entrada struct {
	Evento time.Time
	Modo   Modo
	Regra  Regra
	Dias   int
	Cal    Calendario // nil = só feriados nacionais
	Hoje   time.Time
}

// Resultado é o vencimento calculado com todas as premissas.
type Resultado struct {
	Publicacao, Inicio, Vencimento *time.Time
	DiasUteisAteVencimento         int
	Premissas                      []string
	Excluidos                      []DiaNaoUtil
	Alternativas                   []string
	NaoConsiderado                 []string
	Duvidas                        []string
}

var (
	ErrDias   error = msgErro("Número de dias inválido.")
	ErrEvento error = msgErro("Data do evento inválida.")
	ErrRegra  error = msgErro("Regra do prazo desconhecida.")
	ErrModo   error = msgErro("Modo de contagem desconhecido.")
)

// msgErro é um erro cujo texto é a frase mostrada ao usuário (por isso começa com maiúscula e termina com ponto).
type msgErro string

func (e msgErro) Error() string { return string(e) }

// MaxDias é o teto aceito (evita laço sem fim por digitação errada).
const MaxDias = 3650

// NaoConsideradoFixo é a lista fixa do que o cálculo NÃO considera (aparece em todo resultado).
var NaoConsideradoFixo = []string{
	"termo inicial por juntada de AR/mandado, citação, carga dos autos ou audiência",
	"consulta tácita (10 dias) da intimação eletrônica",
	"indisponibilidade do sistema do tribunal",
	"suspensões de prazo por portaria não cadastradas no calendário",
	"ponto facultativo não cadastrado com fonte",
	"feriado municipal ou estadual não cadastrado",
	"prazo em dobro (CPC arts. 180, 183, 186 e 229): o painel não o aplica; se o caso o exigir, o advogado ajusta manualmente",
	"prazos em horas, meses ou anos",
}
