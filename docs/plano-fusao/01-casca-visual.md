# Fusão — Plano 1: Casca visual do painel B no v2

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Todas as telas que o v2 já tem passam a ter o visual do painel B (tokens, sidebar preta, selos, tabelas, drawer, toast), sem nenhum `on*=`/`style=` e sem relaxar a CSP.

**Architecture:** O CSS do B (`painel.css`) é copiado **sem alteração** para `web/static/painel.css`; uma camada `compat.css` traduz as classes que os templates atuais do v2 já usam (`.cartao`, `.pagina`, `.tabela`, `.selo`, `.botao`…) para o vocabulário visual do B. O `base.html` ganha a casca do B (sidebar + `.main` + faixa de rodapé). O drawer é um `<aside>` controlado por `app.js` com delegação de eventos e atributos `data-*`.

**Tech Stack:** Go 1.26, html/template, htmx local, CSS puro, JS puro (sem build).

**Spec:** `~/projetos/painel-teodoro-v2/docs/plano-fusao/ESPECIFICACAO-FUSAO.md` (cópia de `2º Cérebro/01-PROJETOS/2026-tec-fusao-paineis/ESPECIFICACAO-FUSAO.md`) e inventário `.../01-referencias/painel-B-inventario.md`.

## Global Constraints

- CSP estrita: sem `<script>` inline, sem `onclick`/`on*=`, sem `style=`, sem `unsafe-inline`/`unsafe-eval`; só recursos do próprio domínio.
- Fonte Arial; raio 3px; corpo 14px; sidebar 230px; drawer 480px que **empurra** a página, sem véu, **sem fechar por ESC nem por clique fora** (pedido do advogado).
- Cores de ação: vinho `#7C2D3B`; latão `#A9822F` só para foco/destaque. Sem tema escuro.
- Selos sempre com texto (cor nunca é a única informação).
- Arquivos de produção ≤ 300 linhas.
- A pasta `D:\OneDrive\Desktop\painel-teodoro-entrega-1-1-2026-10-30` é **somente leitura**; copiar dela é permitido, alterar não.
- Comandos Go rodam no WSL: `wsl.exe -e bash -lc 'cd ~/projetos/painel-teodoro-v2 && …'`.

---

## Task 1: Contrato do tema (teste que falha primeiro)

**Files:**
- Create: `web/tema_test.go`
- Create: `internal/app/csp_templates_test.go`

**Interfaces:**
- Produces: testes `TestTemaPainelContrato` (pacote `web`) e `TestTemplatesSemInline` (pacote `app`) que as tarefas seguintes fazem passar.

- [ ] **Step 1: Escrever `web/tema_test.go`**

```go
package web

import (
	"io/fs"
	"strings"
	"testing"
)

func lerStatic(t *testing.T, nome string) string {
	t.Helper()
	b, err := fs.ReadFile(Static(), nome)
	if err != nil {
		t.Fatalf("ler %s: %v", nome, err)
	}
	return string(b)
}

func TestTemaPainelContrato(t *testing.T) {
	css := strings.Join(strings.Fields(lerStatic(t, "painel.css")), "")
	for _, tok := range []string{
		"--ink:#000000;", "--paper:#EEF0F0;", "--panel:#FFFFFF;",
		"--wine:#7C2D3B;", "--wine-dark:#5E2029;", "--brass:#A9822F;",
		"--radius:3px;", ".drawer{width:480px;",
	} {
		if !strings.Contains(css, tok) {
			t.Errorf("painel.css sem %q", tok)
		}
	}
	for _, nome := range []string{"painel.css", "compat.css"} {
		c := lerStatic(t, nome)
		for _, proibido := range []string{"@import", "url(http", "url(//", "prefers-color-scheme", "expression("} {
			if strings.Contains(c, proibido) {
				t.Errorf("%s contém %q", nome, proibido)
			}
		}
	}
}

func TestAppJSSemEvalNemInline(t *testing.T) {
	js := lerStatic(t, "app.js")
	for _, proibido := range []string{"eval(", "new Function", "innerHTML", "document.write"} {
		if strings.Contains(js, proibido) {
			t.Errorf("app.js contém %q", proibido)
		}
	}
}
```

