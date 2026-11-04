# ADR 0010 — Nada se apaga

**Status:** aceita em 04/11/2026 (determinação do advogado, Plano 4C); vale para todo o sistema.

**Contexto:** no escritório, tarefa, audiência, compromisso, diligência e intimação são **ata do que foi feito e decidido**. Apagar apaga prova de diligência
e abre caminho a erro silencioso. O B oferecia exclusão com senha para parte desses registros.

**Decisão:**

1. Tarefa, audiência, aula/reunião/ligação (compromisso), diligência e intimação **não se excluem**. A opção é **cancelar**, com **motivo obrigatório**; o registro
   continua listado, com selo "Cancelada" e com quem/quando/por quê. Cancelar uma tarefa encerra o prazo ligado a ela.
2. **Processo duplicado** não é apagado nem bloqueado: é cadastrado com marcador e **justificativa obrigatória** (Plano 4C).
3. No banco, o `painel_app` não tem `DELETE` nas tabelas novas (calendário forense, índices) e há gatilho contra `DELETE` até para o dono (calendário).
4. Os registros públicos de cache (índices do BCB) são sobrescritos pelo valor mais novo; a tabela de coletas guarda só o estado. Isso não é dado de negócio.
5. **Exceções: nenhuma** para dados de negócio. A remoção de documento por erro de envio, se um dia for necessária, é decisão nova e passa pelo advogado.

**Consequências:** listas têm filtro/selo de cancelado; relatórios contam cancelados à parte; a auditoria registra `*.cancelado`. A migração 0012 só reverte sem dados.
