/* =========================================================
   PAINEL TEODORO - modulo 14-pessoas: view Pessoas: ficha, processos da pessoa, CEP, salvar
   Extraido de web/index.html (linhas 2756-3210) na etapa 4b da
   refatoracao (2026-10-30). Scripts classicos carregados em ordem
   numerica pelo index.html: compartilham o escopo global lexico,
   exatamente como antes dentro da IIFE unica. Sem build toolchain.
   ========================================================= */
'use strict';

/* ---------------------------------------------------------
   VIEW: PESSOAS — cadastro próprio (ficha real desde 2026-09-13). Tabela
   única (não há mais "papel" por pessoa — o papel só existe no contexto de
   cada processo que a referencia, ver processosDaPessoa).
   --------------------------------------------------------- */
const ROTULO_PAPEL_PESSOA = {
  cliente: 'cliente',
  parte_contraria: 'parte contrária',
  terceiro_interessado: 'terceiro interessado'
};

// processosDaPessoa cruza DB.processos com a pessoa (por wikilink) para
// achar em quais processos ela aparece e em que papel — a API de pessoa não
// traz mais isso embutido (ficha de pessoa é independente da de processo).
function processosDaPessoa(pessoa){
  if(!pessoa || !pessoa.arquivo) return [];
  const wikilink = '[[pessoas/'+pessoa.arquivo.replace(/\.md$/,'')+']]';
  const resultado = [];
  DB.processos.forEach(p=>{
    (p.partes||[]).forEach(parte=>{
      if(parte && parte.pessoa===wikilink){
        resultado.push({estadoPasta:p.estadoPasta, arquivoNome:p.arquivoNome, numeroOuTitulo:p.numeroProcesso||p.title||'', papel:parte.papel || 'parte', cliente:!!parte.cliente});
      }
    });
    (p.responsaveis||[]).forEach(resp=>{
      if(resp===wikilink){
        resultado.push({estadoPasta:p.estadoPasta, arquivoNome:p.arquivoNome, numeroOuTitulo:p.numeroProcesso||p.title||'', papel:'Responsável pelo cadastro'});
      }
    });
  });
  return resultado;
}

function tabelaPessoas(lista){
  if(!lista.length) return `<div class="empty"><p>Nenhuma pessoa cadastrada.</p></div>`;
  const rows = lista.map(c=>{
    const idx = DB.clientes.indexOf(c);
    return `<tr>
    <td>${escapeHTML(c.codigo||'—')}</td>
    <td>${escapeHTML(pessoaNome(c))}</td>
    <td>${escapeHTML(c.tipoPessoa||'—')}</td>
    <td class="tabular">${escapeHTML(pessoaDoc(c))}</td>
    <td class="row-actions">
      <button class="btn-text" onclick="App.openPessoaEditForm('${escapeHTML(c.arquivo)}')">Editar</button>
      <button class="btn-danger-text" onclick="App.abrirProcessoDaPessoa(${idx})">Excluir</button>
    </td>
  </tr>`;
  }).join('');
  return `<table><thead><tr><th>Código</th><th>Nome</th><th>Tipo</th><th>Documento</th><th></th></tr></thead><tbody>${rows}</tbody></table>`;
}

// Não exclui nada — abre o(s) processo(s) de origem da pessoa para o
// advogado editar/apagar o campo correspondente ali. Decisão explícita do
// advogado: o texto do botão continua "Excluir", o comportamento é navegação.
// Se a pessoa não aparece em nenhum processo, não há para onde navegar — o
// cadastro da pessoa em si só se apaga pela rota de exclusão real do
// backend, que esta versão do painel não liga a um botão de propósito.
App.abrirProcessoDaPessoa = function(idx){
  const pessoa = DB.clientes[idx];
  const processos = processosDaPessoa(pessoa);
  if(!pessoa || processos.length===0){
    showToast('Esta pessoa não está referenciada em nenhum processo — não há ficha de origem para abrir.', true);
    return;
  }
  if(processos.length===1){
    const proc = processos[0];
    const rotuloPapel = ROTULO_PAPEL_PESSOA[proc.papel] || proc.papel;
    App.openProcessoEditForm(proc.estadoPasta, proc.arquivoNome, `Esta pessoa é ${escapeHTML(rotuloPapel)} neste processo — edite ou apague o campo correspondente abaixo.`);
    return;
  }
  const html = `
    <h2>Escolha o processo</h2>
    <p class="drawer-sub">Esta pessoa aparece em mais de um processo.</p>
    ${processos.map((proc,i)=>`<div class="field"><button type="button" class="btn btn-ghost" style="width:100%;text-align:left;" onclick="App.abrirProcessoEspecificoDaPessoa(${idx},${i})">${escapeHTML(proc.numeroOuTitulo||'(sem número/título)')} — ${escapeHTML(ROTULO_PAPEL_PESSOA[proc.papel]||proc.papel)}</button></div>`).join('')}
    <div class="drawer-actions">
      <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Fechar</button>
    </div>`;
  App.openDrawer(html);
};

