package app

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/intimacoes"
	"painelteodoro/internal/intimacoes/djen"
)

var reInseguro = regexp.MustCompile(`(?i)\son[a-z]+\s*=|\sstyle\s*=|javascript:`)

// Categoria Intimações: colunas do B, origens novas e o botão Ver que abre o drawer "Intimação" (E2).
func TestIntimacoesComOrigemDataJudEBotaoVer(t *testing.T) {
	dj := novoFalsoDataJud(t)
	a := ambMonitor(t, dj, true, "k")
	pid := a.adv.criarProcesso(t, numA)
	ctx := context.Background()
	dj.poe(numA, hitG("G1", mov(26, "Distribuição", "2025-03-10T10:00:00.000Z")))
	_, _ = a.svc.ColetarAgora(ctx, a.ac)
	dj.poe(numA, hitG("G1", mov(26, "Distribuição", "2025-03-10T10:00:00.000Z"), mov(123, "Intimação", "2026-11-03T09:30:00.000Z")))
	_, _ = a.svc.ColetarAgora(ctx, a.ac)
	var iid string
	_ = a.pool.QueryRow(ctx, `SELECT id::text FROM intimacoes WHERE origem='DataJud'`).Scan(&iid)

	r, body := a.adv.get("/agenda")
	if r.StatusCode != 200 {
		t.Fatalf("agenda: %d", r.StatusCode)
	}
	for _, quer := range []string{"<th>Origem</th><th>Processo</th><th>Data</th><th>Status</th><th>Tarefa de análise</th>", "<td>DataJud</td>",
		`data-drawer="/intimacoes/` + iid + `/detalhe"`, `class="btn-text" data-drawer`, ">Ver</button>"} {
		if !strings.Contains(body, quer) {
			t.Errorf("a tabela deveria conter %q", quer)
		}
	}
	if reInseguro.MatchString(body) {
		t.Error("HTML da tabela com on*=/style=")
	}
	_ = pid

	r, d := a.adv.get("/intimacoes/" + iid + "/detalhe")
	if r.StatusCode != 200 {
		t.Fatalf("detalhe: %d", r.StatusCode)
	}
	for _, quer := range []string{"<h2>Intimação</h2>", "Origem", "DataJud", "Tribunal", "TJSP", "Órgão", "1ª Vara", "Processo", numA, "Data", "03/11/2026",
		"Tipo", "Texto completo", "Grau: G1", "Código do movimento: 123", "Link",
		"Movimentação processual (DataJud). Esta fonte não fornece o teor da intimação."} {
		if !strings.Contains(d, quer) {
			t.Errorf("o drawer deveria conter %q:\n%s", quer, d)
		}
	}
	if reInseguro.MatchString(d) {
		t.Error("drawer com on*=/style=")
	}
	if r, _ := a.adv.get("/intimacoes/nao-e-uuid/detalhe"); r.StatusCode != 404 {
		t.Errorf("id inválido = %d", r.StatusCode)
	}
	if r, _ := a.adv.get("/intimacoes/00000000-0000-0000-0000-000000000000/detalhe"); r.StatusCode != 404 {
		t.Errorf("inexistente = %d", r.StatusCode)
	}
}

// DJEN: texto integral e, no drawer, sem o aviso do DataJud; link só https.
func TestDetalheDaIntimacaoDJENMostraTextoIntegral(t *testing.T) {
	c := comunicacao("1", "")
	c.Texto = djen.NormalizarTexto(strings.Repeat("Texto longo da comunicação. ", 200) + "FIM-DO-TEXTO")
	f := &falsoDJEN{itens: []djen.Comunicacao{c}}
	e, ac, _ := ambDJEN(t, f)
	if _, err := e.svc.Intimacoes.ColetarAgora(context.Background(), ac); err != nil {
		t.Fatal(err)
	}
	adv, _ := e.user("adv-det@exemplo.test", auth.Advogado)
	var iid string
	_ = e.pool.QueryRow(context.Background(), `SELECT id::text FROM intimacoes`).Scan(&iid)
	_, d := adv.get("/intimacoes/" + iid + "/detalhe")
	if !strings.Contains(d, "FIM-DO-TEXTO") || strings.Contains(d, "Esta fonte não fornece o teor") || !strings.Contains(d, `href="https://exemplo.test/c/1"`) || !strings.Contains(d, "DJEN") {
		t.Errorf("drawer da intimação do DJEN:\n%.400s", d)
	}
	// Intimação do Plano 5 anterior à 0013 (só resumo): o texto migra sem perda.
	_, _ = e.pool.Exec(context.Background(), `UPDATE intimacoes SET texto='', resumo='Resumo antigo preservado' WHERE id=$1::uuid`, iid)
	if _, d = adv.get("/intimacoes/" + iid + "/detalhe"); !strings.Contains(d, "Resumo antigo preservado") {
		t.Error("sem texto integral, o drawer cai no resumo")
	}
}

