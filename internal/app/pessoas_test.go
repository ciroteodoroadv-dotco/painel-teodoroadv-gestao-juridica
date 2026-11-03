package app

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/auth"
)

var inlineProibido = regexp.MustCompile(`(?i)\son[a-z]+\s*=|\sstyle\s*=|<script>|javascript:`)

func TestPessoasCicloCompleto(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	ass, _ := e.user("ass@exemplo.test", auth.Assistente)

	// Assistente lê, mas não escreve (nem abre os formulários).
	if r, _ := ass.get("/pessoas"); r.StatusCode != 200 {
		t.Fatalf("assistente listando = %d", r.StatusCode)
	}
	if r, _ := ass.post("/pessoas", pessoaForm("Ana Souza", "529.982.247-25")); r.StatusCode != 403 {
		t.Fatalf("assistente criando pessoa = %d, esperado 403", r.StatusCode)
	}
	if r, _ := ass.get("/pessoas/nova"); r.StatusCode != 403 {
		t.Fatalf("assistente abrindo o formulário = %d, esperado 403", r.StatusCode)
	}

	// Validações do servidor: mensagem do B, status 400, texto simples (o JS mostra como toast).
	r, body := adv.post("/pessoas", pessoaForm("Ana Souza", "529.982.247-24"))
	if r.StatusCode != 400 || !strings.Contains(body, "CPF inválido — confira os dígitos informados.") || strings.Contains(r.Header.Get("Content-Type"), "html") {
		t.Fatalf("CPF inválido = %d %q", r.StatusCode, body)
	}
	if r, body := adv.post("/pessoas", pessoaForm("  ", "")); r.StatusCode != 400 || !strings.Contains(body, "Nome completo é obrigatório.") {
		t.Fatalf("nome vazio = %d %q", r.StatusCode, body)
	}

	// Criação: toast com o código; id no cabeçalho.
	r, _ = adv.post("/pessoas", pessoaForm("Ana Souza", "529.982.247-25"))
	if r.StatusCode != 200 || r.Header.Get("X-Painel-Ok") != "Pessoa criada — código PES. 0000001." {
		t.Fatalf("criação = %d %q", r.StatusCode, r.Header.Get("X-Painel-Ok"))
	}
	id := r.Header.Get("X-Painel-Id")

	// LGPD: nada de CPF em claro no banco; só o cifrado e o índice cego.
	var enc, idx []byte
	if err := e.pool.QueryRow(context.Background(), `SELECT documento_enc, documento_idx FROM pessoas WHERE id=$1`, id).Scan(&enc, &idx); err != nil {
		t.Fatal(err)
	}
	if len(enc) == 0 || len(idx) != 32 || strings.Contains(string(enc), "52998224725") {
		t.Fatalf("documento deveria estar cifrado: enc=%d idx=%d", len(enc), len(idx))
	}
	if e.count(`SELECT count(*) FROM pessoas WHERE nome_completo ILIKE '%52998224725%' OR email ILIKE '%52998224725%' OR observacoes ILIKE '%52998224725%'`) != 0 {
		t.Error("CPF vazou em coluna comum")
	}

	// Duplicidade de documento.
	if r, body := adv.post("/pessoas", pessoaForm("Outra Ana", "52998224725")); r.StatusCode != 400 || body != "Já existe uma pessoa com esse documento.\n" {
		t.Fatalf("documento duplicado = %d %q", r.StatusCode, body)
	}

	// Lista: quem escreve vê o documento inteiro; quem só lê, mascarado. Acordeão do B com os quatro grupos.
	_, lista := adv.get("/pessoas")
	if !strings.Contains(lista, "529.982.247-25") || !strings.Contains(lista, "ANA SOUZA") || !strings.Contains(lista, "PES. 0000001") {
		t.Error("a lista de quem escreve deveria trazer nome em maiúsculas, código e documento inteiro")
	}
	for _, rot := range []string{"Cliente", "Parte adversa", "Terceiro interessado", "Sem processo vinculado"} {
		if !strings.Contains(lista, "<span>"+rot+"</span>") {
			t.Errorf("faltou o grupo %q no acordeão", rot)
		}
	}
	_, listaAss := ass.get("/pessoas")
	if strings.Contains(listaAss, "529.982.247-25") || !strings.Contains(listaAss, "***.***.***-25") {
		t.Error("a lista de quem só lê deve trazer o documento mascarado")
	}
	if strings.Contains(listaAss, "Nova pessoa") || strings.Contains(listaAss, ">Editar<") || strings.Contains(listaAss, ">Excluir<") {
		t.Error("quem só lê não deve ver botões de escrita")
	}
	for nome, h := range map[string]string{"adv": lista, "ass": listaAss} {
		if inlineProibido.MatchString(h) {
			t.Errorf("página de pessoas (%s) com on*=/style=/script inline: %q", nome, inlineProibido.FindString(h))
		}
	}
	if !strings.Contains(lista, `data-drawer="/pessoas/nova"`) {
		t.Error("o botão do topo deveria abrir o drawer")
	}

	// Drawer de edição: fragmento que começa por <h2>, mostra o documento e audita a revelação.
	r, ed := adv.do("GET", "/pessoas/"+id+"/editar", nil, map[string]string{"HX-Request": "true"})
	if r.StatusCode != 200 || !strings.HasPrefix(strings.TrimSpace(ed), "<h2>Editar pessoa</h2>") || strings.Contains(ed, "<html") {
		t.Fatalf("drawer de edição = %d %.60q", r.StatusCode, ed)
	}
	if !strings.Contains(ed, `value="529.982.247-25"`) || !strings.Contains(ed, "disabled") ||
		!strings.Contains(ed, "Tipo de pessoa não se troca depois de criada — exclua e recrie, se for o caso.") {
		t.Error("a edição deveria mostrar o CPF, travar o tipo e trazer a dica do B")
	}
	if inlineProibido.MatchString(ed) {
		t.Errorf("drawer com atributo inline: %q", inlineProibido.FindString(ed))
	}
	if e.count(`SELECT count(*) FROM audit_events WHERE action='pessoa.documento_revelado' AND target_id=$1`, id) == 0 {
		t.Error("revelar documento deveria ser auditado")
	}
	if _, nova := adv.get("/pessoas/nova"); !strings.HasPrefix(strings.TrimSpace(nova), "<h2>Nova pessoa</h2>") ||
		!strings.Contains(nova, "Qualificação nos termos do art. 319 do CPC.") || !strings.Contains(nova, "Salvar cadastro") {
		t.Error("drawer 'Nova pessoa' fora do texto do B")
	}

	// Concorrência otimista: a 2ª alteração com a versão antiga é recusada.
	f := pessoaForm("Ana Souza Lima", "529.982.247-25")
	f.Set("version", "1")
	if r, _ := adv.post("/pessoas/"+id, f); r.StatusCode != 200 || r.Header.Get("X-Painel-Ok") != "Pessoa atualizada." {
		t.Fatalf("1ª alteração = %d", r.StatusCode)
	}
	f2 := pessoaForm("Ana Souza Pereira", "529.982.247-25")
	f2.Set("version", "1")
	if r, _ := adv.post("/pessoas/"+id, f2); r.StatusCode != 409 {
		t.Fatalf("alteração com versão velha = %d, esperado 409", r.StatusCode)
	}
	if e.count(`SELECT count(*) FROM pessoas WHERE nome_completo='ANA SOUZA LIMA'`) != 1 {
		t.Error("a alteração válida deveria ter prevalecido")
	}

	// Relatório do titular e exclusão lógica (pessoa sem processo: o drawer pede confirmação).
	if r, body := adv.get("/pessoas/" + id + "/relatorio"); r.StatusCode != 200 || !strings.Contains(body, "529.982.247-25") {
		t.Fatalf("relatório do titular = %d", r.StatusCode)
	}
	if r, _ := ass.get("/pessoas/" + id + "/relatorio"); r.StatusCode != 403 {
		t.Errorf("assistente no relatório do titular = %d, esperado 403", r.StatusCode)
	}
	if _, conf := adv.get("/pessoas/" + id + "/excluir"); !strings.Contains(conf, "Excluir esta ficha? Ela deixa de aparecer nas listas e a exclusão fica registrada na auditoria.") {
		t.Error("o drawer de exclusão deveria trazer o texto de exclusão lógica")
	}
	if r, _ := adv.post("/pessoas/"+id+"/excluir", url.Values{"version": {"2"}}); r.StatusCode != 200 || r.Header.Get("X-Painel-Ok") != "Pessoa excluída." {
		t.Fatalf("exclusão = %d", r.StatusCode)
	}
	if _, body := adv.get("/pessoas"); strings.Contains(body, "ANA SOUZA") {
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

func TestPessoasEstadoVazioECsrf(t *testing.T) {
	e := newEnv(t)
	adv, _ := e.user("adv@exemplo.test", auth.Advogado)
	_, vazio := adv.get("/pessoas")
	if !strings.Contains(vazio, "<h3>Nenhuma pessoa cadastrada</h3>") || !strings.Contains(vazio, `Crie a primeira pessoa pelo botão "Nova pessoa" acima.`) ||
		!strings.Contains(vazio, `placeholder="Buscar por nome…"`) {
		t.Error("estado vazio fora do texto do B")
	}
	// Sem o token CSRF, a escrita é negada.
	f := pessoaForm("Sem Token", "")
	req := adv.semCSRF("POST", "/pessoas", f)
	if req != 403 {
		t.Errorf("POST sem CSRF = %d, esperado 403", req)
	}
	if e.count(`SELECT count(*) FROM pessoas`) != 0 {
		t.Error("nada deveria ter sido gravado")
	}
}
