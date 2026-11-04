/* =========================================================
   PAINEL TEODORO - modulo 11-agenda: view Agenda: categorias, intimações, audiências, diligências
   Extraido de web/index.html (linhas 1975-2648) na etapa 4b da
   refatoracao (2026-10-30). Scripts classicos carregados em ordem
   numerica pelo index.html: compartilham o escopo global lexico,
   exatamente como antes dentro da IIFE unica. Sem build toolchain.
   ========================================================= */
'use strict';

/* ---------------------------------------------------------
   VIEW: AGENDA — painel principal do sistema. Data e dia da semana
   no topo; contadores por categoria (Intimações mantém o histórico; os
   demais itens seguem a regra de itens em aberto) (botões
   horizontais); cada categoria abre em painel vertical, todas podem
   ficar abertas ao mesmo tempo.
   --------------------------------------------------------- */
const TIPOS_AGENDA_CRITICOS = ['Audiência', 'Prazo Fatal'];
function agendaCriticaBadgeClass(tipo){
  return TIPOS_AGENDA_CRITICOS.includes(tipo) ? 'badge-danger' : 'badge-neutral';
}

const CATEGORIAS_AGENDA = [
  {chave:'tarefas',       label:'Tarefas',       tipoAgenda:null},
  {chave:'intimacoes',    label:'Intimações',    tipoAgenda:null},
  {chave:'audiencias',    label:'Audiências',    tipoAgenda:null},
  {chave:'diligencias',   label:'Diligências',   tipoAgenda:null},
  {chave:'reunioes',      label:'Reuniões',      tipoAgenda:'Reunião'},
  {chave:'ligacoes',      label:'Ligações',      tipoAgenda:'Ligação'},
  {chave:'aulas',         label:'Aulas',         tipoAgenda:'Aula'}
];

// normalizarTexto compara rótulos ignorando maiúsculas e acentos
// ("Ligação" casa com "ligacao").
function normalizarTexto(s){
  return String(s||'').normalize('NFD').replace(/[\u0300-\u036f]/g,'').toLowerCase().trim();
}

// Entrega 1.1, item 9: a LEITURA da agenda é unificada — as categorias
// Ligações, Reuniões e Aulas mostram também as tarefas em aberto
// correspondentes, sem fundir os dois armazenamentos: a tarefa continua
// sendo criada e editada na aba Tarefas. Prazo não é categoria (decisão
// do advogado, 2026-10-29): é campo da tarefa, visível na aba Tarefas.
const SUBTITULOS_CATEGORIA_AGENDA = {
  'ligacoes': ['ligacao','ligacoes'],
  'reunioes': ['reuniao','reunioes'],
  'aulas':    ['aula','aulas']
};
function tarefaEmAberto(t){ return !ESTADOS_TAREFA_FINALIZADOS.includes(t.status); }
function tarefasDaCategoriaAgenda(chave){
  const rotulos = SUBTITULOS_CATEGORIA_AGENDA[chave];
  if(!rotulos) return [];
  return DB.tarefas.filter(t => tarefaEmAberto(t) && rotulos.includes(normalizarTexto(t.subtitulo)))
    .sort((a,b) => String(a.prazo||a.dataBase||'').localeCompare(String(b.prazo||b.dataBase||'')));
}
// Tabela das tarefas dentro de uma categoria da agenda. A data exibida
// é o prazo (ou a data base) da tarefa.
function referenciasClientesTarefaAgenda(t){
 if(t.origemTipo==='atendimento'){
  const a=DB.atendimentos.find(x=>x.id===t.origemId);
  return a?referenciasAtendimento(a,'clientes'):[];
 }
 if(t.origemTipo==='processo'){
  const p=DB.processos.find(x=>x.id===t.origemId);
  const partes=p?(p.partes||[]).filter(x=>x.cliente).map(x=>x.pessoa):[];
  return partes.length?partes:(p&&p.clienteNome?[p.clienteNome]:[]);
 }
 return [];
}
function clienteTarefaAgendaHTML(t){
 const refs=referenciasClientesTarefaAgenda(t);
 return refs.map(ref=>{const p=encontrarPessoaPorReferencia(ref);return p?`<span>${escapeHTML(nomePessoaAtendimento(ref))} · ${escapeHTML(p.codigo||'PES. [VERIFICAR]')}</span>`:escapeHTML(nomePessoaAtendimento(ref)||'Referência não localizada');}).join('<br>')||'—';
}
function tabelaTarefasCategoriaAgenda(lista){
 if(!lista.length)return '';
 const rows=lista.map(t=>{const st=computeTarefaStatus(t);return `<tr>
 <td class="tabular">${escapeHTML(t.codigo||'—')}</td>
 <td><strong>${escapeHTML(t.titulo)}</strong></td>
 <td>${clienteTarefaAgendaHTML(t)}</td>
 <td>${escapeHTML((t.criadoPor?nomeExibicaoReferenciaPessoa(t.criadoPor):'')||'—')}</td>
 <td class="tabular">${escapeHTML(fmtDateOrRaw(t.prazo))}</td>
 <td class="tabular">${escapeHTML(fmtDateOrRaw(t.prazoFatal))}</td>
 <td><span class="badge ${st.cls}">${escapeHTML(st.label)}</span></td></tr>`;}).join('');
 return `<div class="agenda-table-scroll"><table class="agenda-tarefas"><thead><tr><th>Código</th><th>Tarefa</th><th>Cliente(s) / PES</th><th>Criado por</th><th>Prazo de entrega</th><th>Prazo fatal</th><th>Status</th></tr></thead><tbody>${rows}</tbody></table></div>`;
}