func TestPaginaColetaMostraMonitorESigilosos(t *testing.T) {
	dj := novoFalsoDataJud(t)
	a := ambMonitor(t, dj, true, "k")
	a.adv.criarProcesso(t, numA)
	adm, _ := a.user("adm3-mon@exemplo.test", auth.Administrador)
	ctx := context.Background()
	_, _ = a.svc.ColetarAgora(ctx, a.ac) // processo sem registro no DataJud
	_, body := adm.get("/coleta-djen")
	for _, quer := range []string{"Monitor de processos (DataJud)", "<th>Processo</th><th>Tribunal</th><th>Último resultado</th><th>Última consulta</th><th>Base</th>", numA, "tjsp",
		"Sem registro no DataJud para este número", "Executar monitor agora", "Processos em segredo de justiça", "não o teor da intimação"} {
		if !strings.Contains(body, quer) {
			t.Errorf("a página Coleta DJEN deveria conter %q", quer)
		}
	}
	if reInseguro.MatchString(body) {
		t.Error("página com on*=/style=")
	}
	// Linha de base registrada.
	dj.poe(numA, hitG("G1", mov(26, "Distribuição", "2025-03-10T10:00:00.000Z")))
	_, _ = a.svc.ColetarAgora(ctx, a.ac)
	if _, body = adm.get("/coleta-djen"); !strings.Contains(body, "Linha de base registrada em 04/11/2026") {
		t.Error("a tela deveria dizer 'Linha de base registrada em {data}'")
	}
	// Só administrador vê a página; o advogado não.
	if r, _ := a.adv.get("/coleta-djen"); r.StatusCode != 403 {
		t.Errorf("advogado na página de coletas = %d", r.StatusCode)
	}
}

func TestBotaoExecutarMonitorSoComIntegracaoLigada(t *testing.T) {
	dj := novoFalsoDataJud(t)
	a := ambMonitor(t, dj, false, "k")
	adm, _ := a.user("adm4-mon@exemplo.test", auth.Administrador)
	_, body := adm.get("/coleta-djen")
	if strings.Contains(body, "Executar monitor agora") || !strings.Contains(body, "Monitor desligado neste ambiente.") {
		t.Error("desligado: sem botão e com o aviso")
	}
}

func TestCadastroDeSigilosoNaTela(t *testing.T) {
	dj := novoFalsoDataJud(t)
	a := ambMonitor(t, dj, true, "k", WithColetorSigiloso(&falsoSigiloso{}))
	pid := a.adv.criarProcesso(t, numB)
	semNum := a.adv.criarProcesso(t, "")
	adm, _ := a.user("adm5-mon@exemplo.test", auth.Administrador)
	ctx := context.Background()
	oab, _ := a.env.svc.Intimacoes.CadastrarOAB(ctx, a.ac, intimacoes.OABIn{Nome: "Adv Tela", Numero: "777777", UF: "SP",
		AutorizacaoBase: intimacoes.BaseAutorizacaoPadrao, AutorizacaoEm: "2026-11-01", Ativa: true})
	r, d := adm.get("/coleta-djen/sigilosos/novo")
	if r.StatusCode != 200 || !strings.Contains(d, numB) || !strings.Contains(d, "777777/SP") || strings.Contains(d, "(sem número)") {
		t.Errorf("drawer: %d (processo sem número não é oferecido)", r.StatusCode)
	}
	if reInseguro.MatchString(d) {
		t.Error("drawer com on*=/style=")
	}
	if r, _ := adm.post("/coleta-djen/sigilosos", url.Values{"processo_id": {semNum}, "oab_id": {oab}}); r.StatusCode != 400 {
		t.Errorf("processo sem número CNJ válido deveria ser recusado: %d", r.StatusCode)
	}
	if r, _ := adm.post("/coleta-djen/sigilosos", url.Values{"processo_id": {pid}, "oab_id": {oab}}); r.StatusCode != 200 {
		t.Fatalf("cadastro: %d", r.StatusCode)
	}
	if _, body := adm.get("/coleta-djen"); !strings.Contains(body, "777777/SP") || !strings.Contains(body, "Desativar") {
		t.Error("o sigiloso cadastrado deveria aparecer com a ação Desativar")
	}
	var sid string
	_ = a.pool.QueryRow(ctx, `SELECT id::text FROM monitor_sigilosos`).Scan(&sid)
	if r, _ := adm.post("/coleta-djen/sigilosos/"+sid+"/desativar", url.Values{}); r.StatusCode != 200 {
		t.Fatalf("desativar: %d", r.StatusCode)
	}
	if a.count(`SELECT count(*) FROM monitor_sigilosos WHERE NOT ativo`) != 1 {
		t.Error("nada se apaga: fica inativo")
	}
	if r, _ := a.adv.post("/coleta-djen/sigilosos", url.Values{"processo_id": {pid}, "oab_id": {oab}}); r.StatusCode != 403 {
		t.Errorf("advogado não cadastra: %d", r.StatusCode)
	}
}

// Rodapé com os três módulos e os textos de estado do B.
func TestRodapeComOsTresModulos(t *testing.T) {
	dj := novoFalsoDataJud(t)
	a := ambMonitor(t, dj, false, "k")
	ctx := context.Background()
	if err := a.svc.IniciarEstado(ctx); err != nil {
		t.Fatal(err)
	}
	if err := a.env.svc.Intimacoes.IniciarEstado(ctx); err != nil {
		t.Fatal(err)
	}
	_, body := a.adv.get("/sync/estado")
	for _, quer := range []string{"DJEN: bloqueado.", "DataJud: bloqueado.", "DJEN por processo sigiloso: bloqueado.", "Coleta desligada neste ambiente."} {
		if !strings.Contains(body, quer) {
			t.Errorf("o rodapé deveria conter %q:\n%s", quer, body)
		}
	}
	if strings.Count(body, `class="sync-error"`) != 3 {
		t.Errorf("os três módulos fora de sucesso levam sync-error: %d", strings.Count(body, `class="sync-error"`))
	}
}