App.abrirProcessoEspecificoDaPessoa = function(idx, procIdx){
  const pessoa = DB.clientes[idx];
  const processos = processosDaPessoa(pessoa);
  const proc = processos[procIdx];
  if(!proc){ return; }
  const rotuloPapel = ROTULO_PAPEL_PESSOA[proc.papel] || proc.papel;
  App.openProcessoEditForm(proc.estadoPasta, proc.arquivoNome, `Esta pessoa é ${escapeHTML(rotuloPapel)} neste processo — edite ou apague o campo correspondente abaixo.`);
};

// categoriaPapelPessoa classifica a pessoa pelos papéis que ela já tem nos
// processos vinculados (via processosDaPessoa) — a ficha da pessoa em si
// não guarda papel fixo (decisão de 2026-09-13: papel é contextual ao
// processo, a mesma pessoa pode ser cliente num caso e parte adversa em
// outro). Quem nunca apareceu em nenhum processo cai em "sem processo".
function categoriaPapelPessoa(pessoa){
  const processos = processosDaPessoa(pessoa);
  if(processos.some(p => p.cliente)) return 'cliente';
  if(processos.some(p => p.papel === 'Terceiro interessado')) return 'terceiro_interessado';
  if(processos.length) return 'parte_contraria';
  return 'sem_processo';
}

const CATEGORIAS_PESSOAS = [
  {chave:'cliente', label:'Cliente'},
  {chave:'parte_contraria', label:'Parte adversa'},
  {chave:'terceiro_interessado', label:'Terceiro interessado'},
  {chave:'sem_processo', label:'Sem processo vinculado'},
];

App.viewPessoas = function(){
  let base = DB.clientes.slice();
  const toolbar = `<div class="toolbar">
    <label class="sr-only" for="buscaPessoas">Buscar pessoas</label>
    <input id="buscaPessoas" class="search" autocomplete="off" placeholder="Buscar por nome…" value="${escapeHTML(state.search)}" oninput="App.setSearch(this.value)">
  </div>`;
  if(state.search){
    base = base.filter(c => pessoaNome(c).toLowerCase().includes(state.search));
  }
  if(base.length===0 && !state.search){
    return toolbar + `<div class="empty"><h3>Nenhuma pessoa cadastrada</h3><p>Crie a primeira pessoa pelo botão "Nova pessoa" acima.</p></div>`;
  }
  const categorias = CATEGORIAS_PESSOAS.map(cat => {
    const itens = base.filter(p => categoriaPapelPessoa(p) === cat.chave);
    return {chave:cat.chave, label:cat.label, count:itens.length, html: itens.length ? tabelaPessoas(itens) : `<div class="empty"><p>Nenhuma pessoa nesta categoria.</p></div>`};
  });
  return toolbar + renderAcordeao('pessoas', categorias);
};

