package auth

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/csrf"
)

// HandlerFunc recebe a sessão já validada (nil nas rotas Public).
type HandlerFunc func(w http.ResponseWriter, r *http.Request, s *Session)

// Route é uma rota declarada com a permissão exigida.
type Route struct {
	Pattern string
	Perm    Perm
}

// Router é o ÚNICO jeito de registrar rota: sem permissão declarada o programa nem sobe.
type Router struct {
	a      *Auth
	mux    *http.ServeMux
	routes []Route
	envios chan struct{} // semáforo: no máximo EnviosSimultaneos arquivos sendo lidos ao mesmo tempo (auditoria 4.3)
}

// EnviosSimultaneos limita quantos arquivos grandes ficam em memória ao mesmo tempo (o contêiner tem 768 MiB).
const EnviosSimultaneos = 3

// NewRouter cria o roteador sobre o mux.
func NewRouter(a *Auth, mux *http.ServeMux) *Router {
	return &Router{a: a, mux: mux, envios: make(chan struct{}, EnviosSimultaneos)}
}

// Routes lista tudo o que foi registrado (usado por testes e auditoria de configuração).
func (rt *Router) Routes() []Route { return append([]Route(nil), rt.routes...) }

// Limites de corpo por rota (auditoria 05/11, item 1.1): a regra mora SÓ aqui. O httpx aplica apenas o teto geral.
const (
	LimiteCorpoPadrao = 1 << 20  // 1 MiB: formulários comuns (inclusive os do drawer, enviados como multipart)
	LimiteCorpoEnvio  = 26 << 20 // 25 MiB de arquivo + folga do multipart: rotas registradas com HandleUpload
)

// Handle registra a rota. perm vazia derruba o programa na inicialização (negar por padrão).
func (rt *Router) Handle(pattern string, perm Perm, h HandlerFunc) {
	rt.handle(pattern, perm, LimiteCorpoPadrao, h)
}

// HandleUpload registra uma rota que recebe ARQUIVO (até 25 MiB). Só rotas declaradas assim aceitam corpo grande.
func (rt *Router) HandleUpload(pattern string, perm Perm, h HandlerFunc) {
	if perm == Public {
		panic("rota pública não recebe arquivo: " + pattern)
	}
	rt.handle(pattern, perm, LimiteCorpoEnvio, h)
}

func (rt *Router) handle(pattern string, perm Perm, limite int64, h HandlerFunc) {
	if perm == "" {
		panic("rota sem permissão declarada: " + pattern)
	}
	rt.routes = append(rt.routes, Route{pattern, perm})
	rt.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		r = rt.a.ensureAnon(w, r)
		r.Body = http.MaxBytesReader(w, r.Body, limite)
		multipart := strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data")
		var s *Session
		if perm == Public {
			// Rotas públicas nunca recebem arquivo: recusa SEM ler o corpo (auditoria F4).
			if multipart {
				http.Error(w, "Tipo de envio não aceito.", http.StatusUnsupportedMediaType)
				return
			}
		} else {
			// Sessão e permissão ANTES de ler qualquer corpo: quem não está autorizado não consome memória.
			if s = rt.a.session(r); s == nil {
				redirectOrDeny(w, r, "/login")
				return
			}
			switch {
			case perm == MFAPending && s.MFAVerified:
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			case perm != MFAPending && perm != AnySession && !s.MFAVerified:
				redirectOrDeny(w, r, "/2fa")
				return
			case perm != MFAPending && perm != AnyLoggedIn && perm != AnySession && !Can(s.Role, perm):
				rt.a.deny(w, r, s, string(perm), http.StatusForbidden)
				return
			}
		}
		if multipart && limite >= LimiteCorpoEnvio {
			select {
			case rt.envios <- struct{}{}:
				defer func() { <-rt.envios }()
			default:
				http.Error(w, "Muitos envios de arquivo ao mesmo tempo. Tente de novo em instantes.", http.StatusServiceUnavailable)
				return
			}
		}
		if multipart && !rt.a.parseMultipart(w, r, limite) {
			return
		}
		if !rt.a.csrfOK(r) {
			rt.a.deny(w, r, s, "csrf", http.StatusForbidden)
			return
		}
		if perm == Public {
			h(w, r, nil)
			return
		}
		if s.MustChange && perm != AnyLoggedIn && perm != AnySession && perm != MFAPending {
			redirectOrDeny(w, r, "/conta/senha")
			return
		}
		h(w, r, s)
	})
}

