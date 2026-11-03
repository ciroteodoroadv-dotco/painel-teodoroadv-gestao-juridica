package app

import (
	"net/url"
	"strings"
	"testing"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/br"
)

func diligenciaForm(desc, data, hora string) url.Values {
	f := compromissoForm("Diligência", desc, data, hora)
	f.Set("responsavel", "Fulano de Tal")
	return f
}

func TestDiligencias(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)
	adm, _ := e.user("adm@exemplo.test", auth.Administrador)
	ctx := t.Context()

	// Mesmas validações do "Novo compromisso" do B.
	if r, b := adv.post("/agenda", diligenciaForm("", dataFutura, "10:00")); r.StatusCode != 400 || !strings.Contains(b, "Descrição, data e hora são obrigatórios.") {
		t.Errorf("sem descrição = %d %s", r.StatusCode, b)
	}
	if r, _ := adm.post("/agenda", diligenciaForm("Protocolo presencial", dataFutura, "10:00")); r.StatusCode != 403 {
		t.Errorf("administrador criando diligência = %d, esperado 403", r.StatusCode)
	}

	// Criar pelo "Novo compromisso" com tipo Diligência: Descrição vira Título, Data vira Data-limite; status Pendente.
	pes := adv.criarPessoa(t, "Dora Lopes", "")
	num := br.MakeCNJ("0000800", "2024", "8", "26", "0100")
	proc := adv.criarProcesso(t, num)
	f := diligenciaForm("Protocolo presencial no fórum", dataFutura, "10:00")
	f.Set("cliente", pes)
	f.Set("processo", proc)
	f.Set("atendimento", "00000000-0000-0000-0000-000000000000") // não vale para diligência (como no B): ignorado
	if m := okMsg(t, adv.mustPost("/agenda", f)); m != "Diligência criada — código DIL. 0000001." {
		t.Errorf("toast = %q", m)
	}
	if m := okMsg(t, ass.mustPost("/agenda", diligenciaForm("Perícia no local", "2030-05-07", "14:00"))); m != "Diligência criada — código DIL. 0000002." {
		t.Errorf("toast = %q", m)
	}
	if e.count(`SELECT count(*) FROM agenda_compromissos`) != 0 {
		t.Error("diligência não pode virar compromisso")
	}
	if e.count(`SELECT count(*) FROM diligencias WHERE codigo='DIL. 0000001' AND titulo='Protocolo presencial no fórum' AND data_limite='2030-05-06'
		AND to_char(hora,'HH24:MI')='10:00' AND responsavel='Fulano de Tal' AND status='Pendente' AND processo_id=$1 AND processo_numero=$2 AND cliente_pessoa_id=$3`, proc, num, pes) != 1 {
		t.Error("diligência gravada fora do esperado")
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='diligencia.criada'`) != 2 {
		t.Error("criação não auditada")
	}

	// Conflito: bloqueada quando cai no mesmo dia e hora de uma audiência (item crítico).
	adv.okPostTeste(t, "/audiencias", audienciaForm("sustentacao-oral", "Sustentação marcada", "2030-08-01", "16:00"))
	r, b := adv.post("/agenda", diligenciaForm("Em cima da audiência", "2030-08-01", "16:00"))
	if r.StatusCode != 400 || !strings.Contains(b, `Bloqueado: conflito de horário com Audiência "Sustentação marcada" em 2030-08-01 às 16:00.`) {
		t.Errorf("conflito = %d %s", r.StatusCode, b)
	}
	if m := okMsg(t, adv.mustPost("/agenda", diligenciaForm("Outra hora", "2030-08-01", "17:00"))); !strings.HasPrefix(m, "Diligência criada") {
		t.Errorf("hora livre = %q", m)
	}

	// Edição: drawer do B, status não se edita, versão otimista.
	var id string
	if err := e.pool.QueryRow(ctx, `SELECT id FROM diligencias WHERE codigo='DIL. 0000001'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	_, frag := adv.do("GET", "/diligencias/"+id+"/editar", nil, map[string]string{"HX-Request": "true"})
	ordem := []string{"<h2>Editar diligência</h2>", `<p class="drawer-sub">Protocolo presencial no fórum</p>`, "Título *", "Processo vinculado",
		"Cliente", "Data-limite *", "Hora", "Local", "Responsável", "Status não se edita por aqui.", "Cancelar", "Salvar alterações"}
	pos := 0
	for _, s := range ordem {
		i := strings.Index(frag[pos:], s)
		if i < 0 {
			t.Fatalf("texto %q ausente ou fora de ordem:\n%s", s, frag)
		}
		pos += i + len(s)
	}
	g := url.Values{"titulo": {"Protocolo remarcado"}, "processo": {num}, "cliente": {pes}, "data_limite": {"2030-05-08"}, "hora": {"11:00"},
		"local": {"Fórum Central"}, "responsavel": {"Beltrano"}, "version": {"1"}, "status": {"Concluída"}}
	if m := okMsg(t, adv.mustPost("/diligencias/"+id, g)); m != "Diligência atualizada." {
		t.Errorf("toast = %q", m)
	}
	if e.count(`SELECT count(*) FROM diligencias WHERE id=$1 AND titulo='Protocolo remarcado' AND status='Pendente' AND responsavel='Beltrano' AND version=2`, id) != 1 {
		t.Error("edição incorreta (o status não pode mudar por aqui)")
	}
	if r, _ := adv.post("/diligencias/"+id, g); r.StatusCode != 409 {
		t.Errorf("versão velha = %d", r.StatusCode)
	}
	g.Set("version", "2")
	g.Set("titulo", "")
	if r, _ := adv.post("/diligencias/"+id, g); r.StatusCode != 400 {
		t.Errorf("sem título = %d", r.StatusCode)
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='diligencia.atualizada'`) != 1 {
		t.Error("edição não auditada")
	}
	if _, err := audit.Verify(ctx, e.pool); err != nil {
		t.Fatal(err)
	}
}

// okPostTeste faz o POST e falha se o status não for 200.
func (b *browser) okPostTeste(t *testing.T, path string, f url.Values) string {
	t.Helper()
	return okMsg(t, b.mustPost(path, f))
}