// "Em aberto" para item de agenda.json = ainda não passou (sem campo de
// status próprio no backend). Audiência e diligência são fichas reais
// (DB.audiencias/DB.diligencias), mesmo critério de data, ver
// audienciasEmAberto/diligenciasEmAberto logo abaixo.
function itensAgendaEmAberto(tipo){
  return DB.agenda.filter(a => a.tipo===tipo && (!a.data || a.data >= todayISO()))
    .sort((a,b) => (a.data+a.hora).localeCompare(b.data+b.hora));
}
function audienciasEmAberto(){
  return DB.audiencias.filter(a => !a.data || a.data >= todayISO())
    .sort((a,b) => (a.data+a.hora).localeCompare(b.data+b.hora));
}
function diligenciasEmAberto(){
  return DB.diligencias.filter(d => !d.dataLimite || d.dataLimite >= todayISO())
    .sort((a,b) => (a.dataLimite+a.hora).localeCompare(b.dataLimite+b.hora));
}

function linhaAgenda(a){
  return `<tr>
    <td><span class="badge ${agendaCriticaBadgeClass(a.tipo)}">${escapeHTML(a.tipo)}</span></td>
    <td>${escapeHTML(a.descricao)}${a.criadoPor?`<div class="hint">Criado por: ${escapeHTML(nomeExibicaoReferenciaPessoa(a.criadoPor))}</div>`:''}${a.cliente?`<div class="hint">${escapeHTML(nomePessoaAtendimento(a.cliente))} · ${escapeHTML((encontrarPessoaPorReferencia(a.cliente)||{}).codigo||'PES. [VERIFICAR]')}</div>`:''}${a.atendimentoVinculado?`<div class="hint">${escapeHTML((DB.atendimentos.find(x=>x.id===a.atendimentoVinculado)||{}).codigo||a.atendimentoVinculado)}</div>`:''}</td>
    <td class="tabular">${fmtDateOrRaw(a.data)}</td>
    <td class="tabular">${escapeHTML(a.hora||'—')}</td>
    <td>${escapeHTML(a.local||'—')}</td>
    <td>${escapeHTML((DB.processos.find(x=>x.id===a.processoVinculado)||{}).codigo||a.processoVinculado||'—')}</td>
    <td class="row-actions">
      <button class="btn-text" onclick="App.openAgendaEditForm(${a.id})">Editar</button>
      <button class="btn-danger-text" onclick="App.deleteAgenda()">Excluir</button>
    </td>
  </tr>`;
}
function tabelaAgenda(lista){
  if(!lista.length) return `<div class="empty"><p>Nenhum item em aberto nesta categoria.</p></div>`;
  return `<table><thead><tr><th>Tipo</th><th>Descrição</th><th>Data</th><th>Hora</th><th>Local</th><th>Processo</th><th></th></tr></thead>
    <tbody>${lista.map(linhaAgenda).join('')}</tbody></table>`;
}

function linhaAudiencia(a){
  return `<tr>
    <td><span class="badge badge-danger">Audiência</span><div class="hint" style="color:var(--text-muted);margin-top:3px;">${escapeHTML(labelTipoAudiencia(a.tipo))}</div></td>
    <td>${escapeHTML(a.codigo||'—')}</td>
    <td>${escapeHTML(a.titulo)}</td>
    <td class="tabular">${fmtDateOrRaw(a.data)}</td>
    <td class="tabular">${escapeHTML(a.hora||'—')}</td>
    <td>${escapeHTML(a.local||'—')}</td>
    <td>${escapeHTML(a.processo||'—')}</td>
    <td class="row-actions">${a.cancelada ? `<span class="badge badge-neutral">Cancelada</span>` : `<button class="btn-text" onclick="App.openAudienciaEditForm('${a.tipo}','${a.arquivo.split('/').slice(1).join('/')}')">Editar</button><button class="btn-danger-text" onclick="App.cancelarAudiencia('${a.tipo}','${a.arquivo.split('/').slice(1).join('/')}')">Cancelar audiência</button>`}</td>
  </tr>`;
}
function tabelaAudiencias(lista){
  if(!lista.length) return `<div class="empty"><p>Nenhuma audiência em aberto nesta categoria.</p></div>`;
  return `<table><thead><tr><th>Tipo do ato</th><th>Código</th><th>Título</th><th>Data</th><th>Hora</th><th>Local</th><th>Processo</th><th></th></tr></thead>
    <tbody>${lista.map(linhaAudiencia).join('')}</tbody></table>`;
}

