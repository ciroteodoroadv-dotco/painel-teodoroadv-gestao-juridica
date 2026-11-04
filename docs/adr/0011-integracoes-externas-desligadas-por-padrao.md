# ADR 0011 — Integrações externas desligadas por padrão

**Status:** aceita em 04/11/2026; vale para DJEN, DataJud, índices do BCB e consulta de CEP.

**Contexto:** cada integração envia ou recebe dado de fora (OAB do advogado, número de processo, CEP de uma pessoa). Antes de ligar uma, o advogado precisa conferir
a fonte, a base legal e o que sai. Um painel que "já nasce falando com a internet" é um risco de privacidade e de resultado errado.

**Decisão:**

| Chave | O que liga | Padrão |
|---|---|---|
| `PAINEL_DJEN_ATIVO` | coleta de intimações do DJEN | desligada |
| `PAINEL_DATAJUD_ATIVO` (+ `PAINEL_DATAJUD_KEY` ou `_KEY_FILE`) | monitor do DataJud | desligada, sem chave |
| `PAINEL_INDICES_ATIVO` | índices do BCB (rodapé) | desligada |
| `PAINEL_CEP_ATIVO` | botão "Buscar CEP" (ViaCEP) | desligada |

1. Ausente ou `false` = **zero rede**: nem agendador nasce; o rodapé mostra "bloqueado — Coleta desligada neste ambiente."; o CEP responde "Consulta de CEP desligada neste ambiente.".
2. Valor inválido derruba a subida (não "liga sem querer").
3. **Como ligar:** uma por vez, no arquivo de ambiente da pilha, reiniciando o app; depois o **teste manual** da lista de `ACEITE-FINAL.md` (1 processo no monitor, 1 OAB no DJEN,
   chave do DataJud etc.). Se algo estranho aparecer, desliga-se a chave e reinicia-se: o histórico fica.
4. Os clientes são **isolados** (URL base https fixa na configuração, timeout, resposta limitada, sem redirecionamento a outro host, `User-Agent` genérico, sem dado do usuário).
5. Nenhum teste automático usa a rede (transporte falso; o teste falha se for chamado). Nenhuma chave real está em arquivo do projeto; as chaves não entram no backup nem no log.
6. O que sai: DJEN e DataJud, o que for cadastrado (OAB, número CNJ); índices, nada (só leitura); CEP, **somente os 8 dígitos**.

**Consequências:** a documentação oficial das fontes deve ser conferida antes de ligar (pendências em `PENDENCIAS.md`); índice e resultado de coleta nunca substituem a consulta à fonte oficial.
