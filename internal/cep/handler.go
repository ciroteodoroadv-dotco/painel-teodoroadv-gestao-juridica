package cep

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/ratelimit"
)

// MsgDesligada é o texto quando a consulta está desligada neste ambiente.
const MsgDesligada = "Consulta de CEP desligada neste ambiente."

// MsgLimite é o texto quando o usuário passou do limite de consultas.
const MsgLimite = "Muitas consultas de CEP em pouco tempo. Aguarde um minuto."

// Buscador é o que o handler precisa do cliente.
type Buscador interface {
	Buscar(ctx context.Context, cep string) (Endereco, error)
}

const (
	validadeCache = 10 * time.Minute
	maxCache      = 500
)

type itemCache struct {
	e   Endereco
	ate time.Time
}

// Handler é a rota autenticada que o formulário de pessoa chama. O navegador nunca fala com o ViaCEP.
type Handler struct {
	A      *auth.Auth
	Ativo  bool
	B      Buscador
	Limite *ratelimit.Limiter // por usuário
	Agora  func() time.Time

	mu    sync.Mutex
	cache map[string]itemCache
}

// Register declara a rota: exige sessão e escrita em Pessoas (e CSRF, por ser POST).
func (h *Handler) Register(rt *auth.Router) {
	rt.Handle("POST /pessoas/cep", auth.PermPessoasWrite, h.buscar)
}

type saida struct {
	Logradouro string `json:"logradouro,omitempty"`
	Bairro     string `json:"bairro,omitempty"`
	Municipio  string `json:"municipio,omitempty"`
	UF         string `json:"uf,omitempty"`
	Fonte      string `json:"fonte,omitempty"`
	Erro       string `json:"erro,omitempty"`
}

func responder(w http.ResponseWriter, status int, s saida) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(s)
}

func (h *Handler) agora() time.Time {
	if h.Agora != nil {
		return h.Agora()
	}
	return time.Now()
}

func (h *Handler) doCache(cep string) (Endereco, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	it, ok := h.cache[cep]
	if !ok || h.agora().After(it.ate) {
		delete(h.cache, cep)
		return Endereco{}, false
	}
	return it.e, true
}

func (h *Handler) guardar(cep string, e Endereco) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cache == nil || len(h.cache) >= maxCache {
		h.cache = map[string]itemCache{}
	}
	h.cache[cep] = itemCache{e: e, ate: h.agora().Add(validadeCache)}
}

// buscar nunca grava o CEP em log: a mensagem de erro mostra só o motivo.
func (h *Handler) buscar(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	cep := Digitos(r.PostFormValue("cep"))
	if len(cep) != 8 {
		responder(w, http.StatusBadRequest, saida{Erro: ErrCEPInvalido.Error()})
		return
	}
	if !h.Ativo || h.B == nil {
		responder(w, http.StatusServiceUnavailable, saida{Erro: MsgDesligada})
		return
	}
	if h.Limite != nil && !h.Limite.Allow(s.UserID) {
		responder(w, http.StatusTooManyRequests, saida{Erro: MsgLimite})
		return
	}
	e, ok := h.doCache(cep)
	if !ok {
		var err error
		if e, err = h.B.Buscar(r.Context(), cep); err != nil {
			switch {
			case errors.Is(err, ErrNaoEncontrado):
				responder(w, http.StatusNotFound, saida{Erro: ErrNaoEncontrado.Error()})
			default:
				responder(w, http.StatusBadGateway, saida{Erro: ErrIndisponivel.Error()})
			}
			return
		}
		h.guardar(cep, e)
	}
	responder(w, http.StatusOK, saida{Logradouro: e.Logradouro, Bairro: e.Bairro, Municipio: e.Municipio, UF: e.UF, Fonte: Fonte})
}
