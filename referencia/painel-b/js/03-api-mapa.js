/* =========================================================
   PAINEL TEODORO - modulo 03-api-mapa: adaptação dos dados reais da API (mapear*API) + slugify
   Extraido de web/index.html (linhas 187-413) na etapa 4b da
   refatoracao (2026-10-30). Scripts classicos carregados em ordem
   numerica pelo index.html: compartilham o escopo global lexico,
   exatamente como antes dentro da IIFE unica. Sem build toolchain.
   ========================================================= */
'use strict';

/* =========================================================
   ADAPTAÇÃO DOS DADOS REAIS DA API (camada nova desta versão)
   ========================================================= */

// GET /api/processos devolve campos em português (numero_processo, juizo,
// comarca, classe, ramo, cliente, polo, parte_contraria, terceiro_interessado,
// valor_causa, distribuido_em, proximo_prazo, estado, arquivo,
// camposFaltantes, ...). Esta função traduz esse formato para os nomes de
// campo que as views abaixo usam, sem alterar o visual nem os componentes
// originais.
function mapearProcessoAPI(p){
  const segs = String(p.arquivo||'').split('/');
  const estadoPasta = segs[0] || '';
  const arquivoNome = segs.slice(1).join('/') || '';
  const listaPartes = Array.isArray(p.partes) ? p.partes : [];
  const parteCliente = listaPartes.find(pt => pt && (pt.cliente===true || pt.cliente==='true'));
  const outrasPartes = listaPartes.filter(pt => pt !== parteCliente);
  return {
    id: p.arquivo,
    estadoPasta,
    arquivoNome,
    codigo: p.codigo || '',
    title: p.title || '',
    numeroProcesso: p.numero_processo || '',
    sistema: p.sistema || '',
    instancia: p.instancia || '',
    juizo: p.juizo || '',
    comarca: p.comarca || '',
    classeProcessual: p.classe || '',
    materiaDireito: p.ramo || '',
    polo: p.polo || '',
    tipoContingencia: p.tipo_contingencia || '',
    // partes é a fonte de verdade nova (lista de {pessoa,papel,cliente});
    // clienteNome/parteContraria(Texto) continuam derivados dela só para não
    // ter que reescrever tabela/busca/cruzamento de Pessoas que já liam esses
    // dois campos — ver processosDaPessoa, que usa `partes` direto.
    partes: listaPartes,
    clienteNome: parteCliente ? parteCliente.pessoa : '',
    parteContraria: outrasPartes.map(pt=>pt.pessoa),
    parteContrariaTexto: outrasPartes.map(pt=>pt.pessoa).join(', '),
    responsaveis: Array.isArray(p.responsaveis) ? p.responsaveis : [],
    valorCausaRaw: p.valor_causa || '',
    dataDistribuicao: p.distribuido_em || '',
    citacao: p.citacao || '',
    audiencia: p.audiencia || '',
    sentenca: p.sentenca || '',
    recursos: p.recursos || '',
    faseExecucao: p.fase_execucao || '',
    atendimentoOrigem: p.atendimento_origem || '',
    observacoes: p.observacoes || '',
    proximoPrazo: p.proximo_prazo || '',
    projetoVinculado: p.projeto_vinculado || '',
    criado: p.criado || '',
    estado: p.estado || '',
    camposFaltantes: Array.isArray(p.camposFaltantes) ? p.camposFaltantes : []
  };
}

// GET /api/pessoas devolve a ficha própria de Pessoa (desde 2026-09-13):
// {arquivo, codigo, tipoPessoa, nomeCompleto, cpf, rg, ..., razaoSocial,
// cnpj, ..., logradouro, ..., email, telefone, observacoes, criado}. Não
// tem mais "papel" nem "processos" embutidos — o papel (cliente/parte
// contrária/terceiro interessado) só existe no contexto de cada processo
// que a referencia (ver processosDaPessoa). id = nome de arquivo
// ("fulano.md"), mesmo formato que resolverReferenciaPessoa aceita no
// backend.
function mapearPessoaAPI(p){
  return {
    id: p.arquivo || '',
    arquivo: p.arquivo || '',
    codigo: p.codigo || '',
    tipoPessoa: p.tipoPessoa || '',
    nomeCompleto: p.nomeCompleto || '',
    cpf: p.cpf || '',
    rg: p.rg || '',
    estadoCivil: p.estadoCivil || '',
    profissao: p.profissao || '',
    nacionalidade: p.nacionalidade || '',
    dataNascimento: p.dataNascimento || '',
    razaoSocial: p.razaoSocial || '',
    nomeFantasia: p.nomeFantasia || '',
    cnpj: p.cnpj || '',
    representanteNome: p.representanteNome || '',
    representanteCpf: p.representanteCpf || '',
    representanteCargo: p.representanteCargo || '',
    logradouro: p.logradouro || '',
    numero: p.numero || '',
    complemento: p.complemento || '',
    bairro: p.bairro || '',
    cep: p.cep || '',
    cidade: p.cidade || '',
    uf: p.uf || '',
    email: p.email || '',
    telefone: p.telefone || '',
    observacoes: p.observacoes || '',
    criado: p.criado || ''
  };
}

