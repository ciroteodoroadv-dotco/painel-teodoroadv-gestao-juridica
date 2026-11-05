/* =========================================================
   PAINEL TEODORO - modulo 09-processos: view Processos: acordeão por estado, ficha, andamento, financeiro, providência
   Extraido de web/index.html (linhas 1320-1758) na etapa 4b da
   refatoracao (2026-10-30). Scripts classicos carregados em ordem
   numerica pelo index.html: compartilham o escopo global lexico,
   exatamente como antes dentro da IIFE unica. Sem build toolchain.
   ========================================================= */
'use strict';

/* ---------------------------------------------------------
   VIEW: PROCESSOS (dados reais via GET /api/processos) — acordeão
   vertical simultâneo por estado: Ativos, Sobrestados, Extintos,
   Arquivados definitivamente.
   --------------------------------------------------------- */
const CATEGORIAS_PROCESSOS = [
  {chave:'ativos',      label:'Ativos',                       estado:'ativo'},
  {chave:'sobrestados', label:'Sobrestados',                  estado:'sobrestado'},
  {chave:'extintos',    label:'Extintos',                     estado:'extinto'},
  {chave:'arquivados',  label:'Arquivados definitivamente',   estado:'arquivado'}
];

function linhaProcesso(p){
  const pendenciaBadge = p.camposFaltantes.length
    ? `<span class="badge badge-warn" title="${escapeHTML(p.camposFaltantes.join('; '))}">Pendências (${p.camposFaltantes.length})</span>`
    : '<span class="hint" style="color:var(--text-muted);">—</span>';
  return `<tr>
    <td>${escapeHTML(p.codigo||'—')}</td>
    <td class="tabular"><strong>${escapeHTML(p.numeroProcesso||'—')}</strong><div class="hint" style="color:var(--text-muted);margin-top:3px;">${escapeHTML(p.title||'')}</div></td>
    <td>${escapeHTML(p.classeProcessual||'—')}<div class="hint" style="color:var(--text-muted);margin-top:3px;">${escapeHTML(p.materiaDireito||'—')}</div></td>
    <td>${escapeHTML(nomeExibicaoReferenciaPessoa(p.clienteNome))}<div class="hint" style="color:var(--text-muted);margin-top:3px;">${escapeHTML(p.polo||'')}</div></td>
    <td>${escapeHTML(p.parteContraria.map(nomeExibicaoReferenciaPessoa).join(', ')||'—')}</td>
    <td class="tabular">${escapeHTML(p.valorCausaRaw||'—')}</td>
    <td class="tabular">${fmtDateOrRaw(p.dataDistribuicao)}</td>
    <td>${pendenciaBadge}</td>
    <td class="row-actions">
      <button class="btn-text" onclick="App.openProcessoEditForm('${p.estadoPasta}','${p.arquivoNome}')">Editar</button>
      <button class="btn-text" onclick="App.acaoMoverEstado('${p.estadoPasta}','${p.arquivoNome}')">Mover estado</button>
      <button class="btn-text" onclick="App.acaoAndamento('${p.estadoPasta}','${p.arquivoNome}')">Andamento</button>
      <button class="btn-text" onclick="App.acaoFinanceiro('${p.estadoPasta}','${p.arquivoNome}')">Financeiro</button>
      <button class="btn-text" onclick="App.acaoProximaProvidencia('${p.estadoPasta}','${p.arquivoNome}')">Próxima providência</button>
      <button class="btn-danger-text" onclick="App.deleteProcesso('${p.estadoPasta}','${p.arquivoNome}')">Excluir</button>
    </td>
  </tr>`;
}
function tabelaProcessos(lista){
  if(!lista.length) return `<div class="empty"><p>Nenhum processo nesta categoria.</p></div>`;
  return `<table><thead><tr>
    <th>Código</th><th>Nº processo</th><th>Classe / Matéria</th><th>Cliente</th><th>Parte contrária</th><th>Valor da causa</th><th>Distribuição</th><th>Pendências</th><th></th>
  </tr></thead><tbody>${lista.map(linhaProcesso).join('')}</tbody></table>`;
}