function linhaDiligencia(d){
  return `<tr>
    <td>${escapeHTML(d.codigo||'—')}</td>
    <td>${escapeHTML(d.titulo)}</td>
    <td class="tabular">${fmtDateOrRaw(d.dataLimite)}</td>
    <td class="tabular">${escapeHTML(d.hora||'—')}</td>
    <td>${escapeHTML(d.local||'—')}</td>
    <td>${escapeHTML(d.responsavel||'—')}</td>
    <td>${escapeHTML(d.status||'—')}</td>
    <td>${escapeHTML(d.processo||'—')}</td>
    <td class="row-actions"><button class="btn-text" onclick="App.openDiligenciaEditForm('${d.arquivo}')">Editar</button></td>
  </tr>`;
}
function tabelaDiligencias(lista){
  if(!lista.length) return `<div class="empty"><p>Nenhuma diligência em aberto.</p></div>`;
  return `<table><thead><tr><th>Código</th><th>Título</th><th>Data-limite</th><th>Hora</th><th>Local</th><th>Responsável</th><th>Status</th><th>Processo</th><th></th></tr></thead>
    <tbody>${lista.map(linhaDiligencia).join('')}</tbody></table>`;
}
function tarefaDaIntimacao(item){
  return DB.tarefas.find(t => t.origemId===item.hash && String(t.origemTipo||'').toLowerCase()===String(item.origem||'').toLowerCase());
}
function statusIntimacaoClasse(status){
  return status==='confirmada' ? 'badge-ok' : status==='descartada' ? 'badge-neutral' : 'badge-warn';
}
function statusIntimacaoLabel(status){
  return status==='confirmada' ? 'Confirmada' : status==='descartada' ? 'Descartada' : 'Em triagem';
}
function tabelaIntimacoes(lista){
  if(!DB.intimacoes){
    const estado = DB.statusFontes.intimacoes;
    return `<div class="form-block"><strong>Intimações indisponíveis.</strong><br>${escapeHTML(estado&&estado.mensagem || 'Nenhuma sincronização de intimações foi executada.')}</div>`;
  }
  if(!lista.length) return `<div class="empty"><p>Nenhuma intimação registrada no histórico.</p></div>`;
  const rows = lista.map(item=>{
    const tarefa = tarefaDaIntimacao(item);
    const acoes = item.status==='triagem'
      ? `<button class="btn-text" onclick="App.atualizarStatusIntimacao('${escapeHTML(item.hash)}','confirmada')">Confirmar</button><button class="btn-danger-text" onclick="App.atualizarStatusIntimacao('${escapeHTML(item.hash)}','descartada')">Descartar</button>`
      : '—';
    const tarefaHtml = tarefa ? `<strong>${escapeHTML(tarefa.titulo)}</strong><div class="hint">${escapeHTML(tarefa.estadoPasta||tarefa.status||'pendente')}</div>` : '<span class="hint">Tarefa não localizada</span>';
    return `<tr>
      <td>${escapeHTML(item.origem||'—')}</td>
      <td>${escapeHTML(item.processo||'—')}</td>
      <td class="tabular">${fmtDateOrRaw(item.dataPublicacao)}</td>
      <td><span class="badge ${statusIntimacaoClasse(item.status)}">${statusIntimacaoLabel(item.status)}</span></td>
      <td>${tarefaHtml}</td>
      <td class="row-actions">${acoes}</td>
    </tr>`;
  }).join('');
  return `<table><thead><tr><th>Origem</th><th>Processo</th><th>Data</th><th>Status</th><th>Tarefa de análise</th><th></th></tr></thead><tbody>${rows}</tbody></table>`;
}

App.atualizarStatusIntimacao = function(hash, status){
  (async ()=>{
    try{
      await Painel.atualizarStatus('intimacoes', hash, status);
      await App_carregarIntimacoes();
      App.renderNav(); App.renderStats(); App.renderTab();
      showToast(status==='descartada' ? 'Intimação descartada e mantida no histórico.' : 'Intimação confirmada.');
    }catch(err){
      showToast(err.message || 'Falha ao atualizar a intimação.', true);
    }
  })();
};

function tabelaTarefasAbertas(lista){
 return tabelaTarefasCategoriaAgenda(lista)||`<div class="empty"><p>Nenhuma tarefa em aberto.</p></div>`;
}

function filtrarPorBusca(itens){
  if(!state.search) return itens;
  const q=normalizarTexto(state.search);
  return itens.filter(it => {
   const refs=it.origemTipo?referenciasClientesTarefaAgenda(it):(it.cliente?[it.cliente]:[]);
   const nomes=refs.map(ref=>{const p=encontrarPessoaPorReferencia(ref);return [nomePessoaAtendimento(ref),p&&p.codigo].filter(Boolean).join(' ');}).join(' ');
   const responsaveis=(it.responsaveis||[]).map(nomeExibicaoReferenciaPessoa).join(' ');
   const ate=DB.atendimentos.find(x=>x.id===it.atendimentoVinculado);
   const pro=DB.processos.find(x=>x.id===it.processoVinculado);
   return normalizarTexto([JSON.stringify(it),nomes,responsaveis,ate&&ate.codigo,pro&&pro.codigo].filter(Boolean).join(' ')).includes(q);
 });
}

