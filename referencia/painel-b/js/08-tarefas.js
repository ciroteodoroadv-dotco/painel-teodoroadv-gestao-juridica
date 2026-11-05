/* =========================================================
   PAINEL TEODORO - modulo 08-tarefas: view Tarefas: tabela, formulário, mover estado, subtarefas, documentos
   Extraido de web/index.html (linhas 928-1318) na etapa 4b da
   refatoracao (2026-10-30). Scripts classicos carregados em ordem
   numerica pelo index.html: compartilham o escopo global lexico,
   exatamente como antes dentro da IIFE unica. Sem build toolchain.
   ========================================================= */
'use strict';

/* ---------------------------------------------------------
   VIEW: TAREFAS
   --------------------------------------------------------- */
function linhaTarefa(t){
  const responsaveisTexto = (t.responsaveis||[]).map(nomeExibicaoReferenciaPessoa).join(', ');
  const cliente = clienteNomeDoVinculo(t);
  const origem = codigoVinculoTarefa(t);
  const prioridade = t.prioridade||'—';
  return `<tr>
    <td>${escapeHTML(t.codigo||'—')}</td>
    <td><div class="tarefa-info">
      <strong>${escapeHTML(t.titulo)}</strong>
      ${t.subtitulo?`<div class="tarefa-subtitulo">${escapeHTML(t.subtitulo)}</div>`:''}
      ${t.criadoPor?`<div class="hint">Criado por: ${escapeHTML(nomeExibicaoReferenciaPessoa(t.criadoPor))}</div>`:''}
      ${cliente?`<div class="tarefa-client">${escapeHTML(cliente)}</div>`:''}
      <div class="tarefa-meta">
        <div class="tarefa-meta-item"><span class="tarefa-meta-label">Origem</span><span class="tarefa-meta-value">${escapeHTML(origem||'—')}</span></div>
        <div class="tarefa-meta-item"><span class="tarefa-meta-label">Responsável</span><span class="tarefa-meta-value">${escapeHTML(responsaveisTexto||'—')}</span></div>
        <div class="tarefa-meta-item"><span class="tarefa-meta-label">Prazo</span><span class="tarefa-meta-value tabular">${escapeHTML(fmtDateOrRaw(t.prazo))}</span></div>
        <div class="tarefa-meta-item"><span class="tarefa-meta-label">Prioridade</span><span class="tarefa-meta-value"><span class="badge ${prioridadeBadgeClass(prioridade)}">${escapeHTML(prioridade)}</span></span></div>
      </div>
      ${t.prazoFatal?`<span class="badge badge-danger">Prazo fatal: ${escapeHTML(fmtDateOrRaw(t.prazoFatal))}</span>`:''}
      ${t.descricao?`<div class="hint" style="color:var(--text-muted);margin-top:3px;">${escapeHTML(t.descricao)}</div>`:''}
    </div></td>
    <td class="row-actions">
      <select aria-label="Mover tarefa ${escapeHTML(t.codigo||t.titulo)} para outro estado" onchange="App.acaoMoverEstadoTarefa(${inlineArg(t.estadoPasta)},${inlineArg(t.arquivoNome)},this.value)">
        <option value="">Mover para...</option>
        ${CATEGORIAS_TAREFA.filter(c=>c.chave!==t.estadoPasta).map(c=>`<option value="${c.chave}">${escapeHTML(c.label)}</option>`).join('')}
      </select>
      <button class="btn-text" onclick="App.openTarefaEditForm(${inlineArg(t.estadoPasta)},${inlineArg(t.arquivoNome)})">Editar</button>
      <button class="btn-text" onclick="App.openTarefaFilha(${inlineArg(t.estadoPasta)},${inlineArg(t.arquivoNome)})">Tarefa vinculada</button>
      <button class="btn-danger-text" onclick="App.deleteTarefa(${inlineArg(t.estadoPasta)},${inlineArg(t.arquivoNome)})">Excluir</button>
    </td>
  </tr>`;
}
function tabelaTarefasDoEstado(lista){
  if(!lista.length) return `<div class="empty"><p>Nenhuma tarefa neste estado.</p></div>`;
  return `<table><thead><tr>
    <th scope="col">Código</th><th scope="col">Tarefa</th><th scope="col">Ações</th>
  </tr></thead><tbody>${lista.map(linhaTarefa).join('')}</tbody></table>`;
}

