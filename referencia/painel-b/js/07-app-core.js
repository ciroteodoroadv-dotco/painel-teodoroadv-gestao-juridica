/* =========================================================
   PAINEL TEODORO - modulo 07-app-core: App: init, navegação, topbar, stats, busca, renderTab, acordeão
   Extraido de web/index.html (linhas 747-926) na etapa 4b da
   refatoracao (2026-10-30). Scripts classicos carregados em ordem
   numerica pelo index.html: compartilham o escopo global lexico,
   exatamente como antes dentro da IIFE unica. Sem build toolchain.
   ========================================================= */
'use strict';

/* =========================================================
   APP — objeto global de controle (render + ações de UI)
   ========================================================= */
const App = {};

// App.adicionarLinhaParte acrescenta uma linha em branco ao repetidor de
// Partes do drawer de processo atualmente aberto (criação ou edição, mesmo
// container #partesRows nos dois formulários).
App.adicionarLinhaParte = function(){
  const container = document.getElementById('partesRows');
  if(container) container.insertAdjacentHTML('beforeend', linhaParteHtml());
};
window.App = App;

App.init = async function(){
  try{
    await Promise.all([App_carregarProcessos(), App_carregarPessoas(), App_carregarTarefas(), App_carregarAtendimentos(), App_carregarAgenda(), App_carregarAudiencias(), App_carregarDiligencias(), App_carregarIndices(), App_carregarIntimacoes()]);
  }catch(err){
    showToast('Não foi possível carregar os dados do servidor.', true);
  }
  App.renderNav();
  App.renderTopbar();
  App.renderStats();
  App.renderTab();
  App.renderIndices();

  // Faixa de índices atualiza a cada 30s — só relê o último estado
  // persistido pelo servidor (GET /api/indices), nunca dispara nova
  // coleta na fonte oficial; a periodicidade real de cada série (ex.: BCB
  // mensal para taxa de crédito) é decidida no servidor, não aqui.
  setInterval(async ()=>{
    try{ await App_carregarIndices(); App.renderIndices(); }catch(err){ /* mantém o último estado exibido */ }
  }, 30000);

  // Re-render em tempo real quando outra aba/sessão gravar a mesma entidade.
  Painel.ao('indices', async ()=>{
    await App_carregarIndices();
    App.renderIndices();
  });
  Painel.ao('intimacoes', async ()=>{
    await App_carregarIntimacoes();
    App.renderNav(); App.renderStats(); App.renderTab();
  });
  Painel.ao('processos', async ()=>{
    try{ await App_carregarProcessos(); }catch(e){}
    App.renderNav(); App.renderStats(); App.renderTab();
  });
  Painel.ao('pessoas', async ()=>{
    try{ await App_carregarPessoas(); }catch(e){}
    App.renderNav(); App.renderStats(); App.renderTab();
  });
  Painel.ao('tarefas', async ()=>{
    try{ await App_carregarTarefas(); }catch(e){}
    App.renderNav(); App.renderStats(); App.renderTab();
  });
  Painel.ao('atendimentos', async ()=>{
    try{ await App_carregarAtendimentos(); }catch(e){}
    App.renderNav(); App.renderStats(); App.renderTab();
  });
  Painel.ao('agenda', async ()=>{
    try{ await App_carregarAgenda(); }catch(e){}
    App.renderNav(); App.renderStats(); App.renderTab();
  });
  Painel.ao('audiencias', async ()=>{
    try{ await App_carregarAudiencias(); }catch(e){}
    App.renderNav(); App.renderStats(); App.renderTab();
  });
  Painel.ao('diligencias', async ()=>{
    try{ await App_carregarDiligencias(); }catch(e){}
    App.renderNav(); App.renderStats(); App.renderTab();
  });
};

App.setTab = function(tab){
  state.tab = tab; state.search=''; state.searchDraft='';
  if(tab==='agenda') state.accOpen={};
  App.renderNav(); App.renderTopbar(); App.renderStats(); App.renderTab();
};

// Ordem de definição = ordem de exibição na navegação lateral, e é
// alfabética por rótulo (Agenda, Atendimentos, Audiências, Pessoas,
// Processos, Tarefas), a pedido do advogado.
const TAB_META = {
  agenda:       {label:'Agenda',        title:'Agenda',        sub:'Painel principal: compromissos e tarefas em aberto, por categoria.'},
  atendimentos: {label:'Atendimentos',  title:'Atendimentos',  sub:'Registro das demandas e dos atendimentos do escritório.'},
  audiencias:   {label:'Audiências',    title:'Audiências',    sub:'Audiências marcadas, agrupadas por tipo de ato.'},
  pessoas:      {label:'Pessoas',       title:'Pessoas',       sub:'Cadastro de clientes, partes adversas e terceiros interessados — art. 319 do CPC.'},
  processos:    {label:'Processos',     title:'Processos',     sub:'Cadastro processual: partes, classe, valor da causa e marcos.'},
  tarefas:      {label:'Tarefas',       title:'Tarefas',       sub:'Toda tarefa nasce de um atendimento (ordem de serviço) ou de um processo.'}
};