App.viewAgenda = function(){
  const cabecalho = `<label class="sr-only" for="buscaAgenda">Buscar agenda</label>
  <input id="buscaAgenda" class="search" autocomplete="off" placeholder="Buscar por título, tipo, cliente ou responsável…" value="${escapeHTML(state.search)}" oninput="App.buscarAgenda(this)">`; 

  const tarefasAbertas = filtrarPorBusca(DB.tarefas.filter(t=>!ESTADOS_TAREFA_FINALIZADOS.includes(t.status)));

  const categorias = CATEGORIAS_AGENDA.map(c=>{
    if(c.chave==='tarefas'){
      return {chave:c.chave, label:c.label, count:tarefasAbertas.length, html: tabelaTarefasAbertas(tarefasAbertas)};
    }
    if(c.chave==='intimacoes'){
      const itens = filtrarPorBusca(DB.intimacoes || []);
      return {chave:c.chave, label:c.label, count:itens.length, html: tabelaIntimacoes(itens)};
    }
    if(c.chave==='audiencias'){
      const itens = filtrarPorBusca(audienciasEmAberto());
      return {chave:c.chave, label:c.label, count:itens.length, html: tabelaAudiencias(itens)};
    }
    if(c.chave==='diligencias'){
      const itens = filtrarPorBusca(diligenciasEmAberto());
      return {chave:c.chave, label:c.label, count:itens.length, html: tabelaDiligencias(itens)};
    }
    // Ligações, Reuniões e Aulas: compromissos + tarefas abertas com o
    // subtítulo correspondente (item 9 — leitura unificada).
    const itens = filtrarPorBusca(itensAgendaEmAberto(c.tipoAgenda));
    const tarefasCat = filtrarPorBusca(tarefasDaCategoriaAgenda(c.chave));
    let html = '';
    if(itens.length) html += tabelaAgenda(itens);
    html += tabelaTarefasCategoriaAgenda(tarefasCat, 'prazo');
    if(!html) html = `<div class="empty"><p>Nenhum item em aberto nesta categoria.</p></div>`;
    return {chave:c.chave, label:c.label, count:itens.length + tarefasCat.length, html};
  });

  return cabecalho + `<div id="agendaResultados">${renderAcordeao('agenda', categorias)}</div>`;
};

App.buscarAgenda = function(input){
 state.search=input.value;
 const template=document.createElement('template');template.innerHTML=App.viewAgenda();
 const next=template.content.querySelector('#agendaResultados');
 document.getElementById('agendaResultados').replaceWith(next);
};

async function recarregarAgendaERenderizar(){
  try{
    await App_carregarAgenda();
    App.renderNav(); App.renderStats(); App.renderTab();
  }catch(err){
    showToast('Ação registrada, mas houve falha ao recarregar a agenda.', true);
  }
}

// Formulário genérico de "Novo compromisso" — cobre Prazo Fatal, Reunião,
// Ligação, Aula e Diligência. Audiência tem drawer próprio (App.openAudienciaForm)
// porque pede campos que os demais não têm (juízo, cliente, tipo de ato).
function opcoesReferenciaAgenda(lista,valor,label){
 let h='<option value="">Sem vínculo</option>';
 for(const item of lista)h+=`<option value="${escapeHTML(item.id)}" ${item.id===valor?'selected':''}>${escapeHTML(label(item))}</option>`;
 if(valor&&!lista.some(x=>x.id===valor))h+=`<option selected value="${escapeHTML(valor)}">${escapeHTML(valor)} · referência legada</option>`;
 return h;
}
function camposVinculosAgenda(a){
 a=a||{};const pessoa=encontrarPessoaPorReferencia(a.cliente);
 return `<div class="field agenda-vinculos"><label>Cliente (opcional)</label><input type="search" aria-label="Buscar cliente do compromisso" placeholder="Buscar nome ou código PES…" oninput="App.filtrarVinculoAgenda(this)"><select name="cliente">${opcoesReferenciaAgenda(DB.clientes,pessoa?pessoa.id:a.cliente,p=>nomePessoaAtendimento(p.id)+' · '+p.codigo)}</select></div>
 <div class="field agenda-vinculos"><label>Atendimento vinculado (opcional)</label><input type="search" aria-label="Buscar atendimento do compromisso" placeholder="Buscar código ATE, cliente ou objeto…" oninput="App.filtrarVinculoAgenda(this)"><select name="atendimentoVinculado">${opcoesReferenciaAgenda(DB.atendimentos,a.atendimentoVinculado,x=>[x.codigo,nomesAtendimento(x,'clientes'),x.descricao].filter(Boolean).join(' · '))}</select></div>
 <div class="field agenda-vinculos"><label>Processo vinculado (opcional)</label><input type="search" aria-label="Buscar processo do compromisso" placeholder="Buscar código PRO, número ou título…" oninput="App.filtrarVinculoAgenda(this)"><select name="processoVinculado">${opcoesReferenciaAgenda(DB.processos,a.processoVinculado,x=>[x.codigo,x.numeroProcesso,x.title].filter(Boolean).join(' · '))}</select></div>`;
}
App.filtrarVinculoAgenda=function(input){const q=normalizarTexto(input.value);input.closest('.agenda-vinculos').querySelectorAll('option').forEach(o=>o.hidden=!!o.value&&!o.selected&&!normalizarTexto(o.textContent).includes(q));};
App.openAgendaForm = function(){
  const html = `
    <h2>Novo compromisso</h2>
    <p class="drawer-sub">Audiência nunca é bloqueada por conflito de horário; os demais compromissos podem ser. Prazo fatal é campo da tarefa (aba Tarefas), não compromisso.</p>
    <form onsubmit="return App.saveAgenda(event)">
      <div class="field"><label>Tipo *</label>
        <select name="tipo" onchange="App.toggleResponsavelDiligencia(this)" required>
          <option value="">Selecione…</option>
          ${OPCOES.agendaTipo.map(t=>`<option>${escapeHTML(t)}</option>`).join('')}
        </select>
      </div>
      <div class="field"><label>Descrição *</label><input type="text" name="descricao" required></div>
      <div class="row2">
        <div class="field"><label>Data *</label><input type="date" name="data" required></div>
        <div class="field"><label>Hora *</label><input type="time" name="hora" required></div>
      </div>
      <div class="field"><label>Local ou link da aula/reunião</label><input type="text" name="local"><p class="hint">Somente registro do local/link. Não inicia gravação, download ou transcrição.</p></div>
      <div id="camposDiligencia" style="display:none">
        <div class="field" id="campoResponsavelDiligencia">
          <label>Responsável pela diligência</label><input type="text" name="responsavel">
        </div>

      </div>
      ${camposVinculosAgenda()}${campoCriadoPorHTML('')}
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Cancelar</button>
        <button type="submit" class="btn btn-primary">Salvar compromisso</button>
      </div>
    </form>`;
  App.openDrawer(html);
};