App.viewTarefas = function(){
  let list = DB.tarefas.slice();
  if(state.search){
    list = list.filter(t => {
      const partes = (t.partes||[]).map(p=>nomeExibicaoReferenciaPessoa(p.pessoa)).join(' ');
      const cliente = clienteNomeDoVinculo(t);
      const origem = codigoVinculoTarefa(t);
      return (t.codigo+' '+t.titulo+' '+(t.subtitulo||'')+' '+cliente+' '+origem+' '+(t.responsaveis||[]).map(nomeExibicaoReferenciaPessoa).join(' ')+' '+partes+' '+(t.descricao||'')+' '+(t.observacoes||'')).toLowerCase().includes(state.search);
    });
  }
  const toolbar = `<form class="toolbar tarefa-search" onsubmit="return App.aplicarBuscaTarefas(event)">
    <label class="sr-only" for="buscaTarefas">Buscar tarefas</label>
    <input id="buscaTarefas" class="search" name="buscaTarefas" autocomplete="off" placeholder="Buscar por título, cliente, código ou responsável…" value="${escapeHTML(state.searchDraft||state.search)}" oninput="App.setSearchDraft(this.value)">
    <button type="submit" class="btn btn-primary btn-small">Buscar</button>
  </form>`;

  if(list.length===0){
    return toolbar + `<div class="empty">
      <h3>Nenhuma tarefa cadastrada</h3>
      <p>Crie a primeira tarefa pelo botão "Nova tarefa" acima.</p>
    </div>`;
  }

  const categorias = CATEGORIAS_TAREFA.map(c=>{
    const itens = list.filter(t=>t.estadoPasta===c.chave);
    return {chave:c.chave, label:c.label, count:itens.length, html: tabelaTarefasDoEstado(itens)};
  });

  return toolbar + renderAcordeao('tarefas', categorias);
};

async function recarregarTarefasERenderizar(){
  try{
    await App_carregarTarefas();
    App.renderNav(); App.renderStats(); App.renderTab();
  }catch(err){
    showToast('Ação registrada, mas houve falha ao recarregar a lista de tarefas.', true);
  }
}

function origemOptionsHTML(tipo, selecionado){
  if(tipo==='atendimento'){
    return DB.atendimentos.map(a=>`<option value="${a.id}" ${a.id===selecionado?'selected':''}>${escapeHTML(a.codigo||'ATE. [VERIFICAR]')} · ${escapeHTML(a.clienteNome||'sem cliente')}</option>`).join('');
  }
  if(tipo==='processo'){
    return DB.processos.map(p=>`<option value="${p.id}" ${p.id===selecionado?'selected':''}>${escapeHTML(p.codigo||'PRO. [VERIFICAR]')} · ${escapeHTML(p.clienteNome||'sem cliente')}</option>`).join('');
  }
  return '';
}
// refreshOrigemOptions não aceita mais 'avulsa' — toda tarefa nasce
// vinculada a Atendimento ou Processo (campo sempre visível e obrigatório).
App.refreshOrigemOptions = function(tipo){
  const sel = document.getElementById('origemIdSelect');
  sel.innerHTML = origemOptionsHTML(tipo);
};