App.viewProcessos = function(){
  let base = DB.processos.slice();
  const toolbar = `<div class="toolbar">
    <label class="sr-only" for="buscaProcessos">Buscar processos</label>
    <input id="buscaProcessos" class="search" autocomplete="off" placeholder="Buscar por número, classe, matéria ou parte…" value="${escapeHTML(state.search)}" oninput="App.setSearch(this.value)">
  </div>`;
  if(state.search){
    base = base.filter(p => (p.numeroProcesso+' '+p.classeProcessual+' '+p.materiaDireito+' '+nomeExibicaoReferenciaPessoa(p.clienteNome)+' '+p.parteContrariaTexto).toLowerCase().includes(state.search));
  }
  if(base.length===0 && !state.search){
    return toolbar + `<div class="empty"><h3>Nenhum processo cadastrado</h3><p>Nenhum processo foi encontrado na carteira processual do servidor.</p></div>`;
  }
  const categorias = CATEGORIAS_PROCESSOS.map(c=>{
    const itens = base.filter(p=>p.estado===c.estado);
    return {chave:c.chave, label:c.label, count:itens.length, html: tabelaProcessos(itens)};
  });
  return toolbar + renderAcordeao('processos', categorias);
};

// SISTEMAS_PROCESSO/INSTANCIAS_PROCESSO alimentam os selects com opção
// "Outro" — mesmo padrão de OPCOES.classeProcessual/materiaDireito.
const SISTEMAS_PROCESSO = ['PJe','Eproc','e-SAJ','Projudi','Físico','Outro'];
const INSTANCIAS_PROCESSO = ['1º grau','2º grau','Tribunais Superiores','Juizado Especial','Outro'];
function selectComOutroHtml(nomeBase, opcoes, valorAtual){
  const conhecido = opcoes.slice(0,-1).includes(valorAtual);
  const selecaoSelect = conhecido ? valorAtual : (valorAtual ? 'Outro' : '');
  const valorOutro = conhecido ? '' : (valorAtual || '');
  return `<select class="${nomeBase}-select" onchange="this.nextElementSibling.style.display=(this.value==='Outro'?'block':'none')">
      <option value="">—</option>${opcoes.map(op=>`<option ${op===selecaoSelect?'selected':''}>${escapeHTML(op)}</option>`).join('')}
    </select>
    <input type="text" class="${nomeBase}-outro" placeholder="Especifique" value="${escapeHTML(valorOutro)}" style="margin-top:6px;display:${selecaoSelect==='Outro'?'block':'none'};">`;
}
function valorDeSelectComOutro(f, nomeBase){
  const sel = f.querySelector('.'+nomeBase+'-select').value;
  return sel==='Outro' ? f.querySelector('.'+nomeBase+'-outro').value.trim() : sel;
}
function atendimentoOrigemOptionsHtml(valorAtual){
  return `<option value="">—</option>` + DB.atendimentos.map(a=>{
    const sel = a.id===valorAtual ? 'selected' : '';
    return `<option value="${escapeHTML(a.id)}" ${sel}>${escapeHTML(a.codigo||'')} · ${escapeHTML(a.tipo||'Atendimento')} · ${escapeHTML(nomeExibicaoReferenciaPessoa(a.clienteNome))} · ${escapeHTML(fmtDateOrRaw(a.dataAbertura))}</option>`;
  }).join('');
}
// campoDataHtml monta um <input type="date"> só quando o valor já é uma data
// AAAA-MM-DD válida (mesmo critério já usado em "Distribuído em") — evita
// que um valor legado fora desse formato quebre o input nativo de data.
function campoDataHtml(nome, valorAtual){
  const v = /^\d{4}-\d{2}-\d{2}$/.test(valorAtual||'') ? valorAtual : '';
  return `<input type="date" name="${nome}" value="${v}">`;
}

