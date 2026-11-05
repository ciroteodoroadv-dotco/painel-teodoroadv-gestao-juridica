package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
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
		"PAINEL_DB_PASSWORD": "s3nha/com:especiais", "PAINEL_DB_APP_PASSWORD": "app-senha-x", "PAINEL_SESSION_KEY": goodKey, "PAINEL_FIELD_KEY": goodKey,
		"PAINEL_ACCESS_TEAM_DOMAIN": "t.cloudflareaccess.com", "PAINEL_ACCESS_AUD": "aud",
		"PAINEL_PUBLIC_URL": "https://painel.exemplo.test",
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
	for _, n := range []string{"PAINEL_DB_APP_PASSWORD", "PAINEL_SESSION_KEY", "PAINEL_FIELD_KEY"} {
		if !strings.Contains(err.Error(), n) {
			t.Errorf("erro deveria citar %s: %v", n, err)
		}
	}
}

// Auditoria 05/11, item 2.1: o servidor web sobe SEM a senha do dono do banco; só a migração a recebe.
func TestSenhaDoDonoEhOpcional(t *testing.T) {
	v := baseVars()
	delete(v, "PAINEL_DB_PASSWORD")
	c, err := Load(envOf(v, nil))
	if err != nil {
		t.Fatal(err)
	}
	if c.TemDono() || c.DatabaseURL != "" || c.AppDatabaseURL == "" {
		t.Fatalf("sem a senha do dono: TemDono=%v DatabaseURL=%q AppDatabaseURL vazia=%v", c.TemDono(), c.DatabaseURL, c.AppDatabaseURL == "")
	}
	v["PAINEL_DB_PASSWORD_FILE"] = "/nao/existe"
	if _, err := Load(envOf(v, nil)); err == nil {
		t.Fatal("arquivo de senha indicado e ilegível deveria ser erro")
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

func TestPublicURL(t *testing.T) {
	v := baseVars()
	v["PAINEL_PUBLIC_URL"] = "http://painel.exemplo.test"
	if _, err := Load(envOf(v, nil)); err == nil {
		t.Error("http em produção deveria ser recusado")
	}
	v["PAINEL_ENV"] = "dev"
	delete(v, "PAINEL_PUBLIC_URL")
	c, err := Load(envOf(v, nil))
	if err != nil || c.PublicURL != "http://localhost:8080" {
		t.Errorf("dev sem URL deveria usar localhost: %v %q", err, c.PublicURL)
	}
	delete(v, "PAINEL_ENV")
	if _, err := Load(envOf(v, nil)); err == nil {
		t.Error("produção sem PAINEL_PUBLIC_URL deveria falhar")
	}
}

// F9: criptografia da conexão com o banco.
func TestSSLModeDoBanco(t *testing.T) {
	urlCom := func(v map[string]string) (string, error) {
		c, err := Load(envOf(v, nil))
		return c.DatabaseURL, err
	}
	externo := func(mods map[string]string) map[string]string {
		v := baseVars()
		v["PAINEL_DB_HOST"] = "bd.exemplo.test"
		for k, x := range mods {
			v[k] = x
		}
		return v
	}
	// Banco fora da rede interna, sem nada informado: padrão seguro.
	if u, err := urlCom(externo(nil)); err != nil || !strings.Contains(u, "sslmode=verify-full") {
		t.Errorf("banco externo deveria usar verify-full por padrão: %v %q", err, u)
	}
	// Em produção, modo fraco com banco externo é recusado.
	for _, fraco := range []string{"disable", "allow", "prefer"} {
		if _, err := urlCom(externo(map[string]string{"PAINEL_DB_SSLMODE": fraco})); err == nil {
			t.Errorf("sslmode=%s com banco externo em produção deveria ser recusado", fraco)
		}
	}
	for _, forte := range []string{"require", "verify-ca", "verify-full"} {
		if u, err := urlCom(externo(map[string]string{"PAINEL_DB_SSLMODE": forte})); err != nil || !strings.Contains(u, "sslmode="+forte) {
			t.Errorf("sslmode=%s deveria ser aceito: %v %q", forte, err, u)
		}
	}
	// Rede interna do Docker ("db") continua com o padrão sem TLS; dev aceita tudo; valor inválido é recusado.
	if u, err := urlCom(baseVars()); err != nil || !strings.Contains(u, "sslmode=disable") {
		t.Errorf("host interno deveria usar disable por padrão: %v %q", err, u)
	}
	if _, err := urlCom(externo(map[string]string{"PAINEL_ENV": "dev", "PAINEL_DB_SSLMODE": "disable"})); err != nil {
		t.Errorf("dev deveria aceitar disable: %v", err)
	}
	if _, err := urlCom(externo(map[string]string{"PAINEL_DB_SSLMODE": "talvez"})); err == nil {
		t.Error("valor inválido deveria ser recusado")
	}
	for _, h := range []string{"db", "localhost", "127.0.0.1", "::1"} {
		if !isInternalHost(h) {
			t.Errorf("%q deveria ser rede interna", h)
		}
	}
	for _, h := range []string{"bd.exemplo.test", "10.0.0.5", "2001:db8::1", ""} {
		if isInternalHost(h) {
			t.Errorf("%q NÃO deveria ser rede interna", h)
		}
	}
}

func TestDJENVemDesligadoPorPadrao(t *testing.T) {
	c, err := Load(envOf(baseVars(), nil))
	if err != nil {
		t.Fatal(err)
	}
	if c.DJENAtivo {
		t.Fatal("a coleta do DJEN deve vir DESLIGADA por padrão")
	}
	if c.DJENURL != "https://comunicaapi.pje.jus.br/api/v1/comunicacao" || c.DJENIntervalo != 3*time.Hour || c.DJENAlertaHoras != 6 {
		t.Fatalf("padrões do DJEN: %q %v %d", c.DJENURL, c.DJENIntervalo, c.DJENAlertaHoras)
	}
}

func TestDJENVariaveis(t *testing.T) {
	v := baseVars()
	v["PAINEL_DJEN_ATIVO"], v["PAINEL_DJEN_INTERVALO"], v["PAINEL_DJEN_ALERTA_HORAS"] = "true", "90m", "4"
	c, err := Load(envOf(v, nil))
	if err != nil || !c.DJENAtivo || c.DJENIntervalo != 90*time.Minute || c.DJENAlertaHoras != 4 {
		t.Fatalf("variáveis do DJEN: %+v %v", c, err)
	}
	for nome, valor := range map[string]string{"PAINEL_DJEN_ATIVO": "talvez", "PAINEL_DJEN_INTERVALO": "5s", "PAINEL_DJEN_ALERTA_HORAS": "0",
		"PAINEL_DJEN_URL": "http://comunicaapi.pje.jus.br/api"} {
		w := baseVars()
		w[nome] = valor
		if _, err := Load(envOf(w, nil)); err == nil {
			t.Errorf("%s=%q deveria ser recusado", nome, valor)
		}
	}
	w := baseVars()
	w["PAINEL_DJEN_INTERVALO"] = "abc"
	if _, err := Load(envOf(w, nil)); err == nil {
		t.Error("intervalo inválido deveria ser recusado")
	}
}

func TestDataJudVemDesligadoPorPadraoESemChave(t *testing.T) {
	c, err := Load(envOf(baseVars(), nil))
	if err != nil {
		t.Fatal(err)
	}
	d := c.DataJud
	if d.Ativo || d.Chave != "" || d.URL != "https://api-publica.datajud.cnj.jus.br" || d.Intervalo != 24*time.Hour || d.AlertaHoras != 36 || d.IntervaloEntreProcessos != time.Second {
		t.Fatalf("padrões do DataJud: %+v", d)
	}
}

func TestDataJudVariaveisEChaveSoPorSegredo(t *testing.T) {
	v := baseVars()
	v["PAINEL_DATAJUD_ATIVO"], v["PAINEL_DATAJUD_INTERVALO"], v["PAINEL_DATAJUD_ALERTA_HORAS"], v["PAINEL_DATAJUD_PAUSA"] = "true", "6h", "48", "2s"
	v["PAINEL_DATAJUD_KEY"] = "CHAVE-DE-TESTE-FICTICIA"
	c, err := Load(envOf(v, nil))
	if err != nil || !c.DataJud.Ativo || c.DataJud.Intervalo != 6*time.Hour || c.DataJud.AlertaHoras != 48 || c.DataJud.Chave != "CHAVE-DE-TESTE-FICTICIA" || c.DataJud.IntervaloEntreProcessos != 2*time.Second {
		t.Fatalf("variáveis do DataJud: %+v %v", c.DataJud, err)
	}
	if strings.Contains(c.String(), "CHAVE-DE-TESTE-FICTICIA") || strings.Contains(fmt.Sprintf("%+v %#v", c, c), "CHAVE-DE-TESTE-FICTICIA") {
		t.Error("a chave do DataJud não pode aparecer na representação da configuração")
	}
	for nome, valor := range map[string]string{"PAINEL_DATAJUD_ATIVO": "talvez", "PAINEL_DATAJUD_INTERVALO": "5s", "PAINEL_DATAJUD_ALERTA_HORAS": "0",
		"PAINEL_DATAJUD_URL": "http://api-publica.datajud.cnj.jus.br", "PAINEL_DATAJUD_PAUSA": "0s"} {
		w := baseVars()
		w[nome] = valor
		if _, err := Load(envOf(w, nil)); err == nil {
			t.Errorf("%s=%q deveria ser recusado", nome, valor)
		}
	}
}

// Plano 7: a calculadora só deixa de avisar "regras não validadas" quando o advogado liga a variável.
func TestCalculadoraRegrasValidadas(t *testing.T) {
	c, err := Load(envOf(baseVars(), nil))
	if err != nil || c.CalculadoraRegrasValidadas {
		t.Fatalf("ausente deve ser false: %+v %v", c.CalculadoraRegrasValidadas, err)
	}
	v := baseVars()
	v["PAINEL_CALCULADORA_REGRAS_VALIDADAS"] = "true"
	if c, err = Load(envOf(v, nil)); err != nil || !c.CalculadoraRegrasValidadas {
		t.Fatalf("true deve ligar: %v", err)
	}
	v["PAINEL_CALCULADORA_REGRAS_VALIDADAS"] = "talvez"
	if _, err = Load(envOf(v, nil)); err == nil {
		t.Error("valor inválido deveria ser recusado")
	}
}

func TestIndicesECEPVemDesligadosPorPadrao(t *testing.T) {
	c, err := Load(envOf(baseVars(), nil))
	if err != nil {
		t.Fatal(err)
	}
	i := c.Indices
	if i.Ativo || i.URL != "https://api.bcb.gov.br" || i.Intervalo != 24*time.Hour || i.ValidadeDias != 40 || c.CEP.Ativo || c.CEP.URL != "https://viacep.com.br" {
		t.Fatalf("padrões: %+v %+v", i, c.CEP)
	}
}

func TestIndicesECEPVariaveis(t *testing.T) {
	v := baseVars()
	v["PAINEL_INDICES_ATIVO"], v["PAINEL_INDICES_INTERVALO"], v["PAINEL_INDICES_VALIDADE_DIAS"], v["PAINEL_CEP_ATIVO"] = "true", "12h", "60", "1"
	c, err := Load(envOf(v, nil))
	if err != nil || !c.Indices.Ativo || c.Indices.Intervalo != 12*time.Hour || c.Indices.ValidadeDias != 60 || !c.CEP.Ativo {
		t.Fatalf("variáveis: %+v %+v %v", c.Indices, c.CEP, err)
	}
	for nome, valor := range map[string]string{"PAINEL_INDICES_ATIVO": "talvez", "PAINEL_INDICES_INTERVALO": "5m", "PAINEL_INDICES_VALIDADE_DIAS": "0",
		"PAINEL_INDICES_URL": "http://api.bcb.gov.br", "PAINEL_CEP_ATIVO": "sim", "PAINEL_CEP_URL": "http://viacep.com.br"} {
		w := baseVars()
		w[nome] = valor
		if _, err := Load(envOf(w, nil)); err == nil {
			t.Errorf("%s=%q deveria ser recusado", nome, valor)
		}
	}
}