App.openTarefaForm = function(tarefaPai, sobrepor){
  const vinculoInicial = tarefaPai ? tarefaPai.origemTipo : (DB.atendimentos.length ? 'atendimento' : (DB.processos.length ? 'processo' : ''));
  const origemInicial = tarefaPai ? tarefaPai.origemId : '';
  const html = `
    <h2>Nova tarefa</h2>
    <p class="drawer-sub">Toda tarefa nasce vinculada a um atendimento ou a um processo já existente.</p>
    <form onsubmit="return App.saveTarefa(event)">
      <div class="field">
        <label>Vínculo *</label>
        <div class="radio-row">
          <label class="radio-opt"><input type="radio" name="origemTipo" value="atendimento" required onchange="App.refreshOrigemOptions(this.value)" ${DB.atendimentos.length===0?'disabled':''} ${vinculoInicial==='atendimento'?'checked':''}> Atendimento</label>
          <label class="radio-opt"><input type="radio" name="origemTipo" value="processo" required onchange="App.refreshOrigemOptions(this.value)" ${DB.processos.length===0?'disabled':''} ${vinculoInicial==='processo'?'checked':''}> Processo</label>
        </div>
      </div>
      <div class="field" id="origemIdField">
        <label>Vínculo específico *</label>
        <select name="origemId" id="origemIdSelect" required>${origemOptionsHTML(vinculoInicial, origemInicial)}</select>
      </div>
      <input type="hidden" name="tarefaPai" value="${escapeHTML(tarefaPai?tarefaPai.codigo||'':'')}">
      <div class="field"><label>Título *</label><input type="text" name="titulo" required placeholder="Ex.: Protocolar embargos à execução"></div>
      <div class="field"><label>Subtítulo</label><input type="text" name="subtitulo" list="sugestoesSubtituloTarefa">
        <datalist id="sugestoesSubtituloTarefa">
          <option value="Ligação"></option><option value="Reunião"></option><option value="Aula"></option>
          <option value="Análise de decisão"></option><option value="Monitoramento"></option>
        </datalist>
        <div class="hint">Tarefas de Ligação, Reunião ou Aula aparecem também na categoria correspondente da agenda.</div>
      </div>
      <fieldset style="border:1px solid var(--border);border-radius:8px;padding:12px;margin-bottom:14px;">
        <legend>Partes</legend>
        <div id="partesTarefaRows"></div>
        <button type="button" class="btn-text" onclick="adicionarLinhaParteTarefa()">+ adicionar parte</button>
      </fieldset>
      <div class="field"><label>Descrição</label><textarea name="descricao"></textarea></div>
      <div class="row2">
        ${campoCriadoPorHTML('')}<div class="field"><label>Data base</label>${campoDataHtml('dataBase','')}</div>
        <div class="field"><label>Prazo de entrega</label>${campoDataHtml('prazo','')}</div>
      </div>
      <div class="field"><label>Prazo fatal</label>${campoDataHtml('prazoFatal','')}</div>
      <fieldset style="border:1px solid var(--border);border-radius:8px;padding:12px;margin-bottom:14px;">
        <legend>Responsáveis existentes</legend>
        <div id="responsaveisTarefaRows"></div>
        <button type="button" class="btn-text" onclick="adicionarLinhaResponsavelTarefa()">+ adicionar responsável</button>
      </fieldset>
      <div class="row2">
        <div class="field"><label>Status</label>
          <select name="estado">${CATEGORIAS_TAREFA.map(c=>`<option value="${c.chave}" ${c.chave==='pendente'?'selected':''}>${escapeHTML(c.label)}</option>`).join('')}</select>
        </div>
        <div class="field"><label>Prioridade</label>
          <select name="prioridade">${OPCOES.prioridade.map(p=>`<option ${p==='Normal'?'selected':''}>${escapeHTML(p)}</option>`).join('')}</select>
        </div>
      </div>
      <div class="field"><label>Observações</label><textarea name="observacoes"></textarea></div>
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Cancelar</button>
        <button type="submit" class="btn btn-primary">Salvar tarefa</button>
      </div>
    </form>`;
  App.openDrawer(html, !!sobrepor);
  if(tarefaPai && tarefaPai.partes && tarefaPai.partes.length){
    tarefaPai.partes.forEach(parte=>adicionarLinhaParteTarefa(parte));
  }else adicionarLinhaParteTarefa();
  if(tarefaPai && tarefaPai.responsaveis && tarefaPai.responsaveis.length){
    tarefaPai.responsaveis.forEach(resp=>adicionarLinhaResponsavelTarefa(resp));
  }else adicionarLinhaResponsavelTarefa();
};

