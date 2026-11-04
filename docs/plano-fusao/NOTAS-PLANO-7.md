# Notas de decisão — Plano 7 (04/11/2026)

Onde o plano foi ambíguo ou duvidoso foi escolhido o caminho conservador (vencimento mais curto, nada resolvido por conta própria).

1. **Emenda do advogado (sem prazo em dobro)**: sem campo "Prazo em dobro" no drawer (só o extra "Calendário"), sem parâmetro `Dobro`, sem `dobro.go`,
   sem o teste dourado nº 7. Todo resultado lista, em "Não considerado": "prazo em dobro (CPC arts. 180, 183, 186 e 229): o painel não o aplica; se o caso
   o exigir, o advogado ajusta manualmente". Teste `TestSemDobro_ListadoComoNaoConsiderado` e verificação de que o drawer não contém "em dobro".
2. **Fontes legais**: planalto.gov.br abriu (CPC e Lei 14.759/2023). Lidos: CPC arts. 216, 219, 220, 224 e cabeça do 231 (a ferramenta resumiu alguns
   parágrafos); Lei 14.759/2023 (publicada em 22/12/2023). **Não consultados** (pendência, marcados "Não lida" em `REGRAS-DE-PRAZO.md`): CPP 798, CC 132,
   CP 10, Lei 5.010/66 art. 62, Lei 662/49, Lei 9.093/95, Lei 11.419/06. Nenhum bloqueio de acesso; foi economia de custo.
3. **Sábado e prazo corrido (CPP/material cível)**: o plano manda "prorrogar se vencer em dia não útil"; o princípio "nunca errar a favor de prazo mais longo"
   manda mostrar o mais curto. Conciliado: prorroga **domingo e feriado** (o texto citado do CPP 798 §3º; CC 132 §1º cita feriado); **sábado não é
   prorrogado**, aparece como vencimento, com a segunda-feira nas alternativas e `[VERIFICAR]`. Depois de uma prorrogação, o dia final tem de ser útil.
4. **CPP no modo DJe**: dois termos possíveis (publicação no dia útil seguinte, ou a própria disponibilização). O sistema calcula os dois e **exibe o mais
   curto**, listando o outro; `[VERIFICAR]` (Lei 11.419/06 não lida).
5. **Prazo material**: o modo (DJe/ciência) é ignorado; conta da data do evento informada (termo mais cedo = vencimento mais curto), dito nas premissas.
6. **Recesso (20/12–20/01)**: aplicado só no prazo em dias úteis do CPC (art. 220, lido). No CPP e nos materiais **não** é aplicado, com `[VERIFICAR]`.
7. **Calendário do tribunal nos prazos que não são do CPC**: só entra o tipo `feriado`; suspensão por portaria/recesso do tribunal não é aplicada a eles
   (`[VERIFICAR]`). No CPC entram todos os tipos com "conta como não útil".
8. **Nacional sempre aplicado no cálculo** (código `calculo.Nacional`: fixos, 20/11 a partir de 2024 e Sexta-feira Santa por Páscoa). A tabela
   `calendario_dias` também recebe a semente dos fixos (ano corrente e 2 seguintes, com a norma), para ficarem **visíveis** na página; o cálculo não depende
   dela. O calendário "NACIONAL" não aparece como escolha na calculadora (já é aplicado).
9. **Citações de norma na semente**: "Lei 662/1949 (com alterações)" para todos os fixos (como pede o plano) e "Lei 9.093/1995, art. 2º, II [VERIFICAR a norma]"
   para a Sexta-feira Santa. Norma de cada data (21/04, 12/10, 02/11) **a conferir** (`REGRAS-DE-PRAZO.md`, linha 12).
10. **Semente da Justiça Federal (Lei 5.010/66)**: só na demonstração (`TRF-DEMO`), marcada `[VERIFICAR]` e com `conta_como_nao_util=false` (a lei não foi
    lida, então não afeta cálculo). O calendário "Justiça Federal" de produção nasce vazio. `TRIBUNAL DE TESTE` (fictício) tem 2 dias com efeito.