// Formulário único de qualificação (art. 319 do CPC), reaproveitado por
// criação e edição — mesmo padrão do artefato paradigma
// (painel-tarefas.html). `rec` é a pessoa já mapeada (mapearPessoaAPI) ao
// editar, ou null ao criar.
function formularioPessoaHTML(rec){
  const pf = !rec || rec.tipoPessoa !== 'Jurídica';
  const sugestaoArquivo = rec ? rec.arquivo.replace(/\.md$/,'') : '';
  return `
    <form onsubmit="return App.savePessoa(event${rec ? `,'${escapeHTML(rec.arquivo)}'` : ',null'})">
      <div class="field">
        <label>Tipo de pessoa</label>
        <div class="radio-row" style="margin-bottom:0;padding-top:9px;">
          <label class="radio-opt"><input type="radio" name="tipoPessoa" value="Física" ${pf?'checked':''} onchange="App.togglePessoaFields('Física')" ${rec?'disabled':''}> Física</label>
          <label class="radio-opt"><input type="radio" name="tipoPessoa" value="Jurídica" ${!pf?'checked':''} onchange="App.togglePessoaFields('Jurídica')" ${rec?'disabled':''}> Jurídica</label>
        </div>
        ${rec ? '<div class="hint">Tipo de pessoa não se troca depois de criada — exclua e recrie, se for o caso.</div>' : ''}
      </div>

      <fieldset id="fsPF" style="display:${pf?'block':'none'}">
        <legend>Pessoa física</legend>
        <div class="field">
          <label>Nome completo</label>
          <input type="text" name="nomeCompleto" style="text-transform:uppercase" oninput="App.nomePessoaMaiusculo(this)" value="${rec?escapeHTML(rec.nomeCompleto||''):''}" ${pf?'required':''}>
        </div>
        <div class="row2">
          <div class="field">
            <label>CPF</label>
            <input type="text" name="cpf" inputmode="numeric" autocomplete="off" maxlength="14" value="${rec?maskCPF(rec.cpf||''):''}" oninput="this.value=maskCPF(this.value)">
          </div>
          <div class="field">
            <label>RG (+ órgão expedidor)</label>
            <input type="text" name="rg" value="${rec?escapeHTML(rec.rg||''):''}" placeholder="00.000.000-0 SSP/UF">
          </div>
        </div>
        <div class="row2">
          <div class="field">
            <label>Estado civil</label>
            <select name="estadoCivil"><option value="">—</option>${OPCOES.estadoCivil.map(e=>`<option ${rec&&rec.estadoCivil===e?'selected':''}>${escapeHTML(e)}</option>`).join('')}</select>
          </div>
          <div class="field">
            <label>Profissão</label>
            <input type="text" name="profissao" value="${rec?escapeHTML(rec.profissao||''):''}">
          </div>
        </div>
        <div class="row2">
          <div class="field">
            <label>Nacionalidade</label>
            <input type="text" name="nacionalidade" value="${rec?escapeHTML(rec.nacionalidade||'Brasileiro(a)'):'Brasileiro(a)'}">
          </div>
          <div class="field">
            <label>Data de nascimento</label>
            <input type="date" name="dataNascimento" value="${rec?rec.dataNascimento||'':''}">
          </div>
        </div>
      </fieldset>

      <fieldset id="fsPJ" style="display:${!pf?'block':'none'}">
        <legend>Pessoa jurídica</legend>
        <div class="field">
          <label>Razão social</label>
          <input type="text" name="razaoSocial" style="text-transform:uppercase" oninput="App.nomePessoaMaiusculo(this)" value="${rec?escapeHTML(rec.razaoSocial||''):''}" ${!pf?'required':''}>
        </div>
        <div class="row2">
          <div class="field">
            <label>Nome fantasia</label>
            <input type="text" name="nomeFantasia" style="text-transform:uppercase" oninput="App.nomePessoaMaiusculo(this)" value="${rec?escapeHTML(rec.nomeFantasia||''):''}">
          </div>
          <div class="field">
            <label>CNPJ</label>
            <input type="text" name="cnpj" inputmode="numeric" autocomplete="off" maxlength="18" value="${rec?maskCNPJ(rec.cnpj||''):''}" oninput="this.value=maskCNPJ(this.value)">
          </div>
        </div>
        <div class="row3">
          <div class="field">
            <label>Representante legal</label>
            <input type="text" name="representanteNome" value="${rec?escapeHTML(rec.representanteNome||''):''}">
          </div>
          <div class="field">
            <label>CPF do representante</label>
            <input type="text" name="representanteCpf" maxlength="14" value="${rec?maskCPF(rec.representanteCpf||''):''}" oninput="this.value=maskCPF(this.value)">
          </div>
          <div class="field">
            <label>Cargo</label>
            <input type="text" name="representanteCargo" value="${rec?escapeHTML(rec.representanteCargo||''):''}">
          </div>
        </div>
      </fieldset>

      <fieldset>
        <legend>Endereço</legend>
        <div class="row3">
          <div class="field" style="grid-column:span 2;">
            <label>Logradouro</label>
            <input type="text" name="logradouro" value="${rec?escapeHTML(rec.logradouro||''):''}">
          </div>
          <div class="field">
            <label>Número</label>
            <input type="text" name="numero" value="${rec?escapeHTML(rec.numero||''):''}">
          </div>
        </div>
        <div class="row3">
          <div class="field">
            <label>Complemento</label>
            <input type="text" name="complemento" value="${rec?escapeHTML(rec.complemento||''):''}">
          </div>
          <div class="field">
            <label>Bairro</label>
            <input type="text" name="bairro" value="${rec?escapeHTML(rec.bairro||''):''}">
          </div>
          <div class="field">
            <label>CEP</label>
            <div style="display:flex;gap:8px;align-items:flex-start;">
              <input type="text" name="cep" maxlength="9" value="${rec?maskCEP(rec.cep||''):''}" oninput="this.value=maskCEP(this.value)">
              <button type="button" class="btn btn-ghost btn-small" onclick="App.buscarCEP(this.form)">Buscar CEP</button>
            </div>
            <div class="hint" data-cep-fonte></div>
          </div>
        </div>
        <div class="row2">
          <div class="field">
            <label>Cidade</label>
            <input type="text" name="cidade" value="${rec?escapeHTML(rec.cidade||''):''}">
          </div>
          <div class="field">
            <label>UF</label>
            <input type="text" name="uf" maxlength="2" value="${rec?escapeHTML(rec.uf||''):''}" style="text-transform:uppercase" oninput="this.value=this.value.toUpperCase()">
          </div>
        </div>
      </fieldset>

      <div class="row2">
        <div class="field">
          <label>E-mail</label>
          <input type="email" name="email" autocomplete="email" value="${rec?escapeHTML(rec.email||''):''}">
        </div>
        <div class="field">
          <label>Telefone</label>
          <input type="tel" name="telefone" autocomplete="tel" inputmode="tel" maxlength="15" value="${rec?maskPhone(rec.telefone||''):''}" oninput="this.value=maskPhone(this.value)">
        </div>
      </div>
      <div class="field">
        <label>Observações</label>
        <textarea name="observacoes">${rec?escapeHTML(rec.observacoes||''):''}</textarea>
      </div>
      ${rec ? '' : `<div class="field"><label>Nome do arquivo da ficha</label>
        <input type="text" name="nomeArquivo" placeholder="deixe em branco para gerar automaticamente">
        <div class="hint">Identificador técnico do arquivo, não é o nome da pessoa nem o código PES. Pode deixar em branco: o sistema gera. Só letras minúsculas, números e hífen.</div>
      </div>`}
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.cancelarPessoaForm()">Cancelar</button>
        <button type="submit" class="btn btn-primary">Salvar ${rec?'alterações':'cadastro'}</button>
      </div>
    </form>`;
}

