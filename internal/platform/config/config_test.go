package config

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

var goodKey = base64.StdEncoding.EncodeToString(make([]byte, 32))

func envOf(vars map[string]string, files map[string]string) Env {
	return Env{
		Getenv: func(k string) string { return vars[k] },
		ReadFile: func(p string) ([]byte, error) {
			if c, ok := files[p]; ok {
				return []byte(c + "\n"), nil
			}
			return nil, errors.New("sem arquivo")
		},
	}
}

func baseVars() map[string]string {
	return map[string]string{
		"PAINEL_DB_HOST": "db", "PAINEL_DB_NAME": "painel", "PAINEL_DB_USER": "painel",
		"PAINEL_DB_PASSWORD": "s3nha/com:especiais", "PAINEL_SESSION_KEY": goodKey, "PAINEL_FIELD_KEY": goodKey,
		"PAINEL_ACCESS_TEAM_DOMAIN": "t.cloudflareaccess.com", "PAINEL_ACCESS_AUD": "aud",
	}
}

func TestProducaoExigeAccess(t *testing.T) {
	v := baseVars()
	delete(v, "PAINEL_ACCESS_AUD")
	if _, err := Load(envOf(v, nil)); err == nil {
		t.Fatal("produção sem AUD deveria falhar")
	}
	v["PAINEL_ENV"] = "dev"
	c, err := Load(envOf(v, nil))
	if err != nil || !c.Dev {
		t.Fatalf("dev sem AUD deveria passar: %v", err)
	}
	v["PAINEL_ENV"] = "staging"
	if _, err := Load(envOf(v, nil)); err == nil {
		t.Fatal("PAINEL_ENV desconhecida deveria falhar")
	}
}

func TestLoadOK(t *testing.T) {
	c, err := Load(envOf(baseVars(), nil))
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != ":8080" || c.LogLevel != "info" {
		t.Fatalf("padrões errados: %s", c)
	}
	if !strings.Contains(c.DatabaseURL, "s3nha%2Fcom:especiais") && !strings.Contains(c.DatabaseURL, "%3A") {
		t.Fatalf("senha não foi escapada na URL: %q", c.DatabaseURL)
	}
}

func TestLoadFalhaSemSegredos(t *testing.T) {
	_, err := Load(envOf(map[string]string{"PAINEL_DB_HOST": "db", "PAINEL_DB_NAME": "x", "PAINEL_DB_USER": "x"}, nil))
	if err == nil {
		t.Fatal("deveria falhar sem segredos")
	}
	for _, n := range []string{"PAINEL_DB_PASSWORD", "PAINEL_SESSION_KEY", "PAINEL_FIELD_KEY"} {
		if !strings.Contains(err.Error(), n) {
			t.Errorf("erro deveria citar %s: %v", n, err)
		}
	}
}

func TestSegredoPorArquivo(t *testing.T) {
	v := baseVars()
	delete(v, "PAINEL_DB_PASSWORD")
	v["PAINEL_DB_PASSWORD_FILE"] = "/run/secrets/db"
	c, err := Load(envOf(v, map[string]string{"/run/secrets/db": "abc"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(c.DatabaseURL, ":abc@") {
		t.Fatalf("senha do arquivo não usada: %q", c.DatabaseURL)
	}
}

func TestChaveCurtaOuInvalida(t *testing.T) {
	for _, bad := range []string{"%%%", base64.StdEncoding.EncodeToString([]byte("curta"))} {
		v := baseVars()
		v["PAINEL_SESSION_KEY"] = bad
		if _, err := Load(envOf(v, nil)); err == nil {
			t.Errorf("chave %q deveria ser recusada", bad)
		}
	}
}

func TestStringNaoVazaSegredo(t *testing.T) {
	c, _ := Load(envOf(baseVars(), nil))
	if s := c.String(); strings.Contains(s, "especiais") || strings.Contains(s, goodKey) {
		t.Fatalf("String vazou segredo: %s", s)
	}
}
