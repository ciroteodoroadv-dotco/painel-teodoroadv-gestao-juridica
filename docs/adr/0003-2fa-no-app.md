# ADR 0003 — 2FA obrigatório dentro do app

**Status:** aceita (11/2026)

**Contexto:** o Access é perímetro; se a conta de e-mail do usuário cair, o perímetro cai junto.

**Decisão:** o app exige segundo fator próprio a todos os papéis: chave física/passkey (WebAuthn) ou
aplicativo autenticador (TOTP). Códigos de recuperação de uso único, guardados com hash. Administrador
recomenda-se com chave física.

**Estado (01/11/2026):** TOTP implementado, com anti-reuso do código e códigos de recuperação. **WebAuthn
(chave física) ainda não** — fica para a parte 2 do M2; até lá, o administrador usa TOTP.

**Consequências:** identidade do app independente do provedor de perímetro; cadastro de fator no primeiro
login; procedimento de recuperação deve estar no runbook.