- [ ] **Step 2: Escrever `internal/app/csp_templates_test.go`**

```go
package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	reHandler = regexp.MustCompile(`(?i)\son[a-z]+\s*=`)
	reStyle   = regexp.MustCompile(`(?i)\sstyle\s*=`)
	reScript  = regexp.MustCompile(`(?i)<script(\s[^>]*)?>`)
)

// Nenhum template pode ter handler inline, style= ou <script> sem src (a CSP bloquearia).
func TestTemplatesSemInline(t *testing.T) {
	raiz := filepath.Join("..", "..")
	n := 0
	err := filepath.WalkDir(raiz, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".html") || !strings.Contains(filepath.ToSlash(p), "/templates/") {
			return err
		}
		n++
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		s := string(b)
		if reHandler.MatchString(s) {
			t.Errorf("%s: handler inline (on*=)", p)
		}
		if reStyle.MatchString(s) {
			t.Errorf("%s: atributo style=", p)
		}
		for _, m := range reScript.FindAllString(s, -1) {
			if !strings.Contains(strings.ToLower(m), "src=") {
				t.Errorf("%s: <script> inline: %s", p, m)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n < 10 {
		t.Fatalf("poucos templates encontrados (%d): caminho errado?", n)
	}
}
```

- [ ] **Step 3: Rodar e ver falhar**

Run: `wsl.exe -e bash -lc 'cd ~/projetos/painel-teodoro-v2 && go test ./web/ -run Tema -v'`
Expected: FAIL (`ler painel.css: file does not exist`). O teste de templates já passa (não há inline hoje) e fica como trava de regressão.

- [ ] **Step 4: Commit**

```bash
git add web/tema_test.go internal/app/csp_templates_test.go
git commit -m "test(fusao): contrato do tema B e trava contra inline em templates"
```

---

## Task 2: Copiar o CSS e o logo do B (sem alterar o original)

**Files:**
- Create: `web/static/painel.css` (cópia literal)
- Create: `web/static/logo.png` (cópia literal; otimização opcional no passo 4)

**Interfaces:**
- Produces: `/static/painel.css`, `/static/logo.png`.

- [ ] **Step 1: Registrar o hash do original antes de copiar**

Run:
```bash
wsl.exe -e bash -lc 'cd "/mnt/d/OneDrive/Desktop/painel-teodoro-entrega-1-1-2026-10-30/painel-teodoro/web" && sha256sum css/painel.css assets/logo-lateral-escuro.png'
```
Anotar os dois hashes em `docs/plano-fusao/HASHES-B.txt` (criar o arquivo com essas duas linhas).

- [ ] **Step 2: Copiar**

Run:
```bash
wsl.exe -e bash -lc 'B="/mnt/d/OneDrive/Desktop/painel-teodoro-entrega-1-1-2026-10-30/painel-teodoro/web"; cd ~/projetos/painel-teodoro-v2 && cp "$B/css/painel.css" web/static/painel.css && cp "$B/assets/logo-lateral-escuro.png" web/static/logo.png && sha256sum web/static/painel.css web/static/logo.png'
```
Expected: hashes idênticos aos do passo 1.

- [ ] **Step 3: Conferir que o original não mudou**

Run o `sha256sum` do passo 1 de novo. Expected: mesmos hashes.

- [ ] **Step 4 (opcional): reduzir o logo**

Se `magick` existir: `magick web/static/logo.png -resize 300x300 web/static/logo.png` (aparece a 150px; 300px cobre telas 2x). Se não existir, manter o original e anotar no ficha.

- [ ] **Step 5: Commit**

```bash
git add web/static/painel.css web/static/logo.png docs/plano-fusao/HASHES-B.txt
git commit -m "feat(ui): copia o CSS e o logo do painel B (somente leitura na origem)"
```

---

## Task 3: Camada de compatibilidade (`compat.css`)

**Files:**
- Create: `web/static/compat.css`