11. **Unicidade**: índice único **parcial** (só dias ativos) em `(calendario_id, data, tipo)`, para permitir recadastrar o dia correto depois de inativar o
    errado (o plano pedia unicidade simples, o que impediria a correção).
12. **Gatilho contra DELETE** em `calendarios` e `calendario_dias` (além do REVOKE ao `painel_app`): "nada se apaga" vale até para o dono do banco.
    A migração 0014 só reverte sem dados de pessoas (semente tem `created_by` nulo).
13. **Quem verifica**: quem grava o dia é registrado como `verificado_por`/`verificado_em`. Ponto facultativo "conta como não útil" só por **advogado**
    (administrador recebe recusa). Dia da semente nasce sem verificador e aparece com `[VERIFICAR]`.
14. **"Criar prazo com este cálculo"**: link para `/prazos/novo` preenchido (data, fundamento, fonte). Acrescentada a fonte **`calculadora`** ao `CHECK` de
    `prazos.fonte` (na própria 0014) e o prazo vindo dela **nasce A CONFERIR mesmo quando o advogado salva** (e volta a "a conferir" se editado): só vale após
    "Confirmar". Fundamento truncado em 300 caracteres.
15. **Erros**: 400 com o texto do motivo, em bloco vermelho (`.form-block-erro` em `compat.css`, no lugar do `style=` do B). Como o htmx não troca 4xx por
    padrão, `web/static/calculadora.js` libera a troca só para `#resultadoPrazo`. `drawer.js` ganhou `htmx.process(drawer)` para ativar o `hx-post` do
    fragmento inserido.
16. **Botão da sidebar**: novo `AddNavAcao` (item que abre drawer; `<a data-drawer>`), na ordem alfabética do menu, que coloca "Calculadora de prazo" logo
    depois de "Audiências" (como no B).
17. **Auditoria**: `prazo.calculado` (regra, modo, dias, "local"/"nacional"; sem dado de cliente), `calendario.criado`, `calendario.dia_criado`,
    `calendario.dia_inativado`, `calendario.semeado`. Chaves novas na lista permitida de metadados (`regra`, `modo`, `dias`, `calendario`, `tipo`, `data`).
18. **`PAINEL_CALCULADORA_REGRAS_VALIDADAS`**: ausente = false; valor inválido derruba a subida. Não foi definida em nenhum arquivo de ambiente.
19. **Número de dias**: teto de 3650 (evita laço por digitação errada); fora disso "Número de dias inválido.".
20. **TDD**: vermelho observado no pacote de cálculo (15 testes falhando contra o esqueleto), na migração (tabela ausente), nos testes de tela (rotas
    desligadas: 4 falhas) e na configuração. **Exceções honestas**: calendário/semente (teste escrito antes; o vermelho foi falha de compilação por
    símbolos ausentes) e semente demo (teste e código no mesmo passo). Casos dourados calculados à mão nos comentários dos testes.
21. **Testes antigos ajustados**: `TestMigracao0012PreservaIntimacoesDescartadas` (agora desce 3 migrações); listas de `TRUNCATE` do harness e de coletas
    incluem as tabelas novas; texto de aviso de `prazos/lista.html` e `form.html`.
22. **Conferência manual no Chrome**: teste gated `TestServirParaConferenciaManual` (só com `PAINEL_SERVIR_MANUAL=1`) sobe o v2 com banco descartável e
    proxy para `web/static`. Sem o 18080, sem credencial real.
23. **Paridade**: único ajuste de CSS por comparação de estilo: `input[type=number]` do B é nativo (178 px, borda 2px inset); o `compat.css` estilizava
    todo input, e foi neutralizado como já era feito para senha/arquivo.
