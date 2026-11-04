---
projeto: 2026-tec-fusao-paineis
tipo: inventario-somente-leitura
data: 2026-11-03
status: PARCIAL — faltam domínios/prazo (agente ainda rodando)
origem: D:\OneDrive\Desktop\painel-teodoro-entrega-1-1-2026-10-30\painel-teodoro (NÃO alterada)
---

# Inventário do painel B — parcial (segurança/infra + frontend)

Feito por leitura (agentes `analista-sonnet`), sem executar nada do B e sem abrir `dados/`, `painel.exe` ou `.env`.
Marcas: [Certeza] com arquivo:linha; [VERIFICAR] = não conferido. Caminhos abaixo são relativos à raiz do B.

## 1. Segurança e infraestrutura — B × v2

**Veredito:** B é painel de máquina única (localhost) com segurança "senha única + cookie assinado". Em quase todos os itens o v2 é melhor.

| Item | B | Veredito |
|---|---|---|
| Login | senha única em texto claro (`PAINEL_ADMIN_PASSWORD`), sem usuário/papel/2FA/limite de tentativas (`seguranca/sessao.go:47`) | v2 melhor |
| Sessão | cookie HMAC stateless 8 h, sem revogação (logout só apaga cookie), sem `Secure`/`__Host-`; se faltar `PAINEL_SESSION_SECRET`, a senha vira a chave (`sessao.go:33-35,56-67`) | v2 melhor |
| **GET/HEAD sem sessão** | todo GET/HEAD passa livre, inclusive dados do escritório e SSE (`sessao.go:96`) | **grave fora de localhost** |
| CSRF | só `SameSite=Strict`, sem token | v2 melhor |
| CSP/cabeçalhos | nenhum (zero ocorrências de CSP, X-Frame, nosniff, CORS) | v2 melhor |
| RBAC / ator | inexistente; `criadoPor` vem do corpo da requisição (`agenda/http.go:241`) | v2 melhor |
| Auditoria | inexistente (só `log.Printf`) | v2 melhor |
| Dados em repouso | `.md`/`.json` em claro dentro do OneDrive; CPF cifrado? [VERIFICAR] | v2 melhor |
| Backup | pontual (só na implantação Linux), sem cifra, restauração não testada; `.bak` da agenda nunca podado | v2 melhor |
| Limite de corpo | sem `MaxBytesReader` (upload 32 MB só) | v2 melhor |
| Path traversal | **bem tratado**: `ficha.NomeArquivoSeguro` (`ficha/validacao.go:11-19`), regex em processo novo, upload com `O_EXCL`+`Rel` (`tarefas/http.go:478-515`), `exec.Command` sem shell | B bom |
| Agenda corrompida | na versão atual **não sobrescreve** (`agenda.go:56-72,133-135`; 500 "original preservado") — a auditoria externa C4 pode referir versão anterior | B ok (resíduo: `gravarArquivo` com `os.WriteFile`, hoje inalcançável, `main.go:95-98`) |
| Escrita de ficha | temp de nome fixo + rename, sem fsync, sem lock entre processos (`ficha/io.go:9-15`); agenda é a melhor (`agenda.go:132-173`) | misto |

**Onde B é melhor / tem ideia aproveitável:** (1) bind só `localhost` fixo no código (`main.go:436`); (2) **zero dependências** (`go.mod` sem `require`, `go 1.21` — fora de suporte, toolchain real [VERIFICAR]); (3) **sandbox systemd** (usuário sem privilégio, `NoNewPrivileges`, `ProtectSystem=strict`, `ReadWritePaths` mínimo, `LoadCredential`, recusa rodar como root); (4) implantação Linux com `go test` antes de instalar, `GOPROXY=off`, `fontes.sha256`, backup com SHA-256; (5) token próprio do WhatsApp isolado da senha, fila desligada por padrão (`PAINEL_WHATSAPP_ENABLED=false`); (6) agenda com `.bak` a cada gravação.

