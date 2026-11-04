/* =========================================================
   PAINEL TEODORO - modulo 12-prazo: calculadora de prazo (ferramenta solta)
   Extraido de web/index.html (linhas 2650-2723) na etapa 4b da
   refatoracao (2026-10-30). Scripts classicos carregados em ordem
   numerica pelo index.html: compartilham o escopo global lexico,
   exatamente como antes dentro da IIFE unica. Sem build toolchain.
   ========================================================= */
'use strict';

/* ---------------------------------------------------------
   CALCULADORA DE PRAZO (Fase 6, escopo restrito) — ferramenta solta,
   não vinculada a ficha nenhuma; só calcula e mostra o resultado.
   ---------------------------------------------------------*/
App.abrirCalculadoraPrazo = function(){
  const html = `
    <h2>Calculadora de prazo</h2>
    <p class="drawer-sub">Ferramenta solta, não vinculada a ficha nenhuma — só calcula e mostra o resultado.</p>
    <form onsubmit="return App.calcularPrazoDrawer(event)">
      <div class="field">
        <label>Data do evento</label>
        <input type="date" name="base" required value="${todayISO()}">
        <div class="hint">Data da disponibilização no Diário ou da ciência/intimação direta.</div>
      </div>
      <div class="field">
        <label>Modo de contagem</label>
        <div class="radio-row">
          <label class="radio-opt"><input type="radio" name="modo" value="dje" checked> Diário oficial (DJe)</label>
          <label class="radio-opt"><input type="radio" name="modo" value="ciencia"> Ciência / intimação direta</label>
        </div>
      </div>
      <div class="field">
        <label>Regra do prazo</label>
        <div class="radio-row">
          <label class="radio-opt"><input type="radio" name="tipo" value="uteis" checked> CPC (dias úteis)</label>
          <label class="radio-opt"><input type="radio" name="tipo" value="corridos"> CPP (prazo processual penal — dias corridos)</label>
	  <label class="radio-opt"><input type="radio" name="tipo" value="material-civel"> Prazo material cível (dias corridos)</label>
	  <label class="radio-opt"><input type="radio" name="tipo" value="material-penal"> Prazo material penal (dias corridos)</label>
        </div>
      </div>
      <div class="field">
        <label>Número de dias</label>
        <input type="number" name="dias" min="1" step="1" required>
      </div>
      <div id="resultadoPrazo"></div>
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" onclick="App.closeDrawer()">Fechar</button>
        <button type="submit" class="btn btn-primary">Calcular</button>
      </div>
    </form>`;
  App.openDrawer(html);
};

App.calcularPrazoDrawer = async function(ev){
  ev.preventDefault();
  const f = ev.target;
  const base = f.base.value;
  const modo = f.modo.value;
  const tipo = f.tipo.value;
  const dias = parseInt(f.dias.value, 10);
  const alvo = document.getElementById('resultadoPrazo');
  if(!Number.isInteger(dias) || dias <= 0){
    alvo.innerHTML = `<div class="form-block" style="border-color:#c0392b;color:#c0392b;">Número de dias inválido.</div>`;
    return false;
  }
  try{
    const resp = await fetch('/api/prazos/calcular', {
      method:'POST', headers:{'Content-Type':'application/json'},
      body: JSON.stringify({base, modo, dias, tipo})
    });
    const corpo = await resp.json().catch(()=>({}));
    if(!resp.ok){
      alvo.innerHTML = `<div class="form-block" style="border-color:#c0392b;color:#c0392b;">${escapeHTML(corpo.erro || 'Falha ao calcular o prazo.')}</div>`;
      return false;
    }
    const venc = fmtDate(corpo.venc);
    const faltam = corpo.diasUteisAteVencimento;
    const faltamTexto = faltam===0 ? 'vence hoje' : faltam>0 ? `faltam ${faltam} dia(s) útil(eis)` : `venceu há ${Math.abs(faltam)} dia(s) útil(eis)`;
    alvo.innerHTML = `<div class="form-block"><strong>Prazo fatal: ${venc}</strong><br>${faltamTexto}.</div>`;
  }catch(err){
    alvo.innerHTML = `<div class="form-block" style="border-color:#c0392b;color:#c0392b;">Falha ao calcular o prazo.</div>`;
  }
  return false;
};
