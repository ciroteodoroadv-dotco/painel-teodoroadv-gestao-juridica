# ADR 0002 — Acesso por Cloudflare Access + Tunnel

**Status:** aceita (11/2026)

**Contexto:** o plano-base exige "sem acesso público direto ao app". O advogado quer abrir o painel por
URL comum no navegador, como em plataformas web de gestão jurídica, sem instalar VPN.

**Decisão:** `cloudflared` na VM abre conexão só de saída; o Cloudflare Access autentica por e-mail
autorizado; o app valida o JWT `Cf-Access-Jwt-Assertion` e recusa requisição sem ele. Nenhuma porta de
entrada aberta.

**Consequências (LGPD):** o Cloudflare termina o TLS e vê o conteúdo em trânsito [Provável]. É operador
(art. 5º, VII) e implica transferência internacional (art. 33). Exige DPA arquivado, registro no ROPA e
base para a transferência. Cifra de campos sensíveis na aplicação reduz o que o terceiro enxerga em
formulários, mas não elimina.

**Alternativas rejeitadas por ora (revisáveis):**
- Tailscale — terceiro vê metadados; exige cliente instalado.
- WireGuard puro — sem terceiro, mas chaves e revogação manuais, ruim para operação leiga.
- Headscale — sem terceiro, mas mais um serviço a manter.

**Gatilho de revisão:** parecer jurídico contrário ao Cloudflare como operador → migrar para Headscale.