**Achados do B (para a auditoria G2):** B-01 ALTA GET sem sessão · B-02 ALTA senha única sem hash/2FA/limite · B-03 MÉDIA senha como chave HMAC · B-04 MÉDIA cookie sem Secure/`__Host-`, sem revogação · B-05 MÉDIA sem CSP/CSRF token · B-06 MÉDIA corpo sem limite · B-07 MÉDIA sem RBAC · B-08 MÉDIA sem trilha · B-09 MÉDIA `iniciar.bat` (`iex` do próprio arquivo, `Unblock-File` em massa, `-ExecutionPolicy Bypass`) e `atualizar-1-3.bat` (robocopy por cima sem verificar hash) · B-10 MÉDIA 3 `.sh` (~380 KB, base64, root) opacos — **não decodificados** · B-11 MÉDIA compose `atendai/evolution-api:latest` + `8080:8080` (só se ligado) · B-12..B-17 BAIXAS (temp fixo, `gravarArquivo`, backup, máscara WhatsApp só CPF/CNPJ, go 1.21, PDF só por extensão).

**Rede externa do B:** viacep, api.bcb.gov.br, comunicaapi.pje.jus.br (DJEN), api-publica.datajud.cnj.jus.br. DJEN consultado por OAB/UF/nome do advogado (`api_server.go:33-37`).

**Testes:** 151 funções `Test*` em 43 arquivos (contagem por grep, não executado); só 2 de sessão; sem teste dedicado de traversal/CSRF/CORS.

**Lacunas [VERIFICAR]:** conteúdo dos GET de pessoas (CPF?); front-matter quebrado (`ficha/frontmatter.go`); payloads dos `.sh`; onde ficam as senhas no Windows; agente externo que consome a fila do WhatsApp; `go test` real.

## 2. Frontend e design (a preservar)

**Arquitetura:** HTML estático + 18 scripts clássicos em ordem numérica (`referencia/painel-b/js/00..99`) + `web/shared/client.js`; sem framework, build, CDN ou fonte externa. Cada tela é `App.viewX()` que devolve string HTML via `innerHTML`; re-render total; sem router/hash (F5 volta à Agenda). API por `fetch`, SSE em `/api/events` (só o nome da entidade), polling de índices (30 s) e sincronização (60 s). `referencia/painel-b/js/00-fetch-auth.js` repete a escrita após re-login em 401. Caminhos absolutos (`/css`, `/js`): `file://` não funciona.

**Telas (abas):** Agenda (padrão; inclui Tarefas, Intimações, Audiências, Diligências, Reuniões, Ligações, Aulas), Atendimentos, Audiências, Pessoas, Processos, Tarefas; Calculadora de prazo (drawer); faixa de índices no rodapé; painel "Estado das coletas". Não há grade de calendário, atalhos de teclado nem ícones.

**Tokens (`referencia/painel-b/css/painel.css`):** `--ink #000`, `--ink-2 #1A1A1A`, `--paper #EEF0F0`, `--panel #FFF`, `--wine #7C2D3B`, `--wine-dark #5E2029`, `--brass #A9822F`, `--text #1E2126`, `--text-muted #5B6470`, `--border #D7DBDD`, `--border-strong #B7BEC2`; fundos de estado danger `#FBE9E9` / warn `#F7EFDD` / ok `#E9E9E9` / neutral `#E7E8E6`; raio 3px; fonte Arial (decisão do advogado, comentada em `painel.css:116-119`); sidebar 230px; drawer 480px (empurra a página, sem véu, **sem ESC nem clique fora — pedido do advogado**); sombras só em drawer/toast/aba ativa; breakpoints 780/781px; sem tema escuro. Logo `referencia/painel-b/assets/logo-lateral-escuro.png` (1200×1200, 108 KB, exibido a 150px).

**Pontos fortes a preservar:** sidebar preta com marca latão e logo circular; vinho como única cor de ação; pílulas de categoria com contador; drawer lateral; tabelas densas zebradas; faixa de índices; estados vazios/erro explícitos, nunca dado presumido; selos sempre com texto.

**CSP estrita — o que impede hoje:** `index.html` limpo; mas ~109 handlers `on*=` e ~35 `style="` dentro de template strings (módulos 06, 07, 08, 09, 10, 11, 12, 14). Cores soltas fora dos tokens (`#c0392b` em `12-prazo.js`, etc.). Solução: delegação de eventos com `data-*` e classes CSS — também elimina a classe de XSS abaixo.

