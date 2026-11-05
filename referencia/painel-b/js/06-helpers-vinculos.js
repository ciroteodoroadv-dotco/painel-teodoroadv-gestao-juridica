/* =========================================================
   PAINEL TEODORO - modulo 06-helpers-vinculos: helpers de dados relacionados: pessoas, partes, responsáveis, vínculos de tarefa
   Extraido de web/index.html (linhas 512-745) na etapa 4b da
   refatoracao (2026-10-30). Scripts classicos carregados em ordem
   numerica pelo index.html: compartilham o escopo global lexico,
   exatamente como antes dentro da IIFE unica. Sem build toolchain.
   ========================================================= */
'use strict';

/* =========================================================
   HELPERS DE DADOS RELACIONADOS
   ========================================================= */
function pessoaNome(p){
  return p.tipoPessoa==='Jurídica' ? (p.razaoSocial||'(sem razão social)') : (p.nomeCompleto||p.nome||'(sem nome)');
}
function pessoaDoc(p){
  return p.tipoPessoa==='Jurídica' ? (p.cnpj||'—') : (p.cpf||'—');
}
function getCliente(id){ return DB.clientes.find(c=>c.id===id); }

// arquivoDoWikilinkPessoa extrai "fulano.md" de "[[pessoas/fulano]]" (mesmo
// formato que o backend grava nos campos cliente/parte_contraria/terceiro_
// interessado). Devolve '' se não for um wikilink de pessoa reconhecível.
function arquivoDoWikilinkPessoa(v){
  const m = /^\[\[pessoas\/(.+)\]\]$/.exec(String(v||'').trim());
  return m ? m[1]+'.md' : '';
}
// encontrarPessoaPorReferencia localiza, em DB.clientes, a pessoa
// correspondente a uma referência vinda da API (wikilink, nome de arquivo
// ou código) — mesmo critério de ficha.ResolverPessoa no backend.
function encontrarPessoaPorReferencia(v){
  const ref = String(v||'').trim();
  if(!ref) return null;
  const arquivo = arquivoDoWikilinkPessoa(ref) || ref;
  return DB.clientes.find(c=>c.arquivo===arquivo || c.id===ref || c.codigo===ref) || null;
}
// nomeExibicaoReferenciaPessoa devolve o texto para mostrar numa célula de
// tabela a partir de uma referência crua (wikilink/arquivo/texto legado):
// nome da pessoa cadastrada se achar, senão o texto bruto sem colchetes.
function nomeExibicaoReferenciaPessoa(v){
  const p = encontrarPessoaPorReferencia(v);
  if(p) return pessoaNome(p);
  const ref = String(v||'').trim();
  const arquivo = arquivoDoWikilinkPessoa(ref);
  return arquivo ? arquivo.replace(/\.md$/,'') : ref;
}
// opcoesPessoas monta as <option> de um <select> de pessoa a partir de
// DB.clientes, pré-selecionando valorAtual (wikilink, arquivo ou texto
// legado). Se valorAtual não corresponder a nenhuma pessoa cadastrada,
// acrescenta uma opção extra com o texto literal, selecionada, para não
// perder o dado legado ao reabrir o formulário sem salvar.
function opcoesPessoas(valorAtual){
  const ref = String(valorAtual||'').trim();
  const achou = ref ? encontrarPessoaPorReferencia(ref) : null;
  let opts = `<option value="">—</option>` + DB.clientes.map(c=>{
    const sel = achou && c.id===achou.id ? 'selected' : '';
    return `<option value="${escapeHTML(c.id)}" ${sel}>${escapeHTML(pessoaNome(c))} — ${escapeHTML(c.tipoPessoa||'?')}${pessoaDoc(c)!=='—'?' ('+escapeHTML(pessoaDoc(c))+')':''}</option>`;
  }).join('');
  if(ref && !achou){
    opts += `<option value="${escapeHTML(ref)}" selected>${escapeHTML(nomeExibicaoReferenciaPessoa(ref))} (texto legado, não cadastrado)</option>`;
  }
  return opts;
}
// PAPEIS_PROCESSUAIS alimenta o <select> de papel de cada linha de Partes —
// "Outro" revela um campo de texto livre ao lado (ver linhaParteHtml).
const PAPEIS_PROCESSUAIS = ['Autor','Réu','Exequente','Executado','Embargante','Terceiro interessado','Assistente','Outro'];

