package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
)

// Gera instantâneos HTML do v2 (seed 100% fictício) para a prova de paridade com o painel B.
// Só roda com PARIDADE_OUT=<pasta vazia>; depois: python3 -m http.server -d <pasta> <porta>.
func TestInstantaneosParaParidade(t *testing.T) {
	out := os.Getenv("PARIDADE_OUT")
	if out == "" {
		t.Skip("PARIDADE_OUT não definida")
	}
	e := newEnv(t)
	adv, uid := e.user("adv@exemplo.test", auth.Advogado)
	if err := SeedDemo(context.Background(), e.pool, fieldKey, e.a.Store, uid); err != nil {
		t.Fatal(err)
	}
	if r, _ := adv.post("/prazos", prazoForm("Prazo fictício de exemplo", 3)); r.StatusCode != 303 {
		t.Fatalf("prazo = %d", r.StatusCode)
	}
	if err := copiaDir(filepath.Join("..", "..", "web", "static"), filepath.Join(out, "static")); err != nil {
		t.Fatal(err)
	}
	paginas := map[string]string{"agenda.html": "/agenda", "audiencias.html": "/audiencias", "tarefas.html": "/tarefas",
		"processos.html": "/processos", "pessoas.html": "/pessoas", "prazos.html": "/prazos", "atendimentos.html": "/atendimentos"}
	var base string
	for nome, caminho := range paginas {
		_, html := adv.get(caminho)
		if err := os.WriteFile(filepath.Join(out, nome), []byte(html), 0o600); err != nil {
			t.Fatal(err)
		}
		if nome == "agenda.html" {
			base = html
		}
	}
	// Plano 2: fragmentos dos drawers de Pessoas e Processos, gravados no mesmo caminho em que a página os
	// pede (data-drawer), para o JS do v2 abri-los a partir do servidor estático do instantâneo.
	var pAna, pPro1 string
	if err := e.pool.QueryRow(context.Background(), `SELECT id FROM pessoas WHERE codigo='PES. 0000001'`).Scan(&pAna); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.QueryRow(context.Background(), `SELECT id FROM processos WHERE codigo='PRO. 0000001'`).Scan(&pPro1); err != nil {
		t.Fatal(err)
	}
	// Plano 3: fragmentos dos drawers de Atendimentos e Tarefas.
	var ate1, tar1 string
	if err := e.pool.QueryRow(context.Background(), `SELECT id FROM atendimentos WHERE codigo='ATE. 0000001'`).Scan(&ate1); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.QueryRow(context.Background(), `SELECT id FROM tarefas WHERE codigo='TAR. 0000001'`).Scan(&tar1); err != nil {
		t.Fatal(err)
	}
	// Plano 4: fragmentos dos drawers de Agenda, Audiências e Diligências.
	var cmp1, aud1, dil1 string
	for sql, dest := range map[string]*string{`SELECT id FROM agenda_compromissos ORDER BY data LIMIT 1`: &cmp1,
		`SELECT id FROM audiencias WHERE codigo='AUD. 0000001'`: &aud1, `SELECT id FROM diligencias WHERE codigo='DIL. 0000001'`: &dil1} {
		if err := e.pool.QueryRow(context.Background(), sql).Scan(dest); err != nil {
			t.Fatal(err)
		}
	}
	for _, caminho := range []string{"/agenda/novo", "/agenda/" + cmp1 + "/editar", "/audiencias/nova", "/audiencias/" + aud1 + "/editar",
		"/audiencias/" + aud1 + "/cancelar", "/diligencias/" + dil1 + "/editar"} {
		_, frag := adv.do("GET", caminho, nil, map[string]string{"HX-Request": "true"})
		alvo := filepath.Join(out, filepath.FromSlash(strings.TrimPrefix(caminho, "/")))
		if err := os.MkdirAll(filepath.Dir(alvo), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(alvo, []byte(frag), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, caminho := range []string{"/atendimentos/novo", "/atendimentos/" + ate1 + "/editar", "/atendimentos/pessoas", "/tarefas/nova",
		"/tarefas/" + tar1 + "/editar", "/tarefas/" + tar1 + "/filha",
		"/pessoas/nova", "/pessoas/" + pAna + "/editar", "/processos/novo",
		"/processos/" + pPro1 + "/editar", "/processos/" + pPro1 + "/estado", "/processos/" + pPro1 + "/andamento",
		"/processos/" + pPro1 + "/financeiro", "/processos/" + pPro1 + "/providencia", "/processos/" + pPro1 + "/excluir"} {
		_, frag := adv.do("GET", caminho, nil, map[string]string{"HX-Request": "true"})
		alvo := filepath.Join(out, filepath.FromSlash(strings.TrimPrefix(caminho, "/")))
		if err := os.MkdirAll(filepath.Dir(alvo), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(alvo, []byte(frag), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Plano 5: a página Coleta DJEN (administrador) e os drawers da OAB.
	adm, _ := e.user("adm@exemplo.test", auth.Administrador)
	_, coleta := adm.get("/coleta-djen")
	if err := os.WriteFile(filepath.Join(out, "coleta-djen.html"), []byte(coleta), 0o600); err != nil {
		t.Fatal(err)
	}
	var oab1 string
	if err := e.pool.QueryRow(context.Background(), `SELECT id FROM oab_monitoradas LIMIT 1`).Scan(&oab1); err != nil {
		t.Fatal(err)
	}
	for _, caminho := range []string{"/coleta-djen/nova", "/coleta-djen/" + oab1 + "/editar"} {
		_, frag := adm.do("GET", caminho, nil, map[string]string{"HX-Request": "true"})
		alvo := filepath.Join(out, filepath.FromSlash(strings.TrimPrefix(caminho, "/")))
		if err := os.MkdirAll(filepath.Dir(alvo), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(alvo, []byte(frag), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Fragmento do rodapé, que o JS (data-refresh) recarrega.
	_, sync := adv.get("/sync/estado")
	if err := os.MkdirAll(filepath.Join(out, "sync"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "sync", "estado"), []byte(sync), 0o600); err != nil {
		t.Fatal(err)
	}
	// Página de componentes: a mesma casca, com o miolo do painel B (acordeão, drawer, máscaras…).
	// Os links de CSS/JS já são absolutos (/static/…).
	ini := strings.Index(base, `<div class="stats" id="stats">`)
	meio := strings.Index(base, `<div id="content">`)
	fim := strings.Index(base, `<section id="syncStatus"`)
	if ini < 0 || meio < ini || fim < meio {
		t.Fatal("casca inesperada")
	}
	comp := base[:ini] + `<div class="stats" id="stats">` + componentesStats + `</div>
` + `<div id="content">` + componentesConteudo + `</div>
` + base[fim:]
	if err := os.WriteFile(filepath.Join(out, "componentes.html"), []byte(comp), 0o600); err != nil {
		t.Fatal(err)
	}
	for nome, corpo := range map[string]string{"drawer-fragmento.html": drawerFragmento, "drawer-fragmento2.html": drawerFragmento2} {
		if err := os.WriteFile(filepath.Join(out, nome), []byte(corpo), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func copiaDir(de, para string) error {
	if err := os.MkdirAll(para, 0o750); err != nil {
		return err
	}
	ents, err := os.ReadDir(de)
	if err != nil {
		return err
	}
	for _, en := range ents {
		b, err := os.ReadFile(filepath.Join(de, en.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(para, en.Name()), b, 0o600); err != nil {
			return err
		}
	}
	return nil
}

const componentesStats = `<div class="acc-heads">
<button class="acc-head" data-acc="agenda:tarefas"><span>Tarefas</span><span class="count">2</span></button>
<button class="acc-head" data-acc="agenda:aulas"><span>Aulas</span><span class="count">1</span></button>
</div>`

const componentesConteudo = `<input id="buscaAgenda" class="search" autocomplete="off" placeholder="Buscar" data-filtro="#tabAula">
<div class="acc">
<div class="acc-item" data-acc="agenda:tarefas"><div class="acc-body"><table><thead><tr><th>Código</th><th>Título</th></tr></thead><tbody>
<tr><td>TAR.0001</td><td>Conferir procuração</td></tr><tr><td>TAR.0002</td><td>Redigir contestação</td></tr></tbody></table></div></div>
<div class="acc-item" data-acc="agenda:aulas"><div class="acc-body"><table id="tabAula"><thead><tr><th>Código</th><th>Título</th></tr></thead><tbody>
<tr><td>AGE.0001</td><td>Aula de Processo</td></tr><tr><td>AGE.0002</td><td>Reunião interna</td></tr></tbody></table></div></div>
</div>
<p><button type="button" class="btn btn-primary" id="abrirDrawer" data-drawer="/drawer-fragmento.html">Nova tarefa</button></p>
<div class="field"><label for="cpf">CPF</label><input type="text" id="cpf" data-mask="cpf" data-validate="cpf"></div>
<div class="field"><label for="tipo">Tipo</label><select id="tipo" data-mostra-quando="Outro" data-mostra="#campoOutro"><option>Reunião</option><option>Outro</option></select></div>
<div class="field" id="campoOutro" hidden><label>Qual?</label><input type="text"></div>
<div id="partesRows"></div>
<template id="tplParte"><div class="row2 row-parte"><input type="text" placeholder="Parte"><button type="button" class="btn-text" data-remove-row=".row-parte">Remover parte</button></div></template>
<button type="button" class="btn-text" id="addParte" data-add-row="#tplParte" data-row-target="#partesRows">+ adicionar parte</button>`

const drawerFragmento = `<h2>Nova tarefa</h2><p class="drawer-sub">Fragmento fictício de teste.</p>
<div class="field"><label for="dTitulo">Título</label><input type="text" id="dTitulo"></div>
<div class="field"><button type="button" class="btn btn-ghost" id="sobrepor" data-drawer="/drawer-fragmento2.html" data-drawer-sobrepor>Abrir sobreposto</button></div>
<div class="drawer-actions"><button type="button" class="btn btn-ghost" id="cancelar" data-drawer-fechar>Cancelar</button></div>`

const drawerFragmento2 = `<h2>Confirmar</h2><p class="drawer-sub">Segundo nível da pilha.</p>
<div class="drawer-actions"><button type="button" class="btn btn-ghost" id="fecharSobreposto" data-drawer-fechar>Fechar</button></div>`
