package app

import (
	"net/url"
	"strings"
	"testing"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/br"
)

func audienciaForm(tipo, titulo, data, hora string) url.Values {
	return url.Values{"tipo": {tipo}, "titulo": {titulo}, "data": {data}, "hora": {hora}, "juizo": {"1ª Vara Cível"},
		"local": {"Sala 2"}, "descricao": {"Levar a procuração."}}
}

var tiposAto = []string{"audiencia-ad-monitoria", "conciliacao-e-mediacao", "inicial-justica-do-trabalho",
	"instrucao-e-julgamento", "sustentacao-oral", "una-justica-do-trabalho"}

func TestAudiencias(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)
	adm, _ := e.user("adm@exemplo.test", auth.Administrador)

	// Validações, com os textos do B.
	for _, c := range []struct {
		f    url.Values
		want string
	}{
		{audienciaForm("", "Instrução", dataFutura, "10:00"), "Selecione o tipo do ato."},
		{audienciaForm("outro-ato", "Instrução", dataFutura, "10:00"), "Selecione o tipo do ato."},
		{audienciaForm("sustentacao-oral", "", dataFutura, "10:00"), "Título, data e hora são obrigatórios."},
		{audienciaForm("sustentacao-oral", "x", "", "10:00"), "Título, data e hora são obrigatórios."},
		{audienciaForm("sustentacao-oral", "x", dataFutura, ""), "Título, data e hora são obrigatórios."},
	} {
		if r, body := adv.post("/audiencias", c.f); r.StatusCode != 400 || !strings.Contains(body, c.want) {
			t.Errorf("%v: status %d, corpo %q", c.f, r.StatusCode, body)
		}
	}

	// Criar uma por tipo de ato: códigos AUD. 0000001.. em sequência (o assistente também escreve).
	for i, tipo := range tiposAto {
		who := adv
		if i%2 == 1 {
			who = ass
		}
		r, body := who.post("/audiencias", audienciaForm(tipo, "Audiência "+tipo, "2030-06-0"+string(rune('1'+i)), "09:00"))
		want := "Audiência criada — código AUD. 000000" + string(rune('1'+i)) + "."
		if m := okMsg(t, res{r, body}); m != want {
			t.Errorf("toast = %q, esperado %q", m, want)
		}
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='audiencia.criada'`) != 6 {
		t.Error("criação não auditada")
	}

	// Audiência nunca é bloqueada: só avisa se coincidir com outra audiência (mesmo dia, mesma hora exata).
	r, body := adv.post("/audiencias", audienciaForm("sustentacao-oral", "Coincidente", "2030-06-01", "09:00"))
	want := `Audiência criada — Atenção: coincide com Audiência "Audiência audiencia-ad-monitoria" em 2030-06-01 às 09:00 — imposição externa, verifique manualmente.`
	if m := okMsg(t, res{r, body}); m != want {
		t.Errorf("aviso = %q", m)
	}
	if n := e.count(`SELECT count(*) FROM audiencias`); n != 7 {
		t.Errorf("audiências = %d, esperado 7", n)
	}

	// Processo vinculado: número que existe grava o vínculo real; número desconhecido guarda só o texto.
	num := br.MakeCNJ("0000700", "2024", "8", "26", "0100")
	proc := adv.criarProcesso(t, num)
	f := audienciaForm("instrucao-e-julgamento", "Com processo", "2030-07-01", "11:00")
	f.Set("processo", num)
	okMsg(t, adv.mustPost("/audiencias", f))
	f.Set("titulo", "Sem processo cadastrado")
	f.Set("processo", br.MakeCNJ("0000999", "2024", "8", "26", "0100"))
	f.Set("hora", "12:00")
	okMsg(t, adv.mustPost("/audiencias", f))
	if e.count(`SELECT count(*) FROM audiencias WHERE titulo='Com processo' AND processo_id=$1 AND processo_numero=$2`, proc, num) != 1 {
		t.Error("o número existente deveria gravar processo_id e o texto")
	}
	if e.count(`SELECT count(*) FROM audiencias WHERE titulo='Sem processo cadastrado' AND processo_id IS NULL AND processo_numero<>''`) != 1 {
		t.Error("o número desconhecido deveria guardar só o texto")
	}

	// Edição: o tipo do ato não muda; o drawer mostra o aviso do B.
	var id string
	var ver int
	if err := e.pool.QueryRow(t.Context(), `SELECT id, version FROM audiencias WHERE titulo='Com processo'`).Scan(&id, &ver); err != nil {
		t.Fatal(err)
	}
	_, frag := adv.do("GET", "/audiencias/"+id+"/editar", nil, map[string]string{"HX-Request": "true"})
	for _, s := range []string{"<h2>Editar audiência</h2>", `<p class="drawer-sub">Com processo</p>`, "Tipo do ato", "disabled",
		"Instrução e Julgamento", "Tipo do ato não se edita aqui (é a pasta da ficha).", "Salvar alterações"} {
		if !strings.Contains(frag, s) {
			t.Errorf("drawer de edição sem %q", s)
		}
	}
	g := audienciaForm("sustentacao-oral", "Com processo (remarcada)", "2030-07-02", "15:00")
	g.Set("processo", num)
	g.Set("version", "1")
	if m := okMsg(t, adv.mustPost("/audiencias/"+id, g)); m != "Audiência atualizada." {
		t.Errorf("toast de edição = %q", m)
	}
	var tipo string
	if err := e.pool.QueryRow(t.Context(), `SELECT tipo_ato FROM audiencias WHERE id=$1`, id).Scan(&tipo); err != nil || tipo != "instrucao-e-julgamento" {
		t.Errorf("tipo do ato mudou na edição: %q", tipo)
	}
	if r, _ := adv.post("/audiencias/"+id, g); r.StatusCode != 409 {
		t.Errorf("versão velha = %d, esperado 409", r.StatusCode)
	}

	// Cancelar: drawer, motivo e senha obrigatórios, reautenticação, ficha preservada, auditoria.
	_, cd := adv.do("GET", "/audiencias/"+id+"/cancelar", nil, map[string]string{"HX-Request": "true"})
	for _, s := range []string{"<h2>Cancelar audiência</h2>", "A ficha é preservada com o motivo do cancelamento. Confirme com a sua senha.",
		"Motivo do cancelamento *", "Será preservado na ficha", "Senha *", "Voltar", "Cancelar audiência</button>"} {
		if !strings.Contains(cd, s) {
			t.Errorf("drawer de cancelamento sem %q", s)
		}
	}
	senhaCerta := "cavalo-bateria-grampo-7"
	if r, _ := ass.post("/audiencias/"+id+"/cancelar", url.Values{"motivo": {"x"}, "senha": {senhaCerta}, "version": {"2"}}); r.StatusCode != 403 {
		t.Errorf("assistente cancelando = %d, esperado 403", r.StatusCode)
	}
	if r, _ := adm.post("/audiencias/"+id+"/cancelar", url.Values{"motivo": {"x"}, "senha": {senhaCerta}, "version": {"2"}}); r.StatusCode != 403 {
		t.Errorf("administrador cancelando = %d, esperado 403", r.StatusCode)
	}
	if r, b := adv.post("/audiencias/"+id+"/cancelar", url.Values{"motivo": {" "}, "senha": {senhaCerta}, "version": {"2"}}); r.StatusCode != 400 || !strings.Contains(b, "motivo") {
		t.Errorf("sem motivo = %d %s", r.StatusCode, b)
	}
	if r, b := adv.post("/audiencias/"+id+"/cancelar", url.Values{"motivo": {"Remarcada"}, "senha": {"errada"}, "version": {"2"}}); r.StatusCode != 400 || !strings.Contains(b, "Senha incorreta") {
		t.Errorf("senha errada = %d %s", r.StatusCode, b)
	}
	if e.count(`SELECT count(*) FROM audiencias WHERE id=$1 AND NOT cancelada`, id) != 1 {
		t.Fatal("a audiência não pode mudar com senha errada")
	}
	if m := okMsg(t, adv.mustPost("/audiencias/"+id+"/cancelar", url.Values{"motivo": {"Remarcada pelo juízo"}, "senha": {senhaCerta}, "version": {"2"}})); m != "Audiência cancelada; a ficha foi preservada." {
		t.Errorf("toast = %q", m)
	}
	if e.count(`SELECT count(*) FROM audiencias WHERE id=$1 AND cancelada AND cancelamento_motivo='Remarcada pelo juízo' AND cancelada_em IS NOT NULL AND cancelada_por IS NOT NULL AND excluido_em IS NULL`, id) != 1 {
		t.Error("a ficha deveria ficar preservada com motivo, quem e quando")
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='audiencia.cancelada'`) != 1 {
		t.Error("cancelamento não auditado")
	}
	if r, _ := adv.post("/audiencias/"+id+"/cancelar", url.Values{"motivo": {"de novo"}, "senha": {senhaCerta}, "version": {"3"}}); r.StatusCode == 200 {
		t.Error("cancelar duas vezes deveria ser recusado")
	}
	if r, _ := adv.post("/audiencias/"+id, func() url.Values { h := g; h.Set("version", "3"); return h }()); r.StatusCode == 200 {
		t.Error("editar audiência cancelada deveria ser recusado")
	}

	// A audiência cancelada não bloqueia mais o horário; a ativa bloqueia compromisso (e diligência).
	if m := okMsg(t, adv.mustPost("/agenda", compromissoForm("Reunião", "No horário da cancelada", "2030-07-02", "15:00"))); m != "Compromisso criado." {
		t.Errorf("horário de audiência cancelada deveria estar livre: %q", m)
	}
	r, b := adv.post("/agenda", compromissoForm("Reunião", "Em cima da audiência", "2030-06-01", "09:00"))
	if r.StatusCode != 400 || !strings.Contains(b, `Bloqueado: conflito de horário com Audiência "Audiência audiencia-ad-monitoria" em 2030-06-01 às 09:00.`) {
		t.Errorf("conflito de compromisso = %d %s", r.StatusCode, b)
	}
	if e.count(`SELECT count(*) FROM agenda_compromissos WHERE descricao='Em cima da audiência'`) != 0 {
		t.Error("compromisso em conflito não pode ser gravado")
	}
	if _, err := audit.Verify(t.Context(), e.pool); err != nil {
		t.Fatal(err)
	}
}