**Interfaces:**
- Consumes: tokens de `painel.css` (`--ink`, `--paper`, `--panel`, `--wine`, `--wine-dark`, `--brass`, `--text`, `--text-muted`, `--border`, `--border-strong`, `--danger-bg`, `--warn-bg`, `--ok-bg`, `--neutral-bg`, `--radius`).
- Produces: estilos para as classes já usadas pelos templates do v2: `.cartao`, `.pagina`, `.apoio`, `.erro`, `.ok`, `.botao`, `button`, `button.secundario`, `button.perigo`, `.tabela`, `.selo`, `.selo-nivel`, `.nivel-*`, `.aviso-prazo`, `.cabeca`, `.busca`, `.linha`, `.grade`, `.bloco`, `.ficha`, `.lista`, `.acoes`, `.paginas`, `.pular`.

- [ ] **Step 1: Escrever `web/static/compat.css`**

```css
/* Camada de compatibilidade: dá o visual do painel B às classes que os templates do v2 já usam.
   Telas novas devem usar as classes do painel.css (.btn, .badge, .field, .empty…). */
h2{font-size:18px;font-weight:600;margin:0 0 8px 0;}
a{color:var(--wine);}
a:hover{color:var(--wine-dark);}
.apoio{color:var(--text-muted);}
.erro{color:var(--wine-dark);font-weight:600;}
.ok{color:var(--ink-2);font-weight:600;}
code{font-family:ui-monospace,Consolas,monospace;font-size:12.5px;}
.pular{position:absolute;left:8px;top:8px;z-index:100;transform:translateY(-150%);background:var(--ink);color:#fff;padding:8px 12px;border-radius:var(--radius);font-weight:600;}
.pular:focus{transform:translateY(0);}

.cartao{max-width:420px;margin:8vh auto;padding:24px;background:var(--panel);border:1px solid var(--border);border-radius:var(--radius);}
.pagina{padding:18px 28px 40px 28px;}
.cabeca{display:flex;flex-wrap:wrap;gap:16px;align-items:center;justify-content:space-between;margin-bottom:14px;}
.cabeca h1{font-size:24px;font-weight:600;margin:0;}
.busca{margin:0 0 14px 0;}
.linha{display:flex;flex-wrap:wrap;gap:12px;align-items:end;}
.linha>div{flex:1 1 200px;}
.grade{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:0 12px;align-items:start;}
.grade .larga{grid-column:1/-1;}
.bloco{margin-top:20px;padding-top:14px;border-top:1px solid var(--border);}

label{display:block;margin:12px 0 5px;font-size:12.5px;font-weight:600;}
input,select,textarea{width:100%;padding:9px 11px;border:1px solid var(--border-strong);border-radius:var(--radius);background:#fff;color:var(--text);}
input:focus,select:focus,textarea:focus{outline:2px solid var(--ink-2);outline-offset:1px;}
input[type=checkbox],input[type=radio]{width:auto;}

button,.botao{display:inline-block;margin-top:14px;padding:9px 16px;border:0;border-radius:var(--radius);background:var(--wine);color:#fff;font-size:13px;font-weight:600;cursor:pointer;text-decoration:none;}
button:hover,.botao:hover{background:var(--wine-dark);color:#fff;}
button.secundario{margin:0;background:transparent;color:var(--text);border:1px solid var(--border-strong);}
button.secundario:hover{background:#fff;color:var(--text);}
button.perigo{background:var(--wine-dark);}
.cabeca .botao,.linha button,.acoes button,.lista button{margin-top:0;}
.acoes button,.lista button{padding:5px 10px;font-size:12px;}
.acoes{display:flex;flex-wrap:wrap;gap:8px;align-items:center;}
.acoes form,.lista form{margin:0;}

.tabela{overflow-x:auto;}
table caption{text-align:left;color:var(--text-muted);padding:0 0 8px 0;font-size:12.5px;}
th,td{white-space:nowrap;}
.quebra th,.quebra td{white-space:normal;}
tbody tr:nth-child(even){background:#F7F8F7;}
tbody tr:hover{background:#EEF2F1;}

.selo{display:inline-block;padding:2px 9px;border-radius:10px;font-size:11.5px;font-weight:600;background:var(--warn-bg);color:#7A5F1E;}
.selo-nivel{display:inline-block;padding:2px 9px;border-radius:10px;font-size:11.5px;font-weight:600;background:var(--neutral-bg);color:var(--text-muted);}
.nivel-atrasado .selo-nivel,.nivel-hoje .selo-nivel{background:var(--danger-bg);color:var(--wine-dark);}
.nivel-d3 .selo-nivel,.nivel-d5 .selo-nivel{background:var(--warn-bg);color:#7A5F1E;}
.nivel-atrasado td,li.nivel-atrasado{background:var(--danger-bg);}
.nivel-hoje td,li.nivel-hoje{background:var(--danger-bg);}
.nivel-encerrado td{opacity:.7;}
.aviso-prazo{padding:10px 14px;border-left:4px solid var(--brass);background:var(--warn-bg);font-size:13px;}

.ficha{display:grid;grid-template-columns:max-content 1fr;gap:6px 20px;margin:14px 0;}
.ficha dt{color:var(--text-muted);}
.ficha dd{margin:0;}
.lista{list-style:none;padding:0;margin:8px 0;}
.lista li{display:flex;flex-wrap:wrap;gap:12px;align-items:center;justify-content:space-between;padding:8px 0;border-bottom:1px solid var(--border);}
.paginas{display:flex;gap:16px;margin-top:14px;align-items:center;}
.chave{letter-spacing:.08em;word-break:break-all;}
.qr{display:block;margin:14px 0;background:#fff;padding:8px;border-radius:var(--radius);}
.codigos{columns:2;list-style:none;padding:0;font-size:16px;}
@media (max-width:640px){.ficha{grid-template-columns:1fr;}.pagina{padding:14px 16px 40px 16px;}}
```

