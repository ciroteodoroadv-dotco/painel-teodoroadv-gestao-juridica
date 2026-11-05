/* =========================================================
   PAINEL TEODORO - modulo 15-drawers: drawer lateral, confirmação/senha em drawer, exports window.*
   Extraido de web/index.html (linhas 3212-3316) na etapa 4b da
   refatoracao (2026-10-30). Scripts classicos carregados em ordem
   numerica pelo index.html: compartilham o escopo global lexico,
   exatamente como antes dentro da IIFE unica. Sem build toolchain.
   ========================================================= */
'use strict';

/* ---------------------------------------------------------
   DRAWER (painel lateral de formulário — padrão do artefato
   paradigma: um único par overlay/drawer, reaproveitado por todas
   as entidades; cada entidade só fornece o HTML do formulário).
   --------------------------------------------------------- */
App.drawerStack = [];
App.drawerPreviousFocus = null;
/* ---------------------------------------------------------
   CONFIRMAÇÃO E SENHA EM DRAWER (Entrega 1.1, item 10) — fim
   das janelinhas nativas ("localhost diz..."): toda pergunta ao
   advogado usa o mesmo padrão visual do Editar.
   --------------------------------------------------------- */
// _resolverDrawer é chamado quando o drawer fecha sem conclusão (clique
// fora, Cancelar): garante que a Promise da pergunta sempre resolve.
App._resolverDrawer = null;

// confirmarEmDrawer abre uma confirmação e resolve true/false.
// sobrepor=true preserva o formulário aberto por baixo (ex.: CEP dentro
// do cadastro de pessoa).
App.confirmarEmDrawer = function(opcoes){
  return new Promise(resolve=>{
    const html = `
      <h2>${escapeHTML(opcoes.titulo)}</h2>
      <p class="drawer-sub">${escapeHTML(opcoes.mensagem)}</p>
      <div class="drawer-actions">
        <button type="button" class="btn btn-ghost" id="btnCancelarConfirm">Cancelar</button>
        <button type="button" class="btn btn-primary" id="btnConfirmarAcao">${escapeHTML(opcoes.labelConfirmar||'Confirmar')}</button>
      </div>`;
    App._resolverDrawer = ()=>resolve(false);
    App.openDrawer(html, !!opcoes.sobrepor);
    const fechar = valor=>{ App._resolverDrawer = null; App.closeDrawer(); resolve(valor); };
    document.getElementById('btnCancelarConfirm').onclick = ()=>fechar(false);
    document.getElementById('btnConfirmarAcao').onclick = ()=>fechar(true);
  });
};

// pedirSenhaEmDrawer abre o pedido de senha e resolve a senha ou null.
App.pedirSenhaEmDrawer = function(titulo, mensagem){
  return new Promise(resolve=>{
    const html = `
      <h2>${escapeHTML(titulo)}</h2>
      <p class="drawer-sub">${escapeHTML(mensagem)}</p>
      <form id="formSenhaDrawer">
        <div class="field"><label>Senha</label><input type="password" name="senha" autocomplete="current-password" required></div>
        <div class="drawer-actions">
          <button type="button" class="btn btn-ghost" id="btnCancelarSenha">Cancelar</button>
          <button type="submit" class="btn btn-primary">Confirmar</button>
        </div>
      </form>`;
    App._resolverDrawer = ()=>resolve(null);
    App.openDrawer(html, true);
    const fechar = valor=>{ App._resolverDrawer = null; App.closeDrawer(); resolve(valor); };
    document.getElementById('btnCancelarSenha').onclick = ()=>fechar(null);
    document.getElementById('formSenhaDrawer').onsubmit = ev=>{ ev.preventDefault(); fechar(ev.target.senha.value); return false; };
  });
};

App.openDrawer = function(html, sobrepor){
  if(sobrepor && !document.getElementById('drawerOverlay').classList.contains('hidden')){
    App.drawerStack.push(document.getElementById('drawer').innerHTML);
  }
  const overlay = document.getElementById('drawerOverlay');
  const drawer = document.getElementById('drawer');
  if(overlay.classList.contains('hidden')) App.drawerPreviousFocus = document.activeElement;
  drawer.innerHTML = html;
  const title = drawer.querySelector('h2');
  if(title){
    title.id = 'drawerTitle';
    drawer.setAttribute('aria-labelledby', 'drawerTitle');
  }
  overlay.classList.remove('hidden');
  overlay.setAttribute('aria-hidden', 'false');
  document.body.classList.add('drawer-aberto');
  const firstFocusable = drawer.querySelector('button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])');
  (firstFocusable || drawer).focus();
};
App.closeDrawer = function(){
  if(App._resolverDrawer){ const r = App._resolverDrawer; App._resolverDrawer = null; r(); }
  if(App.drawerStack.length){
    document.getElementById('drawer').innerHTML = App.drawerStack.pop();
    document.getElementById('drawer').focus();
    return;
  }
  const overlay = document.getElementById('drawerOverlay');
  overlay.classList.add('hidden');
  overlay.setAttribute('aria-hidden', 'true');
  document.body.classList.remove('drawer-aberto');
  document.getElementById('drawer').innerHTML = '';
  if(App.drawerPreviousFocus && typeof App.drawerPreviousFocus.focus === 'function'){
    App.drawerPreviousFocus.focus();
  }
  App.drawerPreviousFocus = null;
};
/* O drawer NAO fecha por clique fora nem por ESC (pedido do advogado,
   2026-10-29): so fecha por acao explicita nos botoes do proprio drawer
   (Fechar/Cancelar/Salvar), para nunca perder dados por acidente. */

/* expõe helpers usados em atributos inline */
window.maskCNJ = maskCNJ;
window.maskCPF = maskCPF;
window.maskCNPJ = maskCNPJ;
window.maskCEP = maskCEP;
window.maskPhone = maskPhone;
window.maskCurrencyInput = maskCurrencyInput;
