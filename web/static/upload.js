/* Envio imediato de arquivo: <input type="file" data-upload="/rota"> (App.adicionarDocumentoTarefa do B).
   Ao escolher o PDF, ele vai por fetch (FormData + CSRF); o servidor confere o tipo pelo CONTEÚDO. */
(function () {
  "use strict";
  var App = (window.App = window.App || {});

  document.addEventListener("change", function (ev) {
    var i = ev.target;
    if (!i.matches || !i.matches("input[type=file][data-upload]")) return;
    var arquivo = i.files && i.files[0];
    if (!arquivo) return;
    var corpo = new FormData();
    corpo.append("documento", arquivo);
    fetch(i.getAttribute("data-upload"), {
      method: "POST",
      body: corpo,
      headers: { "X-CSRF-Token": App.csrf ? App.csrf() : "", "HX-Request": "true" },
      credentials: "same-origin"
    }).then(function (r) {
      if (r.headers.get("X-Painel-Ok") !== null) {
        window.PainelToast(window.App.textoDoCabecalho(r.headers.get("X-Painel-Ok")), false);
        i.value = "";
        return null;
      }
      return r.text().then(function (t) { window.PainelToast("Falha ao juntar documento: " + (t || r.status), true); i.value = ""; });
    }).catch(function () {
      window.PainelToast("Falha ao juntar documento.", true);
      i.value = "";
    });
  });
})();