App.saveTarefa = function(ev){
  ev.preventDefault();
  const f = ev.target;
  const titulo = f.titulo.value.trim();
  if(!titulo){ showToast('Título é obrigatório.', true); return false; }
  const origemTipoEl = f.querySelector('input[name="origemTipo"]:checked');
  if(!origemTipoEl){ showToast('Selecione o vínculo (Atendimento ou Processo).', true); return false; }
  const origemTipo = origemTipoEl.value;
  const origemId = f.origemId.value;
  if(!origemId){ showToast('Selecione o vínculo específico.', true); return false; }
  const subtitulo = f.subtitulo.value.trim();
  const partes = coletarPartesTarefaDoForm(f);
  const responsaveis = coletarResponsaveisTarefaDoForm(f);
  const dataBase = f.dataBase.value;
  const prazo = f.prazo.value;
  const prazoFatal = f.prazoFatal.value;
  const prioridade = f.prioridade.value;
  const estado = f.estado.value;
  const descricao = f.descricao.value.trim();
  const observacoes = f.observacoes.value.trim();
  const nomeArquivo = slugify(titulo) + '-' + Date.now().toString(36);
  const submit = f.querySelector('button[type="submit"]');
  submit.disabled = true;
  submit.textContent = 'Salvando…';
  (async ()=>{
    try{
      const resp = await fetch('/api/tarefas/nova', {
        method:'POST', headers:{'Content-Type':'application/json'},
        body: JSON.stringify({nomeArquivo, titulo, criadoPor:f.criadoPor.value, subtitulo, partes, responsaveis, dataBase, prazo, prazoFatal, prioridade, estado, origemTipo, origemId, tarefaPai:f.tarefaPai.value, descricao, observacoes})
      });
      if(!resp.ok){
        if(resp.status===409) showToast('Já existe uma tarefa com esse nome de arquivo — tente de novo.', true);
        else showToast('Falha ao criar a tarefa: '+await resp.text(), true);
        submit.disabled = false;
        submit.textContent = 'Salvar tarefa';
        return;
      }
      App.closeDrawer();
      showToast('Tarefa criada.');
      await recarregarTarefasERenderizar();
    }catch(err){
      submit.disabled = false;
      submit.textContent = 'Salvar tarefa';
      showToast('Falha ao criar a tarefa.', true);
    }
  })();
  return false;
};

App.acaoMoverEstadoTarefa = async function(estadoPasta, arquivoNome, novoEstado){
  if(!novoEstado) return;
  try{
    await Painel.acao('tarefas', [estadoPasta, arquivoNome], 'mover-estado', {novoEstado}, 'PATCH');
    showToast('Estado da tarefa atualizado.');
    await recarregarTarefasERenderizar();
  }catch(err){
    showToast('Falha ao mover o estado da tarefa.', true);
  }
};

App.openTarefaFilha = function(estadoPasta, arquivoNome){
  const tarefa = DB.tarefas.find(x=>x.estadoPasta===estadoPasta && x.arquivoNome===arquivoNome);
  if(!tarefa){ showToast('Tarefa não encontrada.', true); return; }
  App.openTarefaForm(tarefa, true);
};

App.deleteTarefa = function(estadoPasta, arquivoNome){
  const html = `
    <h2>Excluir tarefa</h2>
    <p class="drawer-sub">Excluir definitivamente esta tarefa? Esta ação não pode ser desfeita. Confirme com a senha de administrador.</p>
    <form onsubmit="return App.confirmarExcluirTarefa(event,${inlineArg(estadoPasta)},${inlineArg(arquivoNome)})">
      <div class="field"><label>Senha de administrador *</label><input type="password" name="senha" autocomplete="current-password" required></div>
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Cancelar</button>
        <button type="submit" class="btn btn-primary">Excluir definitivamente</button>
      </div>
    </form>`;
  App.openDrawer(html);
};
App.confirmarExcluirTarefa = function(ev, estadoPasta, arquivoNome){
  ev.preventDefault();
  const senha = ev.target.senha.value;
  (async ()=>{
    try{
      await Painel.acao('tarefas', [estadoPasta, arquivoNome], 'excluir', {}, 'DELETE',
        {'X-Painel-Admin-Password': senha.trim()});
      App.closeDrawer();
      showToast('Tarefa excluída.');
      await recarregarTarefasERenderizar();
    }catch(err){
      showToast('Não foi possível excluir a tarefa: '+err.message, true);
    }
  })();
  return false;
};

