/* Drawer lateral do painel B (web/js/15-drawers.js), sem HTML inline e com pilha.
   Contrato:
   - [data-drawer="URL"]     abre o fragmento de URL (que começa por <h2>) em #drawer;
   - [data-drawer-sobrepor]  no mesmo elemento: empilha o conteúdo atual em vez de trocá-lo;
   - [data-drawer-fechar]    fecha (desempilha, se houver pilha) e devolve o foco à origem.
   O drawer NÃO fecha por tecla nem por clique fora (pedido do advogado): só pelos botões dele. */
(function () {
  'use strict';
  const App = (window.App = window.App || {});
  App.drawerStack = [];
  App.drawerPreviousFocus = null;

  const overlay = () => document.getElementById('drawerOverlay');
  const drawer = () => document.getElementById('drawer');

  function focoInicial(d) {
    const f = d.querySelector('button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])');
    (f || d).focus();
  }

  // Coloca os nós (de um fragmento do próprio servidor, já escapado) no drawer.
  App.openDrawer = function (nodes, sobrepor) {
    const o = overlay(), d = drawer();
    if (!o || !d) return;
    const aberto = !o.classList.contains('hidden');
    if (sobrepor && aberto) App.drawerStack.push(Array.from(d.childNodes));
    if (!aberto) App.drawerPreviousFocus = document.activeElement;
    d.replaceChildren(...nodes);
    if (window.htmx) window.htmx.process(d); // ativa hx-post/hx-get dos fragmentos inseridos
    const titulo = d.querySelector('h2');
    if (titulo) titulo.id = 'drawerTitle';
    o.classList.remove('hidden');
    o.setAttribute('aria-hidden', 'false');
    document.body.classList.add('drawer-aberto');
    focoInicial(d);
  };

  App.closeDrawer = function () {
    const o = overlay(), d = drawer();
    if (!o || !d) return;
    if (App.drawerStack.length) {
      d.replaceChildren(...App.drawerStack.pop());
      d.focus();
      return;
    }
    o.classList.add('hidden');
    o.setAttribute('aria-hidden', 'true');
    document.body.classList.remove('drawer-aberto');
    d.replaceChildren();
    const f = App.drawerPreviousFocus;
    if (f && typeof f.focus === 'function' && document.contains(f)) f.focus();
    App.drawerPreviousFocus = null;
  };

  // DOMParser não executa scripts: o fragmento é só marcação.
  App.nodesDeHTML = function (html) {
    return Array.from(new DOMParser().parseFromString(html, 'text/html').body.childNodes);
  };

  document.addEventListener('click', function (ev) {
    const alvo = ev.target.closest ? ev.target : ev.target.parentElement;
    if (!alvo) return;
    const abre = alvo.closest('[data-drawer]');
    if (abre) {
      ev.preventDefault();
      const sobrepor = abre.hasAttribute('data-drawer-sobrepor');
      fetch(abre.getAttribute('data-drawer'), { headers: { 'HX-Request': 'true' }, credentials: 'same-origin' })
        .then(function (r) { if (!r.ok) throw new Error(r.status); return r.text(); })
        .then(function (html) {
          App.openDrawer(App.nodesDeHTML(html), sobrepor);
        })
        .catch(function () { if (window.PainelToast) window.PainelToast('Não foi possível abrir o formulário.', true); });
      return;
    }
    if (alvo.closest('[data-drawer-fechar]')) {
      ev.preventDefault();
      App.closeDrawer();
    }
  });
})();
