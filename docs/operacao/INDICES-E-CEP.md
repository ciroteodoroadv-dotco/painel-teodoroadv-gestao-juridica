# Índices do Banco Central e busca de CEP (guia para leigos)

Duas funções do painel conversam com a internet. As duas vêm **desligadas**. Este guia explica o que fazem, como ligar e o que conferir.

## 1. Faixa de índices (a faixa preta no pé da tela)

**O que mostra.** Selic (meta do Copom), IPCA, INPC, IGP-M, Dólar (PTAX) e Salário mínimo, com o valor mais recente publicado pelo Banco Central (sistema SGS) e a **data de referência**.
Passe o mouse sobre um valor para ver a data e a unidade. No fim da faixa: "Ref.: dd/mm/aaaa · sincronizado".

**Regras que o painel segue.**
- Nunca mostra valor "de cabeça": só o que veio do Banco Central, sempre com data.
- Se um índice não for atualizado, **o valor antigo continua aparecendo**, mas marcado **desatualizado**. O mesmo vale quando a data de referência é mais velha que 40 dias (`PAINEL_INDICES_VALIDADE_DIAS`).
- Sem dados, a faixa diz o motivo: "Índices econômicos: não sincronizado", "falha" ou "bloqueado — Coleta desligada neste ambiente.".
- A faixa se renova sozinha a cada 30 segundos, mas **só relê o que já está gravado**; ela nunca vai à internet. Quem vai é a coleta, 1 vez por dia.
- **O índice é informativo.** Para cálculo judicial, confira sempre a fonte oficial.

**Onde cada série vem** (SGS): 432 Selic meta (% ao ano), 433 IPCA (variação mensal, %), 188 INPC (variação mensal, %), 189 IGP-M (variação mensal, %), 1 dólar PTAX (R$), 1619 salário mínimo (R$).
**A conferir antes de confiar:** o significado exato de cada série (por exemplo, se 1619 é mensal ou anual e se a série 1 é a PTAX de venda). O painel mostra o nome que o painel B usava.

**Como ligar (uma vez, depois de conferir):** no arquivo de ambiente, `PAINEL_INDICES_ATIVO=true`; reinicie. Opcionais: `PAINEL_INDICES_INTERVALO` (padrão 24h, mínimo 1h),
`PAINEL_INDICES_VALIDADE_DIAS` (padrão 40), `PAINEL_INDICES_URL` (padrão `https://api.bcb.gov.br`, tem de ser https).
**Conferência manual:** entre como administrador em **Índices** (menu, segundo grupo), clique **Atualizar índices agora** e compare os números com o site do Banco Central. A linha do rodapé "Estado das coletas" deve mostrar "Índices (BCB): última coleta concluída".

## 2. Botão "Buscar CEP" (cadastro de pessoa)

**O que faz.** Você digita o CEP e clica **Buscar CEP**: o painel pergunta ao ViaCEP (serviço público, **não oficial**) e preenche logradouro, bairro, cidade e UF.
Mostra "Fonte consultada: ViaCEP (público, não oficial)". Se algum desses campos já tem texto, o painel pergunta (janela sobreposta) se deve **substituir**; se você cancelar, só os campos vazios são preenchidos.

**Privacidade.** Só os 8 números do CEP saem do painel; nada de nome, CPF ou outro dado. O navegador nunca fala com a internet: quem consulta é o painel. O CEP não vai para o registro de funcionamento.
Há limite de consultas por minuto por usuário e uma memória curta (10 minutos) para o mesmo CEP. **Só o advogado** usa o botão (é quem pode cadastrar pessoas).

**Mensagens.** "Informe um CEP com oito dígitos." · "CEP não encontrado." · "Não foi possível consultar o CEP agora…" (nada é preenchido) · "Consulta de CEP desligada neste ambiente." · "Muitas consultas de CEP em pouco tempo. Aguarde um minuto.".
**Confira sempre o endereço** com o documento da pessoa: a base é pública e pode estar desatualizada.

**Como ligar:** `PAINEL_CEP_ATIVO=true` e reinicie (`PAINEL_CEP_URL` opcional, https). **Conferência manual:** cadastre uma pessoa de teste, digite um CEP que você conhece e compare o resultado.