App.togglePessoaFields = function(tipo){
  document.querySelector('[name=nomeCompleto]').required=tipo==='Física';
 document.querySelector('[name=razaoSocial]').required=tipo==='Jurídica';
  document.getElementById('fsPF').style.display = tipo==='Física' ? 'block' : 'none';
  document.getElementById('fsPJ').style.display = tipo==='Jurídica' ? 'block' : 'none';
};

App.openPessoaForm = function(noAtendimento){
 if(!noAtendimento)App.contextoPessoaAtendimento=null;
  const html = `
    <h2>Nova pessoa</h2>
    <p class="drawer-sub">Qualificação nos termos do art. 319 do CPC.</p>
    ${formularioPessoaHTML(null)}`;
  App.openDrawer(html);
};

App.openPessoaEditForm = function(arquivo){
  const rec = DB.clientes.find(c=>c.arquivo===arquivo);
  if(!rec){ showToast('Pessoa não encontrada.', true); return; }
  const html = `
    <h2>Editar pessoa</h2>
    <p class="drawer-sub">${escapeHTML(pessoaNome(rec))} — ${escapeHTML(rec.codigo||'')}</p>
    ${formularioPessoaHTML(rec)}`;
  App.openDrawer(html);
};

App.buscarCEP = function(form){
  const cep = onlyDigits(form.cep.value);
  if(cep.length!==8){ showToast('Informe um CEP com oito dígitos.', true); return false; }
  const botao = form.querySelector('button[onclick*="buscarCEP"]');
  if(botao) botao.disabled = true;
  (async ()=>{
    try{
      const resultado = await Painel.carregarResultado('/api/cep/'+cep);
      const campos = [
        ['logradouro', resultado.logradouro],
        ['bairro', resultado.bairro],
        ['cidade', resultado.municipio],
        ['uf', resultado.uf]
      ].filter(([,valor])=>String(valor||'').trim());
      const ocupados = campos.filter(([nome])=>String(form.elements[nome].value||'').trim());
      const substituir = ocupados.length > 0 && await App.confirmarEmDrawer({
        titulo:'Endereço pelo CEP',
        mensagem:'Há campos de endereço preenchidos. Deseja substituí-los pelos dados do CEP?',
        labelConfirmar:'Substituir', sobrepor:true
      });
      campos.forEach(([nome,valor])=>{
        const campo = form.elements[nome];
        if(!String(campo.value||'').trim() || substituir) campo.value = nome==='uf' ? String(valor).toUpperCase() : valor;
      });
      const fonte = form.querySelector('[data-cep-fonte]');
      if(fonte) fonte.textContent = 'Fonte consultada: '+(resultado.fonte || 'não informada');
      DB.statusFontes.cep = {status:'sucesso', fonte:resultado.fonte||'', cep};
      showToast(substituir ? 'Endereço preenchido e campos existentes substituídos.' : 'Endereço preenchido nos campos vazios.');
    }catch(err){
      DB.statusFontes.cep = {status:'falha', mensagem:err.message, cep};
      showToast(err.message || 'Falha ao consultar o CEP.', true);
    }finally{
      if(botao) botao.disabled = false;
    }
  })();
  return false;
};

