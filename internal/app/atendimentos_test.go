package app

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
)

// atendimentoForm monta o formulário do drawer "Novo atendimento" (campos do B).
func atendimentoForm(desc string, clientes []string, adversas ...string) url.Values {
	f := url.Values{"descricao": {desc}, "data_abertura": {"2026-11-03"}, "observacoes": {""}}
	for _, c := range clientes {
		f.Add("clientes", c)
	}
	for _, a := range adversas {
		f.Add("partes_adversas", a)
	}
	return f
}

func (b *browser) criarAtendimento(t *testing.T, desc string, clientes []string, adversas ...string) string {
	t.Helper()
	r, body := b.post("/atendimentos", atendimentoForm(desc, clientes, adversas...))
	if r.StatusCode != 200 || r.Header.Get("X-Painel-Id") == "" {
		t.Fatalf("criar atendimento = %d %s", r.StatusCode, body)
	}
	return r.Header.Get("X-Painel-Id")
}

func TestAtendimentosTela(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)
	adm, _ := e.user("adm@exemplo.test", auth.Administrador)
	maria := adv.criarPessoa(t, "Maria Exemplo", "529.982.247-25")
	joao := adv.criarPessoa(t, "João Fictício", "111.444.777-35")

	// Página vazia: textos do B.
	_, vazia := adv.get("/atendimentos")
	for _, w := range []string{"Atendimentos", "Registro das demandas e dos atendimentos do escritório.", "Novo atendimento",
		"Buscar por código, cliente, parte adversa ou objeto…", "Nenhum atendimento cadastrado. Use Novo atendimento para registrar uma demanda.",
		`class="tabela-atendimentos"`} {
		if !strings.Contains(vazia, w) {
			t.Errorf("página vazia sem %q", w)
		}
	}

	// Validação do B: ao menos um cliente.
	if r, body := adv.post("/atendimentos", atendimentoForm("Revisar contrato", nil, joao)); r.StatusCode != 400 || body != "Selecione ao menos um cliente.\n" {
		t.Errorf("sem cliente = %d %q", r.StatusCode, body)
	}
	r, _ := adv.post("/atendimentos", atendimentoForm("Revisar contrato de ação", []string{maria}, joao))
	if r.StatusCode != 200 || r.Header.Get("X-Painel-Ok") != "Atendimento criado." {
		t.Fatalf("criar = %d %q", r.StatusCode, r.Header.Get("X-Painel-Ok"))
	}
	id := r.Header.Get("X-Painel-Id")

	// Lista: colunas na ordem do B, nomes em MAIÚSCULAS, data dd/mm/aaaa, "Editar".
	_, lista := adv.get("/atendimentos")
	cab := "<th>Código</th><th>Cliente(s)</th><th>Parte(s) adversa(s)</th><th>Descrição / objeto</th><th>Data do cadastro</th><th></th>"
	for _, w := range []string{cab, "ATE. 0000001", "MARIA EXEMPLO", "JOÃO FICTÍCIO", "Revisar contrato de ação", "03/11/2026", `data-drawer="/atendimentos/` + id + `/editar"`} {
		if !strings.Contains(lista, w) {
			t.Errorf("lista sem %q", w)
		}
	}
	if inlineProibido.MatchString(lista) {
		t.Error("lista com HTML inline proibido")
	}

	// Drawer "Novo atendimento": rótulos na ordem do B.
	_, novo := adv.do("GET", "/atendimentos/novo", nil, map[string]string{"HX-Request": "true"})
	ordem := []string{"<h2>Novo atendimento</h2>", "Registro da demanda. Cada novo objeto deve ter seu próprio atendimento.", "Criado por",
		"Não informado", "Pessoa que criou este registro. Não define quem executará a atividade.", "<legend>Cliente(s)</legend>",
		"Buscar pessoa por nome ou código PES…", "Selecione uma pessoa", "+ Adicionar cliente", "+ Novo cliente",
		"<legend>Parte(s) adversa(s)</legend>", "+ Adicionar parte adversa", "+ Nova parte adversa", "Descrição / objeto",
		"Data do cadastro", "Observações complementares", "Cancelar", "Salvar atendimento"}
	pos := 0
	for _, w := range ordem {
		i := strings.Index(novo[pos:], w)
		if i < 0 {
			t.Fatalf("drawer novo: %q ausente ou fora de ordem", w)
		}
		pos += i + len(w)
	}
	if !strings.Contains(novo, "MARIA EXEMPLO · PES. 0000001") || inlineProibido.MatchString(novo) {
		t.Error("drawer sem as opções de pessoa ou com HTML inline")
	}
	_, edit := adv.do("GET", "/atendimentos/"+id+"/editar", nil, map[string]string{"HX-Request": "true"})
	if !strings.Contains(edit, "<h2>Editar atendimento</h2>") || !strings.Contains(edit, "Salvar alterações") ||
		!strings.Contains(edit, `value="`+maria+`" selected`) || !strings.Contains(edit, `name="version" value="1"`) {
		t.Error("drawer de edição incompleto")
	}

	// Edição; versão antiga conflita.
	f := atendimentoForm("Objeto novo", []string{joao})
	f.Set("version", "1")
	if r, _ := adv.post("/atendimentos/"+id, f); r.StatusCode != 200 || r.Header.Get("X-Painel-Ok") != "Atendimento atualizado." {
		t.Fatalf("editar = %d %q", r.StatusCode, r.Header.Get("X-Painel-Ok"))
	}
	if r, _ := adv.post("/atendimentos/"+id, f); r.StatusCode != 409 {
		t.Errorf("versão antiga = %d, esperado 409", r.StatusCode)
	}

	// Permissões: assistente escreve (mas sem "Novo cliente", que exige Pessoas); administrador só lê.
	_, assNovo := ass.do("GET", "/atendimentos/novo", nil, map[string]string{"HX-Request": "true"})
	if strings.Contains(assNovo, "+ Novo cliente") || !strings.Contains(assNovo, "+ Adicionar cliente") {
		t.Error("assistente não pode criar pessoa: sem botão 'Novo cliente'")
	}
	if r, _ := ass.post("/atendimentos", atendimentoForm("Do assistente", []string{maria})); r.StatusCode != 200 {
		t.Errorf("assistente criando = %d", r.StatusCode)
	}
	if r, _ := adm.get("/atendimentos"); r.StatusCode != 200 {
		t.Errorf("administrador lendo = %d", r.StatusCode)
	}
	_, admLista := adm.get("/atendimentos")
	if strings.Contains(admLista, "Novo atendimento</a>") || strings.Contains(admLista, "/editar") {
		t.Error("quem só lê não vê botões de escrita")
	}
	if r, _ := adm.post("/atendimentos", atendimentoForm("x", []string{maria})); r.StatusCode != 403 {
		t.Errorf("administrador escrevendo = %d, esperado 403", r.StatusCode)
	}
	if st := adv.semCSRF("POST", "/atendimentos", atendimentoForm("x", []string{maria})); st != 403 && st != 401 {
		t.Errorf("sem CSRF = %d", st)
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action IN ('atendimento.criado','atendimento.atualizado')`) != 3 {
		t.Error("auditoria dos atendimentos incompleta")
	}

	// Opções de pessoa para o rascunho (depois de "Novo cliente").
	_, ops := adv.get("/atendimentos/pessoas")
	if !strings.Contains(ops, "<option value=\"\">Selecione uma pessoa</option>") || !strings.Contains(ops, "JOÃO FICTÍCIO · PES. 0000002") {
		t.Errorf("opções = %s", ops)
	}
}

func TestProcessoComAtendimentoDeOrigem(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	maria := adv.criarPessoa(t, "Maria Exemplo", "529.982.247-25")
	ate := adv.criarAtendimento(t, "Demanda de origem", []string{maria})

	_, novo := adv.do("GET", "/processos/novo", nil, map[string]string{"HX-Request": "true"})
	if strings.Contains(novo, "Disponível quando o módulo Atendimentos for ativado") || strings.Contains(novo, `name="atendimento_origem" disabled`) {
		t.Error("o select de atendimento de origem deveria estar habilitado")
	}
	if !strings.Contains(novo, "ATE. 0000001 · Extrajudicial · MARIA EXEMPLO · 03/11/2026") {
		t.Errorf("opção do atendimento de origem ausente: %s", novo)
	}
	f := processoForm("")
	f.Set("atendimento_origem", ate)
	r, body := adv.post("/processos", f)
	if r.StatusCode != 200 {
		t.Fatalf("criar processo com origem = %d %s", r.StatusCode, body)
	}
	var got string
	if err := e.pool.QueryRow(context.Background(), `SELECT atendimento_origem_id::text FROM processos WHERE id=$1`, r.Header.Get("X-Painel-Id")).Scan(&got); err != nil || got != ate {
		t.Errorf("atendimento_origem_id = %q (%v)", got, err)
	}
	bad := processoForm("")
	bad.Set("atendimento_origem", "00000000-0000-0000-0000-000000000000")
	if r, body := adv.post("/processos", bad); r.StatusCode != 400 || !strings.Contains(body, "Atendimento de origem não encontrado.") {
		t.Errorf("origem inexistente = %d %q", r.StatusCode, body)
	}
}
