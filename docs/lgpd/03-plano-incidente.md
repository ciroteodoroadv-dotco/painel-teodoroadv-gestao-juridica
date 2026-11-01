# Plano de resposta a incidente — MODELO EM BRANCO

| Item | Preencher |
|---|---|
| Quem declara incidente | |
| Contatos de emergência | |
| Critério de gravidade | |
| Prazo interno de avaliação | |
| Comunicação à ANPD e aos titulares (art. 48) — critérios e prazo | |

## Passos técnicos previstos

1. Isolar: desativar o túnel do Cloudflare e revogar sessões (`sessions`).
2. Preservar: copiar logs e rodar `painel audit verify`.
3. Trocar segredos (sessão, cifra, banco) conforme runbook.
4. Restaurar de backup validado, se necessário.
5. Registrar linha do tempo, dados afetados e decisões.
6. Revisão pós-incidente e atualização do ROPA.
