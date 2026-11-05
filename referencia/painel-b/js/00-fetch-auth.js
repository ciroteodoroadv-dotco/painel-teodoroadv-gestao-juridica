/* =========================================================
   PAINEL TEODORO - modulo 00-fetch-auth: wrapper de fetch com re-login em drawer (401 em escrita)
   Extraido de web/index.html (linhas 44-53) na etapa 4b da
   refatoracao (2026-10-30). Scripts classicos carregados em ordem
   numerica pelo index.html: compartilham o escopo global lexico,
   exatamente como antes dentro da IIFE unica. Sem build toolchain.
   ========================================================= */
'use strict';

const fetchOriginal = window.fetch.bind(window);
window.fetch = async function(url, opcoes={}){
  let resposta = await fetchOriginal(url, opcoes);
  const metodo = (opcoes.method||'GET').toUpperCase();
  if(resposta.status!==401 || ['GET','HEAD'].includes(metodo) || String(url).includes('/api/auth/')) return resposta;
  const senha = await App.pedirSenhaEmDrawer('Sessão expirada', 'Informe a senha do painel para continuar.');
  if(!senha) return resposta;
  const login = await fetchOriginal('/api/auth/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({senha})});
  return login.ok ? fetchOriginal(url, opcoes) : resposta;
};
