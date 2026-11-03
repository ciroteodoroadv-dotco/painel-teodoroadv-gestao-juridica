/* Formulário no drawer, acordeão, linhas repetíveis, "Outro" revela campo e filtro de tabela.
   Tudo por atributos data-* e delegação de eventos (sem handler inline). */
(function () {
  "use strict";
  var App = (window.App = window.App || {});

  function el(alvo) { return alvo && alvo.closest ? alvo : (alvo && alvo.parentElement); }

  // ---- Formulário no drawer: <form data-drawer-form action="/x" method="post">
  // 200 + HTML  -> troca o drawer (erro de validação mantém o drawer aberto);
  // X-Painel-Ok -> fecha o drawer, mostra o toast e recarrega o conteúdo da página.
  document.addEventListener("submit", function (ev) {
    var f = ev.target;
    if (!f.matches || !f.matches("form[data-drawer-form]")) return;
    ev.preventDefault();
    var botao = f.querySelector('[type="submit"]');
    var rotulo = botao ? botao.textContent : "";
    if (botao) { botao.disabled = true; botao.textContent = "Salvando…"; }
    var restaurar = function () { if (botao && botao.isConnected) { botao.disabled = false; botao.textContent = rotulo; } };
    fetch(f.getAttribute("action") || location.href, {
      method: (f.getAttribute("method") || "post").toUpperCase(),
      body: new FormData(f),
      headers: { "X-CSRF-Token": App.csrf ? App.csrf() : "", "HX-Request": "true" },
      credentials: "same-origin"
    }).then(function (r) {
      var ok = r.headers.get("X-Painel-Ok");
      var redir = r.headers.get("HX-Redirect");
      if (ok !== null || redir !== null) {
        App.drawerStack.length = 0;
        App.closeDrawer();
        window.PainelToast(ok || "Salvo.", false);
        return App.recarregarConteudo();
      }
      var html = (r.headers.get("Content-Type") || "").indexOf("text/html") === 0;
      return r.text().then(function (t) {
        if (html) { App.openDrawer(App.nodesDeHTML(t), false); return; }
        restaurar();
        window.PainelToast(t || "Não foi possível salvar.", true);
      });
    }).catch(function () {
      restaurar();
      window.PainelToast("Não foi possível salvar.", true);
    });
  });

  // ---- Acordeão: <button class="acc-head" data-acc="GRUPO:CHAVE"> e <div class="acc-item" data-acc="GRUPO:CHAVE">.
  // O grupo "agenda" abre um por vez; os demais abrem vários ao mesmo tempo.
  document.addEventListener("click", function (ev) {
    var b = el(ev.target) && el(ev.target).closest("button[data-acc]");
    if (!b) return;
    var chave = b.getAttribute("data-acc");
    var grupo = chave.split(":")[0];
    var abrir = !b.classList.contains("open");
    if (grupo === "agenda") {
      document.querySelectorAll('[data-acc^="agenda:"]').forEach(function (x) { x.classList.remove("open"); });
    }
    document.querySelectorAll('[data-acc="' + chave + '"]').forEach(function (x) { x.classList.toggle("open", abrir); });
  });

  // ---- Linhas repetíveis.
  document.addEventListener("click", function (ev) {
    var e = el(ev.target);
    if (!e) return;
    var add = e.closest("[data-add-row]");
    if (add) {
      var tpl = document.querySelector(add.getAttribute("data-add-row"));
      var alvo = document.querySelector(add.getAttribute("data-row-target"));
      if (tpl && alvo && tpl.content) alvo.appendChild(tpl.content.cloneNode(true));
      return;
    }
    var rem = e.closest("[data-remove-row]");
    if (rem) {
      var linha = rem.closest(rem.getAttribute("data-remove-row"));
      if (linha) linha.remove();
    }
  });

  // ---- "Outro" revela campo: <select data-mostra-quando="Outro" data-mostra="#idCampo">.
  function aplicarMostra(sel) {
    var alvo = document.querySelector(sel.getAttribute("data-mostra"));
    if (alvo) alvo.hidden = sel.value !== sel.getAttribute("data-mostra-quando");
  }
  document.addEventListener("change", function (ev) {
    if (ev.target.matches && ev.target.matches("select[data-mostra-quando]")) aplicarMostra(ev.target);
  });
  document.addEventListener("DOMContentLoaded", function () {
    document.querySelectorAll("select[data-mostra-quando]").forEach(aplicarMostra);
  });

  // ---- Filtro de tabela no navegador: <input data-filtro="#idTabela"> (sem acento, sem maiúscula).
  function normaliza(s) { return String(s).normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase(); }
  document.addEventListener("input", function (ev) {
    var i = ev.target;
    if (!i.matches || !i.matches("input[data-filtro]")) return;
    var tabela = document.querySelector(i.getAttribute("data-filtro"));
    if (!tabela) return;
    var q = normaliza(i.value.trim());
    tabela.querySelectorAll("tbody tr").forEach(function (tr) {
      tr.hidden = q !== "" && normaliza(tr.textContent).indexOf(q) < 0;
    });
  });
})();
