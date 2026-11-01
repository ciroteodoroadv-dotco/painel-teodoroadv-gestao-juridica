package app

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/auth"
)

func pessoaForm(nome, doc string) url.Values {
	return url.Values{"tipo": {"PF"}, "nome": {nome}, "documento": {doc}, "email": {"p@exemplo.test"},
		"telefone": {"(11) 91234-5678"}, "papel": {"cliente"}}
}

func TestPessoasCicloCompleto(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)

	// Assistente lê, mas não escreve.
	if r, _ := ass.get("/pessoas"); r.StatusCode != 200 {
		t.Fatalf("assistente listando = %d", r.StatusCode)
	}
	if r, _ := ass.post("/pessoas", pessoaForm("Ana Souza", "529.982.247-25")); r.StatusCode != 403 {
		t.Fatalf("assistente criando pessoa = %d, esperado 403", r.StatusCode)
	}

	// Validações.
	if r, body := adv.post("/pessoas", pessoaForm("Ana Souza", "529.982.247-24")); r.StatusCode != 400 || !strings.Contains(body, "CPF inválido") {
		t.Fatalf("CPF inválido = %d", r.StatusCode)
	}
	if r, _ := adv.post("/pessoas", pessoaForm("A", "")); r.StatusCode != 400 {
		t.Fatalf("nome curto = %d", r.StatusCode)
	}

	// Criação.
	r, _ := adv.post("/pessoas", pessoaForm("Ana Souza", "529.982.247-25"))
	id := idFrom(t, r, "/pessoas/")

	// LGPD: nada de CPF em claro no banco; só o cifrado e o índice cego.
	var enc, idx []byte
	if err := e.pool.QueryRow(context.Background(), `SELECT documento_enc, documento_idx FROM pessoas WHERE id=$1`, id).Scan(&enc, &idx); err != nil {
		t.Fatal(err)
	}
	if len(enc) == 0 || len(idx) != 32 || strings.Contains(string(enc), "52998224725") {
		t.Fatalf("documento deveria estar cifrado: enc=%d idx=%d", len(enc), len(idx))
	}
	if e.count(`SELECT count(*) FROM pessoas WHERE nome ILIKE '%52998224725%' OR email ILIKE '%52998224725%'`) != 0 {
		t.Error("CPF vazou em coluna comum")
	}

	// Duplicidade de documento.
	if r, body := adv.post("/pessoas", pessoaForm("Outra Ana", "52998224725")); r.StatusCode != 400 || !strings.Contains(body, "Já existe") {
		t.Fatalf("documento duplicado = %d", r.StatusCode)
	}

	// Detalhe mostra só a máscara; a edição revela e audita.
	_, det := adv.get("/pessoas/" + id)
	if !strings.Contains(det, "***.***.***-25") || strings.Contains(det, "529.982.247-25") {
		t.Error("detalhe deveria mostrar apenas o CPF mascarado")
	}
	_, ed := adv.get("/pessoas/" + id + "/editar")
	if !strings.Contains(ed, "529.982.247-25") {
		t.Error("o formulário de edição deveria mostrar o documento")
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='pessoa.documento_revelado' AND target_id=$1`, id) == 0 {
		t.Error("revelar documento deveria ser auditado")
	}

	// Busca por nome e por CPF completo; fragmento htmx não traz a página inteira.
	if _, body := adv.get("/pessoas?q=ana"); !strings.Contains(body, "Ana Souza") {
		t.Error("busca por nome falhou")
	}
	if _, body := adv.get("/pessoas?q=529.982.247-25"); !strings.Contains(body, "Ana Souza") {
		t.Error("busca por CPF completo falhou")
	}
	if _, body := adv.get("/pessoas?q=zzzz"); strings.Contains(body, "Ana Souza") {
		t.Error("busca sem correspondência trouxe resultado")
	}
	_, frag := adv.do("GET", "/pessoas?q=ana", nil, map[string]string{"HX-Request": "true"})
	if strings.Contains(frag, "<html") || !strings.Contains(frag, "Ana Souza") {
		t.Error("requisição htmx deveria receber só o fragmento da tabela")
	}

	// Concorrência otimista: a 2ª alteração com a versão antiga é recusada.
	f := pessoaForm("Ana Souza Lima", "529.982.247-25")
	f.Set("version", "1")
	if r, _ := adv.post("/pessoas/"+id, f); r.StatusCode != 303 {
		t.Fatalf("1ª alteração = %d", r.StatusCode)
	}
	f2 := pessoaForm("Ana Souza Pereira", "529.982.247-25")
	f2.Set("version", "1")
	if r, _ := adv.post("/pessoas/"+id, f2); r.StatusCode != 409 {
		t.Fatalf("alteração com versão velha = %d, esperado 409", r.StatusCode)
	}
	if e.count(`SELECT count(*) FROM pessoas WHERE nome='Ana Souza Lima'`) != 1 {
		t.Error("a alteração válida deveria ter prevalecido")
	}

	// Relatório do titular e exclusão (marcação, sem apagar).
	if r, body := adv.get("/pessoas/" + id + "/relatorio"); r.StatusCode != 200 || !strings.Contains(body, "529.982.247-25") {
		t.Fatalf("relatório do titular = %d", r.StatusCode)
	}
	if r, _ := ass.get("/pessoas/" + id + "/relatorio"); r.StatusCode != 403 {
		t.Errorf("assistente no relatório do titular = %d, esperado 403", r.StatusCode)
	}
	if r, _ := adv.post("/pessoas/"+id+"/excluir", url.Values{"version": {"2"}}); r.StatusCode != 303 {
		t.Fatalf("exclusão = %d", r.StatusCode)
	}
	if _, body := adv.get("/pessoas"); strings.Contains(body, "Ana Souza") {
		t.Error("pessoa excluída ainda aparece na lista")
	}
	if e.count(`SELECT count(*) FROM pessoas WHERE id=$1 AND excluido_em IS NOT NULL`, id) != 1 {
		t.Error("exclusão deveria apenas marcar o registro")
	}
	for _, a := range []string{"pessoa.criada", "pessoa.alterada", "pessoa.excluida", "pessoa.relatorio_titular"} {
		if e.count(`SELECT count(*) FROM audit_events WHERE action=$1`, a) == 0 {
			t.Errorf("evento %q não foi auditado", a)
		}
	}
	if n, err := audit.Verify(context.Background(), e.pool); err != nil || n == 0 {
		t.Fatalf("cadeia de auditoria: n=%d err=%v", n, err)
	}
}
