package app

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/intimacoes"
	"painelteodoro/internal/intimacoes/djen"
)

func decodifica(t *testing.T, v string) string {
	t.Helper()
	s, err := url.PathUnescape(v)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// agendaComDuasIntimacoes coleta duas intimações (uma com processo) e devolve o ambiente, os navegadores e os ids.
func agendaComDuasIntimacoes(t *testing.T) (e *env, adv, ass, adm *browser, ids []string) {
	t.Helper()
	f := &falsoDJEN{itens: []djen.Comunicacao{comunicacao("1", "0000101-25.2025.8.26.0100"), comunicacao("2", "")}}
	e, ac, _ := ambDJEN(t, f)
	adv, _ = e.user("adv-tela@exemplo.test", auth.Advogado)
	ass, _ = e.user("ass-tela@exemplo.test", auth.Assistente)
	adm, _ = e.user("adm-tela@exemplo.test", auth.Administrador)
	if _, err := e.svc.Intimacoes.ColetarAgora(context.Background(), ac); err != nil {
		t.Fatal(err)
	}
	ls, _, _, err := e.svc.Intimacoes.Lista(context.Background())
	if err != nil || len(ls) != 2 {
		t.Fatalf("lista = %d %v", len(ls), err)
	}
	for _, l := range ls {
		ids = append(ids, l.ID)
	}
	return e, adv, ass, adm, ids
}

func TestAgendaIntimacoesTabelaDoB(t *testing.T) {
	_, adv, ass, adm, ids := agendaComDuasIntimacoes(t)
	_, pg := adv.get("/agenda")
	ordem(t, "tabela", pg, `data-acc="agenda:intimacoes"`, `<table><thead><tr><th>Origem</th><th>Processo</th><th>Data</th><th>Status</th><th>Tarefa de análise</th><th></th></tr></thead>`)
	if !regexp.MustCompile(`<span>Intimações</span><span class="count">2</span>`).MatchString(pg) {
		t.Error("o contador da pílula Intimações deve contar o histórico inteiro")
	}
	if strings.Contains(pg, "Intimações indisponíveis.") {
		t.Error("com intimações registradas não há bloco de indisponibilidade")
	}
	for _, trecho := range []string{`<td>DJEN</td>`, `<td class="tabular">02/11/2026</td>`, `<span class="badge badge-warn">Em triagem</span>`,
		`<strong>Analisar intimação — Intimação`, `<div class="hint">a-confirmar</div>`} {
		if !strings.Contains(pg, trecho) {
			t.Errorf("faltou %q", trecho)
		}
	}
	// Advogado: Confirmar (btn-text) e Descartar (btn-danger-text), um formulário de envio para cada.
	if !strings.Contains(pg, `action="/intimacoes/`+ids[0]+`/decidir"`) || !strings.Contains(pg, `class="btn-text">Confirmar</button>`) ||
		!strings.Contains(pg, `class="btn-danger-text" data-drawer="/intimacoes/`+ids[0]+`/cancelar">Cancelar</button>`) {
		t.Error("o advogado deveria ver Confirmar e Cancelar")
	}
	for nome, b := range map[string]*browser{"assistente": ass, "administrador": adm} {
		_, p := b.get("/agenda")
		if strings.Contains(p, "/decidir") || strings.Contains(p, ">Confirmar<") || strings.Contains(p, ">Cancelar<") {
			t.Errorf("%s não decide intimação: botões não podem aparecer", nome)
		}
		if !strings.Contains(p, `<td class="row-actions"><button type="button" class="btn-text" data-drawer="/intimacoes/`+ids[0]+`/detalhe">Ver</button></td>`) {
			t.Errorf("%s: a coluna de ações só traz o botão Ver", nome)
		}
	}
	if regexp.MustCompile(`\son[a-z]+=|\sstyle=`).MatchString(pg) {
		t.Error("sem handler inline nem style=")
	}
}

func TestDecidirPelaTelaRBACEToasts(t *testing.T) {
	e, adv, ass, adm, ids := agendaComDuasIntimacoes(t)
	for nome, b := range map[string]*browser{"assistente": ass, "administrador": adm} {
		if r, _ := b.post("/intimacoes/"+ids[0]+"/decidir", url.Values{"status": {"confirmada"}}); r.StatusCode != 403 {
			t.Errorf("%s decidindo = %d, esperado 403", nome, r.StatusCode)
		}
	}
	if e.count(`SELECT count(*) FROM intimacoes WHERE status='triagem'`) != 2 {
		t.Fatal("nada deve mudar sem permissão")
	}
	if r, _ := adv.semCSRFDecidir(ids[0]); r != 403 {
		t.Errorf("sem CSRF = %d", r)
	}
	r, _ := adv.post("/intimacoes/"+ids[0]+"/decidir", url.Values{"status": {"confirmada"}})
	if r.StatusCode != 200 || decodifica(t, r.Header.Get("X-Painel-Ok")) != "Intimação confirmada." {
		t.Errorf("confirmar = %d %q", r.StatusCode, r.Header.Get("X-Painel-Ok"))
	}
	r, _ = adv.post("/intimacoes/"+ids[1]+"/cancelar", url.Values{"motivo": {"Duplicada"}})
	if r.StatusCode != 200 || decodifica(t, r.Header.Get("X-Painel-Ok")) != "Intimação cancelada e mantida no histórico." {
		t.Errorf("cancelar = %d %q", r.StatusCode, r.Header.Get("X-Painel-Ok"))
	}
	if r, _ := adv.post("/intimacoes/"+ids[0]+"/decidir", url.Values{"status": {"confirmada"}}); r.StatusCode != 400 {
		t.Errorf("decidir de novo = %d, esperado 400", r.StatusCode)
	}
	if r, _ := adv.post("/intimacoes/"+ids[0]+"/decidir", url.Values{"status": {"talvez"}}); r.StatusCode != 400 {
		t.Errorf("decisão inválida = %d", r.StatusCode)
	}
	if r, _ := adv.post("/intimacoes/00000000-0000-0000-0000-000000000000/decidir", url.Values{"status": {"confirmada"}}); r.StatusCode != 404 {
		t.Errorf("inexistente = %d", r.StatusCode)
	}
	_, pg := adv.get("/agenda")
	if !strings.Contains(pg, `badge-ok">Confirmada</span>`) || !strings.Contains(pg, `badge-neutral">Cancelada</span>`) ||
		strings.Contains(pg, "/decidir") || !regexp.MustCompile(`<span>Intimações</span><span class="count">2</span>`).MatchString(pg) {
		t.Error("decididas ficam no histórico, com selo e sem botões")
	}
}

func (b *browser) semCSRFDecidir(id string) (int, string) {
	return b.semCSRF("POST", "/intimacoes/"+id+"/decidir", url.Values{"status": {"confirmada"}}), ""
}

func TestRodapeEstadoDasColetas(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv-rodape@exemplo.test", auth.Advogado)
	ctx := context.Background()
	_, pg := adv.get("/agenda")
	if !strings.Contains(pg, "Nenhuma coleta configurada neste painel. Não presuma que houve atualização; confira as fontes oficiais.") {
		t.Error("sem nenhuma coleta registrada, o rodapé mantém o texto honesto")
	}
	tent := time.Date(2026, 11, 3, 15, 4, 5, 0, time.UTC)
	if err := e.svc.Coletas.Registrar(ctx, coletasEstadoTeste("sucesso", "OABs: 1.", tent)); err != nil {
		t.Fatal(err)
	}
	_, pg = adv.get("/agenda")
	ordem(t, "rodapé", pg, `<section id="syncStatus" class="sync-status" aria-label="Estado das coletas" data-refresh="/sync/estado">`,
		`<strong>Estado das coletas</strong>`, `<p>Histórico local. Não comprova agendamento ativo nem substitui a conferência nas fontes oficiais.</p>`,
		`<ul><li>DJEN: última coleta concluída. Tentativa: 03/11/2026, 12:04:05 (Brasília). Sucesso: 03/11/2026, 12:04:05 (Brasília). OABs: 1.</li></ul>`)
	if strings.Contains(pg, `class="sync-error"`) {
		t.Error("sucesso não leva sync-error")
	}
	if err := e.svc.Coletas.Registrar(ctx, coletasEstadoTeste("falha", "djen: HTTP 500", tent.Add(time.Hour))); err != nil {
		t.Fatal(err)
	}
	_, pg = adv.get("/agenda")
	if !strings.Contains(pg, `<li class="sync-error">DJEN: erro. Tentativa: 03/11/2026, 13:04:05 (Brasília). Sucesso: 03/11/2026, 12:04:05 (Brasília). djen: HTTP 500</li>`) {
		t.Errorf("falha deve levar sync-error e manter o último sucesso:\n%s", trechoRodape(pg))
	}
	// O fragmento que o JS recarrega: só o miolo, para quem tem sessão.
	r, frag := adv.get("/sync/estado")
	if r.StatusCode != 200 || strings.Contains(frag, "<html") || !strings.HasPrefix(frag, "<strong>Estado das coletas</strong>") || !strings.Contains(frag, `class="sync-error"`) {
		t.Errorf("fragmento = %d %q", r.StatusCode, frag)
	}
	if r, _ := (&browser{e: e, sid: "x"}).get("/sync/estado"); r.StatusCode == 200 {
		t.Error("sem sessão o fragmento não pode ser servido")
	}
}

func trechoRodape(h string) string {
	i := strings.Index(h, `id="syncStatus"`)
	if i < 0 {
		return ""
	}
	return h[i:min(len(h), i+900)]
}

func TestFaixaDeAlertaDaColeta(t *testing.T) {
	// Desligada: nunca há faixa.
	e := newEnv(t)
	adv, _ := e.user("adv-off@exemplo.test", auth.Advogado)
	if _, pg := adv.get("/agenda"); strings.Contains(pg, "Coleta de intimações sem sucesso") {
		t.Error("coleta desligada não gera faixa de alerta")
	}
	// Ligada, sem sucesso há 7 h: a faixa .notice aparece na Agenda.
	e2, _, _, _, _ := agendaComDuasIntimacoes(t)
	adv2, _ := e2.user("adv-on@exemplo.test", auth.Advogado)
	agora := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	if _, err := e2.pool.Exec(context.Background(), `UPDATE coletas SET ultimo_sucesso=$1 WHERE modulo=$2`, agora.Add(-7*time.Hour), intimacoes.Modulo); err != nil {
		t.Fatal(err)
	}
	_, pg := adv2.get("/agenda")
	if !regexp.MustCompile(`<div class="notice" role="status"><strong>Coleta de intimações sem sucesso desde [^<]+</strong> — confira o DJEN manualmente\.</div>`).MatchString(pg) {
		t.Errorf("faixa de alerta ausente")
	}
	if _, err := e2.pool.Exec(context.Background(), `UPDATE coletas SET ultimo_sucesso=$1 WHERE modulo=$2`, agora.Add(-1*time.Hour), intimacoes.Modulo); err != nil {
		t.Fatal(err)
	}
	if _, pg := adv2.get("/agenda"); strings.Contains(pg, "Coleta de intimações sem sucesso") {
		t.Error("com sucesso recente não há faixa")
	}
}
