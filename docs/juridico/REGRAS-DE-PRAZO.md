# Regras de prazo implementadas na Calculadora — para validação do advogado

**Versão:** Plano 7, 04/11/2026. **Situação:** NÃO VALIDADA. Enquanto o advogado não validar este documento e ligar
`PAINEL_CALCULADORA_REGRAS_VALIDADAS=true`, toda tela de resultado traz o aviso vermelho "Regras de cálculo ainda não
validadas pelo advogado — não use para prazo real." O resultado é sempre **A CONFERIR**.

**Princípio:** na dúvida entre dois vencimentos, o sistema mostra o **mais curto** e lista o outro como alternativa.
Nenhuma dúvida jurídica foi resolvida por conta do código: o que está duvidoso está marcado **[VERIFICAR]**.
**Emenda de 04/11/2026:** o painel **não aplica prazo em dobro** (advocacia privada); o resultado lista, entre os itens
não considerados, "prazo em dobro (CPC arts. 180, 183, 186 e 229): o painel não o aplica; se o caso o exigir, o
advogado ajusta manualmente".

## Como ler a coluna "Fonte consultada"

- **Lida** = texto da norma lido em planalto.gov.br nesta rodada (04/11/2026).
- **Não lida** = a norma é citada de memória/do plano; **o texto não foi conferido**. Confira antes de validar.

## Tabela de regras

| # | Regra | Norma | O que o sistema faz | Exemplo numérico | Fonte consultada | [VERIFICAR] | Validado por / data |
|---|---|---|---|---|---|---|---|
| 1 | Dias úteis | CPC art. 219 | Em prazo em dias do CPC, só conta dia útil. | 15 dias úteis, ver nº 6 | Lida | — | __ / __/__/____ |
| 2 | Não úteis | CPC art. 216 | Sábado, domingo e feriado não contam. Feriado = nacional (nº 12) + os do calendário do tribunal **com fonte**. | — | Lida (trecho) | Quais feriados forenses locais valem (cadastro) | __ / __/__/____ |
| 3 | Dia do começo e do vencimento | CPC art. 224 caput | Exclui o dia do começo, inclui o do vencimento. | — | Lida | — | __ / __/__/____ |
| 4 | Publicação no Diário (modo DJe) | CPC art. 224 §2º | Publicação = primeiro dia útil seguinte à disponibilização. | Disponib. sex 06/11/2026 → publicação seg 09/11 | Lida | — | __ / __/__/____ |
| 5 | Início da contagem | CPC art. 224 §3º | Começa no primeiro dia útil que seguir à publicação (DJe) ou à ciência (modo ciência). | Publicação 09/11 → dia 1 = ter 10/11 | Lida | Ciência direta: usada a mesma lógica (primeiro dia útil seguinte). | __ / __/__/____ |
| 6 | Vencimento CPC (caso 1) | arts. 219 e 224 | Dia 1 = 10/11; 20/11 (sex) é feriado; 15º dia útil = ter **01/12/2026**. | 10,11,12,13,16,17,18,19,23,24,25,26,27,30/11, 01/12 | — | — | __ / __/__/____ |
| 7 | Prorrogação (CPC) | CPC art. 224 §1º | Como só se contam dias úteis, o vencimento nunca cai em dia não útil. | — | Lida (trecho) | — | __ / __/__/____ |
| 8 | Suspensão 20/12 a 20/01 | CPC art. 220 | Esses dias não contam no prazo em dias do CPC. Disponib. 18/12/2026, 5 dias úteis: dia 1 = qui 21/01/2027, vencimento **27/01/2027**. | idem | Lida | A publicação feita dentro da suspensão: considerada o dia útil seguinte; o resultado é o mesmo (a contagem só começa em 21/01). | __ / __/__/____ |
| 9 | CPP — dias corridos | CPP art. 798 | Não computa o dia do começo, computa o do vencimento; **prorroga se vencer em domingo ou feriado**. Ciência qui 05/11 + 3 = dom 08/11 → **seg 09/11**. | idem | **Não lida** | (a) **Sábado não é prorrogado** (o sistema mostra o sábado e lista a segunda como alternativa); (b) a suspensão de 20/12 a 20/01 **não** é aplicada ao CPP; (c) no modo DJe o termo inicial pela publicação (Lei 11.419/2006 art. 4º §§3º-4º, **não lida**) — o sistema calcula também contando da disponibilização e **mostra o mais curto**. Disponib. sex 06/11 + 5: publicação → sáb 14/11; disponibilização → qua **11/11** (exibido). | __ / __/__/____ |
| 10 | Prazo material cível | Código Civil art. 132 | Exclui o dia do começo, inclui o do vencimento; prorroga se cair em domingo ou feriado. Evento sex 13/11 + 7 = sex 20/11 (feriado) → **seg 23/11**. O modo (Diário/ciência) é ignorado: conta da data do evento informada. | idem | **Não lida** | Sábado não prorrogado (como nº 9); recesso não aplicado. | __ / __/__/____ |
| 11 | Prazo material penal | Código Penal art. 10 | **Inclui** o dia do começo; não prorroga. Evento sex 06/11 + 5 = **ter 10/11**; evento qui 05/11 + 4 = dom 08/11 (sem prorrogar). | idem | **Não lida** | Recesso e suspensões por portaria não aplicados. | __ / __/__/____ |
| 12 | Feriados nacionais fixos | Lei 662/1949 e alterações | 01/01, 21/04, 01/05, 07/09, 12/10, 02/11, 15/11, 25/12. | — | **Não lida** | **Confirmar a norma de cada data** (21/04 e 02/11 por lei posterior? 12/10 por lei própria?). A coluna de fonte do calendário hoje diz "Lei 662/1949 (com alterações)". | __ / __/__/____ |
| 13 | 20 de novembro | Lei 14.759/2023 | Feriado nacional **a partir de 2024** (lei publicada em 22/12/2023; em 2023 o dia 20/11 conta como útil). | ciência ter 19/11/2024 + 1 útil → qui 21/11/2024; ciência sex 17/11/2023 + 1 → seg 20/11/2023 | Lida | — | __ / __/__/____ |
| 14 | Sexta-feira Santa | Lei 9.093/1995 art. 2º, II (citada) | Feriado nacional móvel (Páscoa − 2 dias). | 2027: Páscoa 28/03; Sexta Santa 26/03 | **Não lida** | **Confirmar a norma e o alcance.** | __ / __/__/____ |
| 15 | Não entram no nacional | — | Quarta-feira de Cinzas, Corpus Christi, Carnaval, Quarta e Quinta-feira Santa **só** pelo calendário do tribunal, com fonte. (O B tratava Cinzas e Corpus Christi como feriado sem fonte, o que atrasava o vencimento.) | Cinzas 10/02/2027: ciência 09/02 + 1 útil → 10/02 (nacional) ou 11/02 (tribunal que o cadastrou) | — | — | __ / __/__/____ |
| 16 | Justiça Federal | Lei 5.010/1966 art. 62 | **Não semeada.** O calendário "Justiça Federal" nasce vazio; cadastre as datas com a fonte. | — | **Não lida** | Semana Santa (quarta a domingo), Carnaval (segunda e terça), 11/08 e 08/12: conferir a lei antes de cadastrar. | __ / __/__/____ |
| 17 | Ponto facultativo | — | Não conta como não útil por padrão; só conta se um **advogado** marcar, com a fonte do tribunal. | — | — | — | __ / __/__/____ |
| 18 | Dia em dobro | CPC arts. 180, 183, 186, 229 | **Não aplicado** (emenda). Avisado em todo resultado. | — | — | — | — |

