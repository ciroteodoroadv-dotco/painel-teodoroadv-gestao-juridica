package app

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/br"
)

func comJust(f url.Values, just string) url.Values {
	f.Set("duplicado_justificativa", just)
	return f
}

func soDigitos(s string) string { return regexp.MustCompile(`\D`).ReplaceAllString(s, "") }

func TestProcessoDuplicadoExigeMarcadorEJustificativa(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv-dup@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass-dup@exemplo.test", auth.Assistente)
	num := br.MakeCNJ("0000101", "2025", "8", "26", "0100")
	outro := br.MakeCNJ("0000202", "2025", "8", "26", "0100")
	ana := adv.criarPessoa(t, "Ana Souza", "")
	p1 := adv.criarProcesso(t, num, [3]string{ana, "Autor", "1"})
	if e.count(`SELECT count(*) FROM processos WHERE duplicado`) != 0 {
		t.Fatal("o primeiro processo com o número não é duplicado")
	}

	// Sem justificativa (ou curta demais), a duplicata é recusada, com ou sem pontuação no número.
	msg := "Informe a justificativa da duplicidade."
	for nome, f := range map[string]url.Values{
		"sem justificativa": processoForm(num), "sem pontuação": processoForm(soDigitos(num)),
		"justificativa curta": comJust(processoForm(num), "curto"), "só espaços": comJust(processoForm(num), "          "),
	} {
		if r, b := adv.post("/processos", f); r.StatusCode != 400 || strings.TrimSpace(b) != msg {
			t.Errorf("%s = %d %q", nome, r.StatusCode, b)
		}
	}
	if e.count(`SELECT count(*) FROM processos`) != 1 {
		t.Fatal("nenhuma duplicata pode ser gravada sem justificativa")
	}
	// Com justificativa, grava com o marcador, o processo de origem e a auditoria.
	r, b := adv.post("/processos", comJust(processoForm(soDigitos(num)), "Reautuado após extinção do primeiro processo"))
	if r.StatusCode != 200 || r.Header.Get("X-Painel-Ok") != "Processo criado." {
		t.Fatalf("duplicata justificada = %d %s", r.StatusCode, b)
	}
	p2 := r.Header.Get("X-Painel-Id")
	if e.count(`SELECT count(*) FROM processos WHERE id=$1::uuid AND duplicado AND duplicado_de=$2::uuid AND duplicado_justificativa='Reautuado após extinção do primeiro processo' AND numero_cnj=$3`, p2, p1, num) != 1 {
		t.Error("marcador, original e justificativa devem ser gravados")
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='processo.duplicado_justificado' AND target_id=$1`, p2) != 1 {
		t.Error("auditoria processo.duplicado_justificado")
	}
	// Terceiro com o mesmo número: aponta para o mais antigo.
	r, _ = adv.post("/processos", comJust(processoForm(num), "Terceira autuação, outro juízo"))
	if r.StatusCode != 200 || e.count(`SELECT count(*) FROM processos WHERE duplicado AND duplicado_de=$1::uuid`, p1) != 2 {
		t.Errorf("a duplicata aponta sempre para o processo mais antigo (%d)", r.StatusCode)
	}

	// Número vazio nunca é duplicata.
	adv.criarProcesso(t, "")
	adv.criarProcesso(t, "")
	if e.count(`SELECT count(*) FROM processos WHERE numero_cnj='' AND duplicado`) != 0 {
		t.Error("número vazio não é duplicata")
	}

	// Lista: selo em vermelho com a justificativa no title; o original não leva selo.
	_, pg := adv.get("/processos")
	if n := strings.Count(pg, `>PROCESSO DUPLICADO</span>`); n != 2 {
		t.Errorf("selos na lista = %d, esperado 2", n)
	}
	if !strings.Contains(pg, `<span class="badge badge-danger" title="Reautuado após extinção do primeiro processo">PROCESSO DUPLICADO</span>`) {
		t.Error("o selo leva a justificativa no title")
	}

	// Editar o próprio processo sem mudar o número não o marca como duplicata (de si mesmo nem do seu duplicado).
	ed := processoForm(num, [3]string{ana, "Autor", "1"})
	ed.Set("version", "1")
	if r, b := adv.post("/processos/"+p1, ed); r.StatusCode != 200 {
		t.Errorf("editar o original = %d %q", r.StatusCode, b)
	}
	if e.count(`SELECT count(*) FROM processos WHERE id=$1::uuid AND duplicado`, p1) != 0 {
		t.Error("o original não vira duplicata ao ser editado")
	}
	// Editar a duplicata sem mudar o número mantém o marcador e a justificativa.
	ed2 := processoForm(num)
	ed2.Set("version", "1")
	if r, b := adv.post("/processos/"+p2, ed2); r.StatusCode != 200 {
		t.Errorf("editar a duplicata = %d %q", r.StatusCode, b)
	}
	if e.count(`SELECT count(*) FROM processos WHERE id=$1::uuid AND duplicado AND duplicado_justificativa<>''`, p2) != 1 {
		t.Error("a duplicata mantém o marcador e a justificativa")
	}
	// Mudar um processo para o número de outro: mesma regra.
	p4 := adv.criarProcesso(t, outro)
	mv := processoForm(num)
	mv.Set("version", "1")
	if r, b := adv.post("/processos/"+p4, mv); r.StatusCode != 400 || strings.TrimSpace(b) != msg {
		t.Errorf("mudar para número existente sem justificativa = %d %q", r.StatusCode, b)
	}
	comJust(mv, "Migrado do sistema antigo com o mesmo número")
	if r, b := adv.post("/processos/"+p4, mv); r.StatusCode != 200 {
		t.Errorf("mudar para número existente com justificativa = %d %q", r.StatusCode, b)
	}
	if e.count(`SELECT count(*) FROM processos WHERE id=$1::uuid AND duplicado AND duplicado_de=$2::uuid`, p4, p1) != 1 {
		t.Error("virou duplicata do original")
	}
	// Corrigir o número desfaz o marcador, mas a auditoria fica.
	fix := processoForm(br.MakeCNJ("0000303", "2025", "8", "26", "0100"))
	fix.Set("version", "2")
	if r, b := adv.post("/processos/"+p4, fix); r.StatusCode != 200 {
		t.Errorf("corrigir o número = %d %q", r.StatusCode, b)
	}
	if e.count(`SELECT count(*) FROM processos WHERE id=$1::uuid AND NOT duplicado AND duplicado_justificativa='' AND duplicado_de IS NULL`, p4) != 1 {
		t.Error("número corrigido: o marcador some")
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='processo.duplicado_justificado' AND target_id=$1`, p4) != 1 {
		t.Error("a justificativa anterior continua na auditoria")
	}

	// Consulta antes de salvar (POST: o número do processo não vai na URL).
	r, bloco := adv.post("/processos/duplicado", url.Values{"numero": {soDigitos(num)}, "ignorar": {""}})
	for _, w := range []string{`class="form-block"`, `<span class="badge badge-danger">PROCESSO DUPLICADO</span>`, "Já existe o processo PRO. 0000001 (ANA SOUZA) com este número.",
		"Justificativa da duplicidade *", `<textarea name="duplicado_justificativa"`} {
		if r.StatusCode != 200 || !strings.Contains(bloco, w) {
			t.Errorf("bloco sem %q (%d)", w, r.StatusCode)
		}
	}
	for nome, f := range map[string]url.Values{"número livre": {"numero": {br.MakeCNJ("0000999", "2025", "8", "26", "0100")}}, "vazio": {"numero": {""}},
		"inválido": {"numero": {"123"}}} {
		if _, b := adv.post("/processos/duplicado", f); strings.TrimSpace(b) != "" {
			t.Errorf("%s: não deveria haver bloco: %q", nome, b)
		}
	}
	// Ignorando o próprio processo, e sem outro (as duplicatas excluídas não contam), não há duplicidade.
	_, _ = e.pool.Exec(context.Background(), `UPDATE processos SET excluido_em=now() WHERE duplicado`)
	if _, b := adv.post("/processos/duplicado", url.Values{"numero": {num}, "ignorar": {p1}}); strings.TrimSpace(b) != "" {
		t.Errorf("ignorando o próprio: não deveria haver bloco: %q", b)
	}
	if r, _ := ass.post("/processos/duplicado", url.Values{"numero": {num}}); r.StatusCode != 403 {
		t.Errorf("assistente consultando = %d", r.StatusCode)
	}
}

func TestDrawerDoProcessoTrazAVerificacaoDeDuplicidade(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv-dup2@exemplo.test", auth.Advogado)
	num := br.MakeCNJ("0000101", "2025", "8", "26", "0100")
	adv.criarProcesso(t, num)
	p2 := ""
	if r, _ := adv.post("/processos", comJust(processoForm(num), "Reautuado após extinção do primeiro")); r.StatusCode == 200 {
		p2 = r.Header.Get("X-Painel-Id")
	}
	_, novo := adv.fragDrawer("/processos/novo")
	if !strings.Contains(novo, `data-verifica-duplicado="/processos/duplicado"`) || !strings.Contains(novo, `data-duplicado-alvo`) {
		t.Error("o campo do número verifica a duplicidade ao sair, antes de salvar")
	}
	if strings.Contains(novo, "PROCESSO DUPLICADO") {
		t.Error("processo novo não mostra o bloco")
	}
	_, ed := adv.fragDrawer("/processos/" + p2 + "/editar")
	ordem(t, "editar duplicado", ed, `id="prNumero"`, `PROCESSO DUPLICADO`, "Já existe o processo PRO. 0000001", "Justificativa da duplicidade *", "Reautuado após extinção do primeiro")
}
