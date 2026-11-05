package indices

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/indices/bcb"
	"painelteodoro/internal/platform/coletas"
)

// Fonte é o que o serviço precisa do cliente do BCB.
type Fonte interface {
	Ultimo(ctx context.Context, codigo int) (bcb.Ponto, error)
}

// Service coleta (só com Ativo) e mostra os índices.
type Service struct {
	Pool         *pgxpool.Pool
	Coletas      *coletas.Store
	Fonte        Fonte
	Ativo        bool
	Intervalo    time.Duration
	ValidadeDias int
	Now          func() time.Time
	mu           sync.Mutex
}

// Resultado de uma rodada.
type Resultado struct {
	Status, Mensagem string
	Gravadas, Falhas int
}

// Repo devolve o acesso à tabela.
func (s *Service) Repo() Repo { return Repo{Pool: s.Pool} }

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) validade() int {
	if s.ValidadeDias <= 0 {
		return 40
	}
	return s.ValidadeDias
}

func estado(status, msg string, t time.Time) coletas.Estado {
	return coletas.Estado{Modulo: Modulo, Origem: Origem, Status: status, Mensagem: msg, UltimaTentativa: &t}
}

// Coletar faz UMA requisição por série. Desligado, não faz nenhuma e deixa o estado "bloqueado". Série que falha
// mantém o valor antigo (com a data antiga e a marca "desatualizado"); nunca entra valor inventado.
func (s *Service) Coletar(ctx context.Context) (Resultado, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	agora := s.now()
	if !s.Ativo || s.Fonte == nil {
		return Resultado{Status: "bloqueado", Mensagem: MsgDesligada}, s.Coletas.Registrar(ctx, estado("bloqueado", MsgDesligada, agora))
	}
	repo := s.Repo()
	var r Resultado
	var falhas []string
	for _, sr := range bcb.Series {
		p, err := s.Fonte.Ultimo(ctx, sr.Codigo)
		if err == nil {
			err = repo.Gravar(ctx, Valor{Codigo: sr.Codigo, Rotulo: sr.Rotulo, Unidade: sr.Unidade, Valor: p.Valor, Referencia: p.Data, ObtidoEm: agora})
		}
		if err != nil {
			if cerr := ctx.Err(); cerr != nil {
				return r, cerr
			}
			r.Falhas++
			falhas = append(falhas, sr.Rotulo)
			if merr := repo.MarcarFalha(ctx, sr.Codigo); merr != nil {
				return r, merr
			}
			continue
		}
		r.Gravadas++
	}
	switch {
	case r.Falhas == 0:
		r.Status = "sucesso"
		r.Mensagem = fmt.Sprintf("%d séries atualizadas.", r.Gravadas)
	case r.Gravadas == 0:
		r.Status = "falha"
		r.Mensagem = "Nenhuma série foi atualizada. Os valores antigos ficam marcados como desatualizados."
	default:
		r.Status = "parcial"
		r.Mensagem = "Sem atualização: " + strings.Join(falhas, ", ") + ". Esses valores ficam marcados como desatualizados."
	}
	return r, s.Coletas.Registrar(ctx, estado(r.Status, r.Mensagem, agora))
}

// IniciarEstado deixa o rodapé com a situação real logo na subida (sem nenhuma requisição).
func (s *Service) IniciarEstado(ctx context.Context) error {
	e, existe, err := s.Coletas.Obter(ctx, Modulo)
	if err != nil {
		return err
	}
	agora := s.now()
	switch {
	case !s.Ativo:
		return s.Coletas.Registrar(ctx, estado("bloqueado", MsgDesligada, agora))
	case !existe:
		return s.Coletas.Registrar(ctx, estado("nao_sincronizado", "", agora))
	case e.Status == "bloqueado":
		return s.Coletas.Registrar(ctx, estado("nao_sincronizado", "Coleta ligada; aguardando a primeira rodada.", agora))
	}
	return nil
}

// Iniciar liga o agendador (1 rodada por Intervalo, padrão 24 h). Só existe com a coleta ATIVA: desligada, sem goroutine e sem rede.
func (s *Service) Iniciar(ctx context.Context, log *slog.Logger, primeira time.Duration) bool {
	if !s.Ativo || s.Fonte == nil {
		return false
	}
	intervalo := s.Intervalo
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
			if r, err := coletas.Protegida(func() (Resultado, error) { return s.Coletar(ctx) }); err != nil {
				if log != nil && ctx.Err() == nil {
					log.Error("índices do BCB não coletados")
				}
			} else if log != nil {
				log.Info("índices do BCB", "status", r.Status, "gravadas", r.Gravadas, "falhas", r.Falhas)
			}
			espera = intervalo
		}
	}()
	return true
}