// parseMultipart lê o corpo multipart (já depois de sessão e permissão). Devolve false se respondeu erro.
func (a *Auth) parseMultipart(w http.ResponseWriter, r *http.Request, limite int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limite) // já aplicado no início da rota; repetido para a regra ficar visível (gosec G120)
	// #nosec G120 -- o corpo já está limitado pelo MaxBytesReader da linha anterior (limite da rota); a análise do gosec não reconhece essa atribuição.
	// Tudo em memória (a imagem é somente leitura e o /tmp de produção tem 16 MiB); o semáforo de envios limita o total.
	err := r.ParseMultipartForm(limite)
	if err == nil {
		return true
	}
	var mbe *http.MaxBytesError
	switch {
	case errors.As(err, &mbe) && limite >= LimiteCorpoEnvio:
		http.Error(w, "Arquivo grande demais (o limite é 25 MiB).", http.StatusRequestEntityTooLarge)
	case errors.As(err, &mbe):
		http.Error(w, "Envio grande demais para este formulário.", http.StatusRequestEntityTooLarge)
	default:
		http.Error(w, "Envio inválido.", http.StatusBadRequest)
	}
	return false
}

func redirectOrDeny(w http.ResponseWriter, r *http.Request, to string) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		http.Redirect(w, r, to, http.StatusSeeOther)
		return
	}
	http.Error(w, "Sessão necessária.", http.StatusUnauthorized)
}

func (a *Auth) session(r *http.Request) *Session {
	c, err := r.Cookie(sidCookie)
	if err != nil {
		return nil
	}
	s, err := a.Store.LookupSession(r.Context(), c.Value, a.now())
	if err != nil {
		return nil
	}
	return s
}

// ensureAnon garante o cookie anônimo (base do CSRF antes do login) e o expõe no contexto.
func (a *Auth) ensureAnon(w http.ResponseWriter, r *http.Request) *http.Request {
	if _, err := r.Cookie(anonCookie); err == nil {
		return r
	}
	tok, err := NewToken()
	if err != nil {
		return r
	}
	setCookie(w, anonCookie, tok, 0)
	return withValue(r, anonKey{}, tok)
}

// csrfOK exige token válido em todo método que altera estado.
func (a *Auth) csrfOK(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	tok := r.Header.Get("X-CSRF-Token")
	if tok == "" {
		tok = r.PostFormValue("csrf_token")
	}
	var id string
	if c, err := r.Cookie(sidCookie); err == nil && c.Value != "" {
		id = c.Value
	} else if c, err := r.Cookie(anonCookie); err == nil {
		id = c.Value
	}
	return csrf.Valid(a.SessionKey, id, tok)
}

// deny responde 403 e audita a negativa (com ator, se houver).
func (a *Auth) deny(w http.ResponseWriter, r *http.Request, s *Session, motivo string, status int) {
	actor := "anonimo"
	if s != nil {
		actor = s.UserID
	}
	// A chave usa o PADRÃO da rota (não o caminho): variar o id na URL não cria eventos novos.
	rota := r.Pattern
	if rota == "" {
		rota = r.URL.Path
	}
	if ok, suprimidos := a.gate.allow(clientIP(r)+"|"+actor+"|"+rota+"|"+motivo, a.now()); ok {
		meta := map[string]string{"motivo": motivo, "metodo": r.Method, "rota": r.URL.Path}
		if suprimidos > 0 {
			meta["suprimidos"] = strconv.Itoa(suprimidos)
		}
		_ = a.audit(r, w, nil, audit.Event{Actor: actor, Action: "acesso.negado", Result: audit.Negado, Meta: meta})
	}
	http.Error(w, "Acesso negado.", status)
}
