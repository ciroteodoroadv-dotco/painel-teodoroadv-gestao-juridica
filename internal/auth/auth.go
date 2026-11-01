package auth

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/csrf"
	"painelteodoro/internal/platform/ratelimit"
)

const (
	sidCookie  = "__Host-sid"
	anonCookie = "__Host-anon"
)

//go:embed templates/*.html
var tplFS embed.FS

// Auth reúne as dependências de identidade e as telas de acesso.
type Auth struct {
	Store      *Store
	Pool       *pgxpool.Pool
	FieldKey   []byte
	SessionKey []byte
	Log        *slog.Logger
	Limiter    *ratelimit.Limiter
	// RequireAccessEmail exige que o e-mail do login seja o mesmo do JWT do Cloudflare Access.
	RequireAccessEmail bool
	Now                func() time.Time
}

func (a *Auth) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

// pageData alimenta todos os templates.
type pageData struct {
	Title, CSRF, Error, Email, Role string
	Secret, Message                 string
	Codes                           []string
	Events                          []audit.Row
	ChainOK                         bool
	ChainMsg                        string
	CanAudit                        bool
}

func (a *Auth) render(w http.ResponseWriter, r *http.Request, status int, page string, d pageData) {
	t, err := template.ParseFS(tplFS, "templates/base.html", "templates/"+page)
	if err != nil {
		a.Log.Error("template inválido", "pagina", page)
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	d.CSRF = csrf.Token(a.SessionKey, csrfID(r))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := t.ExecuteTemplate(w, "base", d); err != nil {
		a.Log.Error("falha ao renderizar", "pagina", page)
	}
}

// audit grava um evento com o ator e o request_id da requisição; falha de auditoria é registrada em log.
func (a *Auth) audit(r *http.Request, w http.ResponseWriter, tx pgx.Tx, ev audit.Event) error {
	if ev.Actor == "" {
		ev.Actor = "anonimo"
	}
	ev.RequestID = w.Header().Get("X-Request-Id")
	var err error
	if tx != nil {
		err = audit.Record(r.Context(), tx, ev)
	} else {
		err = audit.RecordPool(r.Context(), a.Pool, ev)
	}
	if err != nil {
		a.Log.Error("falha ao gravar auditoria", "acao", ev.Action)
	}
	return err
}

func emailHash(email string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(h[:8])
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("Cf-Connecting-Ip"); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func setCookie(w http.ResponseWriter, name, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", MaxAge: maxAge,
		Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode})
}

// csrfID é o identificador ao qual o token CSRF se liga: a sessão, ou o cookie anônimo antes do login.
func csrfID(r *http.Request) string {
	if v, ok := r.Context().Value(sidKey{}).(string); ok {
		return v // sessão recém-criada nesta resposta
	}
	if c, err := r.Cookie(sidCookie); err == nil && c.Value != "" {
		return c.Value
	}
	if v, ok := r.Context().Value(anonKey{}).(string); ok {
		return v
	}
	if c, err := r.Cookie(anonCookie); err == nil {
		return c.Value
	}
	return ""
}

type (
	anonKey struct{}
	sidKey  struct{}
)

var (
	dummyOnce sync.Once
	dummyHash string
)

// burnPassword gasta o mesmo tempo de uma verificação real, para não revelar se o e-mail existe.
func burnPassword(pw string) {
	dummyOnce.Do(func() { dummyHash, _ = HashPassword("senha-descartavel-para-igualar-tempo") })
	VerifyPassword(pw, dummyHash)
}

func withValue(r *http.Request, k, v any) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), k, v))
}