// corpoDrawerProcesso monta o miolo do formulário (campos + Partes +
// Responsáveis + linha do tempo), compartilhado entre criação e edição —
// only a diferença de título/botões fica em cada App.open*Form.
function corpoDrawerProcesso(p){
  p = p || {};
  return `
      <fieldset>
        <legend>Partes</legend>
        <div id="partesRows">${(p.partes&&p.partes.length ? p.partes : [null]).map(linhaParteHtml).join('')}</div>
        <button type="button" class="btn btn-ghost" onclick="App.adicionarLinhaParte()">+ Adicionar parte</button>
        <div class="hint">Cada parte referencia uma Pessoa já cadastrada — use "Nova pessoa" antes, se precisar. Marque "é cliente do escritório" na(s) linha(s) correspondente(s).</div>
      </fieldset>
      <fieldset>
        <legend>Responsáveis pelo cadastro</legend>
        ${responsaveisSelectsHtml(p.responsaveis||[])}
      </fieldset>
      <div class="row2">
        <div class="field"><label>Número do processo (CNJ)</label><input type="text" name="numeroProcesso" oninput="this.value=maskCNJ(this.value)" placeholder="Em branco se ainda não distribuído" value="${escapeHTML(p.numeroProcesso||'')}"></div>
        <div class="field"><label>Valor da causa</label><input type="text" name="valorCausa" oninput="this.value=maskCurrencyInput(this.value)" value="${escapeHTML(p.valorCausaRaw||'')}"></div>
      </div>
      <div class="row2">
        <div class="field"><label>Sistema</label>${selectComOutroHtml('sistema', SISTEMAS_PROCESSO, p.sistema||'')}</div>
        <div class="field"><label>Instância</label>${selectComOutroHtml('instancia', INSTANCIAS_PROCESSO, p.instancia||'')}</div>
      </div>
      <div class="row2">
        <div class="field"><label>Classe processual</label>
          <select name="classe"><option value="">—</option>${OPCOES.classeProcessual.map(c=>`<option ${c===p.classeProcessual?'selected':''}>${escapeHTML(c)}</option>`).join('')}</select>
        </div>
        <div class="field"><label>Matéria / ramo do direito</label>
          <select name="ramo"><option value="">—</option>${OPCOES.materiaDireito.map(c=>`<option ${c===p.materiaDireito?'selected':''}>${escapeHTML(c)}</option>`).join('')}</select>
        </div>
      </div>
      <div class="row2">
        <div class="field"><label>Comarca</label><input type="text" name="comarca" value="${escapeHTML(p.comarca||'')}"></div>
        <div class="field"><label>Juízo</label><input type="text" name="juizo" value="${escapeHTML(p.juizo||'')}"></div>
      </div>
      <div class="field"><label>Contingência (valor da causa é ativa ou passiva para o escritório/cliente)</label>
        <div class="radio-row">
          <label class="radio-opt"><input type="radio" name="tipoContingencia" value="Ativa" ${p.tipoContingencia==='Ativa'?'checked':''}> Ativa</label>
          <label class="radio-opt"><input type="radio" name="tipoContingencia" value="Passiva" ${p.tipoContingencia==='Passiva'?'checked':''}> Passiva</label>
          <label class="radio-opt"><input type="radio" name="tipoContingencia" value="" ${!p.tipoContingencia?'checked':''}> Não se aplica</label>
        </div>
      </div>
      <fieldset>
        <legend>Linha do tempo</legend>
        <div class="row3">
          <div class="field"><label>Distribuição</label>${campoDataHtml('distribuidoEm', p.dataDistribuicao)}</div>
          <div class="field"><label>Citação</label>${campoDataHtml('citacao', p.citacao)}</div>
          <div class="field"><label>Audiência</label>${campoDataHtml('audiencia', p.audiencia)}</div>
        </div>
        <div class="row3">
          <div class="field"><label>Sentença</label>${campoDataHtml('sentenca', p.sentenca)}</div>
          <div class="field"><label>Recursos</label>${campoDataHtml('recursos', p.recursos)}</div>
          <div class="field"><label>Fase de execução (satisfativa)</label>${campoDataHtml('faseExecucao', p.faseExecucao)}</div>
        </div>
        <div class="hint">Audiência aqui é só anotação rápida — a ficha própria da aba Audiências continua sendo a fonte de verdade.</div>
      </fieldset>
      <div class="field"><label>Atendimento de origem (opcional)</label><select name="atendimentoOrigem">${atendimentoOrigemOptionsHtml(p.atendimentoOrigem||'')}</select></div>
      <div class="field"><label>Observações</label><textarea name="observacoes" rows="3">${escapeHTML(p.observacoes||'')}</textarea></div>`;
}