// vinculoTarefaTexto monta o texto estático do "Vínculo específico" no
// Editar Tarefa — mesmo rótulo de origemOptionsHTML, mas não editável: o
// vínculo nasce com a tarefa e não se troca depois (pedido do advogado).
function vinculoTarefaTexto(origemTipo, origemId){
  if(origemTipo==='atendimento'){
    const a = DB.atendimentos.find(x=>x.id===origemId);
    return a ? (a.codigo||'ATE. [VERIFICAR]') : 'ATE. [VERIFICAR]';
  }
  if(origemTipo==='processo'){
    const p = DB.processos.find(x=>x.id===origemId);
    return p ? (p.codigo||'PRO. [VERIFICAR]') : 'PRO. [VERIFICAR]';
  }
  return '—';
}

App.openTarefaEditForm = function(estadoPasta, arquivoNome){
  const t = DB.tarefas.find(x=>x.estadoPasta===estadoPasta && x.arquivoNome===arquivoNome);
  if(!t){ showToast('Tarefa não encontrada.', true); return; }
  const html = `
    <h2>Editar tarefa</h2>
    <p class="drawer-sub">${escapeHTML(t.titulo)}</p>
    <form onsubmit="return App.updateTarefa(event,${inlineArg(estadoPasta)},${inlineArg(arquivoNome)})">
      <input type="hidden" name="origemTipo" value="${escapeHTML(t.origemTipo)}">
      <input type="hidden" name="origemId" value="${escapeHTML(t.origemId)}">
      <div class="field">
        <label>Vínculo específico</label>
        <div class="hint" style="padding:8px 0;">${escapeHTML(vinculoTarefaTexto(t.origemTipo, t.origemId))}</div>
      </div>
      <div class="field"><label>Título *</label><input type="text" name="titulo" required value="${escapeHTML(t.titulo)}"></div>
      <div class="field"><label>Subtítulo</label><input type="text" name="subtitulo" value="${escapeHTML(t.subtitulo)}" list="sugestoesSubtituloTarefa">
        <div class="hint">Ligação, Reunião ou Aula aparecem também na categoria correspondente da agenda.</div>
      </div>
      <fieldset style="border:1px solid var(--border);border-radius:8px;padding:12px;margin-bottom:14px;">
        <legend>Partes</legend>
        <div id="partesTarefaRows">${(t.partes&&t.partes.length?t.partes:[{}]).map(linhaParteTarefaHtml).join('')}</div>
        <button type="button" class="btn-text" onclick="adicionarLinhaParteTarefa()">+ adicionar parte</button>
      </fieldset>
      <div class="field"><label>Descrição</label><textarea name="descricao" placeholder="Registre aqui as atualizações da tarefa">${escapeHTML(t.descricao)}</textarea></div>
      <fieldset style="border:1px solid var(--border);border-radius:8px;padding:12px;margin-bottom:14px;">
        <legend>Responsáveis existentes</legend>
        <div id="responsaveisTarefaRows">${(t.responsaveis&&t.responsaveis.length?t.responsaveis:['']).map(linhaResponsavelTarefaHtml).join('')}</div>
        <button type="button" class="btn-text" onclick="adicionarLinhaResponsavelTarefa()">+ adicionar responsável</button>
      </fieldset>
      ${campoCriadoPorHTML(t.criadoPor)}<div class="field"><label>Data base</label>${campoDataHtml('dataBase',t.dataBase)}</div>
      <div class="row2">
        <div class="field"><label>Prazo de entrega</label><input type="date" name="prazo" value="${/^\d{4}-\d{2}-\d{2}$/.test(t.prazo||'')?t.prazo:''}"></div>
        <div class="field"><label>Prazo fatal</label><input type="date" name="prazoFatal" value="${/^\d{4}-\d{2}-\d{2}$/.test(t.prazoFatal||'')?t.prazoFatal:''}"></div>
      </div>
      <div class="row2">
        <div class="field"><label>Prioridade</label>
          <select name="prioridade">${OPCOES.prioridade.map(p=>`<option ${p===t.prioridade?'selected':''}>${escapeHTML(p)}</option>`).join('')}</select>
        </div>
        <div class="field"><label>Status</label>
          <select name="estado">${CATEGORIAS_TAREFA.map(c=>`<option value="${c.chave}" ${c.chave===t.estadoPasta?'selected':''}>${escapeHTML(c.label)}</option>`).join('')}</select>
        </div>
      </div>
      <div class="field"><label>Observações</label><textarea name="observacoes">${escapeHTML(t.observacoes)}</textarea></div>
      <div class="field">
        <label for="documentoTarefa">Documentos do caso</label>
        <input id="documentoTarefa" name="documento" type="file" accept=".pdf,application/pdf" onchange="App.adicionarDocumentoTarefa(event,${inlineArg(estadoPasta)},${inlineArg(arquivoNome)})">
        <div class="hint">O PDF será guardado em <code>01-documentos</code> do caso e convertido para .md.</div>
      </div>
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Cancelar</button>
        <button type="submit" class="btn btn-primary">Salvar alterações</button>
      </div>
    </form>`;
  App.openDrawer(html);
};

