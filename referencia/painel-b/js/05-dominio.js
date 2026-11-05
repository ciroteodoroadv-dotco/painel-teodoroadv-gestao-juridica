/* =========================================================
   PAINEL TEODORO - modulo 05-dominio: opções de domínio: OPCOES, TIPOS_AUDIENCIA
   Extraido de web/index.html (linhas 474-510) na etapa 4b da
   refatoracao (2026-10-30). Scripts classicos carregados em ordem
   numerica pelo index.html: compartilham o escopo global lexico,
   exatamente como antes dentro da IIFE unica. Sem build toolchain.
   ========================================================= */
'use strict';

/* =========================================================
   OPÇÕES DE DOMÍNIO
   ========================================================= */
const OPCOES = {
  tipoCadastro: ['Cliente','Parte Adversa','Terceiro Interessado'],
  estadoCivil: ['Solteiro(a)','Casado(a)','Divorciado(a)','Viúvo(a)','União Estável','Separado(a) Judicialmente'],
  sistema: ['PJe','e-SAJ','Projudi','e-Proc','Físico (papel)','Outro'],
  instancia: ['1º Grau','2º Grau / Tribunal','Turma Recursal / Juizado Especial','Superior Tribunal de Justiça (STJ)','Supremo Tribunal Federal (STF)','Tribunal Superior do Trabalho (TST)','Instância Administrativa (ex. CARF)','Outra'],
  classeProcessual: ['Procedimento Comum Cível','Execução Fiscal','Execução de Título Extrajudicial','Embargos à Execução Fiscal','Embargos de Terceiro','Mandado de Segurança','Ação Anulatória de Débito Fiscal','Ação Declaratória','Ação de Repetição de Indébito','Cumprimento de Sentença','Medida Cautelar Fiscal','Exceção de Pré-Executividade','Agravo de Instrumento','Apelação Cível','Recurso Especial','Recurso Extraordinário','Outra (especificar)'],
  materiaDireito: ['Tributário','Cível / Processual Civil','Bancário','Consumidor','Empresarial','Previdenciário','Trabalhista','Administrativo','Ambiental','Sucessões e Família','Outra'],
  atendimentoTipo: ['Extrajudicial','Triagem'],
  atendimentoStatus: ['Em triagem','Em andamento','Concluído','Convertido em processo'],
  prioridade: ['Urgente','Alta','Normal','Baixa'],
  tarefaStatus: ['Pendente','Em andamento','Concluída'],
  estadoProcesso: ['ativo','sobrestado','extinto','arquivado'],
  // Audiência saiu daqui — tem ficha própria e drawer próprio (aba
  // Audiências, ver TIPOS_AUDIENCIA/App.openAudienciaForm). Diligência
  // continua neste formulário genérico, mas grava em ficha própria
  // (POST /api/diligencias/nova) — ver App.saveAgenda.
  agendaTipo: ['Reunião','Ligação','Diligência','Aula']
};

// As 6 subpastas fixas de 11-TEODORO-ADVOCACIA/audiencias/ — mesma lista de
// ficha.TiposAudiencia no backend; "chave" é o nome exato da pasta, usado
// como campo "tipo" na API.
const TIPOS_AUDIENCIA = [
  {chave:'audiencia-ad-monitoria',      label:'Audiência Ad Monitoria'},
  {chave:'conciliacao-e-mediacao',      label:'Conciliação e Mediação'},
  {chave:'inicial-justica-do-trabalho', label:'Inicial — Justiça do Trabalho'},
  {chave:'instrucao-e-julgamento',      label:'Instrução e Julgamento'},
  {chave:'sustentacao-oral',            label:'Sustentação Oral'},
  {chave:'una-justica-do-trabalho',     label:'Una — Justiça do Trabalho'}
];
function labelTipoAudiencia(chave){
  const t = TIPOS_AUDIENCIA.find(t=>t.chave===chave);
  return t ? t.label : (chave || '—');
}
