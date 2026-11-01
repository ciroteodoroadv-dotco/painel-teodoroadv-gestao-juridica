# ADR 0004 — PostgreSQL como única fonte transacional

**Status:** aceita (11/2026)

**Contexto:** a versão anterior mantinha cadastros e auditoria em arquivos, criando bancos concorrentes.

**Decisão:** cadastros, tarefas, agenda, sessões e auditoria vivem só no PostgreSQL. Markdown/PDF continuam
como acervo documental, nunca como cadastro ou trilha de auditoria. Documentos binários ficam em volume
privado, referenciados por tabela.

**Consequências:** transações para operações multi-entidade; migrações versionadas com `down`; backup e
restauração tratam banco e volume juntos. Banco sem porta publicada.