App.openProcessoForm = function(prefill){
  prefill = prefill || {};
  const html = `
    <h2>Novo processo</h2>
    <p class="drawer-sub">${prefill.numeroProcesso?'Dados extraídos do recibo — confira tudo antes de salvar.':'Gera a ficha a partir do molde de processo, sempre em estado "ativo".'}</p>
    <form onsubmit="return App.saveProcesso(event)">
      ${corpoDrawerProcesso(prefill)}
      <input type="hidden" name="projetoVinculado" value="${escapeHTML(prefill.projetoVinculado||'')}">
      <div class="field"><label>Nome do arquivo da ficha</label>
        <input type="text" name="nomeArquivo" placeholder="deixe em branco para gerar automaticamente">
        <div class="hint">Só letras minúsculas, números e hífen.</div>
      </div>
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Cancelar</button>
        <button type="submit" class="btn btn-primary">Salvar processo</button>
      </div>
    </form>`;
  App.openDrawer(html);
};

App.saveProcesso = function(ev){
  ev.preventDefault();
  const f = ev.target;
  const partes = coletarPartesDoForm(f);
  const responsaveis = coletarResponsaveisDoForm(f);
  const numeroProcesso = f.numeroProcesso.value.trim();
  const valorCausa = f.valorCausa.value.trim();
  const sistema = valorDeSelectComOutro(f, 'sistema');
  const instancia = valorDeSelectComOutro(f, 'instancia');
  const classe = f.classe.value;
  const ramo = f.ramo.value;
  const comarca = f.comarca.value.trim();
  const juizo = f.juizo.value.trim();
  const tipoContingencia = (f.querySelector('input[name="tipoContingencia"]:checked')||{}).value || '';
  const primeiraParteCliente = partes.find(p=>p.cliente);
  const nomeClienteTexto = primeiraParteCliente ? nomeExibicaoReferenciaPessoa(primeiraParteCliente.pessoa) : '';
  let nomeArquivo = f.nomeArquivo.value.trim();
  if(!nomeArquivo){
    const primeiraOutraParte = partes.find(p=>!p.cliente);
    nomeArquivo = new Date().getFullYear() + '-' + slugify(nomeClienteTexto||'processo') + (primeiraOutraParte ? '-x-'+slugify(nomeExibicaoReferenciaPessoa(primeiraOutraParte.pessoa)) : '');
  }
  (async ()=>{
    try{
      const resp = await fetch('/api/processos/novo', {
        method:'POST', headers:{'Content-Type':'application/json'},
        body: JSON.stringify({
          nomeArquivo, numeroProcesso, sistema, instancia, juizo, comarca, classe, ramo,
          tipoContingencia, partes, responsaveis, valorCausa,
          distribuidoEm: f.distribuidoEm.value, citacao: f.citacao.value,
          audiencia: f.audiencia.value, sentenca: f.sentenca.value,
          recursos: f.recursos.value, faseExecucao: f.faseExecucao.value,
          atendimentoOrigem: f.atendimentoOrigem.value,
          observacoes: f.observacoes.value.trim(),
          polo:'',           projetoVinculado: f.projetoVinculado ? f.projetoVinculado.value : ''
        })
      });
      if(!resp.ok){
        if(resp.status===409) showToast('Já existe uma ficha com esse nome — tente outro nome de arquivo.', true);
        else showToast('Falha ao criar o processo.', true);
        return;
      }
      App.closeDrawer();
      showToast('Processo criado.');
      await recarregarProcessosERenderizar();
    }catch(err){
      showToast('Falha ao criar o processo.', true);
    }
  })();
  return false;
};
App.deleteProcesso = function(estadoPasta, arquivoNome){
  (async ()=>{
    const ok = await App.confirmarEmDrawer({
      titulo:'Excluir processo',
      mensagem:'Excluir definitivamente esta ficha de processo? Esta ação não pode ser desfeita.',
      labelConfirmar:'Excluir definitivamente'
    });
    if(!ok) return;
    try{
      await Painel.acao('processos', [estadoPasta, arquivoNome], 'excluir', {}, 'DELETE');
      showToast('Processo excluído.');
      await recarregarProcessosERenderizar();
    }catch(err){
      showToast('Falha ao excluir o processo.', true);
    }
  })();
};