App.toggleResponsavelDiligencia = function(sel){
  const bloco = document.getElementById('camposDiligencia');
  if(bloco) bloco.style.display = sel.value==='Diligência' ? '' : 'none';
  const ate=sel.form.querySelector('[name=atendimentoVinculado]');
  if(ate){ate.disabled=sel.value==='Diligência';ate.closest('.field').hidden=sel.value==='Diligência';}
 const criador=sel.form.querySelector('[name=criadoPor]');if(criador)criador.closest('.field').hidden=sel.value==='Diligência';
};

// Diligência grava em ficha própria (/api/diligencias/nova); os demais
// tipos deste formulário continuam em dados/agenda.json (/api/agenda/novo).
App.saveAgenda = function(ev){
  ev.preventDefault();
  const f = ev.target;
  const tipo = f.tipo.value;
  if(!OPCOES.agendaTipo.includes(tipo)){ showToast('Selecione o tipo do compromisso.', true); return false; }
  const descricao = f.descricao.value.trim();
  const data = f.data.value;
  const hora = f.hora.value;
  const local = f.local.value.trim();
  const processoVinculado = f.processoVinculado.value.trim();
  if(!descricao || !data || !hora){ showToast('Descrição, data e hora são obrigatórios.', true); return false; }

  if(tipo==='Diligência'){
    const responsavel = f.responsavel.value.trim();
    const cliente = f.cliente ? f.cliente.value.trim() : '';
    const nomeArquivo = slugify(descricao) + '-' + Date.now().toString(36);
    (async ()=>{
      try{
        const resp = await fetch('/api/diligencias/nova', {
          method:'POST', headers:{'Content-Type':'application/json'},
          body: JSON.stringify({nomeArquivo, titulo:descricao, processo:processoVinculado, cliente, local, dataLimite:data, hora, responsavel, descricao:''})
        });
        const corpo = await resp.json().catch(()=>({}));
        if(resp.status===409){
          showToast(corpo.erro || 'Bloqueado: conflito de horário com um compromisso existente.', true);
          return;
        }
        if(!resp.ok){
          showToast(corpo.erro || 'Falha ao criar a diligência.', true);
          return;
        }
        App.closeDrawer();
        showToast('Diligência criada — código ' + (corpo.codigo || '—') + '.');
        await recarregarDiligenciasERenderizar();
      }catch(err){
        showToast('Falha ao criar a diligência.', true);
      }
    })();
    return false;
  }

  (async ()=>{
    try{
      const resp = await fetch('/api/agenda/novo', {
        method:'POST', headers:{'Content-Type':'application/json'},
        body: JSON.stringify({tipo, descricao, data, hora, local, processoVinculado,cliente:f.cliente.value,atendimentoVinculado:f.atendimentoVinculado.value,criadoPor:f.criadoPor.value})
      });
      const corpo = await resp.json().catch(()=>({}));
      if(resp.status===409){
        showToast(corpo.erro || 'Bloqueado: conflito de horário com um compromisso existente.', true);
        return;
      }
      if(!resp.ok){
        showToast(corpo.erro || 'Falha ao criar o compromisso.', true);
        return;
      }
      App.closeDrawer();
      showToast(corpo.aviso ? 'Compromisso criado — ' + corpo.aviso : 'Compromisso criado.');
      await recarregarAgendaERenderizar();
    }catch(err){
      showToast('Falha ao criar o compromisso.', true);
    }
  })();
  return false;
};

