package app

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/br"
)

func TestProcessosCicloCompleto(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)
	num := br.MakeCNJ("0001234", "2024", "8", "26", "0100")

	// Número inválido; assistente sem escrita; sem número é permitido (processo ainda não distribuído).
	bad := []byte(num)
	bad[1] = '9'
	if r, body := adv.post("/processos", processoForm(string(bad))); r.StatusCode != 400 || !strings.Contains(body, "inválido") {
		t.Fatalf("CNJ inválido = %d %q", r.StatusCode, body)
	}
	if r, _ := ass.post("/processos", processoForm(num)); r.StatusCode != 403 {
		t.Fatalf("assistente criando processo = %d, esperado 403", r.StatusCode)
	}
	if r, _ := ass.get("/processos/novo"); r.StatusCode != 403 {
		t.Fatalf("assistente abrindo o formulário = %d, esperado 403", r.StatusCode)
	}
	if r, body := adv.post("/processos", processoForm(strings.NewReplacer(".", "", "-", "").Replace(num))); r.StatusCode != 200 || // sem pontuação
		r.Header.Get("X-Painel-Ok") != "Processo criado." {
		t.Fatalf("criar com número sem pontuação = %d %s", r.StatusCode, body)
	}
	if e.count(`SELECT count(*) FROM processos WHERE numero_cnj=$1`, num) != 1 {
		t.Fatal("o número deveria ser gravado normalizado")
	}
	if r, body := adv.post("/processos", processoForm(num)); r.StatusCode != 400 || !strings.Contains(body, "Já existe um processo cadastrado com este número.") {
		t.Fatalf("processo duplicado = %d %q", r.StatusCode, body)
	}
	a := adv.criarProcesso(t, "")
	b := adv.criarProcesso(t, "")
	if a == b || e.count(`SELECT count(*) FROM processos WHERE numero_cnj=''`) != 2 {
		t.Error("dois processos sem número deveriam ser aceitos")
	}
}

