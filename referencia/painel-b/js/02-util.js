/* =========================================================
   PAINEL TEODORO - modulo 02-util: utilitários: datas, máscaras, validadores, moeda, toast
   Extraido de web/index.html (linhas 77-185) na etapa 4b da
   refatoracao (2026-10-30). Scripts classicos carregados em ordem
   numerica pelo index.html: compartilham o escopo global lexico,
   exatamente como antes dentro da IIFE unica. Sem build toolchain.
   ========================================================= */
'use strict';

/* =========================================================
   UTILITÁRIOS
   ========================================================= */
function uid(){
  return (crypto.randomUUID ? crypto.randomUUID() : 'id_'+Date.now()+'_'+Math.random().toString(36).slice(2));
}
function todayISO(){
  const now = new Date();
  const offset = now.getTimezoneOffset() * 60000;
  return new Date(now.getTime() - offset).toISOString().slice(0,10);
}
function fmtDate(iso){
  if(!iso) return '—';
  const [y,m,d] = iso.split('-').map(Number);
  if(!y || !m || !d) return '—';
  return new Intl.DateTimeFormat('pt-BR').format(new Date(y,m-1,d));
}
// Datas vindas da API real de processos às vezes não são datas de verdade
// (ex.: "[VERIFICAR]" ou "[FALTA: ...]") — usa isto em vez de fmtDate() quando
// o valor pode não ser uma data ISO válida.
function fmtDateOrRaw(v){
  if(!v) return '—';
  if(/^\d{4}-\d{2}-\d{2}$/.test(v)) return fmtDate(v);
  return escapeHTML(v);
}
function escapeHTML(s){
  return String(s==null?'':s).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
}
function inlineArg(s){
  return escapeHTML(JSON.stringify(String(s==null?'':s)));
}
function onlyDigits(s){ return String(s||'').replace(/\D/g,''); }

function applyMask(value, pattern){
  const digits = onlyDigits(value);
  let out='', di=0;
  for(let i=0;i<pattern.length && di<digits.length;i++){
    if(pattern[i]==='#'){ out+=digits[di]; di++; }
    else out+=pattern[i];
  }
  return out;
}
function maskCPF(v){ return applyMask(v,'###.###.###-##'); }
function maskCNPJ(v){ return applyMask(v,'##.###.###/####-##'); }
function maskCEP(v){ return applyMask(v,'#####-###'); }
function maskPhone(v){
  const d = onlyDigits(v).slice(0,11);
  return d.length>10 ? applyMask(d,'(##) #####-####') : applyMask(d,'(##) ####-####');
}
function maskCNJ(v){
  const d = onlyDigits(v).slice(0,20);
  const parts=[7,2,4,1,2,4], seps=['','-','.','.','.','.'];
  let out='', idx=0;
  parts.forEach((len,i)=>{
    if(d.length>idx){ out += seps[i] + d.slice(idx, idx+len); idx+=len; }
  });
  return out;
}
function isValidCPF(cpfRaw){
  const cpf = onlyDigits(cpfRaw);
  if(cpf.length!==11 || /^(\d)\1{10}$/.test(cpf)) return false;
  let sum=0;
  for(let i=0;i<9;i++) sum += parseInt(cpf[i],10)*(10-i);
  let rev = 11 - (sum%11); if(rev>=10) rev=0;
  if(rev !== parseInt(cpf[9],10)) return false;
  sum=0;
  for(let i=0;i<10;i++) sum += parseInt(cpf[i],10)*(11-i);
  rev = 11 - (sum%11); if(rev>=10) rev=0;
  return rev === parseInt(cpf[10],10);
}
function isValidCNPJ(cnpjRaw){
  const cnpj = onlyDigits(cnpjRaw);
  if(cnpj.length!==14 || /^(\d)\1{13}$/.test(cnpj)) return false;
  function calc(base){
    let len = base.length, pos = len-7, soma=0;
    for(let i=len;i>=1;i--){ soma += parseInt(base.charAt(len-i),10)*pos--; if(pos<2) pos=9; }
    const r = soma % 11;
    return r<2 ? 0 : 11-r;
  }
  const d1 = calc(cnpj.substring(0,12));
  if(d1 !== parseInt(cnpj.charAt(12),10)) return false;
  const d2 = calc(cnpj.substring(0,13));
  return d2 === parseInt(cnpj.charAt(13),10);
}
function maskCurrencyInput(v){
  let d = onlyDigits(v);
  if(!d) return '';
  d = d.replace(/^0+(?=\d)/,'');
  while(d.length<3) d = '0'+d;
  const cents = d.slice(-2);
  let intPart = d.slice(0,-2);
  intPart = intPart.replace(/\B(?=(\d{3})+(?!\d))/g,'.');
  return 'R$ '+intPart+','+cents;
}
function currencyToNumber(v){
  const d = onlyDigits(v);
  return d ? parseInt(d,10)/100 : 0;
}

function showToast(msg, isError){
  const t = document.getElementById('toast');
  t.textContent = msg;
  t.className = 'toast'+(isError?' error':'');
  clearTimeout(showToast._h);
  showToast._h = setTimeout(()=>{ t.className='toast hidden'; }, 3200);
}
function toastNaoImplementado(){
  showToast('Ainda não implementado nesta fase do painel.', true);
}