async function recarregarDiligenciasERenderizar(){
  try{
    await App_carregarDiligencias();
    App.renderNav(); App.renderStats(); App.renderTab();
  }catch(err){
    showToast('Ação registrada, mas houve falha ao recarregar as diligências.', true);
  }
}
App.openDiligenciaEditForm = function(arquivoNome){
  const d = DB.diligencias.find(x=>x.arquivo===arquivoNome);
  if(!d){ showToast('Diligência não encontrada.', true); return; }
  const html = `
    <h2>Editar diligência</h2>
    <p class="drawer-sub">${escapeHTML(d.titulo)}</p>
    <form onsubmit="return App.updateDiligencia(event,'${arquivoNome}')">
      <div class="field"><label>Título *</label><input type="text" name="titulo" required value="${escapeHTML(d.titulo)}"></div>
      <div class="field"><label>Processo vinculado</label><input type="text" name="processo" oninput="this.value=maskCNJ(this.value)" value="${escapeHTML(d.processo)}"></div>
      <div class="field"><label>Cliente</label><select name="cliente">${opcoesPessoas(d.cliente)}</select></div>
      <div class="row2">
        <div class="field"><label>Data-limite *</label><input type="date" name="dataLimite" required value="${/^\d{4}-\d{2}-\d{2}$/.test(d.dataLimite)?d.dataLimite:''}"></div>
        <div class="field"><label>Hora</label><input type="time" name="hora" value="${escapeHTML(d.hora)}"></div>
      </div>
      <div class="field"><label>Local</label><input type="text" name="local" value="${escapeHTML(d.local)}"></div>
      <div class="field"><label>Responsável</label><input type="text" name="responsavel" value="${escapeHTML(d.responsavel)}"></div>
      <div class="hint" style="margin-bottom:14px;">Status não se edita por aqui.</div>
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Cancelar</button>
        <button type="submit" class="btn btn-primary">Salvar alterações</button>
      </div>
    </form>`;
  App.openDrawer(html);
};

App.updateDiligencia = function(ev, arquivoNome){
  ev.preventDefault();
  const f = ev.target;
  const payload = {
    title: f.titulo.value.trim(),
    processo: f.processo.value.trim(),
    cliente: f.cliente.value.trim(),
    local: f.local.value.trim(),
    data_limite: f.dataLimite.value,
    hora: f.hora.value,
    responsavel: f.responsavel.value.trim()
  };
  (async ()=>{
    try{
      const resp = await fetch('/api/diligencias/'+encodeURIComponent(arquivoNome)+'/editar', {
        method:'PUT', headers:{'Content-Type':'application/json'},
        body: JSON.stringify(payload)
      });
      const corpo = await resp.json().catch(()=>({}));
      if(!resp.ok){
        showToast(corpo.erro || 'Falha ao atualizar a diligência.', true);
        return;
      }
      App.closeDrawer();
      if(Array.isArray(corpo.naoEncontrados) && corpo.naoEncontrados.length){
        showToast('Diligência atualizada, mas estes campos não existiam na ficha: '+corpo.naoEncontrados.join(', ')+'.', true);
      }else{
        showToast('Diligência atualizada.');
      }
      await recarregarDiligenciasERenderizar();
    }catch(err){
      showToast('Falha ao atualizar a diligência.', true);
    }
  })();
  return false;
};

async function recarregarAudienciasERenderizar(){
  try{
    await App_carregarAudiencias();
    App.renderNav(); App.renderStats(); App.renderTab();
  }catch(err){
    showToast('Ação registrada, mas houve falha ao recarregar as audiências.', true);
  }
}

App.deleteAgenda = function(){ toastNaoImplementado(); };

App.openAgendaEditForm = function(id){
  const a = DB.agenda.find(x=>x.id===id);
  if(!a){ showToast('Item de agenda não encontrado.', true); return; }
  const html = `
    <h2>Editar compromisso</h2>
    <p class="drawer-sub">${escapeHTML(a.descricao)}</p>
    <form onsubmit="return App.updateAgenda(event,'${id}')">
      <div class="field"><label>Tipo *</label>
        <select name="tipo" onchange="App.toggleResponsavelDiligencia(this)" required>
          ${OPCOES.agendaTipo.map(t=>`<option ${t===a.tipo?'selected':''}>${escapeHTML(t)}</option>`).join('')}
        </select>
      </div>
      <div class="field"><label>Descrição *</label><input type="text" name="descricao" required value="${escapeHTML(a.descricao)}"></div>
      <div class="row2">
        <div class="field"><label>Data *</label><input type="date" name="data" required value="${/^\d{4}-\d{2}-\d{2}$/.test(a.data)?a.data:''}"></div>
        <div class="field"><label>Hora *</label><input type="time" name="hora" required value="${escapeHTML(a.hora)}"></div>
      </div>
      <div class="field"><label>Local ou link da aula/reunião</label><input type="text" name="local" value="${escapeHTML(a.local)}"></div>
      ${camposVinculosAgenda(a)}${campoCriadoPorHTML(a.criadoPor)}
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Cancelar</button>
        <button type="submit" class="btn btn-primary">Salvar alterações</button>
      </div>
    </form>`;
  App.openDrawer(html);
};