**XSS (leitura, não explorado):** nomes de arquivo/ids interpolados sem escape em `onclick` (`09-processos.js:36-41,260-261`; `11-agenda.js:112,132,151,412,485,633`; `08-tarefas.js:94,97`; `14-pessoas.js:189`); `escapeHTML` insuficiente dentro de literal JS (`11-agenda.js:177-178`; `14-pessoas.js:53,146`); `inlineArg` correto existe (`02-util.js:38-40`) mas não é uniforme. Risco prático [Provável] baixo se o servidor gerar sempre os nomes; inaceitável se o backend for outro.

**Acessibilidade [Provável, testar no navegador]:** `all:unset` pode apagar o anel de foco em nav/`.btn`/`.acc-head`; `<label>` sem `for`; drawer sem trava de foco; ticker ~3,6:1 de contraste; tabelas com `min-width` 880–1020px (rolagem lateral em mobile).

**Endpoints chamados pelo front (contrato que um backend novo teria de honrar ou adaptar):** GET `/api/{processos,pessoas,tarefas,atendimentos,agenda,audiencias,diligencias,indices,intimacoes}`; POST `/api/auth/login`; tarefas: `nova`, `mover-estado`, `editar`, `excluir` (header `X-Painel-Admin-Password`), `documentos` (multipart PDF); processos: `novo`, `editar`, `mover-estado`, `andamento`, `financeiro`, `proxima-providencia`, `excluir`; atendimentos: `novo`, `editar`; agenda/diligências/audiências: `nova/novo`, `editar`, `cancelar`; `PATCH /api/intimacoes/{hash}/status`; `POST /api/prazos/calcular`; pessoas: `nova`, `editar`; GET `/api/cep/{cep}`, `/api/sincronizacao/status`, `/api/events`. O contrato é acoplado a fichas `.md` (`arquivo="estado/nome.md"`, wikilinks `[[pessoas/x]]`, códigos `PES./ATE./PRO.`, `referencia/painel-b/js/03-api-mapa.js`).

**Prazos na UI do B:** atraso só nas tarefas da Agenda (`computeTarefaStatus`); "Prazo fatal" em selo vermelho; calculadora (CPC úteis / CPP corridos, DJe ou ciência) calculada no **backend**; Intimações em triagem (confirmar/descartar), sem contagem de dias nem urgência; sem tela própria de monitoramento.

**Fácil de portar:** `painel.css`, logo, tokens, padrões de acordeão/drawer/toast/selos/tabela/ticker, máscaras/validadores de `02-util.js`, listas de `05-dominio.js`, esqueleto de `index.html`. **Acoplado ao backend:** `03-api-mapa.js`, caminhos por pasta/estado, SSE, autenticação por senha única, envelopes `status/origem/mensagem`, payloads com mistura camelCase/snake_case.

## 3. Domínios e prazos do B (leitura estática; nada executado)

Lidos por inteiro: motorprazo, intimacoes, servicos/intimacoes+sincronizacao, monitorprocessual, consultaprocessual, cnj, indices, sincronizador, decisoes, cep, configuracao, whatsapp, agenda, rotina. **Só amostragem:** `http.go` de tarefas/processos/atendimentos/pessoas/audiencias/diligencias e o pacote `ficha` (2.204 linhas).

**Confronto com a auditoria externa (C1–C4) no código desta pasta:** C1 (tabela CNJ→TR) — **não confirmado**; `numero.go:40-68` coincide com a Res. CNJ 65 [Provável, conferido de memória, falta abrir o Anexo VI]; há teste `TestLoteC1TabelaCNJ`, mas usa a mesma lista (não é verificação independente). C2 ("visto" sem registro) — **corrigido** (`monitor.go:300-315`; teste `TestLoteC2C3RetomadaSemPerderMovimento`). C4 (agenda) — corrigido (`agenda.go:56-72,133-135`). A pasta parece já conter o "Lote 1"; a versão auditada não está aqui [VERIFICAR contra o ZIP].