## O que o cálculo NÃO considera (texto fixo no resultado)

Termo inicial por juntada de AR/mandado, citação, carga ou audiência; consulta tácita (10 dias) da intimação eletrônica;
indisponibilidade do sistema do tribunal; suspensões por portaria não cadastradas; ponto facultativo não cadastrado;
feriado municipal ou estadual não cadastrado; prazo em dobro; prazos em horas, meses ou anos.

## Calendário do tribunal: como entra no cálculo

- Só dia **ativo**, com **fonte citada** e marcado "conta como não útil" entra. Dia inativado sai do cálculo, mas a linha
  permanece com o motivo.
- No prazo do CPC, entram todos os tipos (feriado, suspensão de prazo, recesso, sem expediente). Nos demais prazos
  (CPP e materiais) entra só o tipo **feriado** — suspensão por portaria não é aplicada a eles **[VERIFICAR]**.
- Sem calendário do tribunal (ou sem dia cadastrado no ano): o resultado diz "Sem calendário verificado deste tribunal:
  considerados só os feriados nacionais".

## Fontes legais consultadas nesta rodada

CPC (Lei 13.105/2015), arts. 216, 219, 220, 224 e cabeça do 231: lidos em planalto.gov.br em 04/11/2026
(a leitura devolveu trechos resumidos de alguns parágrafos). Lei 14.759/2023: lida. **Não consultadas:** CPC arts. 180,
183, 186 e 229 (dobro, fora do escopo); CPP art. 798; Código Civil art. 132; Código Penal art. 10; Lei 5.010/1966 art. 62;
Lei 662/1949; Lei 9.093/1995; Lei 11.419/2006.
