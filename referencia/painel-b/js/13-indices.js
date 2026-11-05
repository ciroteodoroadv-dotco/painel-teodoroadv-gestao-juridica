/* =========================================================
   PAINEL TEODORO - modulo 13-indices: índices econômicos (ticker do rodapé)
   Extraido de web/index.html (linhas 2725-2754) na etapa 4b da
   refatoracao (2026-10-30). Scripts classicos carregados em ordem
   numerica pelo index.html: compartilham o escopo global lexico,
   exatamente como antes dentro da IIFE unica. Sem build toolchain.
   ========================================================= */
'use strict';

/* ---------------------------------------------------------
   ÍNDICES ECONÔMICOS - só na faixa preta do rodapé (decisão do advogado,
   2026-10-29: bloco da sidebar removido; o espaço terá outro uso).
   Os valores só aparecem quando vierem de /api/indices; ausência, falha e
   não sincronizado são estados operacionais explícitos, nunca dados atuais.
   --------------------------------------------------------- */
const ORDEM_INDICES = ['Selic (meta Copom)', 'IPCA', 'INPC', 'IGP-M', 'Dólar (PTAX)', 'Salário mínimo'];
function statusOperacionalLabel(status){
  return ({sucesso:'sincronizado', parcial:'parcial', falha:'falha', nao_sincronizado:'não sincronizado', bloqueado:'bloqueado'})[status] || status || 'ausente';
}

App.renderIndices = function(){
  const estado = DB.statusFontes.indices;
  const itens = DB.indices && typeof DB.indices === 'object' ? ORDEM_INDICES.map(chave=>DB.indices[chave]).filter(Boolean) : [];
  App.renderIndicesTicker(itens, estado);
};

// Faixa preta fixa no rodapé — mesmos dados do painel lateral, formato
// compacto de ticker. Só relê o último estado já persistido (GET
// /api/indices); nunca dispara nova coleta na fonte oficial.
App.renderIndicesTicker = function(itens, estado){
  const alvo = document.getElementById('idxTicker');
  if(!alvo) return;
  if(!itens || !itens.length){
    alvo.innerHTML = `<span class="idxt-status">Índices econômicos: ${escapeHTML(statusOperacionalLabel(estado && estado.status))}${estado && estado.mensagem ? ' — ' + escapeHTML(estado.mensagem) : ''}</span>`;
    return;
  }
  alvo.innerHTML = itens.map(i => `<span class="idxt-item">${escapeHTML(i.rotulo||'—')}<b>${escapeHTML(i.valor||'—')}</b></span>`).join('')
    + `<span class="idxt-status">Ref.: ${escapeHTML((itens[0]&&itens[0].referencia)||'—')} · ${escapeHTML(statusOperacionalLabel(estado && estado.status))}</span>`;
};