func TestProcessoComPartesResponsaveisEPendencias(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)
	maria := adv.criarPessoa(t, "Maria Exemplo", "529.982.247-25")
	banco := adv.criarPessoa(t, "Banco Teste", "")
	joao := adv.criarPessoa(t, "João Fictício", "")
	zelia := adv.criarPessoa(t, "Zélia Sozinha", "")
	_ = zelia
	num := br.MakeCNJ("0000300", "2025", "8", "26", "0100")

	f := url.Values{"numero_processo": {num}, "parte_pessoa": {maria, banco, ""}, "parte_papel": {"Autor", "Outro", ""},
		"parte_papel_outro": {"", "Fiador", ""}, "parte_cliente": {"1", "0", "0"}} // a última linha está em branco
	f.Add("responsavel", joao)
	f.Add("responsavel", "")
	f.Add("responsavel", maria)
	f.Set("valor_causa", "R$ 10.000,00")
	f.Set("sistema", "PJe")
	f.Set("classe", "Execução Fiscal")
	f.Set("tipo_contingencia", "Ativa")
	f.Set("distribuido_em", "2026-09-15")
	r, body := adv.post("/processos", f)
	if r.StatusCode != 200 {
		t.Fatalf("criar com partes = %d %s", r.StatusCode, body)
	}
	pid := r.Header.Get("X-Painel-Id")
	if e.count(`SELECT count(*) FROM processo_partes WHERE processo_id=$1`, pid) != 2 || e.count(`SELECT count(*) FROM processo_partes WHERE processo_id=$1 AND cliente`, pid) != 1 {
		t.Error("deveria haver 2 partes, 1 delas cliente (a linha em branco é descartada)")
	}
	if e.count(`SELECT count(*) FROM processo_partes WHERE processo_id=$1 AND papel='Fiador'`, pid) != 1 {
		t.Error("o papel 'Outro' deveria gravar o texto livre")
	}
	if e.count(`SELECT count(*) FROM processo_responsaveis WHERE processo_id=$1`, pid) != 2 {
		t.Error("deveria haver 2 responsáveis")
	}
	if e.count(`SELECT count(*) FROM processos WHERE id=$1 AND codigo='PRO. 0000001' AND titulo='MARIA EXEMPLO x BANCO TESTE' AND valor_causa=10000.00 AND situacao='ativo'`, pid) != 1 {
		t.Error("código, título derivado, valor e estado inicial")
	}

	// A lista mostra cliente, parte contrária, valor e o selo de pendências com a lista no title.
	_, lista := adv.get("/processos")
	for _, quer := range []string{"PRO. 0000001", "MARIA EXEMPLO", "BANCO TESTE", "R$ 10.000,00", "15/09/2026", `class="badge badge-warn"`,
		"Pendências (4)", "Instância; Matéria / ramo do direito; Comarca; Juízo",
		`placeholder="Buscar por número, classe, matéria ou parte…"`, "Arquivados definitivamente"} {
		if !strings.Contains(lista, quer) {
			t.Errorf("a lista deveria conter %q", quer)
		}
	}
	for _, rot := range []string{"Código", "Nº processo", "Classe / Matéria", "Cliente", "Parte contrária", "Valor da causa", "Distribuição", "Pendências"} {
		if !strings.Contains(lista, "<th>"+rot+"</th>") {
			t.Errorf("cabeçalho %q ausente", rot)
		}
	}
	for _, acao := range []string{"Editar", "Mover estado", "Andamento", "Financeiro", "Próxima providência", "Excluir"} {
		if !strings.Contains(lista, ">"+acao+"</button>") {
			t.Errorf("ação %q ausente", acao)
		}
	}
	if inlineProibido.MatchString(lista) {
		t.Errorf("lista com atributo inline: %q", inlineProibido.FindString(lista))
	}
	_, listaAss := ass.get("/processos")
	if strings.Contains(listaAss, ">Mover estado<") || strings.Contains(listaAss, "Novo processo") {
		t.Error("quem só lê não vê as ações de escrita")
	}

	// Pessoas: as categorias vêm dos processos (cliente / parte adversa / sem processo).
	_, pes := adv.get("/pessoas")
	if !strings.Contains(pes, `data-acc="pessoas:cliente"`) || !strings.Contains(pes, "Nenhuma pessoa nesta categoria.") {
		t.Error("acordeão de pessoas")
	}
	for _, c := range []struct{ grupo, nome string }{{"cliente", "MARIA EXEMPLO"}, {"parte_contraria", "BANCO TESTE"}, {"parte_contraria", "JOÃO FICTÍCIO"}, {"sem_processo", "ZÉLIA SOZINHA"}} {
		ini := strings.Index(pes, `<div class="acc-item" data-acc="pessoas:`+c.grupo+`"`)
		if ini < 0 {
			t.Fatalf("grupo %s ausente", c.grupo)
		}
		fim := strings.Index(pes[ini:], `</div>
    </div>`)
		if fim < 0 || !strings.Contains(pes[ini:ini+fim], c.nome) {
			t.Errorf("%s deveria estar no grupo %s", c.nome, c.grupo)
		}
	}
	// João é só responsável pelo cadastro: pela regra literal do B (qualquer vínculo que não seja cliente/terceiro) cai em Parte adversa.
}

