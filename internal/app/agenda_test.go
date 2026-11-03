package app

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/br"
)

const dataFutura = "2030-05-06"

func compromissoForm(tipo, desc, data, hora string) url.Values {
	return url.Values{"tipo": {tipo}, "descricao": {desc}, "data": {data}, "hora": {hora}, "local": {"Sala 3"}}
}

// res junta a resposta e o corpo de um POST.
type res struct {
	r *http.Response
	b string
}

func (b *browser) mustPost(path string, f url.Values) res {
	r, body := b.post(path, f)
	return res{r, body}
}

// okMsg devolve o texto do toast de sucesso (cabeçalho X-Painel-Ok, decodificado) ou falha o teste.
func okMsg(t *testing.T, x res) string {
	t.Helper()
	if x.r.StatusCode != 200 {
		t.Fatalf("status = %d, corpo: %s", x.r.StatusCode, x.b)
	}
	m, err := url.PathUnescape(x.r.Header.Get("X-Painel-Ok"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestCompromissos(t *testing.T) {
	e := newEnv(t)
	adv, advID := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)
	adm, _ := e.user("adm@exemplo.test", auth.Administrador)

	// Validações, com os textos do B.
	for _, c := range []struct {
		f    url.Values
		want string
	}{
		{compromissoForm("", "Reunião com a cliente", dataFutura, "14:00"), "Selecione o tipo do compromisso."},
		{compromissoForm("Outro", "x", dataFutura, "14:00"), "Selecione o tipo do compromisso."},
		{compromissoForm("Reunião", "", dataFutura, "14:00"), "Descrição, data e hora são obrigatórios."},
		{compromissoForm("Reunião", "x", "", "14:00"), "Descrição, data e hora são obrigatórios."},
		{compromissoForm("Reunião", "x", dataFutura, ""), "Descrição, data e hora são obrigatórios."},
		{compromissoForm("Reunião", "x", dataFutura, "25:61"), "Descrição, data e hora são obrigatórios."},
	} {
		if r, body := adv.post("/agenda", c.f); r.StatusCode != 400 || !strings.Contains(body, c.want) {
			t.Errorf("%v: status %d, corpo %q", c.f, r.StatusCode, body)
		}
	}

	// Criar Reunião, Ligação e Aula (o assistente também escreve na agenda).
	var ids []string
	for i, tipo := range []string{"Reunião", "Ligação", "Aula"} {
		who := adv
		if i == 1 {
			who = ass
		}
		r, body := who.post("/agenda", compromissoForm(tipo, tipo+" de teste", dataFutura, "1"+string(rune('0'+i))+":30"))
		if m := okMsg(t, res{r, body}); m != "Compromisso criado." {
			t.Errorf("toast = %q", m)
		}
	}
	if n := e.count(`SELECT count(*) FROM agenda_compromissos WHERE data='2030-05-06'`); n != 3 {
		t.Fatalf("compromissos = %d", n)
	}
	_ = ids
	if e.count(`SELECT count(*) FROM audit_events WHERE action='compromisso.criado'`) != 3 {
		t.Error("criação não auditada")
	}

	// O administrador só lê; sem CSRF a escrita é negada.
	if r, _ := adm.post("/agenda", compromissoForm("Aula", "x", dataFutura, "09:00")); r.StatusCode != 403 {
		t.Errorf("administrador escrevendo na agenda = %d, esperado 403", r.StatusCode)
	}
	if st := adv.semCSRF("POST", "/agenda", compromissoForm("Aula", "x", dataFutura, "09:00")); st != 403 {
		t.Errorf("sem CSRF = %d", st)
	}

	// Vínculos: precisam existir. Cliente/atendimento/processo/criado por inexistentes são recusados.
	pes := adv.criarPessoa(t, "Dora Lopes", "")
	proc := adv.criarProcesso(t, br.MakeCNJ("0000500", "2024", "8", "26", "0100"), [3]string{pes, "Autor", "1"})
	zero := "00000000-0000-0000-0000-000000000000"
	for campo, v := range map[string]string{"cliente": zero, "atendimento": zero, "processo": zero, "criado_por": zero} {
		f := compromissoForm("Reunião", "Com vínculo", dataFutura, "16:00")
		f.Set(campo, v)
		if r, _ := adv.post("/agenda", f); r.StatusCode != 400 {
			t.Errorf("%s inexistente = %d, esperado 400", campo, r.StatusCode)
		}
	}
	f := compromissoForm("Reunião", "Com vínculo", dataFutura, "16:00")
	f.Set("cliente", pes)
	f.Set("processo", proc)
	f.Set("criado_por", pes)
	if r, body := adv.post("/agenda", f); okMsg(t, res{r, body}) != "Compromisso criado." {
		t.Fatal("compromisso com vínculos não criado")
	}
	var id string
	var ver int
	if err := e.pool.QueryRow(t.Context(), `SELECT id, version FROM agenda_compromissos WHERE descricao='Com vínculo'`).Scan(&id, &ver); err != nil {
		t.Fatal(err)
	}

	// Drawer de edição: subtítulo = descrição; sem bloco de diligência; "Salvar alterações".
	r, frag := adv.do("GET", "/agenda/"+id+"/editar", nil, map[string]string{"HX-Request": "true"})
	if r.StatusCode != 200 || !strings.HasPrefix(strings.TrimSpace(frag), "<h2>Editar compromisso</h2>") ||
		!strings.Contains(frag, `<p class="drawer-sub">Com vínculo</p>`) || !strings.Contains(frag, "Salvar alterações") ||
		strings.Contains(frag, "Responsável pela diligência") || strings.Contains(frag, "Selecione…") {
		t.Fatalf("drawer de edição inesperado: %s", frag)
	}
	g := compromissoForm("Ligação", "Com vínculo (remarcado)", dataFutura, "17:00")
	g.Set("version", "1")
	if m := okMsg(t, adv.mustPost("/agenda/"+id, g)); m != "Compromisso atualizado." {
		t.Errorf("toast de edição = %q", m)
	}
	if r, _ := adv.post("/agenda/"+id, g); r.StatusCode != 409 {
		t.Errorf("versão velha = %d, esperado 409", r.StatusCode)
	}
	g.Set("version", "2")
	g.Set("tipo", "Diligência")
	if r, body := adv.post("/agenda/"+id, g); r.StatusCode != 400 || !strings.Contains(body, "Diligência tem ficha própria") {
		t.Errorf("trocar para Diligência na edição = %d %s", r.StatusCode, body)
	}
	if r, _ := adv.post("/agenda/"+zero, func() url.Values { h := g; h.Set("tipo", "Aula"); return h }()); r.StatusCode != 404 {
		t.Errorf("editar inexistente = %d", r.StatusCode)
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='compromisso.atualizado'`) != 1 {
		t.Error("edição não auditada")
	}
	_ = advID
	if _, err := audit.Verify(t.Context(), e.pool); err != nil {
		t.Fatal(err)
	}
}

// Os textos do drawer "Novo compromisso" são os do B (11-agenda.js), na ordem do B.
func TestDrawerNovoCompromissoTextosDoB(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	adv.criarPessoa(t, "Dora Lopes", "")
	_, frag := adv.do("GET", "/agenda/novo", nil, map[string]string{"HX-Request": "true"})
	ordem := []string{
		"<h2>Novo compromisso</h2>",
		"Audiência nunca é bloqueada por conflito de horário; os demais compromissos podem ser. Prazo fatal é campo da tarefa (aba Tarefas), não compromisso.",
		"Tipo *", "Selecione…", ">Reunião<", ">Ligação<", ">Diligência<", ">Aula<",
		"Descrição *", "Data *", "Hora *", "Local ou link da aula/reunião",
		"Somente registro do local/link. Não inicia gravação, download ou transcrição.",
		"Responsável pela diligência",
		"Cliente (opcional)", "Buscar nome ou código PES…",
		"Atendimento vinculado (opcional)", "Buscar código ATE, cliente ou objeto…",
		"Processo vinculado (opcional)", "Buscar código PRO, número ou título…",
		"Sem vínculo", "Criado por", "Não informado",
		"Pessoa que criou este registro. Não define quem executará a atividade.",
		"Cancelar", "Salvar compromisso",
	}
	pos := 0
	for _, s := range ordem {
		i := strings.Index(frag[pos:], s)
		if i < 0 {
			t.Fatalf("texto %q ausente (ou fora de ordem) no drawer:\n%s", s, frag)
		}
		pos += i + len(s)
	}
	for _, proibido := range []string{"onclick=", "onchange=", "oninput=", "style=", "Prazo Fatal</option>"} {
		if strings.Contains(frag, proibido) {
			t.Errorf("fragmento contém %q", proibido)
		}
	}
}
