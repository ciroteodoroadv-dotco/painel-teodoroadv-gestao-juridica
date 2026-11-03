/* Formulário no drawer, acordeão, linhas repetíveis, campos que aparecem/somem e filtro de tabela.
   Tudo por atributos data-* e delegação de eventos (sem handler inline). */
(function () {
  "use strict";
  var App = (window.App = window.App || {});

  function el(alvo) { return alvo && alvo.closest ? alvo : (alvo && alvo.parentElement); }

  // ---- Formulário no drawer: <form data-drawer-form action="/x" method="post">
  // 200 + HTML  -> troca o drawer (erro de validação mantém o drawer aberto);
  // X-Painel-Ok -> fecha o drawer, mostra o toast e recarrega o conteúdo da página;
  // texto simples (erro) -> toast de erro e o formulário continua como estava.
  document.addEventListener("submit", function (ev) {
    var f = ev.target;
    if (!f.matches || !f.matches("form[data-drawer-form]")) return;
    ev.preventDefault();
    var botao = f.querySelector('[type="submit"]');
    var rotulo = botao ? botao.textContent : "";
    if (botao) { botao.disabled = true; botao.textContent = "Salvando…"; }
    var restaurar = function () {
      if (botao && botao.isConnected) { botao.disabled = false; botao.textContent = rotulo; }
      if (f.hasAttribute("data-reset-ao-falhar")) f.reset(); // ex.: "Mover para..." volta ao título se o servidor recusar
    };
    fetch(f.getAttribute("action") || location.href, {
      method: (f.getAttribute("method") || "post").toUpperCase(),
      body: new FormData(f),
      headers: { "X-CSRF-Token": App.csrf ? App.csrf() : "", "HX-Request": "true" },
      credentials: "same-origin"
    }).then(function (r) {
      var ok = r.headers.get("X-Painel-Ok");
      var redir = r.headers.get("HX-Redirect");
      if (ok !== null && App.aposSalvar) {
        var tratado = App.aposSalvar(r, f); // ex.: pessoa criada de dentro do atendimento volta ao rascunho
        if (tratado) return tratado;
      }
      if (ok !== null || redir !== null) {
        App.drawerStack.length = 0;
        App.closeDrawer();
        window.PainelToast(ok || "Salvo.", false);
        return App.recarregarConteudo().catch(function () {
          var falhas = {
            pessoas: "Ação registrada, mas houve falha ao recarregar a lista de pessoas.",
            processos: "Ação registrada, mas houve falha ao recarregar a lista de processos.",
            atendimentos: "Ação registrada, mas houve falha ao recarregar os atendimentos.",
            tarefas: "Ação registrada, mas houve falha ao recarregar a lista de tarefas."
          };
          window.PainelToast(falhas[document.body.getAttribute("data-tab")] || "Ação registrada, mas houve falha ao recarregar a lista.", true);
        });
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

  // ---- Envia o formulário ao mudar o campo: <select form="id" data-submit-on-change> ("Mover para..." do B).
  document.addEventListener("change", function (ev) {
    var c = ev.target;
    if (!c.matches || !c.matches("[data-submit-on-change]") || !c.value || !c.form) return;
    c.form.requestSubmit();
  });

  // ---- Troca as opções de um select pelo conteúdo de um <template> ao marcar o radio:
  // <input type="radio" data-origem-opcoes="#tplOrigemAtendimento" data-origem-alvo="#origemIdSelect"> (App.refreshOrigemOptions).
  document.addEventListener("change", function (ev) {
    var r = ev.target;
    if (!r.matches || !r.matches("input[type=radio][data-origem-opcoes]")) return;
    var tpl = document.querySelector(r.getAttribute("data-origem-opcoes"));
    var alvo = document.querySelector(r.getAttribute("data-origem-alvo"));
    if (tpl && alvo && tpl.content) alvo.replaceChildren(tpl.content.cloneNode(true));
  });

  // ---- Busca no servidor: <form data-busca-servidor method="get" action="/x"> (Tarefas busca ao enviar, como no B).
  // Recarrega só o conteúdo (os painéis abertos do acordeão continuam abertos) e guarda a busca na URL.
  document.addEventListener("submit", function (ev) {
    var f = ev.target;
    if (!f.matches || !f.matches("form[data-busca-servidor]")) return;
    ev.preventDefault();
    var url = (f.getAttribute("action") || location.pathname) + "?" + new URLSearchParams(new FormData(f)).toString();
    App.recarregarConteudo(url).then(function () { history.replaceState(null, "", url); }).catch(function () {
      window.PainelToast("Não foi possível buscar.", true);
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

  // ---- Revela campo: <select data-mostra-quando="Outro" data-mostra="#idCampo">. Nas linhas repetíveis, onde
  // não há id único, data-mostra="proximo" revela o campo que vem logo depois do select.
  function aplicarMostra(sel) {
    var a = sel.getAttribute("data-mostra");
    var alvo = a === "proximo" ? sel.nextElementSibling : document.querySelector(a);
    if (alvo) alvo.hidden = sel.value !== sel.getAttribute("data-mostra-quando");
  }
  document.addEventListener("change", function (ev) {
    if (ev.target.matches && ev.target.matches("select[data-mostra-quando]")) aplicarMostra(ev.target);
  });
  document.addEventListener("DOMContentLoaded", function () {
    document.querySelectorAll("select[data-mostra-quando]").forEach(aplicarMostra);
  });

  // ---- Alterna blocos pela opção marcada: <input type="radio" name="x" data-exibe="#bloco">.
  // Aparece o bloco do radio marcado; somem os dos outros radios do grupo. Campos com data-req dentro do
  // bloco visível passam a ser obrigatórios (e deixam de ser no oculto), como no B.
  document.addEventListener("change", function (ev) {
    var r = ev.target;
    if (!r.matches || !r.matches("input[type=radio][data-exibe]")) return;
    (r.form || document).querySelectorAll('input[type=radio][data-exibe][name="' + r.name + '"]').forEach(function (o) {
      var bloco = document.querySelector(o.getAttribute("data-exibe"));
      if (!bloco) return;
      bloco.hidden = !o.checked;
      bloco.querySelectorAll("[data-req]").forEach(function (c) { c.required = o.checked; });
    });
  });

  // ---- Caixa de marcar que vira 0/1: <input type="hidden" name="x" value="0"><input type="checkbox" data-espelha>.
  // O campo escondido (logo antes) leva a marcação, mantendo as listas paralelas das linhas repetíveis alinhadas.
  document.addEventListener("change", function (ev) {
    var c = ev.target;
    if (!c.matches || !c.matches("input[type=checkbox][data-espelha]")) return;
    var h = c.previousElementSibling;
    if (h) h.value = c.checked ? "1" : "0";
  });

  // ---- Maiúsculas ao digitar: <input data-maiusculo> (toLocaleUpperCase('pt-BR'), mantendo o cursor).
  document.addEventListener("input", function (ev) {
    var i = ev.target;
    if (!i.matches || !i.matches("input[data-maiusculo]")) return;
    var a = i.selectionStart, b = i.selectionEnd;
    i.value = i.value.toLocaleUpperCase("pt-BR");
    if (a !== null) i.setSelectionRange(a, b);
  });

  // ---- Buscar CEP: a consulta externa só será ligada em plano futuro (integração desligada por padrão).
  document.addEventListener("click", function (ev) {
    var e = el(ev.target), b = e && e.closest("[data-buscar-cep]");
    if (!b) return;
    var campo = b.form && b.form.elements.cep;
    var digitos = campo ? campo.value.replace(/\D/g, "") : "";
    if (digitos.length !== 8) { window.PainelToast("Informe um CEP com oito dígitos.", true); return; }
    window.PainelToast("Consulta de CEP indisponível neste painel.", true);
  });

  // ---- Filtro de tabela no navegador: <input data-filtro="SELETOR"> (sem acento, sem maiúscula).
  // Casa com o atributo data-busca da linha, se houver; senão, com o texto dela. Dentro de um acordeão,
  // atualiza o número do botão do grupo e troca a tabela vazia pelo aviso <div data-vazio-filtro hidden>.
  function normaliza(s) { return String(s).normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase(); }
  document.addEventListener("input", function (ev) {
    var i = ev.target;
    if (!i.matches || !i.matches("input[data-filtro]")) return;
    var q = normaliza(i.value.trim());
    document.querySelectorAll(i.getAttribute("data-filtro")).forEach(function (tabela) {
      var visiveis = 0;
      tabela.querySelectorAll("tbody tr").forEach(function (tr) {
        var texto = tr.hasAttribute("data-busca") ? tr.getAttribute("data-busca") : tr.textContent;
        tr.hidden = q !== "" && normaliza(texto).indexOf(q) < 0;
        if (!tr.hidden) visiveis++;
      });
      var vazio = tabela.parentElement && tabela.parentElement.querySelector("[data-vazio-filtro]");
      if (vazio) { vazio.hidden = visiveis > 0; if (!i.hasAttribute("data-manter-tabela")) tabela.hidden = visiveis === 0; }
      var item = tabela.closest("[data-acc]");
      var cont = item && document.querySelector('button.acc-head[data-acc="' + item.getAttribute("data-acc") + '"] .count');
      if (cont) cont.textContent = String(visiveis);
    });
  });
})();