func TestProcessoAcoesMoverAndamentoFinanceiroProvidencia(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	pid := adv.criarProcesso(t, br.MakeCNJ("0000400", "2025", "8", "26", "0100"))
	ctx := context.Background()

	// Drawers: fragmentos com os textos do B.
	for rota, quer := range map[string][]string{
		"estado":      {"<h2>Mover estado do processo</h2>", "Novo estado *", `<option value="arquivado">arquivado</option>`, "Ex.: trânsito em julgado, acordo homologado, suspensão…", ">Mover</button>"},
		"andamento":   {"<h2>Registrar andamento</h2>", "O que aconteceu *", "Documento relacionado", "Padrão: hoje.", "Ex.: Juntada de contestação"},
		"financeiro":  {"<h2>Atualizar financeiro</h2>", "Item *", `placeholder="Ex.: 1.500,00"`, `<option value="Parcelado">`, "Ex.: pago, pendente"},
		"providencia": {"<h2>Próxima providência</h2>", "O que fazer *", "Até quando", "Ex.: Protocolar recurso"},
	} {
		_, h := adv.get("/processos/" + pid + "/" + rota)
		if !strings.HasPrefix(strings.TrimSpace(h), "<h2>") {
			t.Errorf("%s: o fragmento deveria começar por <h2>", rota)
		}
		for _, q := range quer {
			if !strings.Contains(h, q) {
				t.Errorf("%s: faltou %q", rota, q)
			}
		}
		if inlineProibido.MatchString(h) {
			t.Errorf("%s: atributo inline %q", rota, inlineProibido.FindString(h))
		}
	}

	// Mover estado: grava no andamento (com o motivo), audita; versão velha → 409; estado igual → recusa.
	if r, body := adv.post("/processos/"+pid+"/estado", url.Values{"version": {"1"}, "novo_estado": {"sobrestado"}, "motivo": {"acordo de teste"}}); r.StatusCode != 200 ||
		r.Header.Get("X-Painel-Ok") != "Estado do processo atualizado." {
		t.Fatalf("mover estado = %d %s", r.StatusCode, body)
	}
	if e.count(`SELECT count(*) FROM processos WHERE id=$1 AND situacao='sobrestado'`, pid) != 1 ||
		e.count(`SELECT count(*) FROM processo_andamentos WHERE processo_id=$1 AND o_que LIKE '%acordo de teste%'`, pid) != 1 ||
		e.count(`SELECT count(*) FROM audit_events WHERE action='processo.estado_movido' AND target_id=$1`, pid) != 1 {
		t.Error("mover estado: estado, motivo no andamento e auditoria")
	}
	if r, _ := adv.post("/processos/"+pid+"/estado", url.Values{"version": {"1"}, "novo_estado": {"extinto"}}); r.StatusCode != 409 {
		t.Errorf("versão velha = %d, esperado 409", r.StatusCode)
	}
	if r, _ := adv.post("/processos/"+pid+"/estado", url.Values{"version": {"2"}, "novo_estado": {"sobrestado"}}); r.StatusCode != 400 {
		t.Errorf("mesmo estado = %d, esperado 400", r.StatusCode)
	}
	if r, _ := adv.post("/processos/"+pid+"/estado", url.Values{"version": {"2"}, "novo_estado": {"suspenso"}}); r.StatusCode != 400 {
		t.Errorf("estado antigo 'suspenso' = %d, esperado 400", r.StatusCode)
	}

	// Andamento: ordenado por data (mais recente primeiro); texto obrigatório.
	for _, d := range []string{"2026-01-10", "2026-03-05", "2026-02-01"} {
		if r, body := adv.post("/processos/"+pid+"/andamento", url.Values{"o_que": {"Evento de " + d}, "data": {d}}); r.StatusCode != 200 || r.Header.Get("X-Painel-Ok") != "Andamento registrado." {
			t.Fatalf("andamento = %d %s", r.StatusCode, body)
		}
	}
	if r, _ := adv.post("/processos/"+pid+"/andamento", url.Values{"o_que": {"  "}}); r.StatusCode != 400 {
		t.Errorf("andamento vazio = %d", r.StatusCode)
	}
	_, det := adv.get("/processos/" + pid)
	i1, i2, i3 := strings.Index(det, "Evento de 2026-03-05"), strings.Index(det, "Evento de 2026-02-01"), strings.Index(det, "Evento de 2026-01-10")
	if i1 < 0 || !(i1 < i2 && i2 < i3) {
		t.Errorf("andamentos fora de ordem: %d %d %d", i1, i2, i3)
	}

	// Financeiro: o mesmo item atualiza em vez de duplicar.
	for _, v := range []string{"R$ 1.000,00", "R$ 1.500,00"} {
		if r, _ := adv.post("/processos/"+pid+"/financeiro", url.Values{"item": {"Honorários"}, "valor": {v}, "situacao": {"Pendente"}}); r.StatusCode != 200 ||
			r.Header.Get("X-Painel-Ok") != "Dado financeiro atualizado." {
			t.Fatalf("financeiro = %d", r.StatusCode)
		}
	}
	if e.count(`SELECT count(*) FROM processo_financeiro WHERE processo_id=$1 AND valor='R$ 1.500,00'`, pid) != 1 || e.count(`SELECT count(*) FROM processo_financeiro`) != 1 {
		t.Error("o item financeiro repetido deveria ser atualizado, não duplicado")
	}

	// Próxima providência.
	if r, _ := adv.post("/processos/"+pid+"/providencia", url.Values{"o_que": {"Protocolar recurso"}, "ate_quando": {"2026-12-01"}, "quem": {"Advogado Exemplo"}}); r.StatusCode != 200 ||
		r.Header.Get("X-Painel-Ok") != "Próxima providência registrada." {
		t.Fatalf("providência = %d", r.StatusCode)
	}
	if e.count(`SELECT count(*) FROM processos WHERE id=$1 AND prox_o_que='Protocolar recurso' AND prox_ate_quando='2026-12-01'`, pid) != 1 {
		t.Error("próxima providência não gravada")
	}
	if r, _ := adv.post("/processos/"+pid+"/providencia", url.Values{"o_que": {"x"}, "ate_quando": {"31/12/2026"}}); r.StatusCode != 400 {
		t.Errorf("data inválida = %d", r.StatusCode)
	}
	for _, a := range []string{"processo.criado", "processo.andamento_registrado", "processo.financeiro_atualizado", "processo.proxima_providencia"} {
		if e.count(`SELECT count(*) FROM audit_events WHERE action=$1`, a) == 0 {
			t.Errorf("faltou auditoria %q", a)
		}
	}
	_ = ctx
}

