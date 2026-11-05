/* Indicador somente leitura. Nao executa coletas nem cria agendamentos. */
'use strict';
(() => {
 let buscando=false;
 const rotulos={intimacoes:'DJEN',monitor:'DataJud','monitor-segredos':'DJEN por processo sigiloso'};
 const estados={sucesso:'última coleta concluída',parcial:'coleta parcial',falha:'erro',bloqueado:'bloqueado',nao_sincronizado:'sem tentativa registrada'};
 function horario(s){if(!s)return 'não registrado';const d=new Date(s);return Number.isNaN(d.getTime())?'horário inválido':d.toLocaleString('pt-BR',{timeZone:'America/Sao_Paulo'})+' (Brasília)';}
 async function atualizar(){
  const el=document.getElementById('syncStatus');if(!el||buscando)return;buscando=true;
  try{
   const resp=await fetch('/api/sincronizacao/status',{cache:'no-store'});if(!resp.ok)throw Error('indisponível');const dados=await resp.json();if(!Array.isArray(dados.fontes)||dados.fontes.length===0)throw Error('inválido');
   el.replaceChildren();const titulo=document.createElement('strong');titulo.textContent='Estado das coletas';el.append(titulo);
   const aviso=document.createElement('p');aviso.textContent='Histórico local. Não comprova agendamento ativo nem substitui a conferência nas fontes oficiais.';el.append(aviso);
   const lista=document.createElement('ul');for(const f of dados.fontes){const li=document.createElement('li');if(f.status!=='sucesso')li.className='sync-error';li.textContent=`${rotulos[f.modulo]||f.modulo}: ${estados[f.status]||'estado desconhecido'}. Tentativa: ${horario(f.ultima_tentativa)}. Sucesso: ${horario(f.ultimo_sucesso)}. ${f.mensagem||''}`;lista.append(li);}el.append(lista);
  }catch{el.replaceChildren();const t=document.createElement('strong');t.textContent='Estado das coletas indisponível';const p=document.createElement('p');p.textContent='Não foi possível ler o histórico local. Não presuma que houve atualização. Confira as fontes oficiais.';el.append(t,p);}
  finally{buscando=false;}
 }
 document.addEventListener('DOMContentLoaded',atualizar);
 document.addEventListener('visibilitychange',()=>{if(!document.hidden)atualizar();});
 if(typeof Painel!=='undefined')for(const m of ['intimacoes','monitor','monitor-segredos'])Painel.ao(m,atualizar);
 // Atualiza somente o pequeno resumo local, sem chamadas às fontes externas.
 setInterval(()=>{if(!document.hidden)atualizar();},60000);
})();
