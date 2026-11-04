/* Calculadora de prazo (Plano 7). O resultado chega por htmx em #resultadoPrazo.
   O servidor responde 400 com a MENSAGEM do motivo (o B perdia o motivo): o htmx não troca respostas 4xx/5xx
   por padrão, então liberamos a troca só para esse alvo. Sem HTML inline, sem eval. */
(function () {
  "use strict";
  document.addEventListener("htmx:beforeSwap", function (ev) {
    var alvo = ev.detail && ev.detail.target;
    if (alvo && alvo.id === "resultadoPrazo" && ev.detail.xhr && ev.detail.xhr.status >= 400) {
      ev.detail.shouldSwap = true;
      ev.detail.isError = false;
    }
  });
  // Falha de rede: o mesmo texto do B.
  document.addEventListener("htmx:sendError", function (ev) {
    var alvo = document.getElementById("resultadoPrazo");
    if (!alvo || !ev.target || ev.target.getAttribute("hx-target") !== "#resultadoPrazo") return;
    var bloco = document.createElement("div");
    bloco.className = "form-block form-block-erro";
    bloco.textContent = "Falha ao calcular o prazo.";
    alvo.replaceChildren(bloco);
  });
})();
