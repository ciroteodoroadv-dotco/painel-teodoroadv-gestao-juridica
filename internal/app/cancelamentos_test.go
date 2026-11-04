package app

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
)

// frag pede um fragmento de drawer (cabeçalho HX-Request) e devolve o status e o HTML.
func (b *browser) fragDrawer(path string) (int, string) {
	r, body := b.do("GET", path, nil, map[string]string{"HX-Request": "true"})
	return r.StatusCode, body
}

func motivoForm(version, motivo string) url.Values {
	return url.Values{"version": {version}, "motivo": {motivo}}
}

func TestCancelarCompromissoNaoExclui(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv-canc1@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass-canc1@exemplo.test", auth.Assistente)
	adm, _ := e.user("adm-canc1@exemplo.test", auth.Administrador)
	for _, tit := range []string{"Aula de teste", "Reunião de teste"} {
		tipo := map[bool]string{true: "Aula", false: "Reunião"}[tit == "Aula de teste"]
		if r, b := adv.post("/agenda", url.Values{"tipo": {tipo}, "descricao": {tit}, "data": {"2030-01-10"}, "hora": {"10:00"}}); r.StatusCode != 200 {
			t.Fatalf("criar compromisso = %d %s", r.StatusCode, b)
		}
	}
	var id, outro string
	_ = e.pool.QueryRow(context.Background(), `SELECT id::text FROM agenda_compromissos WHERE descricao='Aula de teste'`).Scan(&id)
	_ = e.pool.QueryRow(context.Background(), `SELECT id::text FROM agenda_compromissos WHERE descricao='Reunião de teste'`).Scan(&outro)

	_, pg := adv.get("/agenda")
	if strings.Contains(pg, "Excluir") || strings.Contains(pg, "Ainda não implementado") {
		t.Error("a Agenda não oferece mais Excluir")
	}
	if !strings.Contains(pg, `<button type="button" class="btn-danger-text" data-drawer="/agenda/`+id+`/cancelar">Cancelar</button>`) {
		t.Error("o advogado deveria ver o botão Cancelar (btn-danger-text) do compromisso")
	}
	_, pgAss := ass.get("/agenda")
	if !strings.Contains(pgAss, `/agenda/`+id+`/editar`) || strings.Contains(pgAss, `/agenda/`+id+`/cancelar`) {
		t.Error("o assistente edita mas não cancela")
	}
	for nome, b := range map[string]*browser{"assistente": ass, "administrador": adm} {
		if st, _ := b.fragDrawer("/agenda/" + id + "/cancelar"); st != 403 {
			t.Errorf("%s abrindo o drawer = %d", nome, st)
		}
		if r, _ := b.post("/agenda/"+id+"/cancelar", motivoForm("1", "x")); r.StatusCode != 403 {
			t.Errorf("%s cancelando = %d", nome, r.StatusCode)
		}
	}
	st, frag := adv.fragDrawer("/agenda/" + id + "/cancelar")
	if st != 200 {
		t.Fatalf("drawer = %d", st)
	}
	for _, w := range []string{"<h2>Cancelar compromisso</h2>", "O registro é preservado com o motivo do cancelamento.", "Motivo do cancelamento *",
		"Será preservado na ficha", ">Voltar</button>", ">Cancelar compromisso</button>", `name="version" value="1"`} {
		if !strings.Contains(frag, w) {
			t.Errorf("drawer sem %q", w)
		}
	}
	if strings.Contains(frag, "Senha") {
		t.Error("cancelar compromisso não pede senha")
	}
	if r, b := adv.post("/agenda/"+id+"/cancelar", motivoForm("1", "  ")); r.StatusCode != 400 || strings.TrimSpace(b) != "Informe o motivo do cancelamento." {
		t.Errorf("sem motivo = %d %q", r.StatusCode, b)
	}
	if r, _ := adv.post("/agenda/"+outro+"/cancelar", motivoForm("9", "x y")); r.StatusCode != 409 {
		t.Errorf("versão velha = %d", r.StatusCode)
	}
	r, _ := adv.post("/agenda/"+id+"/cancelar", motivoForm("1", "Aula remarcada para fevereiro"))
	if r.StatusCode != 200 || decodifica(t, r.Header.Get("X-Painel-Ok")) != "Compromisso cancelado; o registro foi preservado." {
		t.Fatalf("cancelar = %d %q", r.StatusCode, r.Header.Get("X-Painel-Ok"))
	}
	if e.count(`SELECT count(*) FROM agenda_compromissos WHERE id=$1::uuid AND cancelado AND cancelamento_motivo='Aula remarcada para fevereiro' AND cancelado_em IS NOT NULL AND cancelado_por IS NOT NULL AND excluido_em IS NULL`, id) != 1 {
		t.Error("o registro é preservado com motivo, quem e quando")
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='compromisso.cancelado' AND target_id=$1`, id) != 1 {
		t.Error("auditoria compromisso.cancelado")
	}
	_, pg = adv.get("/agenda")
	if !strings.Contains(pg, "Aula de teste") || !strings.Contains(pg, `<span class="badge badge-neutral">Cancelado</span>`) ||
		strings.Contains(pg, `/agenda/`+id+`/editar`) || strings.Contains(pg, `/agenda/`+id+`/cancelar`) {
		t.Error("o cancelado continua listado, com selo e sem ações")
	}
	if r, b := adv.post("/agenda/"+id, url.Values{"version": {"2"}, "tipo": {"Aula"}, "descricao": {"x"}, "data": {"2030-01-10"}, "hora": {"10:00"}}); r.StatusCode != 400 ||
		strings.TrimSpace(b) != "Compromisso cancelado não pode ser editado." {
		t.Errorf("editar cancelado = %d %q", r.StatusCode, b)
	}
	if r, b := adv.post("/agenda/"+id+"/cancelar", motivoForm("2", "de novo")); r.StatusCode != 400 || strings.TrimSpace(b) != "Este compromisso já está cancelado." {
		t.Errorf("cancelar de novo = %d %q", r.StatusCode, b)
	}
	if st, _ := adv.fragDrawer("/agenda/" + id + "/editar"); st != 400 {
		t.Errorf("drawer de edição do cancelado = %d", st)
	}
}

func TestCancelarDiligenciaNaoExclui(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv-canc2@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass-canc2@exemplo.test", auth.Assistente)
	if r, b := adv.post("/agenda", url.Values{"tipo": {"Diligência"}, "descricao": {"Vistoria de teste"}, "data": {"2030-01-11"}, "hora": {"11:00"}, "responsavel": {"Fulano Exemplo"}}); r.StatusCode != 200 {
		t.Fatalf("criar diligência = %d %s", r.StatusCode, b)
	}
	var id string
	_ = e.pool.QueryRow(context.Background(), `SELECT id::text FROM diligencias`).Scan(&id)
	_, pg := adv.get("/agenda")
	if !strings.Contains(pg, `<button type="button" class="btn-danger-text" data-drawer="/diligencias/`+id+`/cancelar">Cancelar</button>`) {
		t.Error("o advogado deveria ver Cancelar na diligência")
	}
	_, pgAss := ass.get("/agenda")
	if strings.Contains(pgAss, `/diligencias/`+id+`/cancelar`) || !strings.Contains(pgAss, `/diligencias/`+id+`/editar`) {
		t.Error("o assistente edita mas não cancela diligência")
	}
	if r, _ := ass.post("/diligencias/"+id+"/cancelar", motivoForm("1", "x")); r.StatusCode != 403 {
		t.Errorf("assistente cancelando = %d", r.StatusCode)
	}
	_, frag := adv.fragDrawer("/diligencias/" + id + "/cancelar")
	for _, w := range []string{"<h2>Cancelar diligência</h2>", "O registro é preservado com o motivo do cancelamento.", "Motivo do cancelamento *", ">Cancelar diligência</button>"} {
		if !strings.Contains(frag, w) {
			t.Errorf("drawer sem %q", w)
		}
	}
	if r, b := adv.post("/diligencias/"+id+"/cancelar", motivoForm("1", "")); r.StatusCode != 400 || strings.TrimSpace(b) != "Informe o motivo do cancelamento." {
		t.Errorf("sem motivo = %d %q", r.StatusCode, b)
	}
	r, _ := adv.post("/diligencias/"+id+"/cancelar", motivoForm("1", "Perdeu o objeto"))
	if r.StatusCode != 200 || decodifica(t, r.Header.Get("X-Painel-Ok")) != "Diligência cancelada; o registro foi preservado." {
		t.Fatalf("cancelar = %d %q", r.StatusCode, r.Header.Get("X-Painel-Ok"))
	}
	if e.count(`SELECT count(*) FROM diligencias WHERE id=$1::uuid AND status='Cancelada' AND cancelado AND cancelamento_motivo='Perdeu o objeto' AND cancelado_por IS NOT NULL AND excluido_em IS NULL`, id) != 1 {
		t.Error("a diligência é preservada com status Cancelada, motivo e autor")
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='diligencia.cancelada' AND target_id=$1`, id) != 1 {
		t.Error("auditoria diligencia.cancelada")
	}
	_, pg = adv.get("/agenda")
	if !strings.Contains(pg, "Vistoria de teste") || !regexp.MustCompile(`<td>Cancelada</td>`).MatchString(pg) || !strings.Contains(pg, `<span class="badge badge-neutral">Cancelada</span>`) ||
		strings.Contains(pg, `/diligencias/`+id+`/editar`) {
		t.Error("a cancelada continua listada, com status Cancelada, selo e sem ações")
	}
	if r, b := adv.post("/diligencias/"+id, url.Values{"version": {"2"}, "titulo": {"x"}, "data_limite": {"2030-01-11"}}); r.StatusCode != 400 ||
		strings.TrimSpace(b) != "Diligência cancelada não pode ser editada." {
		t.Errorf("editar cancelada = %d %q", r.StatusCode, b)
	}
}