// linhaParteHtml monta uma linha da lista dinâmica de Partes do drawer de
// processo — select de Pessoa (reaproveita opcoesPessoas), select de papel
// processual (+ texto livre se "Outro") e checkbox "é cliente do
// escritório". parte (opcional) pré-preenche a linha na edição.
function linhaParteHtml(parte){
  parte = parte || {pessoa:'', papel:'', cliente:false};
  const papelConhecido = PAPEIS_PROCESSUAIS.slice(0,-1).includes(parte.papel);
  const papelSelect = papelConhecido ? parte.papel : (parte.papel ? 'Outro' : '');
  const papelOutroValor = papelConhecido ? '' : (parte.papel || '');
  return `<div class="row-parte" style="border:1px solid var(--border);border-radius:8px;padding:10px;margin-bottom:8px;">
    <div class="row2">
      <div class="field"><label>Pessoa</label><select class="parte-pessoa">${opcoesPessoas(parte.pessoa)}</select></div>
      <div class="field"><label>Papel processual</label>
        <select class="parte-papel" onchange="this.nextElementSibling.style.display=(this.value==='Outro'?'block':'none')">
          <option value="">—</option>${PAPEIS_PROCESSUAIS.map(op=>`<option ${op===papelSelect?'selected':''}>${escapeHTML(op)}</option>`).join('')}
        </select>
        <input type="text" class="parte-papel-outro" placeholder="Especifique o papel" value="${escapeHTML(papelOutroValor)}" style="margin-top:6px;display:${papelSelect==='Outro'?'block':'none'};">
      </div>
    </div>
    <label class="radio-opt"><input type="checkbox" class="parte-cliente" ${(parte.cliente===true||parte.cliente==='true')?'checked':''}> É cliente do escritório</label>
    <button type="button" class="btn-text" onclick="this.closest('.row-parte').remove()">Remover parte</button>
  </div>`;
}
// coletarPartesDoForm lê todas as .row-parte do form e devolve o array
// {pessoa,papel,cliente} pronto para o payload — linha sem pessoa
// selecionada é descartada (linha em branco deixada pelo usuário).
function coletarPartesDoForm(f){
  return Array.from(f.querySelectorAll('.row-parte')).map(row=>{
    const pessoa = row.querySelector('.parte-pessoa').value.trim();
    const selectPapel = row.querySelector('.parte-papel').value;
    const papel = selectPapel==='Outro' ? row.querySelector('.parte-papel-outro').value.trim() : selectPapel;
    const cliente = row.querySelector('.parte-cliente').checked;
    return {pessoa, papel, cliente};
  }).filter(p=>p.pessoa);
}
// responsaveisSelectsHtml monta os 4 <select> fixos de "Responsáveis pelo
// cadastro" — slots vazios são permitidos e filtrados ao salvar.
function responsaveisSelectsHtml(valoresAtuais){
  const lista = Array.isArray(valoresAtuais) ? valoresAtuais : [];
  let html = '';
  for(let i=0;i<4;i++){
    html += `<div class="field"><label>Responsável ${i+1}</label><select class="responsavel-select">${opcoesPessoas(lista[i]||'')}</select></div>`;
  }
  return html;
}
function coletarResponsaveisDoForm(f){
  return Array.from(f.querySelectorAll('.responsavel-select')).map(el=>el.value.trim()).filter(Boolean);
}