- [ ] **Step 2: Rodar o teste do tema**

Run: `wsl.exe -e bash -lc 'cd ~/projetos/painel-teodoro-v2 && go test ./web/ -v'`
Expected: `TestTemaPainelContrato` PASS (os tokens vêm do `painel.css` copiado), `TestAppJSSemEvalNemInline` PASS.

- [ ] **Step 3: Commit**

```bash
git add web/static/compat.css
git commit -m "feat(ui): camada de compatibilidade das classes do v2 com o tema B"
```

---

## Task 4: Casca do B em `base.html` (sidebar, main, rodapé)

**Files:**
- Modify: `internal/auth/templates/base.html`
- Modify: `web/static/compat.css` (estilo de `nav.tabs a`, sair e rodapé)
- Delete (do repo): `web/static/app.css` — substituído por `painel.css` + `compat.css`. Conferir antes: `grep -rn "app.css" --include=*.go --include=*.html .` deve listar só `base.html`.

**Interfaces:**
- Consumes: campos de página já existentes: `.Title`, `.CSRF`, `.Role`, `.Email`, `.Nav` (itens com `.Href` e `.Label`), `.CanUsers`, `.CanAudit`.
- Produces: estrutura `#app > .sidebar + .main`, usada por todas as telas logadas; telas sem sessão (login, MFA, setup) continuam em `main.cartao` **sem** sidebar.

- [ ] **Step 1: Substituir `base.html`**

