package cfaccess

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type ctxKey struct{}

// Email devolve o e-mail validado pelo Access (vazio no modo dev).
func Email(ctx context.Context) string { s, _ := ctx.Value(ctxKey{}).(string); return s }

// WithEmail marca o contexto como já validado pelo Access (usado em testes e por quem valida por outro caminho).
func WithEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, ctxKey{}, strings.ToLower(email))
}

// HTTPFetcher busca o JWKS em https://<time>/cdn-cgi/access/certs.
func HTTPFetcher(teamDomain string) Fetcher {
	c := &http.Client{Timeout: 5 * time.Second}
	return func(ctx context.Context) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+teamDomain+"/cdn-cgi/access/certs", nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	}
}

// Middleware exige JWT válido em tudo, exceto /healthz e /readyz quando a chamada NÃO veio
// pelo Cloudflare (sem Cf-Ray): o healthcheck do Docker não passa pelo Access, mas o mundo externo
// nunca alcança essas rotas pelo túnel. Com v == nil (modo dev) nada é exigido.
func Middleware(v *Verifier, log *slog.Logger, next http.Handler) http.Handler {
	if v == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internal := (r.URL.Path == "/healthz" || r.URL.Path == "/readyz") && r.Header.Get("Cf-Ray") == ""
		if internal {
			next.ServeHTTP(w, r)
			return
		}
		email, err := v.Verify(r.Context(), r.Header.Get("Cf-Access-Jwt-Assertion"))
		if err != nil {
			log.Warn("acesso recusado", "motivo", err.Error(), "caminho", r.URL.Path)
			http.Error(w, "Acesso negado.", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, strings.ToLower(email))))
	})
}
