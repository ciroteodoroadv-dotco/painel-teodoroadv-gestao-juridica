/* Atendimento: busca dentro das listas de pessoas e "Novo cliente / Nova parte adversa" (App.novaPessoaNoAtendimento
   e App.voltarPessoaAtendimento do B). Sem handler inline: tudo por atributos data-* e delegação. */
(function () {
  "use strict";
  var App = (window.App = window.App || {});
  var papelPendente = "clientes";

  function normaliza(s) { return String(s).normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase(); }

  // <input data-filtra-opcoes>: esconde, nos selects do mesmo fieldset, as opções que não casam com a busca
  // (a opção em branco e a selecionada ficam), como App.filtrarPessoasAtendimento do B.
  document.addEventListener("input", function (ev) {
    var i = ev.target;
    if (!i.matches || !i.matches("input[data-filtra-opcoes]")) return;
    var q = normaliza(i.value);
    var grupo = i.closest("fieldset");
    if (!grupo) return;
    grupo.querySelectorAll("select option").forEach(function (o) {
      o.hidden = !!o.value && !o.selected && normaliza(o.textContent).indexOf(q) < 0;
    });
  });

  // "+ Novo cliente" / "+ Nova parte adversa": o rascunho fica na pilha do drawer (com o que foi digitado).
  document.addEventListener("click", function (ev) {
    var alvo = ev.target.closest ? ev.target : ev.target.parentElement;
    var b = alvo && alvo.closest("[data-nova-pessoa]");
    if (b) papelPendente = b.getAttribute("data-nova-pessoa");
  });

  function rascunhoNaPilha() {
    var topo = App.drawerStack && App.drawerStack[App.drawerStack.length - 1];
    return !!topo && topo.some(function (n) { return n.nodeType === 1 && n.id === "formAtendimento"; });
  }

  function trocaOpcoes(select, modelo) {
    var v = select.value;
    select.replaceChildren.apply(select, Array.prototype.map.call(modelo.childNodes, function (n) { return n.cloneNode(true); }));
    select.value = v;
  }

  // Chamado por forms.js depois de um envio bem-sucedido. Pessoa criada de dentro do atendimento: fecha só o
  // "Nova pessoa", volta ao rascunho e acrescenta a pessoa nova à lista de onde se clicou.
  App.aposSalvar = function (resp, form) {
    var id = resp.headers.get("X-Painel-Id");
    if (!id || form.getAttribute("action") !== "/pessoas" || !rascunhoNaPilha()) return null;
    return fetch("/atendimentos/pessoas", { headers: { "HX-Request": "true" }, credentials: "same-origin" })
      .then(function (r) { return r.text(); })
      .then(function (html) {
        var modelo = new DOMParser().parseFromString(html, "text/html").querySelector("select");
        App.closeDrawer(); // desempilha: volta ao rascunho do atendimento
        var f = document.getElementById("formAtendimento");
        if (!f || !modelo) return;
        f.querySelectorAll(".atendimento-pessoa-linha select").forEach(function (s) { trocaOpcoes(s, modelo); });
        f.querySelectorAll("template").forEach(function (t) {
          t.content.querySelectorAll("select").forEach(function (s) { trocaOpcoes(s, modelo); });
        });
        var nova = modelo.querySelector('option[value="' + id + '"]');
        var criado = f.querySelector('select[name="criado_por"]');
        if (criado && nova) criado.appendChild(nova.cloneNode(true));
        var alvo = f.querySelector("#ate-" + papelPendente), tpl = f.querySelector("#tpl-" + papelPendente);
        if (!alvo || !tpl) return;
        alvo.querySelectorAll(".atendimento-pessoa-linha").forEach(function (l) { // linhas em branco saem, como no B
          if (!l.querySelector("select").value) l.remove();
        });
        alvo.appendChild(tpl.content.cloneNode(true));
        alvo.lastElementChild.querySelector("select").value = id;
        window.PainelToast("Pessoa criada e vinculada ao rascunho do atendimento.", false);
      });
  };
})();