```html
{{define "base"}}<!doctype html>
<html lang="pt-BR">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex, nofollow">
<meta name="csrf-token" content="{{.CSRF}}">
<meta name="htmx-config" content='{"allowEval":false,"includeIndicatorStyles":false}'>
<title>{{.Title}} · Painel Teodoro</title>
<link rel="stylesheet" href="/static/painel.css">
<link rel="stylesheet" href="/static/compat.css">
<script src="/static/htmx.min.js" defer></script>
<script src="/static/webauthn.js" defer></script>
<script src="/static/app.js" defer></script>
</head>
<body>
<a class="pular" href="#conteudo">Pular para o conteúdo</a>
{{if .Role}}
<div id="app">
  <aside class="sidebar">
    <a class="brand" href="/" aria-label="Painel Teodoro — início"><img class="seal-logo" src="/static/logo.png" alt="" width="150" height="150"></a>
    <nav class="tabs" aria-label="Principal">
      <a href="/">Hoje</a>
      {{range .Nav}}<a href="{{.Href}}">{{.Label}}</a>{{end}}
      <a href="/conta">Minha conta</a>
      {{if .CanUsers}}<a href="/admin/usuarios">Usuários</a>{{end}}
      {{if .CanAudit}}<a href="/admin/auditoria">Auditoria</a>{{end}}
    </nav>
    <form method="post" action="/logout" class="sair">
      <input type="hidden" name="csrf_token" value="{{.CSRF}}">
      <span class="quem">{{.Email}}<br>{{.Role}}</span>
      <button type="submit" class="secundario">Sair</button>
    </form>
  </aside>
  <div class="main">
    <main id="conteudo" class="pagina">
{{template "content" .}}
    </main>
  </div>
</div>
<aside id="drawer" class="drawer-overlay hidden" aria-hidden="true">
  <div class="drawer" role="dialog" aria-labelledby="drawer-titulo" aria-modal="false" tabindex="-1">
    <div id="drawer-corpo"></div>
  </div>
</aside>
<div id="toast" class="toast hidden" role="status" aria-live="polite"></div>
{{else}}
<main id="conteudo" class="cartao">
{{template "content" .}}
</main>
{{end}}
</body>
</html>{{end}}
```

- [ ] **Step 2: Acrescentar ao fim de `compat.css` o estilo dos links da sidebar**

```css
nav.tabs a{display:flex;align-items:center;justify-content:space-between;padding:9px 12px;border-radius:var(--radius);font-size:13.5px;color:#C3CBD1;text-decoration:none;}
nav.tabs a:hover{background:var(--ink-2);color:#fff;}
nav.tabs a[aria-current=page]{background:var(--ink-2);color:#fff;box-shadow:inset 2px 0 0 var(--brass);}
a.brand{display:flex;justify-content:center;margin-bottom:28px;}
.sidebar .sair{margin-top:20px;padding-top:14px;border-top:1px solid #262626;display:flex;flex-direction:column;gap:8px;}
.sidebar .quem{color:#8B98A3;font-size:12px;overflow-wrap:anywhere;}
.sidebar button.secundario{color:#D9DEE2;border-color:#3a3a3a;}
.sidebar button.secundario:hover{background:var(--ink-2);color:#fff;}
:focus-visible{outline:3px solid var(--brass);outline-offset:2px;}
@media (max-width:780px){.sidebar .sair{margin:0 0 0 16px;padding:0;border:0;flex-direction:row;align-items:center;}a.brand{margin-bottom:0;}}
```

(O CSS do B usa `nav.tabs button`; os links precisam do equivalente. O `:focus-visible` fica explícito porque `all:unset` do B apaga o anel em botões — correção de acessibilidade que não muda o visual.)

- [ ] **Step 3: Marcar a aba atual**

Em `web/static/app.js`, dentro do IIFE, acrescentar (sem `innerHTML`):

```js
  // Marca no menu o link da página atual.
  document.addEventListener("DOMContentLoaded", function () {
    var atual = location.pathname;
    document.querySelectorAll("nav.tabs a").forEach(function (a) {
      var h = a.getAttribute("href");
      var ativo = h === "/" ? atual === "/" : atual === h || atual.indexOf(h + "/") === 0;
      if (ativo) a.setAttribute("aria-current", "page");
    });
  });
```

- [ ] **Step 4: Remover `app.css` e rodar tudo**

Run:
```bash
wsl.exe -e bash -lc 'cd ~/projetos/painel-teodoro-v2 && grep -rn "app.css" --include=*.go --include=*.html . ; git rm -q web/static/app.css && go vet ./... && go test ./... 2>&1 | tail -15'
```
Expected: o `grep` só mostra o que já foi trocado (nada); `go test ./...` verde (inclui `TestTemplatesSemInline` e `TestTemaPainelContrato`). Se algum teste de integração checar `app.css`, trocar para `painel.css`.

