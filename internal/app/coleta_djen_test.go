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
	"painelteodoro/internal/platform/coletas"
)

func coletasEstadoTeste(status, msg string, t time.Time) coletas.Estado {
	return coletas.Estado{Modulo: intimacoes.Modulo, Origem: "DJEN", Status: status, Mensagem: msg, UltimaTentativa: &t}
}

func formOAB(nome, numero, uf, base, data string) url.Values {
	return url.Values{"nome": {nome}, "numero": {numero}, "uf": {uf}, "autorizacao_base": {base}, "autorizacao_em": {data}, "ativa": {"1"}}
}

func TestPaginaColetaDJENSoAdministrador(t *testing.T) {
	e := newEnv(t)
	adm, _ := e.user("adm-pg@exemplo.test", auth.Administrador)
	adv, _ := e.user("adv-pg@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass-pg@exemplo.test", auth.Assistente)
	for nome, b := range map[string]*browser{"advogado": adv, "assistente": ass} {
		for _, c := range []string{"/coleta-djen", "/coleta-djen/nova"} {
			if r, _ := b.get(c); r.StatusCode != 403 {
				t.Errorf("%s em GET %s = %d, esperado 403", nome, c, r.StatusCode)
			}
		}
		if r, _ := b.post("/coleta-djen", formOAB("X Y", "1", "SP", "Base", "2026-11-01")); r.StatusCode != 403 {
			t.Errorf("%s criando OAB = %d", nome, r.StatusCode)
		}
		if r, _ := b.post("/coleta-djen/executar", nil); r.StatusCode != 403 {
			t.Errorf("%s executando coleta = %d", nome, r.StatusCode)
		}
		if _, pg := b.get("/agenda"); strings.Contains(pg, "Coleta DJEN") {
			t.Errorf("%s não vê o item Coleta DJEN no menu", nome)
		}
	}
	r, pg := adm.get("/coleta-djen")
	if r.StatusCode != 200 {
		t.Fatalf("admin = %d", r.StatusCode)
	}
	ordem(t, "página", pg, `<h1>Coleta DJEN</h1>`, `Nova OAB`, `Coleta desligada neste ambiente.`)
	if strings.Contains(pg, "Executar coleta agora") {
		t.Error("com a coleta desligada não há botão de executar")
	}
	if !strings.Contains(pg, `href="/coleta-djen"`) {
		t.Error("o administrador vê Coleta DJEN no grupo do sistema")
	}
	if !strings.Contains(pg, "O DJEN não é em tempo real") || !strings.Contains(pg, "segredo de justiça") || !strings.Contains(pg, "nunca significa") {
		t.Error("a página deve avisar: não é tempo real, segredo de justiça, sem resultado não é sem prazo")
	}
	if strings.Contains(strings.ToLower(pg), "consentimento") {
		t.Error("a página não fala em consentimento individual do advogado parceiro")
	}
}

