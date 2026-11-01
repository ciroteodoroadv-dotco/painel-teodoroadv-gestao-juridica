// Comportamentos comuns do painel. Sem inline: a CSP só aceita scripts do próprio domínio.
(function () {
  "use strict";

  // Aviso ao sair de formulário alterado (marque o <form> com data-protege).
  var sujo = false;
  document.addEventListener("input", function (ev) {
    if (ev.target.closest && ev.target.closest("form[data-protege]")) sujo = true;
  });
  document.addEventListener("submit", function () { sujo = false; });
  window.addEventListener("beforeunload", function (ev) {
    if (sujo) { ev.preventDefault(); ev.returnValue = ""; }
  });

  // htmx: envia o token CSRF em toda requisição.
  document.addEventListener("htmx:configRequest", function (ev) {
    var m = document.querySelector('meta[name="csrf-token"]');
    if (m) ev.detail.headers["X-CSRF-Token"] = m.content;
  });

  // Confirmação em duas etapas sem diálogo do navegador: botão data-confirmar pede um segundo clique.
  document.addEventListener("click", function (ev) {
    var b = ev.target.closest("button[data-confirmar]");
    if (!b || b.dataset.armado) return;
    ev.preventDefault();
    b.dataset.armado = "1";
    b.dataset.rotulo = b.textContent;
    b.textContent = b.getAttribute("data-confirmar");
    setTimeout(function () { delete b.dataset.armado; b.textContent = b.dataset.rotulo; }, 5000);
  });
})();
