/* Botão "Buscar CEP" do cadastro de pessoa (App.buscarCEP do painel B, web/js/14-pessoas.js).
   O navegador só fala com o painel (POST /pessoas/cep); o painel consulta o ViaCEP. Só os 8 dígitos saem.
   Sem HTML inline: o botão é <button data-buscar-cep data-cep-url="/pessoas/cep">. */
(function () {
  "use strict";
  var App = (window.App = window.App || {});

  function alvo(ev) { return ev.target && ev.target.closest ? ev.target : ev.target && ev.target.parentElement; }
  function aviso(msg, erro) { if (window.PainelToast) window.PainelToast(msg, !!erro); }

  // Confirmação SOBREPOSTA (confirmarEmDrawer com sobrepor do B): empilha sobre o formulário, que volta intacto ao fechar.
  App.confirmarSobreposto = function (opcoes) {
    return new Promise(function (resolve) {
      var h2 = document.createElement("h2");
      h2.textContent = opcoes.titulo;
      var p = document.createElement("p");
      p.className = "drawer-sub";
      p.textContent = opcoes.mensagem;
      var acoes = document.createElement("div");
      acoes.className = "drawer-actions";
      var cancelar = document.createElement("button");
      cancelar.type = "button";
      cancelar.className = "btn btn-ghost";
      cancelar.textContent = "Cancelar";
      var ok = document.createElement("button");
      ok.type = "button";
      ok.className = "btn btn-primary";
      ok.textContent = opcoes.rotuloConfirmar || "Confirmar";
      acoes.append(cancelar, ok);
      var fechar = function (valor) { App.closeDrawer(); resolve(valor); };
      cancelar.addEventListener("click", function () { fechar(false); });
      ok.addEventListener("click", function () { fechar(true); });
      App.openDrawer([h2, p, acoes], true);
    });
  };

  function consultar(url, cep) {
    return fetch(url, {
      method: "POST",
      body: new URLSearchParams({ cep: cep }),
      headers: { "X-CSRF-Token": App.csrf ? App.csrf() : "", "Content-Type": "application/x-www-form-urlencoded", "Accept": "application/json" },
      credentials: "same-origin"
    }).then(function (r) {
      return r.json().catch(function () { return {}; }).then(function (j) {
        if (!r.ok || j.erro) throw new Error(j.erro || "Falha ao consultar o CEP.");
        return j;
      });
    });
  }

  function preencher(form, resultado, substituir) {
    [["logradouro", resultado.logradouro], ["bairro", resultado.bairro], ["cidade", resultado.municipio], ["uf", resultado.uf]].forEach(function (par) {
      var campo = form.elements[par[0]], valor = String(par[1] || "").trim();
      if (!campo || !valor) return;
      if (String(campo.value || "").trim() && !substituir) return;
      campo.value = par[0] === "uf" ? valor.toUpperCase() : valor;
      campo.dispatchEvent(new Event("input", { bubbles: true }));
    });
    var fonte = form.querySelector("[data-cep-fonte]");
    if (fonte) fonte.textContent = "Fonte consultada: " + (resultado.fonte || "não informada");
  }

  document.addEventListener("click", function (ev) {
    var e = alvo(ev), b = e && e.closest("[data-buscar-cep]");
    if (!b) return;
    ev.preventDefault();
    var form = b.form, campo = form && form.elements.cep;
    var cep = campo ? campo.value.replace(/\D/g, "") : "";
    if (cep.length !== 8) { aviso("Informe um CEP com oito dígitos.", true); return; }
    if (b.disabled) return;
    b.disabled = true;
    consultar(b.getAttribute("data-cep-url") || "/pessoas/cep", cep).then(function (res) {
      var ocupados = [["logradouro", res.logradouro], ["bairro", res.bairro], ["cidade", res.municipio], ["uf", res.uf]].filter(function (par) {
        var c = form.elements[par[0]];
        return String(par[1] || "").trim() && c && String(c.value || "").trim();
      });
      var pergunta = ocupados.length
        ? App.confirmarSobreposto({ titulo: "Endereço pelo CEP", mensagem: "Há campos de endereço preenchidos. Deseja substituí-los pelos dados do CEP?", rotuloConfirmar: "Substituir" })
        : Promise.resolve(false);
      return pergunta.then(function (substituir) {
        preencher(form, res, substituir);
        aviso(substituir ? "Endereço preenchido e campos existentes substituídos." : "Endereço preenchido nos campos vazios.", false);
      });
    }).catch(function (err) {
      aviso(err && err.message ? err.message : "Falha ao consultar o CEP.", true);
    }).then(function () { b.disabled = false; });
  });
})();
