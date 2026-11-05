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

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/cfaccess"
	"painelteodoro/internal/platform/coletas"
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
	// WA é nil quando a chave física está desligada (sem PAINEL_PUBLIC_URL válido).
	WA      *webauthn.WebAuthn
	cer     ceremonies
	fl      flash
	nav     []navItem
	faixas  []FaixaFunc
	widgets []WidgetFunc
	sync    func(context.Context) ([]coletas.Estado, error)
	ticker  func(context.Context) (template.HTML, error)
	gate    denyGate
	cadeia  pontoDaCadeia
}

func (a *Auth) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

// PageData alimenta todos os templates.
type PageData struct {
	Title, CSRF, Error, Email, Role string
	Secret, Message                 string
	Codes                           []string
	Events                          []audit.Row
	ChainOK                         bool
	ChainMsg                        string
	CanAudit                        bool
	Next                            string
	HasTOTP, HasKey, MustChange     bool
	Keys                            []KeyInfo
	Users                           []UserRow
	Notice                          string
	CanUsers                        bool
	Nav, NavSistema                 []NavLink
	Head                            Head
	Tab                             string
	Faixas                          []Faixa
	Widgets                         []Widget
	Sync                            SyncView
	Ticker                          template.HTML // miolo do rodapé de índices (vem do domínio indices)
	Data                            any           // dados específicos da página de um domínio
}

// Widget é um bloco da tela "Hoje" fornecido por um domínio (HTML já escapado pelo próprio domínio).
type Widget struct {
	Title string
	Body  template.HTML
}

// WidgetFunc produz o bloco de um domínio para a tela "Hoje".
type WidgetFunc func(ctx context.Context, s *Session) (Widget, error)

// AddWidget registra um bloco da tela "Hoje".
func (a *Auth) AddWidget(f WidgetFunc) { a.widgets = append(a.widgets, f) }

var baseTpl = template.Must(template.ParseFS(tplFS, "templates/base.html"))

// CSRFFor devolve o token CSRF da requisição (para fragmentos htmx que contêm formulários).
func (a *Auth) CSRFFor(r *http.Request) string { return csrf.Token(a.SessionKey, csrfID(r)) }

// BaseTemplate devolve uma cópia do layout para um domínio acrescentar o seu "content".
func BaseTemplate() *template.Template { return template.Must(baseTpl.Clone()) }

// Render desenha uma página de domínio dentro do layout comum.
func (a *Auth) Render(w http.ResponseWriter, r *http.Request, status int, t *template.Template, d PageData) {
	d.CSRF = csrf.Token(a.SessionKey, csrfID(r))
	d.CanAudit = Can(Role(d.Role), PermAuditRead)
	d.CanUsers = Can(Role(d.Role), PermUsersManage)
	d.Nav, d.NavSistema = a.menu(r.Context(), Role(d.Role), r.URL.Path)
	if d.Role != "" {
		d.NavSistema = sistemaFixos(d.NavSistema, Role(d.Role), r.URL.Path)
	}
	d.Tab = tabOf(r.URL.Path)
	if d.Role != "" {
		d.Sync = a.syncView(r.Context())
		d.Ticker = a.tickerView(r.Context())
	}
	if d.Head.Title == "" {
		d.Head.Title = d.Title
	}
	if d.Role != "" && d.Head.Chip == "" && d.Tab == "agenda" {
		d.Head.Chip = hojeISO(a.now())
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := t.ExecuteTemplate(w, "base", d); err != nil {
		a.Log.Error("falha ao renderizar página de domínio")
	}
}

func (a *Auth) render(w http.ResponseWriter, r *http.Request, status int, page string, d PageData) {
	t, err := template.ParseFS(tplFS, "templates/base.html", "templates/"+page)
	if err != nil {
		a.Log.Error("template inválido", "pagina", page)
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	a.Render(w, r, status, t, d)
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

// clientIP só confia em Cf-Connecting-Ip quando o JWT do Cloudflare Access foi validado nesta requisição
// (então o cabeçalho é do Cloudflare). Sem isso — modo dev, por exemplo — o cabeçalho é forjável e é ignorado.
func clientIP(r *http.Request) string {
	if cfaccess.Email(r.Context()) != "" {
		if ip := net.ParseIP(strings.TrimSpace(r.Header.Get("Cf-Connecting-Ip"))); ip != nil {
			return ip.String()
		}
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

// denyGate limita a auditoria de negativas (403/CSRF): no máximo 1 evento por origem+rota+motivo por minuto,
// com contagem das suprimidas no evento seguinte (auditoria F5). A negativa em si NUNCA é suprimida.
type denyGate struct {
	mu sync.Mutex
	m  map[string]*gateEntry
}

type gateEntry struct {
	last       time.Time
	suppressed int
}

const denyWindow = time.Minute

// allow diz se este evento deve ser gravado e quantos foram suprimidos desde o último gravado.
func (g *denyGate) allow(key string, now time.Time) (bool, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.m == nil {
		g.m = map[string]*gateEntry{}
	}
	if len(g.m) > 10000 {
		for k, v := range g.m {
			if now.Sub(v.last) > denyWindow {
				delete(g.m, k)
			}
		}
	}
	e := g.m[key]
	if e == nil || now.Sub(e.last) >= denyWindow {
		n := 0
		if e != nil {
			n = e.suppressed
		}
		if e != nil || len(g.m) <= 10000 { // nunca deixa o mapa crescer sem limite
			g.m[key] = &gateEntry{last: now}
		}
		return true, n
	}
	e.suppressed++
	return false, 0
}
