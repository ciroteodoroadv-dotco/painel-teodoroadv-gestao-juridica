/* =========================================================
   PAINEL TEODORO - modulo 01-estado: estado em memória: DB (espelho da API) + state de UI
   Extraido de web/index.html (linhas 55-75) na etapa 4b da
   refatoracao (2026-10-30). Scripts classicos carregados em ordem
   numerica pelo index.html: compartilham o escopo global lexico,
   exatamente como antes dentro da IIFE unica. Sem build toolchain.
   ========================================================= */
'use strict';

/* =========================================================
   ESTADO / BANCO DE DADOS (espelho em memória do que veio da API)
   ========================================================= */
const DB = {
  clientes: [],     // pessoas: derivado de GET /api/pessoas (nome, papel, origem) — só leitura
                     // papel: "cliente" | "parte_contraria" | "terceiro_interessado"
  atendimentos: [], // derivado de GET /api/atendimentos, mapeado por mapearAtendimentoAPI()
  processos: [],    // derivado de GET /api/processos, mapeado por mapearProcessoAPI()
  tarefas: [],      // derivado de GET /api/tarefas, mapeado por mapearTarefaAPI()
  agenda: [],       // derivado de GET /api/agenda (dados/agenda.json) — só Prazo Fatal/Reunião/Ligação/Aula
  audiencias: [],   // derivado de GET /api/audiencias (fichas .md reais), mapeado por mapearAudienciaAPI()
  diligencias: [],  // derivado de GET /api/diligencias (fichas .md reais), mapeado por mapearDiligenciaAPI()
  indices: null,     // ausente até uma resposta real de GET /api/indices
  intimacoes: null,  // ausente até uma resposta real de GET /api/intimacoes
  statusFontes: { indices: null, intimacoes: null, cep: null }
};

// accOpen: estado de quais painéis de acordeão estão abertos, por grupo
// (agenda/pessoas/processos/audiencias) e por categoria dentro do grupo —
// vários podem ficar abertos ao mesmo tempo, cada grupo é independente.
const state = { tab: 'agenda', search: '', searchDraft: '', accOpen: {} };