// --- Partes e Responsáveis do drawer de Tarefa ---
// Mesmo padrão de linha dinâmica de Partes do Processo, mas sem o checkbox
// "é cliente do escritório" (não faz sentido numa tarefa) — decisão tomada
// com o advogado ao planejar a reforma dos drawers de Tarefa.
function linhaParteTarefaHtml(parte){
  parte = parte || {pessoa:'', papel:''};
  const papelConhecido = PAPEIS_PROCESSUAIS.slice(0,-1).includes(parte.papel);
  const papelSelect = papelConhecido ? parte.papel : (parte.papel ? 'Outro' : '');
  const papelOutroValor = papelConhecido ? '' : (parte.papel || '');
  return `<div class="row-parte-tarefa" style="border:1px solid var(--border);border-radius:8px;padding:10px;margin-bottom:8px;">
    <div class="row2">
      <div class="field"><label>Pessoa</label><select class="parte-tarefa-pessoa">${opcoesPessoas(parte.pessoa)}</select></div>
      <div class="field"><label>Papel</label>
        <select class="parte-tarefa-papel" onchange="this.nextElementSibling.style.display=(this.value==='Outro'?'block':'none')">
          <option value="">—</option>${PAPEIS_PROCESSUAIS.map(op=>`<option ${op===papelSelect?'selected':''}>${escapeHTML(op)}</option>`).join('')}
        </select>
        <input type="text" class="parte-tarefa-papel-outro" placeholder="Especifique o papel" value="${escapeHTML(papelOutroValor)}" style="margin-top:6px;display:${papelSelect==='Outro'?'block':'none'};">
      </div>
    </div>
    <button type="button" class="btn-text" onclick="this.closest('.row-parte-tarefa').remove()">Remover parte</button>
  </div>`;
}
// Funções globais simples (não App.*) porque este trecho roda antes de
// "const App = {}" mais abaixo no arquivo — atribuir a App aqui daria
// ReferenceError de TDZ ("Cannot access 'App' before initialization"). Tudo
// dentro do <script> está numa IIFE, então precisam ir em window.* à mão
// para o onclick inline (que roda no escopo global) enxergá-las.
function adicionarLinhaParteTarefa(parte){
  document.getElementById('partesTarefaRows').insertAdjacentHTML('beforeend', linhaParteTarefaHtml(parte));
}
window.adicionarLinhaParteTarefa = adicionarLinhaParteTarefa;
function coletarPartesTarefaDoForm(f){
  return Array.from(f.querySelectorAll('.row-parte-tarefa')).map(row=>{
    const pessoa = row.querySelector('.parte-tarefa-pessoa').value.trim();
    const selectPapel = row.querySelector('.parte-tarefa-papel').value;
    const papel = selectPapel==='Outro' ? row.querySelector('.parte-tarefa-papel-outro').value.trim() : selectPapel;
    return {pessoa, papel, cliente:false};
  }).filter(p=>p.pessoa);
}
// Responsável de Tarefa é lista dinâmica de verdade (Processo usa 4 selects
// fixos para Responsáveis; aqui não há teto — pedido explícito do advogado).
function linhaResponsavelTarefaHtml(wikilink){
  return `<div class="row-responsavel-tarefa" style="display:flex;gap:8px;align-items:flex-end;margin-bottom:8px;">
    <div class="field" style="flex:1;"><label>Responsável</label><select class="responsavel-tarefa-select">${opcoesPessoas(wikilink||'')}</select></div>
    <button type="button" class="btn-text" onclick="this.closest('.row-responsavel-tarefa').remove()">Remover</button>
  </div>`;
}
function adicionarLinhaResponsavelTarefa(responsavel){
  document.getElementById('responsaveisTarefaRows').insertAdjacentHTML('beforeend', linhaResponsavelTarefaHtml(responsavel));
}
window.adicionarLinhaResponsavelTarefa = adicionarLinhaResponsavelTarefa;
function coletarResponsaveisTarefaDoForm(f){
  return Array.from(f.querySelectorAll('.responsavel-tarefa-select')).map(el=>el.value.trim()).filter(Boolean);
}

function liberarCamposAdm(botao){
  const caixa = botao.closest('#admGateTarefa');
  const senha = caixa.querySelector('.adm-senha-input').value;
  if(!senha){
    showToast('Informe a senha de ADM.', true);
    return;
  }
  const form = botao.closest('form');
  form.dataset.adminPassword = senha;
  form.querySelectorAll('.campo-adm').forEach(el=>el.removeAttribute('disabled'));
  caixa.remove();
  showToast('Campos liberados para edição.');
};
window.liberarCamposAdm = liberarCamposAdm;

function getAtendimento(id){ return DB.atendimentos.find(a=>a.id===id); }
function getProcesso(id){ return DB.processos.find(p=>p.id===id); }

