// Chave de segurança (WebAuthn). Sem inline: a CSP do painel só aceita scripts do próprio domínio.
(function () {
  "use strict";
  var csrf = (document.querySelector('meta[name="csrf-token"]') || {}).content || "";

  function dec(s) {
    s = s.replace(/-/g, "+").replace(/_/g, "/");
    while (s.length % 4) s += "=";
    var bin = atob(s), out = new Uint8Array(bin.length);
    for (var i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
    return out.buffer;
  }
  function enc(buf) {
    var b = new Uint8Array(buf), s = "";
    for (var i = 0; i < b.length; i++) s += String.fromCharCode(b[i]);
    return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  }
  function msg(text) {
    var el = document.getElementById("wa-msg");
    if (el) el.textContent = text;
  }
  async function post(url, body) {
    var r = await fetch(url, {
      method: "POST", credentials: "same-origin",
      headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
      body: body === undefined ? "{}" : JSON.stringify(body)
    });
    var j = await r.json().catch(function () { return {}; });
    if (!r.ok) throw new Error(j.erro || "Falha na operação.");
    return j;
  }

  async function registrar() {
    var nome = (document.getElementById("wa-nome") || {}).value || "";
    var campoSenha = document.getElementById("wa-senha"); // só em "Minha conta": chave adicional pede a senha atual
    var opts = await post("/2fa/webauthn/register/begin", campoSenha ? { senha: campoSenha.value } : undefined);
    if (campoSenha) campoSenha.value = "";
    var pk = opts.publicKey;
    pk.challenge = dec(pk.challenge);
    pk.user.id = dec(pk.user.id);
    (pk.excludeCredentials || []).forEach(function (c) { c.id = dec(c.id); });
    var cred = await navigator.credentials.create({ publicKey: pk });
    var res = await post("/2fa/webauthn/register/finish?nome=" + encodeURIComponent(nome), {
      id: cred.id, rawId: enc(cred.rawId), type: cred.type,
      response: {
        attestationObject: enc(cred.response.attestationObject),
        clientDataJSON: enc(cred.response.clientDataJSON),
        transports: cred.response.getTransports ? cred.response.getTransports() : []
      },
      clientExtensionResults: cred.getClientExtensionResults()
    });
    location.href = res.redirect;
  }

  async function entrar() {
    var opts = await post("/2fa/webauthn/login/begin");
    var pk = opts.publicKey;
    pk.challenge = dec(pk.challenge);
    (pk.allowCredentials || []).forEach(function (c) { c.id = dec(c.id); });
    var a = await navigator.credentials.get({ publicKey: pk });
    var res = await post("/2fa/webauthn/login/finish", {
      id: a.id, rawId: enc(a.rawId), type: a.type,
      response: {
        authenticatorData: enc(a.response.authenticatorData),
        clientDataJSON: enc(a.response.clientDataJSON),
        signature: enc(a.response.signature),
        userHandle: a.response.userHandle ? enc(a.response.userHandle) : ""
      },
      clientExtensionResults: a.getClientExtensionResults()
    });
    location.href = res.redirect;
  }

  document.addEventListener("click", function (ev) {
    var btn = ev.target.closest("[data-wa]");
    if (!btn) return;
    if (!window.PublicKeyCredential) { msg("Este navegador não suporta chave de segurança."); return; }
    var fn = btn.getAttribute("data-wa") === "registrar" ? registrar : entrar;
    btn.disabled = true;
    msg("Siga as instruções do navegador…");
    fn().catch(function (e) {
      msg(e && e.name === "NotAllowedError" ? "Operação cancelada ou expirada." : (e.message || "Falha."));
      btn.disabled = false;
    });
  });
})();