func TestCadastroDeOABComBaseEDataDaAutorizacao(t *testing.T) {
	e := newEnv(t)
	adm, uid := e.user("adm-oab@exemplo.test", auth.Administrador)
	_, frag := adm.do("GET", "/coleta-djen/nova", nil, map[string]string{"HX-Request": "true"})
	ordem(t, "drawer", frag, "Nome do advogado", "Número da OAB", "UF", "Base da autorização", intimacoes.BaseAutorizacaoPadrao, "Data da autorização", "Ativa")
	hoje := time.Now().In(time.FixedZone("BRT", -3*3600)).Format("2006-01-02")
	if !strings.Contains(frag, `value="`+hoje+`"`) && !strings.Contains(frag, `name="autorizacao_em"`) {
		t.Error("data da autorização no formulário")
	}
	if strings.Contains(strings.ToLower(frag), "consentimento") {
		t.Error("sem a palavra consentimento")
	}
	for nome, f := range map[string]url.Values{
		"sem base": formOAB("Advogado Fictício", "123456", "SP", " ", "2026-11-01"),
		"sem data": formOAB("Advogado Fictício", "123456", "SP", "Base", ""),
		"sem uf":   formOAB("Advogado Fictício", "123456", "", "Base", "2026-11-01"),
		"número":   formOAB("Advogado Fictício", "12a456", "SP", "Base", "2026-11-01"),
	} {
		if r, _ := adm.post("/coleta-djen", f); r.StatusCode != 400 {
			t.Errorf("%s: %d, esperado 400", nome, r.StatusCode)
		}
	}
	r, _ := adm.post("/coleta-djen", formOAB("Advogado Fictício", "123456", "sp", "Distribuição dos processos em nome do advogado", "2026-11-01"))
	if r.StatusCode != 200 || decodifica(t, r.Header.Get("X-Painel-Ok")) != "OAB cadastrada." {
		t.Fatalf("cadastro = %d %q", r.StatusCode, r.Header.Get("X-Painel-Ok"))
	}
	if e.count(`SELECT count(*) FROM oab_monitoradas WHERE uf='SP' AND created_by=$1::uuid AND ativa`, uid) != 1 {
		t.Error("OAB gravada com UF em maiúsculas e quem cadastrou")
	}
	if r, _ := adm.post("/coleta-djen", formOAB("Outro Nome", "123456", "SP", "Base", "2026-11-01")); r.StatusCode != 400 {
		t.Errorf("OAB repetida = %d", r.StatusCode)
	}
	_, pg := adm.get("/coleta-djen")
	for _, trecho := range []string{"Advogado Fictício", "123456/SP", "Distribuição dos processos em nome do advogado", "01/11/2026", "Ativa"} {
		if !strings.Contains(pg, trecho) {
			t.Errorf("tabela de OABs sem %q", trecho)
		}
	}
	var id string
	_ = e.pool.QueryRow(context.Background(), `SELECT id::text FROM oab_monitoradas`).Scan(&id)
	_, ed := adm.do("GET", "/coleta-djen/"+id+"/editar", nil, map[string]string{"HX-Request": "true"})
	if !strings.Contains(ed, `value="Advogado Fictício"`) || !strings.Contains(ed, `name="version"`) {
		t.Error("o drawer de edição traz os dados e a versão")
	}
	if r, _ := adm.post("/coleta-djen/"+id, url.Values{"version": {"1"}, "nome": {"Advogado Renomeado"}, "autorizacao_base": {"Nova base"}, "autorizacao_em": {"2026-11-02"}}); r.StatusCode != 200 {
		t.Errorf("editar = %d", r.StatusCode)
	}
	if e.count(`SELECT count(*) FROM oab_monitoradas WHERE advogado_nome='Advogado Renomeado' AND NOT ativa AND version=2`) != 1 {
		t.Error("edição grava os dados, desativa (checkbox desmarcado) e sobe a versão")
	}
	if r, _ := adm.post("/coleta-djen/"+id, url.Values{"version": {"1"}, "nome": {"X Y"}, "autorizacao_base": {"b b"}, "autorizacao_em": {"2026-11-02"}}); r.StatusCode != 409 {
		t.Errorf("versão velha = %d, esperado 409", r.StatusCode)
	}
}

func TestExecutarColetaAgora(t *testing.T) {
	// Desligada: recusa com a mensagem do estado, sem rede.
	e := newEnv(t)
	adm, _ := e.user("adm-exec@exemplo.test", auth.Administrador)
	r, body := adm.post("/coleta-djen/executar", nil)
	if r.StatusCode != 400 || !strings.Contains(body, "Coleta desligada neste ambiente.") {
		t.Errorf("desligada = %d %q", r.StatusCode, body)
	}
	// Ligada, com o DJEN falso.
	f := &falsoDJEN{itens: []djen.Comunicacao{comunicacao("1", "")}}
	e2, _, _ := ambDJEN(t, f)
	adm2, _ := e2.user("adm-exec2@exemplo.test", auth.Administrador)
	_, pg := adm2.get("/coleta-djen")
	if !strings.Contains(pg, "Coleta ativa") || !strings.Contains(pg, "Executar coleta agora") {
		t.Error("ligada: mostra o estado e o botão")
	}
	r, _ = adm2.post("/coleta-djen/executar", nil)
	if r.StatusCode != 200 || !strings.HasPrefix(decodifica(t, r.Header.Get("X-Painel-Ok")), "Coleta concluída") {
		t.Fatalf("executar = %d %q", r.StatusCode, r.Header.Get("X-Painel-Ok"))
	}
	if f.n() != 1 || e2.count(`SELECT count(*) FROM intimacoes`) != 1 {
		t.Errorf("chamadas=%d", f.n())
	}
	if msg := decodifica(t, r.Header.Get("X-Painel-Ok")); regexp.MustCompile(`(?i)nenhuma intimação nova|sem prazo`).MatchString(msg) {
		t.Errorf("a mensagem não pode soar como garantia: %q", msg)
	}
	// Falha do DJEN: erro claro, estado "falha" e janela intacta.
	f.mu.Lock()
	f.err = context.DeadlineExceeded
	f.mu.Unlock()
	r, body = adm2.post("/coleta-djen/executar", nil)
	if r.StatusCode != 500 && r.StatusCode != 400 {
		t.Errorf("falha do DJEN = %d %q", r.StatusCode, body)
	}
	est, _, _ := e2.svc.Coletas.Obter(context.Background(), intimacoes.Modulo)
	if est.Status != "falha" {
		t.Errorf("estado = %+v", est)
	}
}
