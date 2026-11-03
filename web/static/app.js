// Comportamentos comuns do painel. Sem inline: a CSP só aceita scripts do próprio domínio.
(function () {
  "use strict";
  var App = (window.App = window.App || {});

  // Aviso ao sair de formulário alterado (marque o <form> com data-protege).
  var sujo = false;
  document.addEventListener("input", function (ev) {
    if (ev.target.closest && ev.target.closest("form[data-protege]")) sujo = true;
  });
  document.addEventListener("submit", function () { sujo = false; });
  window.addEventListener("beforeunload", function (ev) {
    // Só avisa se o formulário alterado ainda está na tela (um drawer cancelado não deve travar a navegação).
    if (sujo && document.querySelector("form[data-protege]")) { ev.preventDefault(); ev.returnValue = ""; }
  });

  function csrf() {
    var m = document.querySelector('meta[name="csrf-token"]');
    return m ? m.content : "";
  }
  App.csrf = csrf;

  // htmx: envia o token CSRF em toda requisição.
  document.addEventListener("htmx:configRequest", function (ev) {
    ev.detail.headers["X-CSRF-Token"] = csrf();
  });

  // Confirmação em duas etapas sem diálogo do navegador: botão data-confirmar pede um segundo clique.
  document.addEventListener("click", function (ev) {
    var b = ev.target.closest && ev.target.closest("button[data-confirmar]");
    if (!b || b.dataset.armado) return;
    ev.preventDefault();
    b.dataset.armado = "1";
    b.dataset.rotulo = b.textContent;
    b.textContent = b.getAttribute("data-confirmar");
    setTimeout(function () { delete b.dataset.armado; b.textContent = b.dataset.rotulo; }, 5000);
  });

  // Toast do painel B (showToast): 3,2 s; erro em vinho escuro.
  var tToast = null;
  window.PainelToast = function (msg, erro) {
    var toast = document.getElementById("toast");
    if (!toast) return;
    toast.textContent = msg;
    toast.className = "toast" + (erro ? " error" : "");
    clearTimeout(tToast);
    tToast = setTimeout(function () { toast.className = "toast hidden"; }, 3200);
  };
  document.addEventListener("toast", function (ev) {
    var d = ev.detail || {};
    window.PainelToast(d.msg || "", d.erro);
  });

  // Recarrega #content, #stats, o menu e as faixas a partir da própria página (sem recarregar a janela).
  App.recarregarConteudo = function (url) {
    return fetch(url || location.href, { credentials: "same-origin" })
      .then(function (r) { return r.text(); })
      .then(function (html) {
        // O acordeão mantém os painéis abertos depois de recarregar (state.accOpen do B).
        var abertos = Array.prototype.map.call(document.querySelectorAll(".acc-item.open"), function (x) { return x.getAttribute("data-acc"); });
        var doc = new DOMParser().parseFromString(html, "text/html");
        ["content", "stats", "navTabs", "navSistema", "topbar", "syncStatus"].forEach(function (id) {
          var novo = doc.getElementById(id), atual = document.getElementById(id);
          if (novo && atual) atual.replaceChildren.apply(atual, Array.prototype.slice.call(novo.childNodes));
        });
        abertos.forEach(function (k) {
          document.querySelectorAll('[data-acc="' + k + '"]').forEach(function (x) { x.classList.add("open"); });
        });
      });
  };

  // Rodapé "Estado das coletas" (16-sync-status.js do B): <section data-refresh="/sync/estado"> recarrega só o miolo,
  // a cada 60 s e ao voltar à aba. Só lê o histórico local do painel; nunca dispara coleta nem consulta fonte externa.
  var buscandoSync = false;
  function syncIndisponivel(el) {
    var t = document.createElement("strong");
    t.textContent = "Estado das coletas indisponível";
    var p = document.createElement("p");
    p.textContent = "Não foi possível ler o histórico local. Não presuma que houve atualização. Confira as fontes oficiais.";
    el.replaceChildren(t, p);
  }
  App.atualizarSync = function () {
    var el = document.querySelector("[data-refresh]");
    if (!el || buscandoSync) return Promise.resolve();
    buscandoSync = true;
    return fetch(el.getAttribute("data-refresh"), { cache: "no-store", credentials: "same-origin" })
      .then(function (r) {
        if (!r.ok || r.headers.get("X-Painel-Fragmento") !== "sync") throw new Error("indisponível");
        return r.text();
      })
      .then(function (html) { el.replaceChildren.apply(el, App.nodesDeHTML(html)); })
      .catch(function () { syncIndisponivel(el); })
      .then(function () { buscandoSync = false; });
  };
  document.addEventListener("visibilitychange", function () { if (!document.hidden) App.atualizarSync(); });
  setInterval(function () { if (!document.hidden) App.atualizarSync(); }, 60000);
})();