func TestProcessoEdicaoConflitoEExclusao(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	num := br.MakeCNJ("0000500", "2025", "8", "26", "0100")
	pid := adv.criarProcesso(t, num)

	_, ed := adv.get("/processos/" + pid + "/editar")
	for _, q := range []string{"<h2>Editar processo</h2>", num, "Título da ficha", "Polo", "Projeto vinculado",
		`Estado (ativo/sobrestado/...) não se edita aqui — use "Mover estado" na lista.`, "Salvar alterações",
		`value="` + num + `"`} {
		if !strings.Contains(ed, q) {
			t.Errorf("drawer de edição sem %q", q)
		}
	}
	f := processoForm(num)
	f.Set("version", "1")
	f.Set("comarca", "São Paulo")
	f.Set("titulo", "Título editado")
	if r, _ := adv.post("/processos/"+pid, f); r.StatusCode != 200 || r.Header.Get("X-Painel-Ok") != "Processo atualizado." {
		t.Fatalf("alteração = %d", r.StatusCode)
	}
	if r, _ := adv.post("/processos/"+pid, f); r.StatusCode != 409 {
		t.Fatalf("versão velha = %d, esperado 409", r.StatusCode)
	}
	if e.count(`SELECT count(*) FROM processos WHERE id=$1 AND comarca='São Paulo' AND titulo='Título editado' AND situacao='ativo'`, pid) != 1 {
		t.Error("alteração não gravada (e o estado não pode mudar por aqui)")
	}

	// Exclusão lógica: com tarefa em aberto pede confirmação extra.
	adv.criarTarefa(t, tarefaForm("processo", pid, "Tarefa vinculada"))
	_, conf := adv.get("/processos/" + pid + "/excluir")
	if !strings.Contains(conf, "Há 1 tarefa(s)/prazo(s) em aberto vinculados.") || !strings.Contains(conf, "Excluir esta ficha? Ela deixa de aparecer nas listas") {
		t.Error("o drawer de exclusão deveria avisar do trabalho em aberto")
	}
	if r, body := adv.post("/processos/"+pid+"/excluir", url.Values{"version": {"2"}}); r.StatusCode != 400 || !strings.Contains(body, "Há 1 tarefa(s)/prazo(s) em aberto vinculados.") {
		t.Fatalf("sem confirmação extra = %d %q", r.StatusCode, body)
	}
	if r, _ := adv.post("/processos/"+pid+"/excluir", url.Values{"version": {"2"}, "confirmo": {"1"}}); r.StatusCode != 200 || r.Header.Get("X-Painel-Ok") != "Processo excluído." {
		t.Fatalf("exclusão = %d", r.StatusCode)
	}
	if e.count(`SELECT count(*) FROM processos WHERE id=$1 AND excluido_em IS NOT NULL`, pid) != 1 {
		t.Error("exclusão deveria apenas marcar")
	}
	if _, lista := adv.get("/processos"); strings.Contains(lista, num) {
		t.Error("processo excluído ainda aparece")
	}
}