**Motor de prazo (`motorprazo`) — NÃO reaproveitar como motor legal:**
- Calendário único nacional embutido (`motorprazo.go:17-47`), sem tribunal/comarca/município; sem prazo em dobro (arts. 180/183/186/229), sem termo inicial por juntada/citação/carga, sem indisponibilidade do sistema.
- **Erra contra o advogado:** trata Quarta-feira de Cinzas e Corpus Christi como feriado em todo o país, sem fonte (`:44,:46`) → vencimento pode sair **depois** do real. Semana Santa incompleta: só a sexta (`:45`); a Lei 5.010/66 art. 62, II cobre quarta a domingo.
- Tela mostra "Prazo fatal: data" sem "a conferir" (`referencia/painel-b/js/12-prazo.js:78`).
- Tarefas criadas a partir de intimação levam `prazo_fatal: true` com data só "próximo dia de semana" (`servicos/intimacoes.go:321,339`); `DataBase` = disponibilização, não publicação do art. 224.
- Estrutura aproveitável como ideia: algoritmo da Páscoa (conferido à mão 2026/2027), recesso, prorrogação para dia útil.

**Reaproveitar (conceito/lógica, com testes próprios no v2):** coletor DJEN (`intimacoes.go`: paginação até 100 págs, teto 10.000, 429, página repetida, sobreposição de 7 dias); dedupe por hash e tarefa idempotente (`servicos/intimacoes.go`); estado por processo e "visto só após registro" (`monitor.go`); contrato uniforme de resultado de sincronização (`sincronizador`); `agenda.gravarComBackup`; índices BCB (`indices.go`); filtro de segredo de justiça por OAB do sócio (`monitor.go:169-199`).

**Não reaproveitar:** `cnj` e `consultaprocessual` (cascas sem função); `decisoes` como classificador de recurso (palavras literais); máscara WhatsApp de CPF/CNPJ como anonimização; `cep` (promete "Correios/contingência" que não existe); frontmatter montado à mão em tarefas/diligências (`tarefas/http.go:681-700`, quebra com aspas); caminho padrão fixo `C:\Users\User\…`.

**Riscos do B relevantes para prazo:** todo movimento do DataJud vira "intimação" + tarefa (ruído, `monitor.go:288-297`); `Hits[0]` perde processos com mais de um grau (`coletor.go:129-133`); concluir qualquer tarefa-filha baixa a intimação (`tarefas/http.go:148-160`); exclusão de processo sem lixeira (`processos/http.go:446-466`); escrita tmp+rename sem fsync (`ficha/io.go:9-14`); histórico de sincronização lido por inteiro a cada coleta (`armazenador.go:116-184`); erro de `/api/prazos/calcular` vem como texto e a tela espera JSON (`motorprazo/http.go:36` × `12-prazo.js:72`).

## 4. Mapa de capacidades A × B

| Capacidade | v2 (A) | B | Observação |
|---|---|---|---|
| Autenticação/2FA/RBAC/auditoria | completo | senha única, sem RBAC/trilha | A |
| Backup cifrado + restauração testada | sim | pontual | A |
| Dados | PostgreSQL, CPF cifrado | fichas `.md` + JSON no OneDrive, em claro | A (migração B→A é projeto à parte) |
| Cadastros (pessoas, processos, tarefas, agenda, documentos) | sim | sim, mais ricos (atendimentos, audiências, diligências, financeiro, próxima providência) | B tem campos/telas que A não tem |
| Prazos | situação a_conferir/confirmado, advogado confirma | calculadora CPC/CPP + tarefas de intimação | nenhum calcula certo; calendário forense por tribunal é lacuna dos dois |
| Coleta DJEN/DataJud | só desenhada (Fases B/C) | existe e é testada | B adianta o roteiro |
| Índices econômicos, CEP | não | sim | portar |
| WhatsApp (fila) | não (F13 só ADR) | fila local, desligada | adiar |
| Visual/UX | funcional, simples | desenhado pelo advogado | **B vence (requisito)** |
| CSP estrita | sim | não (109 handlers inline) | refatorar ao portar |

## 5. Pendente
- Confirmar tabela CNJ no Anexo VI da Res. CNJ 65; comparar com o ZIP auditado.
- Conteúdo de `dados/` (pessoas/CPF) e `.sh` grandes: não abertos, por decisão.
- Decisão do advogado: arquitetura A/B/C do frontend (ver resposta ao advogado em 03/11/2026).
