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

  // Marca no menu o link da página atual.
  document.addEventListener("DOMContentLoaded", function () {
    var atual = location.pathname;
    document.querySelectorAll("nav.tabs a").forEach(function (a) {
      var h = a.getAttribute("href");
      var ativo = h === "/" ? atual === "/" : atual === h || atual.indexOf(h + "/") === 0;
      if (ativo) a.setAttribute("aria-current", "page");
    });
  });

  // ---- Toast
  var toast = document.getElementById("toast");
  var tToast = null;
  window.PainelToast = function (msg, erro) {
    if (!toast) return;
    toast.textContent = msg;
    toast.classList.toggle("error", !!erro);
    toast.classList.remove("hidden");
    clearTimeout(tToast);
    tToast = setTimeout(function () { toast.classList.add("hidden"); }, 3200);
  };
  document.addEventListener("toast", function (ev) {
    var d = ev.detail || {};
    window.PainelToast(d.msg || "", d.erro);
  });

  // ---- Drawer lateral (visual do painel B): sem ESC, sem clique fora, empurra a página.
  var drawer = document.getElementById("drawer");
  var corpo = document.getElementById("drawer-corpo");
  var voltarFoco = null;

  function abrirDrawer() {
    if (!drawer) return;
    drawer.classList.remove("hidden");
    drawer.setAttribute("aria-hidden", "false");
    document.body.classList.add("drawer-aberto");
    var alvo = drawer.querySelector("input,select,textarea,button") || drawer.querySelector(".drawer");
    alvo.focus();
  }
  function fecharDrawer() {
    if (!drawer) return;
    drawer.classList.add("hidden");
    drawer.setAttribute("aria-hidden", "true");
    document.body.classList.remove("drawer-aberto");
    if (corpo) corpo.replaceChildren();
    if (voltarFoco && voltarFoco.focus) voltarFoco.focus();
    voltarFoco = null;
  }
  document.addEventListener("click", function (ev) {
    var a = ev.target.closest("[data-drawer]");
    if (a && corpo) {
      ev.preventDefault();
      voltarFoco = a;
      fetch(a.getAttribute("data-drawer"), { headers: { "HX-Request": "true" }, credentials: "same-origin" })
        .then(function (r) { if (!r.ok) throw new Error(r.status); return r.text(); })
        .then(function (html) {
          // Fragmento do próprio servidor (html/template, já escapado). DOMParser não executa scripts.
          var doc = new DOMParser().parseFromString(html, "text/html");
          corpo.replaceChildren.apply(corpo, Array.prototype.slice.call(doc.body.childNodes));
          if (window.htmx) window.htmx.process(corpo);
          abrirDrawer();
        })
        .catch(function () { window.PainelToast("Não foi possível abrir o formulário.", true); });
      return;
    }
    if (ev.target.closest("[data-drawer-fechar]")) fecharDrawer();
  });
  document.addEventListener("htmx:afterSwap", function (ev) {
    if (ev.target && ev.target.id === "drawer-corpo") abrirDrawer();
  });
})();