func TestCancelarIntimacaoNaoDescarta(t *testing.T) {
	e, adv, ass, adm, ids := agendaComDuasIntimacoes(t)
	_, pg := adv.get("/agenda")
	if strings.Contains(pg, "Descartar") || strings.Contains(pg, "descartada") {
		t.Error("Descartar deu lugar a Cancelar")
	}
	if !strings.Contains(pg, `<button type="button" class="btn-danger-text" data-drawer="/intimacoes/`+ids[0]+`/cancelar">Cancelar</button>`) ||
		!strings.Contains(pg, `class="btn-text">Confirmar</button>`) {
		t.Error("o advogado vê Confirmar e Cancelar nas intimações em triagem")
	}
	for nome, b := range map[string]*browser{"assistente": ass, "administrador": adm} {
		if st, _ := b.fragDrawer("/intimacoes/" + ids[0] + "/cancelar"); st != 403 {
			t.Errorf("%s abrindo o drawer = %d", nome, st)
		}
		if r, _ := b.post("/intimacoes/"+ids[0]+"/cancelar", url.Values{"motivo": {"x y"}}); r.StatusCode != 403 {
			t.Errorf("%s cancelando = %d", nome, r.StatusCode)
		}
	}
	_, frag := adv.fragDrawer("/intimacoes/" + ids[0] + "/cancelar")
	for _, w := range []string{"<h2>Cancelar intimação</h2>", "O registro é preservado com o motivo do cancelamento.", "Motivo do cancelamento *", ">Voltar</button>", ">Cancelar intimação</button>"} {
		if !strings.Contains(frag, w) {
			t.Errorf("drawer sem %q", w)
		}
	}
	if r, b := adv.post("/intimacoes/"+ids[0]+"/cancelar", url.Values{"motivo": {" "}}); r.StatusCode != 400 || strings.TrimSpace(b) != "Informe o motivo do cancelamento." {
		t.Errorf("sem motivo = %d %q", r.StatusCode, b)
	}
	r, _ := adv.post("/intimacoes/"+ids[0]+"/cancelar", url.Values{"motivo": {"Publicada em duplicidade"}})
	if r.StatusCode != 200 || decodifica(t, r.Header.Get("X-Painel-Ok")) != "Intimação cancelada e mantida no histórico." {
		t.Fatalf("cancelar = %d %q", r.StatusCode, r.Header.Get("X-Painel-Ok"))
	}
	if e.count(`SELECT count(*) FROM intimacoes WHERE id=$1::uuid AND status='cancelada' AND cancelamento_motivo='Publicada em duplicidade' AND cancelado_por IS NOT NULL AND cancelado_em IS NOT NULL`, ids[0]) != 1 {
		t.Error("registra motivo, quem e quando")
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='intimacao.cancelada' AND target_id=$1`, ids[0]) != 1 {
		t.Error("auditoria intimacao.cancelada")
	}
	_, pg = adv.get("/agenda")
	if !strings.Contains(pg, `<span class="badge badge-neutral">Cancelada</span>`) || strings.Contains(pg, `/intimacoes/`+ids[0]+`/cancelar`) || strings.Contains(pg, `/intimacoes/`+ids[0]+`/decidir`) ||
		!regexp.MustCompile(`<span>Intimações</span><span class="count">2</span>`).MatchString(pg) {
		t.Error("a cancelada permanece no histórico, com selo e sem ações")
	}
	if r, b := adv.post("/intimacoes/"+ids[0]+"/decidir", url.Values{"status": {"confirmada"}}); r.StatusCode != 400 || strings.TrimSpace(b) != "Esta intimação já foi decidida." {
		t.Errorf("confirmar a cancelada = %d %q", r.StatusCode, b)
	}
	if r, _ := adv.post("/intimacoes/"+ids[1]+"/decidir", url.Values{"status": {"descartada"}}); r.StatusCode != 400 {
		t.Errorf("'descartada' deixou de existir: %d", r.StatusCode)
	}
	if r, _ := adv.post("/intimacoes/"+ids[1]+"/decidir", url.Values{"status": {"cancelada"}}); r.StatusCode != 400 {
		t.Errorf("cancelar passa pelo drawer com motivo, não pelo decidir: %d", r.StatusCode)
	}
}