App.updateAgenda = function(ev, id){
  ev.preventDefault();
  const f = ev.target;
  const payload = {
    tipo: f.tipo.value,
    descricao: f.descricao.value.trim(),
    data: f.data.value,
    hora: f.hora.value,
    local: f.local.value.trim(),
    processoVinculado: f.processoVinculado.value.trim(),
 cliente:f.cliente.value,atendimentoVinculado:f.atendimentoVinculado.value,criadoPor:f.criadoPor.value
  };
  (async ()=>{
    try{
      const resp = await fetch('/api/agenda/'+encodeURIComponent(id)+'/editar', {
        method:'PUT', headers:{'Content-Type':'application/json'},
        body: JSON.stringify(payload)
      });
      const corpo = await resp.json().catch(()=>({}));
      if(resp.status===409){
        showToast(corpo.erro || 'Bloqueado: conflito de horário com um compromisso existente.', true);
        return;
      }
      if(!resp.ok){
        showToast(corpo.erro || 'Falha ao atualizar o compromisso.', true);
        return;
      }
      App.closeDrawer();
      showToast('Compromisso atualizado.');
      await recarregarAgendaERenderizar();
    }catch(err){
      showToast('Falha ao atualizar o compromisso.', true);
    }
  })();
  return false;
};

/* ---------------------------------------------------------
   VIEW: AUDIÊNCIAS — acordeão horizontal pelas 6 pastas reais de
   11-TEODORO-ADVOCACIA/audiencias/ (TIPOS_AUDIENCIA), fonte de verdade em
   ficha .md real (DB.audiencias), não mais item de agenda.json.
   --------------------------------------------------------- */
App.viewAudiencias = function(){
  const toolbar = `<div class="toolbar">
    <label class="sr-only" for="buscaAudiencias">Buscar audiências</label>
    <input id="buscaAudiencias" class="search" autocomplete="off" placeholder="Buscar por título, processo, cliente ou juízo…" value="${escapeHTML(state.search)}" oninput="App.setSearch(this.value)">
  </div>`;
  const categorias = TIPOS_AUDIENCIA.map(t=>{
    const itens = filtrarPorBusca(DB.audiencias.filter(a=>a.tipo===t.chave).sort((a,b)=>(a.data+a.hora).localeCompare(b.data+b.hora)));
    return {chave:t.chave, label:t.label, count:itens.length, html: tabelaAudiencias(itens)};
  });
  return toolbar + renderAcordeao('audiencias', categorias);
};

// Drawer próprio de audiência: tipo do ato (as 6 pastas reais), processo,
// cliente, juízo, data/hora/local — grava em /api/audiencias/nova.
App.openAudienciaForm = function(){
  const html = `
    <h2>Nova audiência</h2>
    <p class="drawer-sub">Audiência é item crítico: nunca é bloqueada por conflito de horário, só avisa se coincidir com outro item crítico.</p>
    <form onsubmit="return App.saveAudiencia(event)">
      <div class="field"><label>Tipo do ato *</label>
        <select name="tipo" required>
          <option value="">Selecione…</option>
          ${TIPOS_AUDIENCIA.map(t=>`<option value="${t.chave}">${escapeHTML(t.label)}</option>`).join('')}
        </select>
      </div>
      <div class="field"><label>Título *</label><input type="text" name="titulo" required placeholder="Ex.: Audiência de instrução — fulano x banco"></div>
      <div class="field"><label>Processo vinculado</label><input type="text" name="processo" oninput="this.value=maskCNJ(this.value)"></div>
      <div class="field"><label>Cliente</label><select name="cliente">${opcoesPessoas('')}</select></div>
      <div class="row2">
        <div class="field"><label>Data *</label><input type="date" name="data" required></div>
        <div class="field"><label>Hora *</label><input type="time" name="hora" required></div>
      </div>
      <div class="field"><label>Juízo</label><input type="text" name="juizo"></div>
      <div class="field"><label>Local ou link da aula/reunião</label><input type="text" name="local"><p class="hint">Somente registro do local/link. Não inicia gravação, download ou transcrição.</p></div>
      <div class="field"><label>Observações</label><textarea name="descricao" rows="3"></textarea></div>
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Cancelar</button>
        <button type="submit" class="btn btn-primary">Salvar audiência</button>
      </div>
    </form>`;
  App.openDrawer(html);
};

App.saveAudiencia = function(ev){
  ev.preventDefault();
  const f = ev.target;
  const tipo = f.tipo.value;
  if(!TIPOS_AUDIENCIA.some(t=>t.chave===tipo)){ showToast('Selecione o tipo do ato.', true); return false; }
  const titulo = f.titulo.value.trim();
  const processo = f.processo.value.trim();
  const cliente = f.cliente.value.trim();
  const data = f.data.value;
  const hora = f.hora.value;
  const juizo = f.juizo.value.trim();
  const local = f.local.value.trim();
  const descricao = f.descricao.value.trim();
  if(!titulo || !data || !hora){ showToast('Título, data e hora são obrigatórios.', true); return false; }
  const nomeArquivo = slugify(titulo) + '-' + Date.now().toString(36);
  (async ()=>{
    try{
      const resp = await fetch('/api/audiencias/nova', {
        method:'POST', headers:{'Content-Type':'application/json'},
        body: JSON.stringify({nomeArquivo, tipo, titulo, processo, cliente, data, hora, juizo, local, descricao})
      });
      const corpo = await resp.json().catch(()=>({}));
      if(!resp.ok){
        showToast(corpo.erro || 'Falha ao criar a audiência.', true);
        return;
      }
      App.closeDrawer();
      showToast(corpo.aviso ? 'Audiência criada — ' + corpo.aviso : 'Audiência criada — código ' + (corpo.codigo || '—') + '.');
      await recarregarAudienciasERenderizar();
    }catch(err){
      showToast('Falha ao criar a audiência.', true);
    }
  })();
  return false;
};