App.updateTarefa = function(ev, estadoPasta, arquivoNome){
  ev.preventDefault();
  const f = ev.target;
  const titulo = f.titulo.value.trim();
  if(!titulo){ showToast('Título é obrigatório.', true); return false; }
  const payload = {
    title: titulo,
    subtitulo: f.subtitulo.value.trim(),
    descricao: f.descricao.value.trim(),
    partes: coletarPartesTarefaDoForm(f),
    criado_por:f.criadoPor.value,
    responsaveis: coletarResponsaveisTarefaDoForm(f),
    data_base: f.dataBase.value,
    prazo: f.prazo.value,
    prazo_fatal: f.prazoFatal.value,
    prioridade: f.prioridade.value,
    observacoes: f.observacoes.value.trim()
  };

  const novoEstado = f.estado.value;
  const submit = f.querySelector('button[type="submit"]');
  submit.disabled = true;
  submit.textContent = 'Salvando…';
  (async ()=>{
    try{
      const corpo = await Painel.acao('tarefas', [estadoPasta, arquivoNome], 'editar', payload, 'PUT');
      if(novoEstado && novoEstado!==estadoPasta){
        await Painel.acao('tarefas', [estadoPasta, arquivoNome], 'mover-estado', {novoEstado}, 'PATCH');
      }
      App.closeDrawer();
      if(corpo && Array.isArray(corpo.naoEncontrados) && corpo.naoEncontrados.length){
        showToast('Tarefa atualizada, mas estes campos não existiam na ficha: '+corpo.naoEncontrados.join(', ')+'.', true);
      }else{
        showToast('Tarefa atualizada.');
      }
      await recarregarTarefasERenderizar();
    }catch(err){
      submit.disabled = false;
      submit.textContent = 'Salvar alterações';
      showToast(err.message || 'Falha ao atualizar a tarefa.', true);
    }
  })();
  return false;
};

App.adicionarDocumentoTarefa = async function(ev, estadoPasta, arquivoNome){
  const arquivo = ev.target.files && ev.target.files[0];
  if(!arquivo) return;
  const form = new FormData();
  form.append('documento', arquivo);
  try{
    const resp = await fetch(`/api/tarefas/${estadoPasta}/${arquivoNome}/documentos`, {method:'POST', body:form});
    if(!resp.ok) throw new Error(await resp.text());
    const resultado = await resp.json();
    showToast('Documento guardado e convertido: '+resultado.markdown);
    if(resultado.recibo && resultado.processo){
      App.openProcessoForm(resultado.processo);
    }
    ev.target.value = '';
  }catch(err){
    showToast('Falha ao juntar documento: '+err.message, true);
  }
};
