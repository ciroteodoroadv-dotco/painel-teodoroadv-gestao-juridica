# ADR 0003 — 2FA obrigatório dentro do app

**Status:** aceita (11/2026)

**Contexto:** o Access é perímetro; se a conta de e-mail do usuário cair, o perímetro cai junto.

**Decisão:** o app exige segundo fator próprio a todos os papéis: chave física/passkey (WebAuthn) ou
aplicativo autenticador (TOTP). Códigos de recuperação de uso único, guardados com hash. Administrador
recomenda-se com chave física.

**Consequências:** identidade do app independente do provedor de perímetro; cadastro de fator no primeiro
login; procedimento de recuperação deve estar no runbook.