// Status real vem direto da pasta (6 estados do fluxo de tarefas do
// acervo) — ver EstadosTarefa em servidor/ficha/tarefa.go.
const ROTULOS_ESTADO_TAREFA = {
  'a-confirmar':            {label:'A confirmar',            cls:'badge-warn'},
  'pendente':               {label:'Pendente',                cls:'badge-warn'},
  'em-execucao':            {label:'Em execução',             cls:'badge-neutral'},
  'revisao':                {label:'Revisão',                  cls:'badge-neutral'},
  'concluidas-com-sucesso': {label:'Concluída',               cls:'badge-ok'},
  'concluidas-sem-sucesso': {label:'Concluída (sem êxito)',   cls:'badge-neutral'},
  'canceladas':             {label:'Cancelada',                cls:'badge-danger'}
};
// Ordem do acordeão de Tarefas = ordem do fluxo real (ver EstadosTarefa em
// servidor/ficha/tarefa.go). Mesmo padrão de CATEGORIAS_AGENDA.
const CATEGORIAS_TAREFA = [
  {chave:'a-confirmar',            label:'A confirmar'},
  {chave:'pendente',               label:'Pendente'},
  {chave:'em-execucao',            label:'Em execução'},
  {chave:'revisao',                label:'Revisão'},
  {chave:'concluidas-com-sucesso', label:'Concluída com sucesso'},
  {chave:'concluidas-sem-sucesso', label:'Concluída sem sucesso'},
  {chave:'canceladas',             label:'Cancelada'}
];
function computeTarefaStatus(t){
  const finalizada = t.status==='concluidas-com-sucesso' || t.status==='concluidas-sem-sucesso' || t.status==='canceladas';
  const atrasada = t.prazo && t.prazo < todayISO() && !finalizada;
  const base = ROTULOS_ESTADO_TAREFA[t.status] || {label:t.status||'Sem estado', cls:'badge-neutral'};
  if(atrasada) return {label:base.label+' · atrasada', cls:'badge-danger'};
  return base;
}
function prioridadeBadgeClass(pr){
  return pr==='Urgente' ? 'prioridade-alta' : pr==='Alta' ? 'prioridade-media' : pr==='Baixa' ? 'prioridade-baixa' : 'prioridade-normal';
}
function codigoVinculoTarefa(t){
  if(t.origemTipo==='atendimento'){
    const a = DB.atendimentos.find(x=>x.id===t.origemId);
    return a && a.codigo ? a.codigo : 'ATE. [VERIFICAR]';
  }
  if(t.origemTipo==='processo'){
    const p = DB.processos.find(x=>x.id===t.origemId);
    return p && p.codigo ? p.codigo : 'PRO. [VERIFICAR]';
  }
  return t.origemId || '';
}
function clienteNomeDoVinculo(t){
  let referencia = '';
  if(t.origemTipo==='atendimento'){
    const a = DB.atendimentos.find(x=>x.id===t.origemId);
    referencia = a ? a.clienteNome : '';
  }else if(t.origemTipo==='processo'){
    const p = DB.processos.find(x=>x.id===t.origemId);
    referencia = p ? p.clienteNome : '';
  }
  return nomeExibicaoReferenciaPessoa(referencia);
}

// Autoria é independente dos responsáveis históricos: não preencher identidade por inferência.
function campoCriadoPorHTML(ref){
 const p=encontrarPessoaPorReferencia(ref||'');
 let opts='<option value="">Não informado</option>'+DB.clientes.map(x=>`<option value="${escapeHTML(x.id)}" ${p&&x.id===p.id?'selected':''}>${escapeHTML(pessoaNome(x))} · ${escapeHTML(x.codigo||'PES. [VERIFICAR]')}</option>`).join('');
 if(ref&&!p)opts+=`<option selected value="${escapeHTML(ref)}">${escapeHTML(nomeExibicaoReferenciaPessoa(ref))} · referência legada</option>`;
 return `<div class="field"><label>Criado por</label><select name="criadoPor">${opts}</select><p class="hint">Pessoa que criou este registro. Não define quem executará a atividade.</p></div>`;
}
