/* =========================================================
   PAINEL TEODORO - modulo 04-api-carga: carregadores: App_carregar* (GET via Painel.carregar)
   Extraido de web/index.html (linhas 415-472) na etapa 4b da
   refatoracao (2026-10-30). Scripts classicos carregados em ordem
   numerica pelo index.html: compartilham o escopo global lexico,
   exatamente como antes dentro da IIFE unica. Sem build toolchain.
   ========================================================= */
'use strict';

const App_carregarIndices = async function(){
  try{
    const raw = await Painel.carregarResultado('/api/indices');
    DB.indices = raw && raw.indices && typeof raw.indices === 'object' ? raw.indices : null;
    DB.statusFontes.indices = {
      status: raw && raw.status || 'nao_sincronizado',
      origem: raw && raw.origem || 'BCB SGS',
      referencia: raw && raw.data_ref || '',
      mensagem: raw && (raw.mensagem || raw.erro) || ''
    };
  }catch(err){
    DB.indices = null;
    DB.statusFontes.indices = {status:'falha', origem:'BCB SGS', referencia:'', mensagem:err.message};
  }
};
const App_carregarIntimacoes = async function(){
  try{
    const raw = await Painel.carregarResultado('/api/intimacoes');
    DB.intimacoes = Array.isArray(raw && raw.intimacoes) ? raw.intimacoes.map(mapearIntimacaoAPI) : null;
    DB.statusFontes.intimacoes = {
      status: raw && raw.status || 'nao_sincronizado',
      mensagem: raw && (raw.mensagem || raw.erro) || '',
      fontes: raw && raw.fontes || {}
    };
  }catch(err){
    DB.intimacoes = null;
    DB.statusFontes.intimacoes = {status:'falha', mensagem:err.message, fontes:{}};
  }
};

const App_carregarProcessos = async function(){
  const raw = await Painel.carregar('processos');
  DB.processos = raw.map(mapearProcessoAPI);
};
const App_carregarPessoas = async function(){
  const raw = await Painel.carregar('pessoas');
  DB.clientes = raw.map(mapearPessoaAPI);
};
const App_carregarTarefas = async function(){
  const raw = await Painel.carregar('tarefas');
  DB.tarefas = raw.map(mapearTarefaAPI);
};
const App_carregarAtendimentos = async function(){
  const raw = await Painel.carregar('atendimentos');
  DB.atendimentos = raw.map(mapearAtendimentoAPI);
};
const App_carregarAgenda = async function(){
  const raw = await Painel.carregar('agenda');
  DB.agenda = raw.map(mapearAgendaAPI);
};
const App_carregarAudiencias = async function(){
  const raw = await Painel.carregar('audiencias');
  DB.audiencias = raw.map(mapearAudienciaAPI);
};
const App_carregarDiligencias = async function(){
  const raw = await Painel.carregar('diligencias');
  DB.diligencias = raw.map(mapearDiligenciaAPI);
};