func TestExcluirPessoaLevaAoProcessoDeOrigem(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ana := adv.criarPessoa(t, "Ana Origem", "")
	bia := adv.criarPessoa(t, "Bia Origem", "")
	p1 := adv.criarProcesso(t, br.MakeCNJ("0000600", "2025", "8", "26", "0100"), [3]string{ana, "Autor", "1"}, [3]string{bia, "Réu", "0"})
	p2 := adv.criarProcesso(t, br.MakeCNJ("0000700", "2025", "8", "26", "0100"), [3]string{bia, "Terceiro interessado", "0"})

	// Pessoa em um só processo: o "Excluir" abre esse processo (redireciona para o drawer de edição).
	r, _ := adv.get("/pessoas/" + ana + "/excluir")
	if r.StatusCode != 303 || r.Header.Get("Location") != "/processos/"+p1+"/editar?da_pessoa="+ana {
		t.Fatalf("1 processo: %d %s", r.StatusCode, r.Header.Get("Location"))
	}
	_, ed := adv.get("/processos/" + p1 + "/editar?da_pessoa=" + ana)
	if !strings.Contains(ed, "Esta pessoa é Autor neste processo — edite ou apague o campo correspondente abaixo.") {
		t.Error("aviso do B ao abrir o processo a partir da pessoa")
	}
	// Em vários: "Escolha o processo".
	_, esc := adv.get("/pessoas/" + bia + "/excluir")
	if !strings.Contains(esc, "<h2>Escolha o processo</h2>") || !strings.Contains(esc, "Esta pessoa aparece em mais de um processo.") ||
		!strings.Contains(esc, "/processos/"+p2+"/editar?da_pessoa="+bia) || !strings.Contains(esc, "— Terceiro interessado") {
		t.Errorf("Escolha o processo: %.400s", esc)
	}
	// O servidor também recusa excluir pessoa vinculada.
	if r, body := adv.post("/pessoas/"+bia+"/excluir", url.Values{"version": {"1"}}); r.StatusCode != 400 || !strings.Contains(body, "parte em processo") {
		t.Fatalf("excluir pessoa vinculada = %d %q", r.StatusCode, body)
	}
	// Categoria: Bia é só terceira interessada → grupo próprio.
	_, pes := adv.get("/pessoas")
	ini := strings.Index(pes, `data-acc="pessoas:terceiro_interessado"`)
	if ini < 0 {
		t.Fatal("sem grupo de terceiro interessado")
	}
	if !strings.Contains(pes[strings.Index(pes, `<div class="acc-item" data-acc="pessoas:terceiro_interessado"`):], "BIA ORIGEM") {
		t.Error("Bia deveria estar em Terceiro interessado")
	}
	_ = e
}

// Operação multi-entidade: se a auditoria falhar no meio, NADA fica gravado (transação única).
func TestFalhaNoMeioNaoDeixaEstadoParcial(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	pes := adv.criarPessoa(t, "Carlos Dias", "")
	ctx := context.Background()
	if _, err := e.pool.Exec(ctx, `ALTER TABLE audit_events RENAME TO audit_events_off`); err != nil {
		t.Fatal(err)
	}
	r, _ := adv.post("/processos", processoForm(br.MakeCNJ("0000077", "2024", "8", "26", "0100"), [3]string{pes, "Autor", "1"}))
	if _, err := e.pool.Exec(ctx, `ALTER TABLE audit_events_off RENAME TO audit_events`); err != nil {
		t.Fatal(err)
	}
	if r.StatusCode < 500 {
		t.Fatalf("a falha deveria virar erro interno, veio %d", r.StatusCode)
	}
	if e.count(`SELECT count(*) FROM processos`) != 0 || e.count(`SELECT count(*) FROM processo_partes`) != 0 {
		t.Fatal("processo ou parte ficou gravado sem auditoria: estado parcial")
	}
	// O número do código também não foi consumido (a transação desfez).
	if r, _ := adv.post("/processos", processoForm("", [3]string{pes, "Autor", "1"})); r.StatusCode != 200 {
		t.Fatalf("repetição = %d", r.StatusCode)
	}
	if e.count(`SELECT count(*) FROM processos WHERE codigo='PRO. 0000001'`) != 1 {
		t.Error("o código deveria ser reaproveitado após a falha")
	}
}
