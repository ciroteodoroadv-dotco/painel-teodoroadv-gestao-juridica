# Roteiro de aceite final — para o advogado

Marque cada caixa quando conferir. Onde estiver escrito "se algo estranho aparecer", pare, anote a tela e o que viu, e me chame: nada aqui exige mexer em arquivo.
Ninguém, nem eu, entra no seu painel real sem você: este roteiro é para **você** fazer, com o seu login.

## 1. Primeiro login
- [ ] Entre com o e-mail e a **senha provisória** que o administrador criou. O painel pede para trocar a senha (use uma frase longa, só sua).
- [ ] Cadastre o **segundo fator** (aplicativo autenticador no celular, ou chave de segurança). Confira que o código de 6 dígitos funciona.
- [ ] Anote os **códigos de recuperação** (aparecem uma vez) e guarde em lugar seguro, fora do computador.
- [ ] Saia (menu "Sair") e entre de novo para ter certeza de que funciona.

## 2. Conferir cada tela contra o painel B
Abra o painel B ao lado. Em cada tela, olhe: título e subtítulo, botão principal (cor, texto), colunas da lista, ordem dos campos do formulário, textos dos avisos.
- [ ] **Agenda** (página inicial): faixa de avisos, compromissos, aulas, audiências, diligências, intimações; abrir uma aula; criar e cancelar (com motivo).
- [ ] **Atendimentos**: "+ Novo cliente" abre por cima do formulário; vincular a processo.
- [ ] **Audiências**: criar; cancelar (pede sua senha e o motivo).
- [ ] **Diligências**: criar e cancelar com motivo.
- [ ] **Pessoas**: cadastrar pessoa física e jurídica (máscaras de CPF/CNPJ/CEP; CPF inválido é recusado).
- [ ] **Processos**: cadastrar; cadastrar um **duplicado** (o painel exige a justificativa).
- [ ] **Tarefas**: criar tarefa com **prazo fatal** (aparece o selo A CONFERIR; só você confirma); mover para **Cancelada** (não existe excluir).
- [ ] **Calculadora de prazo** (botão no menu): calcule um prazo e veja o resultado **A CONFERIR** e a lista do que não foi considerado; use "Criar prazo com este cálculo".
- [ ] **Rodapé**: faixa preta de índices e bloco "Estado das coletas" (com tudo desligado devem dizer "bloqueado — Coleta desligada neste ambiente.").
- [ ] Diferenças que você achar: anote a tela e o que deveria estar diferente.

## 3. Ligar as integrações, uma por vez (cada uma só depois do teste manual)
Peça ao administrador para mudar **uma** chave, reiniciar o painel, e faça o teste. Se algo estranho aparecer, a chave volta a `false` e o painel reinicia: nada se perde.
- [ ] **Índices do Banco Central** (`PAINEL_INDICES_ATIVO`): em "Índices", clique "Atualizar índices agora"; compare os 6 números com o site do Banco Central (`docs/operacao/INDICES-E-CEP.md`). Confirme o significado de cada série (principalmente salário mínimo e dólar).
- [ ] **Buscar CEP** (`PAINEL_CEP_ATIVO`): numa pessoa de teste, digite um CEP que você conhece e confira o endereço.
- [ ] **DJEN** (`PAINEL_DJEN_ATIVO`): cadastre **1 OAB** (a sua, com a base da autorização) e rode "Executar coleta agora"; confira as intimações no site do DJEN.
- [ ] **DataJud** (`PAINEL_DATAJUD_ATIVO` + chave `PAINEL_DATAJUD_KEY`): coloque **1 processo de cliente** no monitor e confira as movimentações no site do tribunal. A chave pública do DataJud você obtém no site do CNJ.
- [ ] Depois de cada uma: olhe o bloco "Estado das coletas" e confirme "última coleta concluída".

## 4. Validar as regras de prazo
- [ ] Leia `docs/juridico/REGRAS-DE-PRAZO.md` (Plano 7). Para cada linha marcada `[VERIFICAR]` ou "Não lida" decida: **confirmar**, **corrigir** ou **manter como está**.
- [ ] Só quando estiver satisfeito, peça para ligar `PAINEL_CALCULADORA_REGRAS_VALIDADAS=true` (ela só tira o aviso "regras não validadas"; o resultado continua A CONFERIR).
- [ ] Confira o **calendário forense**: cadastre os feriados/pontos facultativos do seu tribunal **com a fonte** (portaria).

## 5. Decisões que dependem de você
- [ ] Campo **"Nome do arquivo da ficha"** (Pessoas): o painel omitiu; manter assim ou incluir?
- [ ] **Tarefa de monitoramento** dos processos que já existiam antes: criar retroativamente (preenchimento) ou só dos novos?
- [ ] **Mensagens de recarga** (aviso quando a coleta ficar parada): texto e prazo do aviso.
- [ ] **Semana Santa/Carnaval** no calendário da Justiça Federal: quais dias contam como não úteis (lei não lida; está `[VERIFICAR]`).
- [ ] As demais decisões em `PENDENCIAS.md` (tabela com quem decide e o risco).

## 6. A chave de backup (o passo mais importante)
- [ ] A **chave privada** do backup (arquivo `identity`) fica **fora** do computador e fora do OneDrive da pasta do painel: em um pendrive guardado e numa cópia impressa em lugar seguro. Sem ela o backup não abre.
- [ ] Faça um backup agora e rode o **teste de restauração** (`docs/operacao/COMECAR-USO-REAL.md`, passo 2 e 3). Deve terminar com "cópia restaurada íntegra e utilizável".
- [ ] Guarde também cópia da pasta de segredos (`deploy/secrets/`): sem `field_key` os dados cifrados não abrem.

## 7. Fim
- [ ] Nada do que apareceu fugiu do que você esperava. Assine/anote a data do aceite: ____ / ____ / ______