// GET /api/tarefas devolve {arquivo, estado, title, subtitulo, responsavel,
// responsaveis, data_base, prazo, prioridade, origem_tipo, origem_id,
// prazo_fatal, partes, observacoes, criado, descricao} — arquivo vem como
// "{estado}/{nomeArquivo}.md". prazo_fatal deixou de ser booleano — agora é
// uma data (string AAAA-MM-DD ou vazia); "responsaveis" é lista, com
// "responsavel" mantido só como fallback de leitura de fichas antigas.
function mapearTarefaAPI(t){
  const partes = String(t.arquivo||'').split('/');
  const responsaveis = Array.isArray(t.responsaveis) && t.responsaveis.length ? t.responsaveis
    : (t.responsavel ? [t.responsavel] : []);
  return {
    id: t.arquivo,
    estadoPasta: t.estado || partes[0] || '',
    arquivoNome: partes.slice(1).join('/') || '',
    codigo: t.codigo || '',
    titulo: t.title || '',
    subtitulo: t.subtitulo || '',
    criadoPor: t.criado_por || '',
    responsavel: t.responsavel || '',
    responsaveis: responsaveis,
    dataBase: t.data_base || '',
    descricao: t.descricao || '',
    observacoes: t.observacoes || '',
    origemTipo: t.origem_tipo || '',
    origemId: t.origem_id || '',
    tarefaPai: t.tarefa_pai || '',
    prazoFatal: /^\d{4}-\d{2}-\d{2}$/.test(t.prazo_fatal||'') ? t.prazo_fatal : '',
    prazo: t.prazo || '',
    prioridade: t.prioridade || '',
    partes: Array.isArray(t.partes) ? t.partes : [],
    status: t.estado || ''
  };
}

// GET /api/atendimentos devolve {arquivo, title, atendimento_tipo, cliente,
// status, data_abertura, criado, descricao} — arquivo é só o nome do
// arquivo (pasta flat, sem subpasta de estado).
function mapearAtendimentoAPI(a){
  return {
 criadoPor:a.criado_por||'',
    id: a.arquivo,
    arquivoNome: a.arquivo || '',
    codigo: a.codigo || '',
    tipo: a.atendimento_tipo || '',
    clienteNome: a.cliente || '',
 clientes: a.clientes || (a.cliente?[a.cliente]:[]),
 partesAdversas: a.partes_adversas || [],
 observacoes: a.observacoes || '',
    descricao: a.descricao || '',
    dataAbertura: a.data_abertura || '',
    status: a.status || ''
  };
}

// GET /api/agenda devolve {id, tipo, descricao, data, hora, local,
// processoVinculado} — só os 4 tipos flexíveis/gerais que ainda vivem em
// dados/agenda.json (Prazo Fatal, Reunião, Ligação, Aula). Audiência e
// Diligência têm ficha própria (ver mapearAudienciaAPI/mapearDiligenciaAPI).
function mapearAgendaAPI(a){
  return {
    id: a.id,
    criadoPor:a.criadoPor||'',
    tipo: a.tipo || '',
    descricao: a.descricao || '',
    data: a.data || '',
    hora: a.hora || '',
    local: a.local || '',
    processoVinculado: a.processoVinculado || '',
 cliente:a.cliente||'',
 atendimentoVinculado:a.atendimentoVinculado||''
  };
}

// GET /api/audiencias devolve uma Audiencia por ficha real em
// 11-TEODORO-ADVOCACIA/audiencias/{tipo}/ — "arquivo" já vem como
// "{tipo}/{nome}.md", exigido para qualquer ação futura sobre o registro.
function mapearAudienciaAPI(a){
  return {
    arquivo: a.arquivo || '',
    tipo: a.tipo || '',
    codigo: a.codigo || '',
    titulo: a.title || '',
    processo: a.processo || '',
    cliente: a.cliente || '',
    data: a.data || '',
    hora: a.hora || '',
    juizo: a.juizo || '',
    local: a.local || '',
    descricao: a.descricao || '',
    cancelada: a.cancelada === 'true',
    motivoCancelamento: a.motivo_cancelamento || ''
  };
}

// GET /api/diligencias devolve uma Diligencia por ficha real (pasta flat)
// em 11-TEODORO-ADVOCACIA/diligencias/.
function mapearDiligenciaAPI(d){
  return {
    arquivo: d.arquivo || '',
    codigo: d.codigo || '',
    titulo: d.title || '',
    processo: d.processo || '',
    cliente: d.cliente || '',
    local: d.local || '',
    dataLimite: d.data_limite || '',
    hora: d.hora || '',
    responsavel: d.responsavel || '',
    status: d.status || '',
    descricao: d.descricao || ''
  };
}

function mapearIntimacaoAPI(i){
  return {
    id: i.hash || i.id || '',
    hash: i.hash || '',
    origem: i.origem || '',
    tribunal: i.tribunal || '',
    orgao: i.orgao || '',
    processo: i.processo || '',
    tipo: i.tipo || '',
    dataPublicacao: i.data_publ || '',
    resumo: i.resumo || '',
    link: i.link || '',
    status: i.status || 'triagem',
    rascunhoPrazo: i.rascunho_prazo || null
  };
}

// Gera um nome de arquivo seguro (minúsculo, sem acento, sem espaço) a
// partir de um texto livre — usado para sugerir nomeArquivo nas criações.
function slugify(texto){
  return String(texto||'').toLowerCase()
    .normalize('NFD').replace(/[̀-ͯ]/g,'')
    .replace(/[^a-z0-9]+/g,'-').replace(/(^-|-$)/g,'');
}