// Abre o mesmo desenho do form de criação, pré-preenchido, para edição.
// avisoExtra (opcional) é usado quando este form é aberto a partir da aba
// Pessoas (App.abrirProcessoDaPessoa) — mostra de onde a pessoa aparece.
App.openProcessoEditForm = function(estadoPasta, arquivoNome, avisoExtra){
  const p = DB.processos.find(x=>x.estadoPasta===estadoPasta && x.arquivoNome===arquivoNome);
  if(!p){ showToast('Processo não encontrado.', true); return; }
  const html = `
    <h2>Editar processo</h2>
    <p class="drawer-sub">${escapeHTML(p.numeroProcesso || p.title || '')}</p>
    ${avisoExtra ? `<p class="drawer-sub">${avisoExtra}</p>` : ''}
    <form onsubmit="return App.updateProcesso(event,'${estadoPasta}','${arquivoNome}')">
      ${corpoDrawerProcesso(p)}
      <div class="field"><label>Título da ficha</label><input type="text" name="title" value="${escapeHTML(p.title)}"></div>
      <div class="field"><label>Polo</label><input type="text" name="polo" value="${escapeHTML(p.polo)}"></div>
      <div class="field"><label>Projeto vinculado</label><input type="text" name="projetoVinculado" value="${escapeHTML(p.projetoVinculado)}"></div>
      <div class="hint" style="margin-bottom:14px;">Estado (ativo/sobrestado/...) não se edita aqui — use "Mover estado" na lista.</div>
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Cancelar</button>
        <button type="submit" class="btn btn-primary">Salvar alterações</button>
      </div>
    </form>`;
  App.openDrawer(html);
};

App.updateProcesso = function(ev, estadoPasta, arquivoNome){
  ev.preventDefault();
  const f = ev.target;
  const payload = {
    title: f.title.value.trim(),
    numero_processo: f.numeroProcesso.value.trim(),
    sistema: valorDeSelectComOutro(f, 'sistema'),
    instancia: valorDeSelectComOutro(f, 'instancia'),
    juizo: f.juizo.value.trim(),
    comarca: f.comarca.value.trim(),
    classe: f.classe.value,
    ramo: f.ramo.value,
    polo: f.polo.value.trim(),
    tipo_contingencia: (f.querySelector('input[name="tipoContingencia"]:checked')||{}).value || '',
    partes: coletarPartesDoForm(f),
    responsaveis: coletarResponsaveisDoForm(f),
    valor_causa: f.valorCausa.value.trim(),
    distribuido_em: f.distribuidoEm.value,
    citacao: f.citacao.value,
    audiencia: f.audiencia.value,
    sentenca: f.sentenca.value,
    recursos: f.recursos.value,
    fase_execucao: f.faseExecucao.value,
    atendimento_origem: f.atendimentoOrigem.value,
    observacoes: f.observacoes.value.trim(),
    projeto_vinculado: f.projetoVinculado.value.trim()
  };
  (async ()=>{
    try{
      const corpo = await Painel.acao('processos', [estadoPasta, arquivoNome], 'editar', payload, 'PUT');
      App.closeDrawer();
      if(corpo && Array.isArray(corpo.naoEncontrados) && corpo.naoEncontrados.length){
        showToast('Processo atualizado, mas estes campos não existiam na ficha: '+corpo.naoEncontrados.join(', ')+'.', true);
      }else{
        showToast('Processo atualizado.');
      }
      await recarregarProcessosERenderizar();
    }catch(err){
      showToast('Falha ao atualizar o processo.', true);
    }
  })();
  return false;
};