// Os textos do drawer "Nova audiência" são os do B (11-agenda.js), na ordem do B.
func TestDrawerNovaAudienciaTextosDoB(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	_, frag := adv.do("GET", "/audiencias/nova", nil, map[string]string{"HX-Request": "true"})
	ordem := []string{
		"<h2>Nova audiência</h2>",
		"Audiência é item crítico: nunca é bloqueada por conflito de horário, só avisa se coincidir com outro item crítico.",
		"Tipo do ato *", "Selecione…", "Audiência Ad Monitoria", "Conciliação e Mediação", "Inicial — Justiça do Trabalho",
		"Instrução e Julgamento", "Sustentação Oral", "Una — Justiça do Trabalho",
		"Título *", "Ex.: Audiência de instrução — fulano x banco", "Processo vinculado", "Cliente", "Data *", "Hora *", "Juízo",
		"Local ou link da aula/reunião", "Somente registro do local/link. Não inicia gravação, download ou transcrição.",
		"Observações", "Cancelar", "Salvar audiência",
	}
	pos := 0
	for _, s := range ordem {
		i := strings.Index(frag[pos:], s)
		if i < 0 {
			t.Fatalf("texto %q ausente (ou fora de ordem):\n%s", s, frag)
		}
		pos += i + len(s)
	}
	if !strings.Contains(frag, `data-mask="cnj"`) {
		t.Error("o campo do processo deveria ter a máscara CNJ")
	}
}

