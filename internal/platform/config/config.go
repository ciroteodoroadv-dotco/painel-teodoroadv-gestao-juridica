// Package config carrega a configuração e FALHA se faltar algum segredo.
// Segredos vêm de variável de ambiente ou de arquivo (VARIAVEL_FILE, padrão Docker secrets).
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

const minKeyBytes = 32

// Config reúne o que o app precisa para subir. Nunca imprima seus campos de segredo.
type Config struct {
	Addr     string
	LogLevel string
	// Dev desliga a exigência do JWT do Cloudflare Access. Só vale com PAINEL_ENV=dev.
	Dev              bool
	AccessTeamDomain string
	AccessAUD        string
	// PublicURL é o endereço pelo qual o navegador acessa o painel (define o domínio da chave de segurança).
	PublicURL string
	// DocsDir é a pasta privada (cifrada) dos documentos enviados.
	DocsDir     string
	DatabaseURL string // conexão do DONO do banco (migrações); contém a senha
	// AppDatabaseURL é a conexão do papel restrito usado em tempo de execução; contém a senha.
	AppDatabaseURL string
	AppDBPassword  string
	SessionKey     []byte
	FieldKey       []byte
}

// String é seguro para log: não revela nenhum segredo.
func (c Config) String() string {
	return fmt.Sprintf("Config{Addr:%s LogLevel:%s segredos:[redigido]}", c.Addr, c.LogLevel)
}

// Env abstrai o ambiente para permitir teste sem tocar no sistema real.
type Env struct {
	Getenv   func(string) string
	ReadFile func(string) ([]byte, error)
}

// OSEnv usa o ambiente e o disco reais.
func OSEnv() Env { return Env{Getenv: os.Getenv, ReadFile: os.ReadFile} }

// Load lê toda a configuração. Reúne todos os erros para o operador corrigir de uma vez.
func Load(e Env) (Config, error) {
	var errs []error
	get := func(name string) string {
		v, err := secret(e, name)
		if err != nil {
			errs = append(errs, err)
		}
		return v
	}

	dbPass := get("PAINEL_DB_PASSWORD")
	appPass := get("PAINEL_DB_APP_PASSWORD")
	sessionKey := decodeKey(get("PAINEL_SESSION_KEY"), "PAINEL_SESSION_KEY", &errs)
	fieldKey := decodeKey(get("PAINEL_FIELD_KEY"), "PAINEL_FIELD_KEY", &errs)

	dbHost := required(e, "PAINEL_DB_HOST", &errs)
	dbName := required(e, "PAINEL_DB_NAME", &errs)
	dbUser := required(e, "PAINEL_DB_USER", &errs)

	dev := false
	switch env := orDefault(e.Getenv("PAINEL_ENV"), "prod"); env {
	case "dev":
		dev = true
	case "prod":
	default:
		errs = append(errs, fmt.Errorf("PAINEL_ENV inválida: use dev ou prod"))
	}
	team := strings.TrimSpace(e.Getenv("PAINEL_ACCESS_TEAM_DOMAIN"))
	aud := strings.TrimSpace(e.Getenv("PAINEL_ACCESS_AUD"))
	if !dev && (team == "" || aud == "") {
		errs = append(errs, errors.New("em produção é obrigatório definir PAINEL_ACCESS_TEAM_DOMAIN e PAINEL_ACCESS_AUD (Cloudflare Access)"))
	}

	public := strings.TrimSpace(e.Getenv("PAINEL_PUBLIC_URL"))
	if public == "" && dev {
		public = "http://localhost:8080"
	}
	if u, err := url.Parse(public); err != nil || u.Hostname() == "" || (u.Scheme != "https" && !(dev && u.Scheme == "http")) {
		errs = append(errs, errors.New("PAINEL_PUBLIC_URL inválida: use https://seu-dominio (http só em dev, com localhost)"))
	}

	ssl, err := resolveSSLMode(e, dbHost, dev)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return Config{
		Addr:             orDefault(e.Getenv("PAINEL_ADDR"), ":8080"),
		LogLevel:         orDefault(e.Getenv("PAINEL_LOG_LEVEL"), "info"),
		Dev:              dev,
		AccessTeamDomain: team,
		AccessAUD:        aud,
		PublicURL:        strings.TrimRight(public, "/"),
		DocsDir:          orDefault(e.Getenv("PAINEL_DOCS_DIR"), "/data/documentos"),
		DatabaseURL:      dbURL(e, dbHost, dbName, dbUser, dbPass, ssl),
		AppDBPassword:    appPass,
		AppDatabaseURL:   dbURL(e, dbHost, dbName, "painel_app", appPass, ssl),
		SessionKey:       sessionKey,
		FieldKey:         fieldKey,
	}, nil
}

func dbURL(e Env, host, name, user, pass, ssl string) string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, pass),
		Host:     net.JoinHostPort(host, orDefault(e.Getenv("PAINEL_DB_PORT"), "5432")),
		Path:     name,
		RawQuery: "sslmode=" + ssl,
	}
	return u.String()
}

// isInternalHost: nome de um só rótulo (serviço do Docker, ex.: "db") ou loopback = rede interna.
func isInternalHost(h string) bool {
	h = strings.ToLower(strings.TrimSpace(h))
	return h == "localhost" || h == "127.0.0.1" || h == "::1" || (h != "" && !strings.ContainsAny(h, ".:"))
}

// resolveSSLMode decide a criptografia da conexão com o banco (auditoria F9). Na rede interna do Docker o
// padrão é "disable"; com o banco FORA dela o padrão é "verify-full" e, em produção, só se aceita
// require/verify-ca/verify-full.
func resolveSSLMode(e Env, host string, dev bool) (string, error) {
	mode := strings.TrimSpace(e.Getenv("PAINEL_DB_SSLMODE"))
	interno := isInternalHost(host)
	if mode == "" {
		if interno {
			return "disable", nil
		}
		return "verify-full", nil
	}
	switch mode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		return "", fmt.Errorf("PAINEL_DB_SSLMODE inválida: %q", mode)
	}
	forte := mode == "require" || mode == "verify-ca" || mode == "verify-full"
	if !dev && !interno && !forte {
		return "", fmt.Errorf("PAINEL_DB_SSLMODE=%s não é aceito em produção com o banco fora da rede interna do Docker: use verify-full (ou verify-ca/require)", mode)
	}
	return mode, nil
}

func secret(e Env, name string) (string, error) {
	if path := e.Getenv(name + "_FILE"); path != "" {
		b, err := e.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("%s_FILE: não consegui ler o arquivo de segredo", name)
		}
		if v := strings.TrimSpace(string(b)); v != "" {
			return v, nil
		}
	} else if v := strings.TrimSpace(e.Getenv(name)); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("segredo ausente: defina %s ou %s_FILE", name, name)
}

func required(e Env, name string, errs *[]error) string {
	v := strings.TrimSpace(e.Getenv(name))
	if v == "" {
		*errs = append(*errs, fmt.Errorf("variável ausente: %s", name))
	}
	return v
}

func decodeKey(v, name string, errs *[]error) []byte {
	if v == "" {
		return nil // erro de ausência já foi registrado
	}
	b, err := base64.StdEncoding.DecodeString(v)
	if err != nil || len(b) < minKeyBytes {
		*errs = append(*errs, fmt.Errorf("%s inválida: precisa ser base64 de pelo menos %d bytes", name, minKeyBytes))
		return nil
	}
	return b
}

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