App.openAudienciaEditForm = function(tipo, arquivoNome){
  const a = DB.audiencias.find(x=>x.tipo===tipo && x.arquivo===tipo+'/'+arquivoNome);
  if(!a){ showToast('Audiência não encontrada.', true); return; }
  const html = `
    <h2>Editar audiência</h2>
    <p class="drawer-sub">${escapeHTML(a.titulo)}</p>
    <form onsubmit="return App.updateAudiencia(event,'${tipo}','${arquivoNome}')">
      <div class="field"><label>Tipo do ato</label><input type="text" value="${escapeHTML(labelTipoAudiencia(tipo))}" disabled></div>
      <div class="field"><label>Título *</label><input type="text" name="titulo" required value="${escapeHTML(a.titulo)}"></div>
      <div class="field"><label>Processo vinculado</label><input type="text" name="processo" oninput="this.value=maskCNJ(this.value)" value="${escapeHTML(a.processo)}"></div>
      <div class="field"><label>Cliente</label><select name="cliente">${opcoesPessoas(a.cliente)}</select></div>
      <div class="row2">
        <div class="field"><label>Data *</label><input type="date" name="data" required value="${/^\d{4}-\d{2}-\d{2}$/.test(a.data)?a.data:''}"></div>
        <div class="field"><label>Hora *</label><input type="time" name="hora" required value="${escapeHTML(a.hora)}"></div>
      </div>
      <div class="field"><label>Juízo</label><input type="text" name="juizo" value="${escapeHTML(a.juizo)}"></div>
      <div class="field"><label>Local ou link da aula/reunião</label><input type="text" name="local" value="${escapeHTML(a.local)}"></div>
      <div class="hint" style="margin-bottom:14px;">Tipo do ato não se edita aqui (é a pasta da ficha).</div>
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Cancelar</button>
        <button type="submit" class="btn btn-primary">Salvar alterações</button>
      </div>
    </form>`;
  App.openDrawer(html);
};

App.updateAudiencia = function(ev, tipo, arquivoNome){
  ev.preventDefault();
  const f = ev.target;
  const payload = {
    title: f.titulo.value.trim(),
    processo: f.processo.value.trim(),
    cliente: f.cliente.value.trim(),
    data: f.data.value,
    hora: f.hora.value,
    juizo: f.juizo.value.trim(),
    local: f.local.value.trim()
  };
  (async ()=>{
    try{
      const resp = await fetch('/api/audiencias/'+encodeURIComponent(tipo)+'/'+encodeURIComponent(arquivoNome)+'/editar', {
        method:'PUT', headers:{'Content-Type':'application/json'},
        body: JSON.stringify(payload)
      });
      const corpo = await resp.json().catch(()=>({}));
      if(!resp.ok){
        showToast(corpo.erro || 'Falha ao atualizar a audiência.', true);
        return;
      }
      App.closeDrawer();
      if(Array.isArray(corpo.naoEncontrados) && corpo.naoEncontrados.length){
        showToast('Audiência atualizada, mas estes campos não existiam na ficha: '+corpo.naoEncontrados.join(', ')+'.', true);
      }else{
        showToast('Audiência atualizada.');
      }
      await recarregarAudienciasERenderizar();
    }catch(err){
      showToast('Falha ao atualizar a audiência.', true);
    }
  })();
  return false;
};

App.cancelarAudiencia = function(tipo, arquivoNome){
  const html = `
    <h2>Cancelar audiência</h2>
    <p class="drawer-sub">A ficha é preservada com o motivo do cancelamento. Confirme com a sua senha.</p>
    <form onsubmit="return App.confirmarCancelarAudiencia(event,${inlineArg(tipo)},${inlineArg(arquivoNome)})">
      <div class="field"><label>Motivo do cancelamento *</label><textarea name="motivo" required placeholder="Será preservado na ficha"></textarea></div>
      <div class="field"><label>Senha *</label><input type="password" name="senha" autocomplete="current-password" required></div>
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Voltar</button>
        <button type="submit" class="btn btn-primary">Cancelar audiência</button>
      </div>
    </form>`;
  App.openDrawer(html);
};
App.confirmarCancelarAudiencia = function(ev, tipo, arquivoNome){
  ev.preventDefault();
  const motivo = ev.target.motivo.value.trim();
  const senha = ev.target.senha.value;
  (async()=>{ try{
    const resp=await fetch('/api/audiencias/'+encodeURIComponent(tipo)+'/'+encodeURIComponent(arquivoNome)+'/cancelar',{method:'POST',headers:{'Content-Type':'application/json','X-Painel-Admin-Password':senha},body:JSON.stringify({motivo})});
    if(!resp.ok){ const corpo=await resp.json().catch(()=>({})); showToast(corpo.erro||'Não foi possível cancelar a audiência.',true); return; }
    App.closeDrawer();
    showToast('Audiência cancelada; a ficha foi preservada.'); await recarregarAudienciasERenderizar();
  }catch(_){ showToast('Falha ao cancelar a audiência.',true); } })();
  return false;
};
