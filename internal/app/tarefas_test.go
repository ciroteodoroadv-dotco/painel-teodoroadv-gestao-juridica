package app

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/auth"
)

// tarefaForm monta o formulário do drawer "Nova tarefa" (campos do B).
func tarefaForm(tipo, origem, titulo string) url.Values {
	return url.Values{"origem_tipo": {tipo}, "origem_id": {origem}, "titulo": {titulo}, "estado": {"pendente"}, "prioridade": {"Normal"}}
}

func (b *browser) criarTarefa(t *testing.T, f url.Values) string {
	t.Helper()
	r, body := b.post("/tarefas", f)
	if r.StatusCode != 200 || r.Header.Get("X-Painel-Id") == "" {
		t.Fatalf("criar tarefa = %d %s", r.StatusCode, body)
	}
	return r.Header.Get("X-Painel-Id")
}

func (b *browser) frag(path string) string {
	b.e.t.Helper()
	_, body := b.do("GET", path, nil, map[string]string{"HX-Request": "true"})
	return body
}

// enviarDocTarefa faz o POST multipart que o upload.js faz ao escolher o PDF.
func (b *browser) enviarDocTarefa(id, nome string, conteudo []byte) (*http.Response, string) {
	b.e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("documento", nome)
	_, _ = fw.Write(conteudo)
	_ = mw.Close()
	req, _ := http.NewRequest("POST", b.e.srv.URL+"/tarefas/"+id+"/documentos", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-CSRF-Token", b.token())
	req.AddCookie(&http.Cookie{Name: "__Host-sid", Value: b.sid})
	resp, err := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(req)
	if err != nil {
		b.e.t.Fatal(err)
	}
	defer resp.Body.Close()
	var out bytes.Buffer
	_, _ = out.ReadFrom(resp.Body)
	return resp, out.String()
}

var (
	reTemplate = regexp.MustCompile(`(?s)<template.*?</template>`)
	reRotulo   = regexp.MustCompile(`(?s)<(?:label|legend)[^>]*>(.*?)</(?:label|legend)>`)
	reTag      = regexp.MustCompile(`(?s)<[^>]*>`)
)

// rotulos extrai, em ordem, o texto de todos os <label>/<legend> fora dos <template> (a lista que o B mostra ao abrir).
func rotulos(html string) []string {
	html = reTemplate.ReplaceAllString(html, "")
	var out []string
	for _, m := range reRotulo.FindAllStringSubmatch(html, -1) {
		if t := strings.TrimSpace(reTag.ReplaceAllString(m[1], "")); t != "" {
			out = append(out, t)
		}
	}
	return out
}

type cenario struct {
	e                *env
	adv, ass, adm    *browser
	maria, joao, ate string
	pro              string
}

func novoCenario(t *testing.T) *cenario {
	e := newEnv(t)
	c := &cenario{e: e}
	c.adv, _ = e.user("adv@exemplo.test", auth.Advogado)
	c.ass, _ = e.user("ass@exemplo.test", auth.Assistente)
	c.adm, _ = e.user("adm@exemplo.test", auth.Administrador)
	c.maria = c.adv.criarPessoa(t, "Maria Exemplo", "529.982.247-25")
	c.joao = c.adv.criarPessoa(t, "João Fictício", "111.444.777-35")
	c.ate = c.adv.criarAtendimento(t, "Demanda de exemplo", []string{c.maria})
	c.pro = c.adv.criarProcesso(t, "", [3]string{c.joao, "Autor", "1"})
	e.semMonitoramento() // estes testes contam tarefas: a de monitoramento (Plano 6) tem testes próprios
	return c
}

func TestTarefasPaginaEDrawersComOsTextosDoB(t *testing.T) {
	c := novoCenario(t)
	// Sem tarefas: estado vazio do B.
	_, vazia := c.adv.get("/tarefas")
	for _, w := range []string{"Toda tarefa nasce de um atendimento (ordem de serviço) ou de um processo.", "Nova tarefa",
		`class="toolbar tarefa-search"`, "Buscar por título, cliente, código ou responsável…", "Buscar</button>",
		"<h3>Nenhuma tarefa cadastrada</h3>", `Crie a primeira tarefa pelo botão "Nova tarefa" acima.`} {
		if !strings.Contains(vazia, w) {
			t.Errorf("página vazia sem %q", w)
		}
	}
	id := c.adv.criarTarefa(t, func() url.Values {
		f := tarefaForm("atendimento", c.ate, "Protocolar embargos à execução")
		f.Set("subtitulo", "Ligação")
		f.Set("descricao", "Conferir os cálculos")
		f.Set("prazo", "2026-11-10")
		f.Set("prazo_fatal", "2026-11-12")
		f.Set("prioridade", "Urgente")
		f.Set("criado_por", c.joao)
		f.Add("responsavel", c.joao)
		f.Add("parte_pessoa", c.maria)
		f.Add("parte_papel", "Autor")
		f.Add("parte_papel_outro", "")
		return f
	}())
	_, pg := c.adv.get("/tarefas")

	// 7 painéis do acordeão, nesta ordem e com estes rótulos.
	var heads []string
	for _, m := range regexp.MustCompile(`<button class="acc-head" data-acc="tarefas:([a-z-]+)"><span>([^<]+)</span><span class="count">(\d+)</span>`).FindAllStringSubmatch(pg, -1) {
		heads = append(heads, m[1]+"|"+m[2]+"|"+m[3])
	}
	want := []string{"a-confirmar|A confirmar|0", "pendente|Pendente|1", "em-execucao|Em execução|0", "revisao|Revisão|0",
		"concluidas-com-sucesso|Concluída com sucesso|0", "concluidas-sem-sucesso|Concluída sem sucesso|0", "canceladas|Cancelada|0"}
	if !reflect.DeepEqual(heads, want) {
		t.Errorf("acc-heads = %v", heads)
	}
	// Linha da tarefa: estrutura do B.
	for _, w := range []string{`<th scope="col">Código</th><th scope="col">Tarefa</th><th scope="col">Ações</th>`, "TAR. 0000001",
		`<div class="tarefa-subtitulo">Ligação</div>`, `<div class="hint">Criado por: JOÃO FICTÍCIO</div>`, `<div class="tarefa-client">MARIA EXEMPLO</div>`,
		`<span class="tarefa-meta-label">Origem</span><span class="tarefa-meta-value">ATE. 0000001</span>`,
		`<span class="tarefa-meta-label">Responsável</span><span class="tarefa-meta-value">JOÃO FICTÍCIO</span>`,
		`<span class="tarefa-meta-value tabular">10/11/2026</span>`, `<span class="badge prioridade-alta">Urgente</span>`,
		`<span class="badge badge-danger">Prazo fatal: 12/11/2026</span>`, `Conferir os cálculos`, `<option value="">Mover para...</option>`,
		`<option value="em-execucao">Em execução</option>`, `data-drawer="/tarefas/` + id + `/editar">Editar</button>`,
		`data-drawer-sobrepor>Tarefa vinculada</button>`,
		"Nenhuma tarefa neste estado."} {
		if !strings.Contains(pg, w) {
			t.Errorf("página sem %q", w)
		}
	}
	if strings.Contains(pg, `<option value="pendente">Pendente</option>`) {
		t.Error("Mover para... não pode oferecer o estado atual")
	}
	if inlineProibido.MatchString(pg) {
		t.Error("página com HTML inline proibido")
	}
	// Busca ao enviar (GET ?q=), sem acento.
	if _, b := c.adv.get("/tarefas?q=embargos"); !strings.Contains(b, "TAR. 0000001") {
		t.Error("busca por título deveria achar")
	}
	if _, b := c.adv.get("/tarefas?q=calculos"); !strings.Contains(b, "TAR. 0000001") {
		t.Error("busca sem acento (descrição) deveria achar")
	}
	if _, b := c.adv.get("/tarefas?q=zzzz"); strings.Contains(b, "TAR. 0000001") || !strings.Contains(b, "Nenhuma tarefa cadastrada") {
		t.Error("busca sem resultado deveria mostrar o vazio do B")
	}

	// "Nova tarefa": rótulos na ordem do B.
	novo := c.adv.frag("/tarefas/nova")
	ordem := []string{"Vínculo *", "Atendimento", "Processo", "Vínculo específico *", "Título *", "Subtítulo", "Partes", "Pessoa", "Papel",
		"Descrição", "Criado por", "Data base", "Prazo de entrega", "Prazo fatal", "Responsáveis existentes", "Responsável", "Status", "Prioridade", "Observações"}
	if got := rotulos(novo); !reflect.DeepEqual(got, ordem) {
		t.Errorf("rótulos de Nova tarefa:\n got %v\nwant %v", got, ordem)
	}
	pos := 0
	for _, w := range []string{"<h2>Nova tarefa</h2>", "Toda tarefa nasce vinculada a um atendimento ou a um processo já existente.",
		`placeholder="Ex.: Protocolar embargos à execução"`, "Ligação", "Reunião", "Aula", "Análise de decisão", "Monitoramento",
		"Tarefas de Ligação, Reunião ou Aula aparecem também na categoria correspondente da agenda.", "Remover parte", "+ adicionar parte",
		"Não informado", "Pessoa que criou este registro. Não define quem executará a atividade.", "+ adicionar responsável",
		"Cancelar", "Salvar tarefa"} {
		i := strings.Index(novo[pos:], w)
		if i < 0 {
			t.Fatalf("Nova tarefa: %q ausente ou fora de ordem", w)
		}
		pos += i
	}
	for _, w := range []string{`value="atendimento" required`, ` checked> Atendimento`, `<option value="` + c.ate + `">ATE. 0000001 · MARIA EXEMPLO</option>`,
		`<option value="a-confirmar">A confirmar</option>`, `<option value="pendente" selected>Pendente</option>`, `<option selected>Normal</option>`,
		`<option>Urgente</option>`, `<option>Baixa</option>`, `<option>Terceiro interessado</option>`, `<option>Outro</option>`} {
		if !strings.Contains(novo, w) {
			t.Errorf("Nova tarefa sem %q", w)
		}
	}
	if inlineProibido.MatchString(novo) {
		t.Error("drawer com HTML inline proibido")
	}

	// "Editar tarefa": ordem do B, sem trocar o vínculo.
	edit := c.adv.frag("/tarefas/" + id + "/editar")
	ordemEdit := []string{"Vínculo específico", "Título *", "Subtítulo", "Partes", "Pessoa", "Papel", "Descrição", "Responsáveis existentes", "Responsável",
		"Criado por", "Data base", "Prazo de entrega", "Prazo fatal", "Prioridade", "Status", "Observações", "Documentos do caso"}
	if got := rotulos(edit); !reflect.DeepEqual(got, ordemEdit) {
		t.Errorf("rótulos de Editar tarefa:\n got %v\nwant %v", got, ordemEdit)
	}
	for _, w := range []string{"<h2>Editar tarefa</h2>", `<p class="drawer-sub">Protocolar embargos à execução</p>`, "ATE. 0000001",
		"Registre aqui as atualizações da tarefa", `accept=".pdf,application/pdf"`, "O PDF será guardado, cifrado, junto ao caso.",
		"Ligação, Reunião ou Aula aparecem também na categoria correspondente da agenda.", "Salvar alterações", `value="2026-11-12"`} {
		if !strings.Contains(edit, w) {
			t.Errorf("Editar tarefa sem %q", w)
		}
	}
	if strings.Contains(edit, `name="origem_tipo"`) || strings.Contains(edit, `name="origem_id"`) {
		t.Error("Editar não pode permitir trocar o vínculo")
	}
	// "Tarefa vinculada": herda vínculo, partes e responsáveis só como pré-preenchimento.
	filha := c.adv.frag("/tarefas/" + id + "/filha")
	if !strings.Contains(filha, "<h2>Nova tarefa</h2>") || !strings.Contains(filha, `name="tarefa_pai" value="`+id+`"`) ||
		!strings.Contains(filha, ` checked> Atendimento`) || !strings.Contains(filha, `value="`+c.ate+`" selected`) ||
		!strings.Contains(filha, `value="`+c.maria+`" selected`) || !strings.Contains(filha, `value="`+c.joao+`" selected`) {
		t.Error("Tarefa vinculada deveria pré-preencher vínculo, partes e responsáveis")
	}
	// Tarefa é ata: não há como excluir (nem botão, nem drawer, nem rota).
	if strings.Contains(c.adv.frag("/tarefas/"+id+"/editar"), "Excluir") {
		t.Error("o drawer de edição não oferece exclusão")
	}
	if r, _ := c.adv.get("/tarefas/" + id + "/excluir"); r.StatusCode != 404 {
		t.Errorf("GET /tarefas/{id}/excluir = %d, esperado 404", r.StatusCode)
	}
}

func TestTarefasPermissoesEBotoes(t *testing.T) {
	c := novoCenario(t)
	id := c.adv.criarTarefa(t, tarefaForm("processo", c.pro, "Tarefa do processo"))
	// Quem só lê (administrador) não vê botões; assistente e advogado escrevem; ninguém exclui.
	_, adm := c.adm.get("/tarefas")
	for _, w := range []string{"Nova tarefa</a>", "/editar", "Mover para...", "Tarefa vinculada"} {
		if strings.Contains(adm, w) {
			t.Errorf("administrador (só leitura) vê %q", w)
		}
	}
	_, ass := c.ass.get("/tarefas")
	if !strings.Contains(ass, "Editar</button>") || !strings.Contains(ass, "Mover para...") || strings.Contains(ass, "Excluir") {
		t.Error("assistente deveria ver Editar e Mover, mas não Excluir")
	}
	if r, _ := c.adm.post("/tarefas", tarefaForm("processo", c.pro, "x")); r.StatusCode != 403 {
		t.Errorf("administrador criando = %d", r.StatusCode)
	}
	for nome, b := range map[string]*browser{"assistente": c.ass, "advogado": c.adv, "administrador": c.adm} {
		if r, _ := b.post("/tarefas/"+id+"/excluir", url.Values{"senha": {"cavalo-bateria-grampo-7"}, "version": {"1"}}); r.StatusCode != 404 {
			t.Errorf("%s excluindo = %d, esperado 404 (a rota não existe)", nome, r.StatusCode)
		}
	}
	if st := c.adv.semCSRF("POST", "/tarefas", tarefaForm("processo", c.pro, "x")); st != 403 && st != 401 {
		t.Errorf("sem CSRF = %d", st)
	}
	for _, p := range []string{"/tarefas", "/tarefas/nova", "/atendimentos"} {
		if r, _ := (&browser{e: c.e, sid: "inexistente"}).get(p); r.StatusCode != 303 {
			t.Errorf("GET %s sem sessão = %d", p, r.StatusCode)
		}
	}
}

func TestTarefasValidacoesMoverECancelar(t *testing.T) {
	c := novoCenario(t)
	for nome, f := range map[string]url.Values{
		"Título é obrigatório.":                          tarefaForm("atendimento", c.ate, " "),
		"Selecione o vínculo (Atendimento ou Processo).": tarefaForm("", "", "x"),
		"Selecione o vínculo específico.":                tarefaForm("atendimento", "", "x"),
		"Vínculo não encontrado.":                        tarefaForm("processo", "00000000-0000-0000-0000-000000000000", "x"),
	} {
		if r, body := c.adv.post("/tarefas", f); r.StatusCode != 400 || strings.TrimSpace(body) != nome {
			t.Errorf("%q: %d %q", nome, r.StatusCode, body)
		}
	}
	f := tarefaForm("atendimento", c.ate, "Automática")
	f.Set("estado", "a-confirmar")
	id := c.ass.criarTarefa(t, f)
	// Só advogado tira de "A confirmar".
	if r, body := c.ass.post("/tarefas/"+id+"/mover", url.Values{"estado": {"pendente"}, "version": {"1"}}); r.StatusCode != 400 ||
		strings.TrimSpace(body) != "Somente advogado confirma tarefas a confirmar." {
		t.Errorf("assistente confirmando = %d %q", r.StatusCode, body)
	}
	if r, _ := c.adv.post("/tarefas/"+id+"/mover", url.Values{"estado": {"pendente"}, "version": {"1"}}); r.StatusCode != 200 ||
		r.Header.Get("X-Painel-Ok") != "Estado da tarefa atualizado." {
		t.Errorf("advogado movendo = %d %q", r.StatusCode, r.Header.Get("X-Painel-Ok"))
	}
	if r, _ := c.adv.post("/tarefas/"+id+"/mover", url.Values{"estado": {"revisao"}, "version": {"1"}}); r.StatusCode != 409 {
		t.Errorf("versão antiga = %d", r.StatusCode)
	}
	// Editar: toast do B; vínculo imutável.
	ed := tarefaForm("processo", c.pro, "Automática (editada)")
	ed.Set("version", "2")
	ed.Set("estado", "pendente")
	if r, _ := c.adv.post("/tarefas/"+id, ed); r.StatusCode != 200 || r.Header.Get("X-Painel-Ok") != "Tarefa atualizada." {
		t.Fatalf("editar = %d %q", r.StatusCode, r.Header.Get("X-Painel-Ok"))
	}
	if e := c.e.count(`SELECT count(*) FROM tarefas WHERE id=$1 AND origem_tipo='atendimento' AND processo_id IS NULL`, id); e != 1 {
		t.Error("o vínculo não pode mudar na edição")
	}
	// Filha: guarda o pai e mostra o código dele na lista.
	fh := tarefaForm("atendimento", c.ate, "Filha")
	fh.Set("tarefa_pai", id)
	c.adv.criarTarefa(t, fh)
	if _, pg := c.adv.get("/tarefas"); !strings.Contains(pg, "Tarefa vinculada a: TAR. 0000001") {
		t.Error("a lista deveria mostrar o código da tarefa-pai")
	}
	// Cancelar (Mover para Cancelada): a tarefa permanece, com código e histórico; nada é apagado.
	if r, _ := c.adv.post("/tarefas/"+id+"/mover", url.Values{"estado": {"canceladas"}, "version": {"3"}}); r.StatusCode != 200 {
		t.Fatalf("cancelar = %d", r.StatusCode)
	}
	_, pg := c.adv.get("/tarefas")
	if !strings.Contains(pg, "Automática (editada)") || !strings.Contains(pg, `<span class="count">`) {
		t.Error("a tarefa cancelada continua na lista")
	}
	if c.e.count(`SELECT count(*) FROM tarefas WHERE id=$1 AND excluido_em IS NULL AND estado='canceladas'`, id) != 1 ||
		c.e.count(`SELECT count(*) FROM tarefas WHERE excluido_em IS NOT NULL`) != 0 {
		t.Error("cancelar não apaga nada")
	}
	if c.e.count(`SELECT count(*) FROM audit_events WHERE action IN ('tarefa.criada','tarefa.alterada','tarefa.estado_alterado')`) < 5 {
		t.Error("auditoria da tarefa incompleta")
	}
	if _, err := audit.Verify(context.Background(), c.e.pool); err != nil {
		t.Fatal(err)
	}
}

func TestTarefaPrazoFatalNoMesmoCampoComSeloAConferir(t *testing.T) {
	c := novoCenario(t)
	fa := tarefaForm("atendimento", c.ate, "Do assistente")
	fa.Set("prazo_fatal", "2026-12-02")
	ida := c.ass.criarTarefa(t, fa)
	fv := tarefaForm("atendimento", c.ate, "Do advogado")
	fv.Set("prazo_fatal", "2026-12-03")
	idv := c.adv.criarTarefa(t, fv)

	sit := func(id string) string {
		var s string
		if err := c.e.pool.QueryRow(context.Background(), `SELECT situacao FROM prazos WHERE tarefa_id=$1 AND excluido_em IS NULL`, id).Scan(&s); err != nil {
			t.Fatalf("prazo da tarefa %s: %v", id, err)
		}
		return s
	}
	if sit(ida) != "a_conferir" || sit(idv) != "confirmado" {
		t.Errorf("situações = %s / %s", sit(ida), sit(idv))
	}
	// O assistente vê o selo A CONFERIR no mesmo campo do prazo fatal, mas não o botão de confirmar.
	_, pg := c.ass.get("/tarefas")
	if !strings.Contains(pg, `Prazo fatal: 02/12/2026</span> <span class="badge badge-warn">A CONFERIR</span>`) || strings.Contains(pg, "Confirmar prazo") {
		t.Error("assistente deveria ver A CONFERIR e nenhum botão de confirmar")
	}
	if strings.Contains(pg, `Prazo fatal: 03/12/2026</span> <span class="badge badge-warn">`) {
		t.Error("prazo confirmado não leva o selo A CONFERIR")
	}
	// O advogado vê o botão que usa o fluxo do domínio Prazos; o assistente não pode chamá-lo.
	_, pgAdv := c.adv.get("/tarefas")
	if !strings.Contains(pgAdv, `action="/tarefas/`+ida+`/confirmar-prazo"`) || !strings.Contains(pgAdv, ">Confirmar prazo</button>") {
		t.Error("advogado deveria ver 'Confirmar prazo'")
	}
	if r, _ := c.ass.post("/tarefas/"+ida+"/confirmar-prazo", nil); r.StatusCode != 403 {
		t.Errorf("assistente confirmando prazo = %d, esperado 403", r.StatusCode)
	}
	// Editar com a mesma data não duplica o prazo nem derruba a situação.
	ed := tarefaForm("atendimento", c.ate, "Do assistente (editada)")
	ed.Set("prazo_fatal", "2026-12-02")
	ed.Set("version", "1")
	if r, _ := c.ass.post("/tarefas/"+ida, ed); r.StatusCode != 200 {
		t.Fatalf("editar = %d", r.StatusCode)
	}
	if c.e.count(`SELECT count(*) FROM prazos WHERE tarefa_id=$1`, ida) != 1 {
		t.Error("editar não pode duplicar o prazo")
	}
	if r, _ := c.adv.post("/tarefas/"+ida+"/confirmar-prazo", nil); r.StatusCode != 200 || r.Header.Get("X-Painel-Ok") != "Prazo confirmado." {
		t.Fatalf("advogado confirmando = %d", r.StatusCode)
	}
	if sit(ida) != "confirmado" {
		t.Errorf("depois de confirmar = %s", sit(ida))
	}
	// Mudar a data (assistente) derruba a confirmação; o prazo de tarefa não se edita pela tela de Prazos.
	ed.Set("prazo_fatal", "2026-12-09")
	ed.Set("version", "2")
	if r, _ := c.ass.post("/tarefas/"+ida, ed); r.StatusCode != 200 {
		t.Fatalf("editar data = %d", r.StatusCode)
	}
	if sit(ida) != "a_conferir" {
		t.Errorf("mudar a data deveria voltar a 'a conferir', ficou %s", sit(ida))
	}
	var pid string
	_ = c.e.pool.QueryRow(context.Background(), `SELECT id FROM prazos WHERE tarefa_id=$1`, ida).Scan(&pid)
	if r, body := c.adv.post("/prazos/"+pid, url.Values{"descricao": {"x"}, "data_fatal": {"2026-12-30"}, "fonte": {"manual"}, "version": {"3"}}); r.StatusCode != 400 ||
		!strings.Contains(body, "pertence a uma tarefa") {
		t.Errorf("editar prazo de tarefa na tela de Prazos = %d %q", r.StatusCode, body)
	}
	// Apagar a data cancela o prazo; concluir cumpre o confirmado.
	ed.Set("prazo_fatal", "")
	ed.Set("version", "3")
	if r, _ := c.ass.post("/tarefas/"+ida, ed); r.StatusCode != 200 {
		t.Fatalf("apagar a data = %d", r.StatusCode)
	}
	if sit(ida) != "cancelado" {
		t.Errorf("apagar a data deveria cancelar o prazo, ficou %s", sit(ida))
	}
	if r, _ := c.adv.post("/tarefas/"+idv+"/mover", url.Values{"estado": {"concluidas-com-sucesso"}, "version": {"1"}}); r.StatusCode != 200 {
		t.Fatalf("concluir = %d", r.StatusCode)
	}
	if sit(idv) != "cumprido" {
		t.Errorf("concluir deveria cumprir o prazo confirmado, ficou %s", sit(idv))
	}
}

func TestTarefaDocumentosDoCaso(t *testing.T) {
	c := novoCenario(t)
	id := c.adv.criarTarefa(t, tarefaForm("processo", c.pro, "Com documento"))
	r, body := c.adv.enviarDocTarefa(id, "peticao.pdf", pdfBytes())
	if r.StatusCode != 200 || r.Header.Get("X-Painel-Ok") != "Documento guardado." {
		t.Fatalf("PDF válido = %d %q", r.StatusCode, body)
	}
	var proc, tar string
	if err := c.e.pool.QueryRow(context.Background(), `SELECT coalesce(processo_id::text,''), coalesce(tarefa_id::text,'') FROM documentos WHERE nome_original='peticao.pdf'`).Scan(&proc, &tar); err != nil || proc != c.pro || tar != id {
		t.Errorf("documento ligado a %q/%q (%v)", proc, tar, err)
	}
	// PDF falso (extensão .pdf, conteúdo de imagem) e arquivo que não é PDF são recusados.
	png := []byte("\x89PNG\r\n\x1a\n" + "falso")
	if r, _ := c.adv.enviarDocTarefa(id, "falso.pdf", png); r.StatusCode != 400 {
		t.Errorf("PDF falso = %d, esperado 400", r.StatusCode)
	}
	if r, body := c.adv.enviarDocTarefa(id, "foto.png", png); r.StatusCode != 400 || !strings.Contains(body, "Envie um arquivo PDF.") {
		t.Errorf("PNG = %d %q", r.StatusCode, body)
	}
	if r, _ := c.adm.enviarDocTarefa(id, "a.pdf", pdfBytes()); r.StatusCode != 403 {
		t.Errorf("administrador enviando = %d, esperado 403", r.StatusCode)
	}
	if c.e.count(`SELECT count(*) FROM documentos`) != 1 {
		t.Error("só o PDF válido deveria ter sido guardado")
	}
	// Tarefa de atendimento: o documento vai ligado ao atendimento.
	ida := c.adv.criarTarefa(t, tarefaForm("atendimento", c.ate, "No atendimento"))
	if r, body := c.adv.enviarDocTarefa(ida, "recibo.pdf", pdfBytes()); r.StatusCode != 200 {
		t.Fatalf("PDF no atendimento = %d %s", r.StatusCode, body)
	}
	if c.e.count(`SELECT count(*) FROM documentos WHERE atendimento_id=$1 AND tarefa_id=$2`, c.ate, ida) != 1 {
		t.Error("o documento deveria ficar ligado ao atendimento e à tarefa")
	}
}