- [ ] **Step 5: Conferir a CSP no navegador**

Subir a pilha dev (`deploy/` conforme `docs/operacao/RUNBOOK.md`), abrir `http://localhost:18080/login` e a tela de Tarefas; `read_console_messages` com padrão `Content Security Policy|Refused` deve voltar vazio.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat(ui): casca do painel B (sidebar, main, drawer, toast) nas telas logadas"
```

---

## Task 5: Drawer e toast por delegação de eventos

**Files:**
- Modify: `web/static/app.js`
- Create: `web/static/drawer.js` (se `app.js` passar de 300 linhas, senão manter em `app.js`)
- Test: manual no navegador + `TestAppJSSemEvalNemInline`

**Interfaces:**
- Produces (contrato para os planos seguintes):
  - Link ou botão com `data-drawer="/caminho"` → faz `GET /caminho` com cabeçalho `HX-Request: true`, coloca o HTML retornado (um fragmento que começa com `<h2 id="drawer-titulo">`) em `#drawer-corpo` e abre o drawer.
  - Elemento com `data-drawer-fechar` dentro do drawer → fecha. **Não** há ESC nem clique fora.
  - `window.PainelToast(texto, erro bool)` mostra o toast por 3,2 s (usa `textContent`).
  - Resposta HTMX com cabeçalho `HX-Trigger: {"toast":{"msg":"…","erro":false}}` mostra o toast.
  - `body.drawer-aberto` é ligado/desligado (o CSS do B já empurra a página).

- [ ] **Step 1: Acrescentar ao IIFE de `app.js`**

```js
  // ---- Drawer lateral (visual do painel B): sem ESC, sem clique fora, empurra a página.
  var drawer = document.getElementById("drawer");
  var corpo = document.getElementById("drawer-corpo");
  var voltarFoco = null;

  function abrirDrawer() {
    if (!drawer) return;
    drawer.classList.remove("hidden");
    drawer.setAttribute("aria-hidden", "false");
    document.body.classList.add("drawer-aberto");
    var alvo = drawer.querySelector("input,select,textarea,button");
    (alvo || drawer.querySelector(".drawer")).focus();
  }
  function fecharDrawer() {
    if (!drawer) return;
    drawer.classList.add("hidden");
    drawer.setAttribute("aria-hidden", "true");
    document.body.classList.remove("drawer-aberto");
    if (corpo) corpo.replaceChildren();
    if (voltarFoco && voltarFoco.focus) voltarFoco.focus();
    voltarFoco = null;
  }
  document.addEventListener("click", function (ev) {
    var a = ev.target.closest("[data-drawer]");
    if (a && corpo) {
      ev.preventDefault();
      voltarFoco = a;
      fetch(a.getAttribute("data-drawer"), { headers: { "HX-Request": "true" }, credentials: "same-origin" })
        .then(function (r) { if (!r.ok) throw new Error(r.status); return r.text(); })
        .then(function (html) {
          // Fragmento do próprio servidor (html/template, já escapado). DOMParser não executa scripts.
          var doc = new DOMParser().parseFromString(html, "text/html");
          corpo.replaceChildren.apply(corpo, Array.prototype.slice.call(doc.body.childNodes));
          if (window.htmx) window.htmx.process(corpo);
          abrirDrawer();
        })
        .catch(function () { window.PainelToast("Não foi possível abrir o formulário.", true); });
      return;
    }
    if (ev.target.closest("[data-drawer-fechar]")) fecharDrawer();
  });
  document.addEventListener("htmx:afterSwap", function (ev) {
    if (ev.target && ev.target.id === "drawer-corpo") abrirDrawer();
  });

  // ---- Toast
  var toast = document.getElementById("toast");
  var tToast = null;
  window.PainelToast = function (msg, erro) {
    if (!toast) return;
    toast.textContent = msg;
    toast.classList.toggle("error", !!erro);
    toast.classList.remove("hidden");
    clearTimeout(tToast);
    tToast = setTimeout(function () { toast.classList.add("hidden"); }, 3200);
  };
  document.addEventListener("toast", function (ev) {
    var d = ev.detail || {};
    window.PainelToast(d.msg || "", d.erro);
  });
```