async function recarregarProcessosERenderizar(){
  try{
    await App_carregarProcessos();
    // Pessoas é dado derivado dos processos (cliente/parte contrária/terceiro
    // interessado) — precisa recarregar junto, senão fica com dado velho
    // depois de editar/excluir/criar processo.
    await App_carregarPessoas();
    App.renderNav(); App.renderStats(); App.renderTab();
  }catch(err){
    showToast('Ação registrada, mas houve falha ao recarregar a lista de processos.', true);
  }
}

// executarAcaoProcesso é o miolo comum das ações rápidas da ficha de
// processo (Entrega 1.1, item 10): o formulário em drawer monta o
// payload e delega — sem janelinhas nativas.
App.executarAcaoProcesso = async function(estadoPasta, arquivoNome, acao, payload, metodo, mensagemOk, mensagemErro){
  try{
    await Painel.acao('processos', [estadoPasta, arquivoNome], acao, payload, metodo);
    App.closeDrawer();
    showToast(mensagemOk);
    await recarregarProcessosERenderizar();
  }catch(err){
    showToast(mensagemErro, true);
  }
};

App.acaoMoverEstado = function(estadoPasta, arquivoNome){
  const html = `
    <h2>Mover estado do processo</h2>
    <p class="drawer-sub">Ao sair de ativo/sobrestado, a tarefa de monitoramento do processo é baixada em cadeia; ao voltar para um desses estados, uma nova é criada.</p>
    <form onsubmit="return App.salvarMoverEstado(event,${inlineArg(estadoPasta)},${inlineArg(arquivoNome)})">
      <div class="field"><label>Novo estado *</label>
        <select name="novoEstado">${OPCOES.estadoProcesso.map(e=>`<option value="${escapeHTML(e)}">${escapeHTML(e)}</option>`).join('')}</select>
      </div>
      <div class="field"><label>Motivo</label><textarea name="motivo" placeholder="Ex.: trânsito em julgado, acordo homologado, suspensão…"></textarea></div>
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Cancelar</button>
        <button type="submit" class="btn btn-primary">Mover</button>
      </div>
    </form>`;
  App.openDrawer(html);
};
App.salvarMoverEstado = function(ev, estadoPasta, arquivoNome){
  ev.preventDefault();
  const f = ev.target;
  App.executarAcaoProcesso(estadoPasta, arquivoNome, 'mover-estado',
    {novoEstado: f.novoEstado.value, motivo: f.motivo.value.trim()}, 'PATCH',
    'Estado do processo atualizado.', 'Falha ao mover o estado do processo.');
  return false;
};

