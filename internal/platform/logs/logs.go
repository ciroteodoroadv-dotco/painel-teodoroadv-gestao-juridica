// Package logs cria o logger JSON. Chaves que costumam carregar dado pessoal ou segredo
// são redigidas por padrão, como rede de proteção. A regra principal continua sendo:
// logar só ids, método, rota e status — nunca conteúdo de cliente.
package logs

import (
	"io"
	"log/slog"
	"strings"
)

const redacted = "[redigido]"

var sensitive = []string{
	"password", "senha", "secret", "token", "authorization", "cookie", "key", "chave",
	"cpf", "cnpj", "email", "nome", "name", "telefone", "phone", "documento", "endereco",
}

// exact guarda chaves curtas demais para busca por trecho (ex.: "rg" casaria com "target").
var exact = map[string]bool{"rg": true}

// New devolve um logger JSON que escreve em w.
func New(w io.Writer, level string) *slog.Logger {
	var lv slog.Level
	if err := lv.UnmarshalText([]byte(level)); err != nil {
		lv = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lv, ReplaceAttr: redact}))
}

func redact(_ []string, a slog.Attr) slog.Attr {
	if isSensitive(a.Key) {
		return slog.String(a.Key, redacted)
	}
	return a
}

func isSensitive(key string) bool {
	k := strings.ToLower(key)
	if exact[k] {
		return true
	}
	for _, s := range sensitive {
		if strings.Contains(k, s) {
			return true
		}
	}
	return false
}