**Nota de segurança:** o código não usa `innerHTML` (o teste `TestAppJSSemEvalNemInline` o proíbe); o fragmento do servidor entra por `DOMParser` + `replaceChildren`, que não executa scripts.

- [ ] **Step 2: Rodar os testes**

Run: `wsl.exe -e bash -lc 'cd ~/projetos/painel-teodoro-v2 && go test ./web/ ./internal/app/ 2>&1 | tail'`
Expected: PASS.

- [ ] **Step 3: Verificar no navegador (aceite do drawer)**

Com a pilha dev no ar, abrir uma tela com `data-drawer` (criada no Plano 2) **ou** colar no console: `document.body.insertAdjacentHTML` é proibido pela CSP de testes? Não — console não é a página; usar `read_page`/`computer` para conferir. Critérios:
1. Abrir empurra a página 480px; a sidebar não some.
2. Apertar ESC **não** fecha; clicar fora **não** fecha.
3. `data-drawer-fechar` fecha e devolve o foco ao link de origem.
4. Console sem erro de CSP.

Registrar o resultado em `docs/plano-fusao/ACEITE-VISUAL.md`.

- [ ] **Step 4: Commit**

```bash
git add web/static/app.js
git commit -m "feat(ui): drawer lateral e toast por delegação de eventos (sem ESC, sem clique fora)"
```

---

## Task 6: Selos, vazios e botões nas telas existentes

**Files:**
- Modify: `internal/tarefas/templates/lista.html`, `internal/processos/templates/*.html`, `internal/pessoas/templates/*.html`, `internal/agenda/templates/lista.html`, `internal/prazos/templates/*.html`, `internal/documentos/templates/*.html`
- Test: `go test ./...`

**Interfaces:**
- Consumes: classes do B (`.badge .badge-danger|warn|ok|neutral`, `.empty`, `.btn .btn-primary|ghost|small`, `.row-actions`).

- [ ] **Step 1: Localizar usos**

Run: `wsl.exe -e bash -lc 'cd ~/projetos/painel-teodoro-v2 && grep -rn "class=\"selo\|class=\"botao\|class=\"erro\|apoio\">Nenhum\|class=\"acoes' internal --include=*.html'`

- [ ] **Step 2: Regras de troca (aplicar em cada ocorrência)**

| De | Para |
|---|---|
| `<span class="selo">a conferir pelo advogado</span>` | `<span class="badge badge-warn">A CONFERIR</span>` |
| `<strong class="erro">atrasada</strong>` | `<span class="badge badge-danger">atrasada</span>` |
| `<a class="botao" …>` de ação principal | `<a class="btn btn-primary" …>` |
| `<td class="acoes">` | `<td class="row-actions">` |
| `<p class="apoio">Nenhum…</p>` (lista vazia) | `<div class="empty"><h3>Nada por aqui</h3><p>Nenhum registro encontrado.</p></div>` |

Os textos dos selos continuam exatamente os mesmos (a regra "a conferir" não pode sumir).

- [ ] **Step 3: Rodar**

