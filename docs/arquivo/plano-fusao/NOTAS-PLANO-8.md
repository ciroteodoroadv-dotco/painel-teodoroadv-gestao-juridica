# Notas de decisão — Plano 8 (04/11/2026)

Onde o plano foi ambíguo ou duvidoso foi escolhido o caminho conservador.

## Tarefas 1 e 2 (índices e CEP)

1. **Documentação oficial.** A página de dados abertos do BCB para a série 432 respondeu **HTTP 404** ao WebFetch (a URL era um palpite); não houve
   tentativa de contornar. Foi lida **uma** resposta real do endpoint do plano (`.../bcdata.sgs.433/dados/ultimos/1?formato=json`, só leitura, fora dos testes),
   que confirmou o formato `[{"data":"dd/mm/aaaa","valor":"0.82"}]` (ponto decimal, data da **primeira dia do mês de referência** para séries mensais). A página do
   ViaCEP abriu: URL `/ws/{8 dígitos}/json/`; CEP inexistente devolve `"erro"` (a documentação diz `"true"` em texto; o cliente aceita texto **ou** booleano);
   CEP malformado devolve HTTP 400. **Não conferido na fonte:** o significado exato de cada série (ex.: SGS 1619 é anual ou mensal? SGS 1 é PTAX de venda ou
   de compra?). O rodapé mostra o rótulo do B, e o significado fica como **pendência de conferência** (ver `INDICES-E-CEP.md`).
2. **Validade de 40 dias.** Séries mensais ficam com referência no dia 1 do mês; o último IPCA publicado passa de 40 dias pouco antes de sair o seguinte.
   Se a série 1619 (salário mínimo) for anual, ela ficará "desatualizada" boa parte do ano — o comportamento é o do plano (conservador: mostra a marca em vez de
   esconder a data). O advogado pode ajustar `PAINEL_INDICES_VALIDADE_DIAS`.
3. **"Desatualizado"** = referência mais velha que a validade **ou** a última coleta daquela série falhou (coluna `falhou`, migração 0015). O valor antigo e a data antiga
   ficam visíveis; nunca entra valor novo inventado. A marca aparece no item (`<em class="idxt-velho">`), no `title` e no fim da linha de estado.
4. **Rodapé vem do serviço.** `Auth.SetTickerSource` + `PageData.Ticker`; sem o domínio de índices o layout mostra "Índices econômicos: não sincronizado" (como antes).
   O JS `app.js` passou a atualizar **todo** elemento `[data-refresh]` (cada um com seu `data-intervalo` e conferência de `X-Painel-Fragmento`); o rodapé de coletas
   continua a 60 s, o de índices a 30 s. Falha de leitura mantém o último texto do ticker (como o B).
5. **Página do administrador** (`/admin/indices`, item "Índices" do segundo grupo; permissão `coleta:gerir` = só administrador): estado, tabela e botão
   "Atualizar índices agora". Desligado, o botão não aparece e o POST responde 400 "Coleta desligada neste ambiente." sem rede.
6. **Agendador**: 1 rodada a cada `PAINEL_INDICES_INTERVALO` (padrão 24 h; mínimo 1 h), primeira rodada 3 min após a subida. Pausa de 1 s entre séries.
7. **CEP**: só o **advogado** tem `pessoas:escrever` (RBAC do v2); administrador e assistente recebem 403 na consulta. Limite padrão: 20 consultas por minuto por
   usuário (só as consultas válidas e com integração ligada contam); cache em memória de 10 min por CEP (até 500 entradas). Código HTTP: 400 (CEP inválido),
   404 (não encontrado), 429 (limite), 502 (fonte indisponível), 503 (desligado). Nenhum dos textos de erro leva o CEP, e o handler não escreve log.
8. **Confirmação sobreposta** do B (`confirmarEmDrawer` com `sobrepor`) foi reimplementada em `cep.js` (`App.confirmarSobreposto`) sobre a pilha do `drawer.js`.
   Igual ao B: cancelar a confirmação ainda preenche os campos **vazios** e avisa "Endereço preenchido nos campos vazios.".
9. **TDD.** Vermelho observado: cliente BCB (símbolos ausentes), ticker/serviço (idem), CEP em tela (`WithCEP` ausente). **Exceções honestas** (teste e código escritos
   no mesmo passo, sem execução vermelha isolada): `config/indices.go` (`TestIndices*`) e `internal/cep/cliente.go` (`cliente_test.go`).
10. **Testes antigos ajustados por mudança de regra:** `migracoes_0012_test` (a reversão agora desce quatro migrações), `TestRodapeEstadoDasColetas` (atributos novos no `<section>`).
11. **Verificação no Chrome** (pilha de teste na porta 18090, banco descartável, transportes falsos de BCB e ViaCEP, usuário fictício): ticker relido pelo `data-refresh` (texto
    marcado trocado pelo servidor em ~30 s, 1 requisição); botão Buscar CEP: CEP curto recusado, campos vazios preenchidos, UF em maiúsculas, fonte exibida, botão
    desabilitado durante a consulta, confirmação sobreposta com Cancelar (formulário intacto) e Substituir. **Não verificado:** consulta real a BCB/ViaCEP (proibida).
    Observação: uma aba anterior do grupo do navegador trocava de página sozinha (origem desconhecida, não era o painel); foi fechada e a verificação refeita em aba nova.
