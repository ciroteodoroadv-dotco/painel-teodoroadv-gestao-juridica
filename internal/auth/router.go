package auth

import (
	"errors"
	"net/http"
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
}

// NewRouter cria o roteador sobre o mux.
func NewRouter(a *Auth, mux *http.ServeMux) *Router { return &Router{a: a, mux: mux} }

// Routes lista tudo o que foi registrado (usado por testes e auditoria de configuração).
func (rt *Router) Routes() []Route { return append([]Route(nil), rt.routes...) }

// Handle registra a rota. perm vazia derruba o programa na inicialização (negar por padrão).
func (rt *Router) Handle(pattern string, perm Perm, h HandlerFunc) {
	if perm == "" {
		panic("rota sem permissão declarada: " + pattern)
	}
	rt.routes = append(rt.routes, Route{pattern, perm})
	rt.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		r = rt.a.ensureAnon(w, r)
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			if err := r.ParseMultipartForm(26 << 20); err != nil {
				var mbe *http.MaxBytesError
				if errors.As(err, &mbe) {
					http.Error(w, "Arquivo grande demais (o limite é 25 MiB).", http.StatusRequestEntityTooLarge)
				} else {
					http.Error(w, "Envio inválido.", http.StatusBadRequest)
				}
				return
			}
		}
		if !rt.a.csrfOK(r) {
			rt.a.deny(w, r, nil, "csrf", http.StatusForbidden)
			return
		}
		if perm == Public {
			h(w, r, nil)
			return
		}
		s := rt.a.session(r)
		if s == nil {
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
		if s.MustChange && perm != AnyLoggedIn && perm != AnySession && perm != MFAPending {
			redirectOrDeny(w, r, "/conta/senha")
			return
		}
		h(w, r, s)
	})
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
	_ = a.audit(r, w, nil, audit.Event{Actor: actor, Action: "acesso.negado", Result: audit.Negado,
		Meta: map[string]string{"motivo": motivo, "metodo": r.Method, "rota": r.URL.Path}})
	http.Error(w, "Acesso negado.", status)
}