App.renderNav = function(){
  const nav = document.getElementById('navTabs');
  const contagem = {
    agenda: DB.agenda.length + DB.audiencias.length + DB.diligencias.length,
    atendimentos: DB.atendimentos.length,
    audiencias: DB.audiencias.length,
    pessoas: DB.clientes.length,
    processos: DB.processos.length,
    tarefas: DB.tarefas.length
  };
  const botoesTab = Object.keys(TAB_META).map(key=>{
    return `<button class="${state.tab===key?'active':''}" onclick="App.setTab('${key}')">
      <span>${TAB_META[key].label}</span><span class="count">${contagem[key]}</span>
    </button>`;
  });
  const botaoCalculadora = `<button onclick="App.abrirCalculadoraPrazo()">Calculadora de prazo</button>`;
  const posAudiencias = Object.keys(TAB_META).indexOf('audiencias');
  botoesTab.splice(posAudiencias + 1, 0, botaoCalculadora);
  nav.innerHTML = botoesTab.join('');
};

App.renderTopbar = function(){
  const meta = TAB_META[state.tab];
  const addLabel = {agenda:'Novo compromisso', atendimentos:'Novo atendimento', audiencias:'Nova audiência', pessoas:'Nova pessoa', processos:'Novo processo', tarefas:'Nova tarefa'}[state.tab];
  const openFn = {agenda:'App.openAgendaForm', atendimentos:'App.openAtendimentoForm', audiencias:'App.openAudienciaForm', pessoas:'App.openPessoaForm', processos:'App.openProcessoForm', tarefas:'App.openTarefaForm'}[state.tab];
  document.getElementById('topbar').innerHTML = `
    <div>
      <div class="title-row"><h1>${meta.title}</h1>${state.tab==='agenda'?`<time class="agenda-date-chip" datetime="${todayISO()}">${escapeHTML(new Intl.DateTimeFormat('pt-BR',{weekday:'short',day:'2-digit',month:'2-digit',year:'numeric'}).format(new Date()))}</time>`:''}</div>
      <p class="sub">${meta.sub}</p>
    </div>
    <button class="btn btn-primary" onclick="${openFn}()">${addLabel}</button>
  `;
};

const ESTADOS_TAREFA_FINALIZADOS = ['concluidas-com-sucesso','concluidas-sem-sucesso','canceladas'];
// A barra de estatísticas vira o próprio conjunto de botões de categoria da aba
// (populados por renderAcordeao); aqui só limpamos antes de renderAcordeao repovoar,
// ou para abas sem acordeão (Tarefas, Atendimentos), que ficam sem essa barra.
App.renderStats = function(){
  document.getElementById('stats').innerHTML = '';
};

App.setSearch = function(v){ state.search = v.toLowerCase(); App.renderTab(); };
App.setSearchDraft = function(v){ state.searchDraft = v; };
App.aplicarBuscaTarefas = function(ev){
  if(ev) ev.preventDefault();
  state.search = String(state.searchDraft||'').trim().toLowerCase();
  App.renderTab();
  return false;
};

App.renderTab = function(){
  document.body.dataset.tab=state.tab;
  const c = document.getElementById('content');
  if(state.tab==='tarefas') c.innerHTML = App.viewTarefas();
  else if(state.tab==='processos') c.innerHTML = App.viewProcessos();
  else if(state.tab==='atendimentos') c.innerHTML = App.viewAtendimentos();
  else if(state.tab==='audiencias') c.innerHTML = App.viewAudiencias();
  else if(state.tab==='pessoas') c.innerHTML = App.viewPessoas();
  else c.innerHTML = App.viewAgenda();
};

/* ---------------------------------------------------------
   COMPONENTE: ACORDEÃO VERTICAL SIMULTÂNEO
   Botões de categoria ficam sempre na horizontal (linha de cima);
   os painéis abrem empilhados verticalmente logo abaixo, e vários
   podem ficar abertos ao mesmo tempo (cada chave é independente).
   --------------------------------------------------------- */
App.toggleAcc = function(grupo, chave){
  state.accOpen[grupo] = state.accOpen[grupo] || {};
  const abrir = !state.accOpen[grupo][chave];
  if(grupo==='agenda') state.accOpen[grupo]={};
  state.accOpen[grupo][chave] = abrir;
  App.renderTab();
};
function accAberto(grupo, chave){
  return !!(state.accOpen[grupo] && state.accOpen[grupo][chave]);
}
// categorias: [{chave, label, count, html}]
function renderAcordeao(grupo, categorias){
  const heads = categorias.map(c => `
    <button class="acc-head ${accAberto(grupo,c.chave)?'open':''}" onclick="App.toggleAcc('${grupo}','${c.chave}')">
      <span>${escapeHTML(c.label)}</span><span class="count">${c.count}</span>
    </button>`).join('');
  // Os botões de categoria ocupam a barra de estatísticas (topo), grandes, no lugar
  // dos antigos cartões estáticos sem função.
  document.getElementById('stats').innerHTML = `<div class="acc-heads">${heads}</div>`;
  const bodies = categorias.map(c => `
    <div class="acc-item ${accAberto(grupo,c.chave)?'open':''}">
      <div class="acc-body">${c.html}</div>
    </div>`).join('');
  return `<div class="acc">${bodies}</div>`;
}