func TestPaginaAudiencias(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)
	okMsg(t, adv.mustPost("/audiencias", audienciaForm("sustentacao-oral", "Sustentação do caso Exemplo", "2030-06-10", "10:00")))
	okMsg(t, adv.mustPost("/audiencias", audienciaForm("sustentacao-oral", "Outra sustentação", "2030-06-05", "10:00")))
	_, pg := adv.get("/audiencias")
	for _, s := range []string{"<h1>Audiências</h1>", "Audiências marcadas, agrupadas por tipo de ato.", "Nova audiência",
		`placeholder="Buscar por título, processo, cliente ou juízo…"`, `id="buscaAudiencias"`,
		"Tipo do ato", "Código", "Título", "Data", "Hora", "Local", "Processo", "Nenhuma audiência em aberto nesta categoria.",
		"AUD. 0000001", "Sustentação do caso Exemplo", "10/06/2030", `data-drawer="/audiencias/nova"`} {
		if !strings.Contains(pg, s) {
			t.Errorf("página de audiências sem %q", s)
		}
	}
	// As 6 pílulas, na ordem do B, e cada tipo de ato com o seu painel; ordenadas por data e hora.
	pos := 0
	for _, s := range []string{"Audiência Ad Monitoria", "Conciliação e Mediação", "Inicial — Justiça do Trabalho", "Instrução e Julgamento",
		"Sustentação Oral", "Una — Justiça do Trabalho"} {
		i := strings.Index(pg[pos:], `<span>`+s+`</span>`)
		if i < 0 {
			t.Fatalf("pílula %q ausente ou fora de ordem", s)
		}
		pos += i + 1
	}
	if strings.Index(pg, "Outra sustentação") > strings.Index(pg, "Sustentação do caso Exemplo") {
		t.Error("as audiências deveriam estar em ordem de data e hora")
	}
	// Vários painéis podem ficar abertos (nenhum aberto de início); o assistente não vê "Cancelar audiência".
	if strings.Contains(pg, `class="acc-head open"`) {
		t.Error("nenhum painel deveria começar aberto")
	}
	if !strings.Contains(pg, "Cancelar audiência") {
		t.Error("o advogado deveria ver Cancelar audiência")
	}
	if _, pa := ass.get("/audiencias"); strings.Contains(pa, ">Cancelar audiência<") || !strings.Contains(pa, ">Editar<") {
		t.Error("o assistente edita, mas não cancela")
	}
	if strings.Contains(pg, "onclick=") || strings.Contains(pg, "style=") {
		t.Error("a página não pode ter script ou estilo inline")
	}
}