App.savePessoa = function(ev, arquivoExistente){
  ev.preventDefault();
  const f = ev.target;
  const tipoPessoa = f.tipoPessoa.value;
  if(tipoPessoa==='Física' && f.cpf.value && !isValidCPF(f.cpf.value)){
    showToast('CPF inválido — confira os dígitos informados.', true); return false;
  }
  if(tipoPessoa==='Jurídica' && f.cnpj.value && !isValidCNPJ(f.cnpj.value)){
    showToast('CNPJ inválido — confira os dígitos informados.', true); return false;
  }
  const campos = {
    tipoPessoa,
    nomeCompleto: f.nomeCompleto.value.trim().toLocaleUpperCase('pt-BR'),
    cpf: onlyDigits(f.cpf.value),
    rg: f.rg.value.trim(),
    estadoCivil: f.estadoCivil.value,
    profissao: f.profissao.value.trim(),
    nacionalidade: f.nacionalidade.value.trim(),
    dataNascimento: f.dataNascimento.value,
    razaoSocial: f.razaoSocial.value.trim().toLocaleUpperCase('pt-BR'),
    nomeFantasia: f.nomeFantasia.value.trim().toLocaleUpperCase('pt-BR'),
    cnpj: onlyDigits(f.cnpj.value),
    representanteNome: f.representanteNome.value.trim(),
    representanteCpf: onlyDigits(f.representanteCpf.value),
    representanteCargo: f.representanteCargo.value.trim(),
    logradouro: f.logradouro.value.trim(),
    numero: f.numero.value.trim(),
    complemento: f.complemento.value.trim(),
    bairro: f.bairro.value.trim(),
    cep: onlyDigits(f.cep.value),
    cidade: f.cidade.value.trim(),
    uf: f.uf.value.trim().toUpperCase(),
    email: f.email.value.trim(),
    telefone: onlyDigits(f.telefone.value),
    observacoes: f.observacoes.value.trim()
  };
  if(arquivoExistente){
    (async ()=>{
      try{
        // PUT /api/pessoas/{arquivo}/editar espera chaves snake_case (whitelist
        // camposEditaveisPessoa em main.go) — diferente do POST de criação.
        const payload = {
          tipo_pessoa: campos.tipoPessoa, nome_completo: campos.nomeCompleto, cpf: campos.cpf,
          rg: campos.rg, estado_civil: campos.estadoCivil, profissao: campos.profissao,
          nacionalidade: campos.nacionalidade, data_nascimento: campos.dataNascimento,
          razao_social: campos.razaoSocial, nome_fantasia: campos.nomeFantasia, cnpj: campos.cnpj,
          representante_nome: campos.representanteNome, representante_cpf: campos.representanteCpf,
          representante_cargo: campos.representanteCargo, logradouro: campos.logradouro,
          numero: campos.numero, complemento: campos.complemento, bairro: campos.bairro,
          cep: campos.cep, cidade: campos.cidade, uf: campos.uf, email: campos.email,
          telefone: campos.telefone
        };
        const resp = await fetch('/api/pessoas/'+encodeURIComponent(arquivoExistente)+'/editar', {
          method:'PUT', headers:{'Content-Type':'application/json'},
          body: JSON.stringify(payload)
        });
        const corpo = await resp.json().catch(()=>({}));
        if(!resp.ok){
          showToast(corpo.erro || 'Falha ao atualizar a pessoa.', true);
          return;
        }
        App.closeDrawer();
        if(Array.isArray(corpo.naoEncontrados) && corpo.naoEncontrados.length){
          showToast('Pessoa atualizada, mas estes campos não existiam na ficha: '+corpo.naoEncontrados.join(', ')+'.', true);
        }else{
          showToast('Pessoa atualizada.');
        }
        await recarregarPessoasERenderizar();
      }catch(err){
        showToast('Falha ao atualizar a pessoa.', true);
      }
    })();
    return false;
  }
  const nomeExibicao = tipoPessoa==='Jurídica' ? campos.razaoSocial : campos.nomeCompleto;
  if(!nomeExibicao){
    showToast(tipoPessoa==='Jurídica' ? 'Razão social é obrigatória.' : 'Nome completo é obrigatório.', true);
    return false;
  }
  let nomeArquivo = f.nomeArquivo ? f.nomeArquivo.value.trim() : '';
  if(!nomeArquivo) nomeArquivo = slugify(nomeExibicao);
  (async ()=>{
    try{
      const resp = await fetch('/api/pessoas/nova', {
        method:'POST', headers:{'Content-Type':'application/json'},
        body: JSON.stringify(Object.assign({nomeArquivo}, campos))
      });
      const corpo = await resp.json().catch(()=>({}));
      if(!resp.ok){
        if(resp.status===409) showToast('Já existe uma pessoa com esse nome de arquivo — tente outro.', true);
        else showToast(corpo.erro || 'Falha ao criar a pessoa.', true);
        return;
      }
      if(App.contextoPessoaAtendimento){
        await App_carregarPessoas();
        App.voltarPessoaAtendimento(corpo.arquivo);
        showToast('Pessoa criada e vinculada ao rascunho do atendimento.');
        App.renderNav();App.renderStats();
        return;
      }
      App.closeDrawer();
      showToast('Pessoa criada — código ' + (corpo.codigo || '—') + '.');
      await recarregarPessoasERenderizar();
    }catch(err){
      showToast('Falha ao criar a pessoa.', true);
    }
  })();
  return false;
};

async function recarregarPessoasERenderizar(){
  try{
    await App_carregarPessoas();
    App.renderNav(); App.renderStats(); App.renderTab();
  }catch(err){
    showToast('Ação registrada, mas houve falha ao recarregar a lista de pessoas.', true);
  }
}

App.cancelarPessoaForm=function(){if(!App.voltarPessoaAtendimento())App.closeDrawer();};
App.nomePessoaMaiusculo=function(input){const inicio=input.selectionStart,fim=input.selectionEnd;input.value=input.value.toLocaleUpperCase('pt-BR');if(inicio!==null)input.setSelectionRange(inicio,fim);};