App.acaoAndamento = function(estadoPasta, arquivoNome){
  const html = `
    <h2>Registrar andamento</h2>
    <p class="drawer-sub">O que aconteceu no processo — vai para a linha do tempo da ficha.</p>
    <form onsubmit="return App.salvarAndamento(event,${inlineArg(estadoPasta)},${inlineArg(arquivoNome)})">
      <div class="field"><label>O que aconteceu *</label><textarea name="oQue" required placeholder="Ex.: Juntada de contestação"></textarea></div>
      <div class="field"><label>Documento relacionado</label><input type="text" name="documento" placeholder="Opcional"></div>
      <div class="field"><label>Data do andamento</label>${campoDataHtml('data', todayISO())}<div class="hint">Padrão: hoje.</div></div>
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Cancelar</button>
        <button type="submit" class="btn btn-primary">Registrar</button>
      </div>
    </form>`;
  App.openDrawer(html);
};
App.salvarAndamento = function(ev, estadoPasta, arquivoNome){
  ev.preventDefault();
  const f = ev.target;
  const payload = {oQue: f.oQue.value.trim(), documento: f.documento.value.trim()};
  if(f.data.value.trim()) payload.data = f.data.value.trim();
  App.executarAcaoProcesso(estadoPasta, arquivoNome, 'andamento', payload, 'POST',
    'Andamento registrado.', 'Falha ao registrar o andamento.');
  return false;
};

App.acaoFinanceiro = function(estadoPasta, arquivoNome){
  const html = `
    <h2>Atualizar financeiro</h2>
    <p class="drawer-sub">Honorários, custas e demais itens financeiros do processo.</p>
    <form onsubmit="return App.salvarFinanceiro(event,${inlineArg(estadoPasta)},${inlineArg(arquivoNome)})">
      <div class="field"><label>Item *</label><input type="text" name="item" required placeholder="Ex.: honorários, custas"></div>
      <div class="row2">
        <div class="field"><label>Valor</label><input type="text" name="valor" placeholder="Ex.: 1.500,00"></div>
        <div class="field"><label>Situação</label><input type="text" name="situacao" list="sugestoesSituacaoFinanceiro" placeholder="Ex.: pago, pendente">
          <datalist id="sugestoesSituacaoFinanceiro"><option value="Pago"></option><option value="Pendente"></option><option value="Parcelado"></option></datalist>
        </div>
      </div>
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Cancelar</button>
        <button type="submit" class="btn btn-primary">Salvar</button>
      </div>
    </form>`;
  App.openDrawer(html);
};
App.salvarFinanceiro = function(ev, estadoPasta, arquivoNome){
  ev.preventDefault();
  const f = ev.target;
  App.executarAcaoProcesso(estadoPasta, arquivoNome, 'financeiro',
    {item: f.item.value.trim(), valor: f.valor.value.trim(), situacao: f.situacao.value.trim()}, 'PATCH',
    'Dado financeiro atualizado.', 'Falha ao atualizar o financeiro.');
  return false;
};

App.acaoProximaProvidencia = function(estadoPasta, arquivoNome){
  const html = `
    <h2>Próxima providência</h2>
    <p class="drawer-sub">O próximo passo do processo, com prazo e responsável.</p>
    <form onsubmit="return App.salvarProximaProvidencia(event,${inlineArg(estadoPasta)},${inlineArg(arquivoNome)})">
      <div class="field"><label>O que fazer *</label><textarea name="oQue" required placeholder="Ex.: Protocolar recurso"></textarea></div>
      <div class="row2">
        <div class="field"><label>Até quando</label>${campoDataHtml('ateQuando','')}</div>
        <div class="field"><label>Responsável</label><input type="text" name="quem"></div>
      </div>
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Cancelar</button>
        <button type="submit" class="btn btn-primary">Registrar</button>
      </div>
    </form>`;
  App.openDrawer(html);
};
App.salvarProximaProvidencia = function(ev, estadoPasta, arquivoNome){
  ev.preventDefault();
  const f = ev.target;
  App.executarAcaoProcesso(estadoPasta, arquivoNome, 'proxima-providencia',
    {oQue: f.oQue.value.trim(), ateQuando: f.ateQuando.value, quem: f.quem.value.trim()}, 'PATCH',
    'Próxima providência registrada.', 'Falha ao registrar a próxima providência.');
  return false;
};
