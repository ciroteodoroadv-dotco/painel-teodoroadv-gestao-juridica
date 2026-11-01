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
	Addr        string
	LogLevel    string
	DatabaseURL string // contém a senha do banco
	SessionKey  []byte
	FieldKey    []byte
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
	sessionKey := decodeKey(get("PAINEL_SESSION_KEY"), "PAINEL_SESSION_KEY", &errs)
	fieldKey := decodeKey(get("PAINEL_FIELD_KEY"), "PAINEL_FIELD_KEY", &errs)

	dbHost := required(e, "PAINEL_DB_HOST", &errs)
	dbName := required(e, "PAINEL_DB_NAME", &errs)
	dbUser := required(e, "PAINEL_DB_USER", &errs)

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return Config{
		Addr:        orDefault(e.Getenv("PAINEL_ADDR"), ":8080"),
		LogLevel:    orDefault(e.Getenv("PAINEL_LOG_LEVEL"), "info"),
		DatabaseURL: dbURL(e, dbHost, dbName, dbUser, dbPass),
		SessionKey:  sessionKey,
		FieldKey:    fieldKey,
	}, nil
}

func dbURL(e Env, host, name, user, pass string) string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, pass),
		Host:     net.JoinHostPort(host, orDefault(e.Getenv("PAINEL_DB_PORT"), "5432")),
		Path:     name,
		RawQuery: "sslmode=" + orDefault(e.Getenv("PAINEL_DB_SSLMODE"), "disable"),
	}
	return u.String()
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
