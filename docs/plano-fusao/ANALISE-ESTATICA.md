# Análise estática de segurança

Gerado por `scripts/analise-estatica.sh` em 2026-11-04 (contêiner `golang:1.26.10-bookworm`; gosec v2.30.0, staticcheck v0.8.2, govulncheck v1.9.0).

| Ferramenta | Resultado |
|---|---|
| gosec | alta: 0 · média: 0 · baixa: 0 (achados abertos); suprimidos com `#nosec` justificado: 12 |
| staticcheck | 0 achado(s) |
| govulncheck | nenhuma vulnerabilidade alcançável pelo código |

## staticcheck
```
```
## govulncheck
```
=== Symbol Results ===

No vulnerabilities found.

Your code is affected by 0 vulnerabilities.
This scan also found 0 vulnerabilities in packages you import and 1
vulnerability in modules you require, but your code doesn't appear to call these
vulnerabilities.
Use '-show verbose' for more details.
```
## gosec (achados abertos e suprimidos)
```
HIGH   G115 internal/auth/totp.go:46 [suprimido: step = unix/30, sempre positivo]
HIGH   G115 internal/auth/password.go:62 [suprimido: len(want) já validado entre 16 e 64]
HIGH   G704 internal/cep/cliente.go:115 [suprimido: falso positivo: a base é a URL https da configuração (PAINEL_CEP_URL, validada em NovoCliente) e o CEP já foi validado como 8 dígitos.]
HIGH   G704 cmd/painel/main.go:173 [suprimido: falso positivo: host fixo (127.0.0.1) e porta vinda de PAINEL_ADDR, configuração do operador; não é entrada de usuário.]
HIGH   G704 internal/cep/cliente.go:122 [suprimido: idem: destino fixo pela configuração; só os 8 dígitos do CEP variam.]
MEDIUM G705 internal/documentos/handler.go:171 [suprimido: falso positivo: o arquivo é cifrado em repouso e sai como anexo (attachment) com nosniff e CSP "sandbox; default-src 'none'"; nunca é interpretado como HTML.]
MEDIUM G120 internal/auth/router.go:92 [suprimido: o corpo já está limitado a 26 MiB pelo MaxBytesReader da linha anterior (e pelo httpx); a análise do gosec não reconhece essa atribuição.]
MEDIUM G203 internal/prazos/handler.go:245 [suprimido: saída do html/template]
MEDIUM G203 internal/monitor/handler.go:79 [suprimido: saída de html/template, já escapada]
MEDIUM G203 internal/indices/fragmento.go:90 [suprimido: saída de html/template, já escapada]
MEDIUM G304 internal/documentos/storage.go:75 [suprimido: p vem de path(): nome validado por ^[0-9a-f]{64}$]
MEDIUM G304 internal/documentos/storage.go:42 [suprimido: Dir + nome aleatório gerado aqui]
```