Run: `wsl.exe -e bash -lc 'cd ~/projetos/painel-teodoro-v2 && go vet ./... && go test ./... 2>&1 | tail -15'`
Expected: PASS. Se algum teste procura o texto `a conferir pelo advogado`, ajustar o teste para `A CONFERIR` **e** manter uma asserção de que o texto aparece.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat(ui): selos, estados vazios e botões no vocabulário do painel B"
```

---

## Task 7: Aceite visual contra o painel B

**Files:**
- Create: `docs/plano-fusao/ACEITE-VISUAL.md`

**Por que manual:** o painel B só renderiza com o backend dele, que lê os dados reais do escritório. **Não executamos o B.** A referência são capturas de tela que o advogado tira do próprio painel.

- [ ] **Step 1: Pedir ao advogado as capturas de referência**

Pedir (em chat) capturas, com **dados fictícios ou borrados**, das telas: Agenda, Tarefas, Processos, Pessoas, Atendimentos, Audiências, com e sem drawer aberto, e a faixa do rodapé. Guardar em `2º Cérebro/01-PROJETOS/2026-tec-fusao-paineis/01-referencias/capturas-B/` (não no repositório).

- [ ] **Step 2: Capturar as telas equivalentes do v2** (Chrome, mesma largura: 1440 e 400 px) e comparar lado a lado: sidebar, cores, tipografia, espaçamentos, selos, tabela, drawer.

- [ ] **Step 3: Preencher a tabela em `ACEITE-VISUAL.md`**

```markdown
| Tela | Largura | Igual? | Diferenças | Ação |
|---|---|---|---|---|
```

- [ ] **Step 4: Advogado aprova ou lista ajustes.** Ajustes viram tarefas no Plano 1.1; aprovação libera o Plano 2.

- [ ] **Step 5: Atualizar a ficha** (`2026-tec-fusao-paineis/00-FICHA-DO-PROJETO.md`: estado, próximo passo, decisões) e commitar `docs/plano-fusao/`.

```bash
git add docs/plano-fusao
git commit -m "docs(fusao): aceite visual do plano 1"
```

---

## Self-review (feito)

- **Cobertura da especificação:** §4.1 tokens (Tasks 2–3), §4.2 componentes (Tasks 4–6; acordeão `acc-head` e ticker ficam no Plano 2 e Plano 8 porque dependem de dados), §4.3 interação sem inline (Tasks 1 e 5), §4.4 hash/URL (já existe: as telas do v2 têm rota própria), §4.5 logo (Task 2 passo 4), §4.6 acessibilidade (Task 4 passo 2: foco), §8 critérios 2, 3 (hash do B em Task 2) e 4 (textos "A CONFERIR" preservados em Task 6). §8.1 depende da Task 7.
- **Sem placeholders:** a única decisão aberta é a otimização do logo (condicional explícita).
- **Consistência:** `data-drawer`, `data-drawer-fechar`, `PainelToast`, `#drawer`, `#drawer-corpo`, `#toast` usados do mesmo jeito nas Tasks 4 e 5.
- **Dark mode:** removido de propósito (o B não tem). Registrar no ADR 0009 ao fim do Plano 1.

## Planos seguintes (um por subsistema; cada um entrega software testável)

| # | Plano | O que herda do B (o melhor que vale aproveitar) |
|---|---|---|
| 2 | Agenda + Tarefas no visual do B | acordeão de categorias com contador; conflito de horário (crítico = audiência/prazo fatal); tarefas em 6 estados; PDF anexo validado por `%PDF`; baixa de intimação só quando **todas** as tarefas-filhas fecham (corrige `tarefas/http.go:148-160` do B) |
| 3 | Processos e Pessoas completos | financeiro, próxima providência, andamentos; partes/responsáveis; CEP (ViaCEP) com timeout; máscaras CPF/CNPJ |
| 4 | Atendimentos, Audiências, Diligências | 6 tipos de audiência; cancelamento com reautenticação (no v2: confirmação com 2º fator) |
| 5 | Coletor DJEN + Intimações | paginação até 100, teto 10.000, 429, página repetida, sobreposição de 7 dias, dedupe por hash, tarefa idempotente; **sem** `prazo_fatal` automático: gera *sugestão* para o advogado confirmar |
| 6 | Monitor DataJud + segredo de justiça | estado por processo, "visto só depois de registrado", filtro por OAB do sócio; **todos** os graus (não só `Hits[0]`); só movimentos relevantes viram tarefa |
| 7 | Calculadora e calendário forense | algoritmo da Páscoa, recesso, prorrogação; calendário **por tribunal com fonte citada** (Semana Santa quarta–domingo, sem Cinzas/Corpus Christi por padrão); dobro e termo inicial por juntada/citação; sempre "A CONFERIR" |
| 8 | Índices BCB e rodapé | séries SGS 432, 433, 188, 189, 1, 1619; ticker; nunca valor presumido; painel "Estado das coletas" com última tentativa e último sucesso |
