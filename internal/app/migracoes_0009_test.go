package app

import (
	"context"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
)

func TestMigracao0009(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	_, uid := e.user("m9@exemplo.test", auth.Advogado)
	t.Cleanup(func() { _, _ = e.pool.Exec(ctx, `TRUNCATE pessoas, processos, atendimentos CASCADE`) })

	var pes, proc, ate string
	if err := e.pool.QueryRow(ctx, `INSERT INTO pessoas (codigo, tipo_pessoa, nome_completo, created_by, updated_by)
		VALUES ('PES. 0000001','Física','MARIA EXEMPLO',$1,$1) RETURNING id`, uid).Scan(&pes); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.QueryRow(ctx, `INSERT INTO processos (codigo, numero_cnj, situacao, created_by, updated_by)
		VALUES ('PRO. 0000001','','ativo',$1,$1) RETURNING id`, uid).Scan(&proc); err != nil {
		t.Fatal(err)
	}
	insAte := func(codigo, tipo, status, desc string) error {
		return e.pool.QueryRow(ctx, `INSERT INTO atendimentos (codigo, tipo, status, descricao, data_abertura, created_by, updated_by)
			VALUES ($1,$2,$3,$4,'2026-11-03',$5,$5) RETURNING id`, codigo, tipo, status, desc, uid).Scan(&ate)
	}
	if err := insAte("ATE. 0000001", "Outro", "Em triagem", "x"); err == nil {
		t.Error("tipo fora da lista deveria ser recusado")
	}
	if err := insAte("ATE. 0000001", "Triagem", "Arquivado", "x"); err == nil {
		t.Error("status fora da lista deveria ser recusado")
	}
	if err := insAte("ATE. 0000001", "Extrajudicial", "Em triagem", "Revisar contrato"); err != nil {
		t.Fatalf("atendimento válido: %v", err)
	}
	if err := insAte("ATE. 0000001", "Extrajudicial", "Em triagem", "outro"); err == nil {
		t.Error("código repetido deveria ser recusado")
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO atendimento_pessoas (atendimento_id, pessoa_id, papel, ordem) VALUES ($1,$2,'cliente',0)`, ate, pes); err != nil {
		t.Errorf("pessoa do atendimento: %v", err)
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO atendimento_pessoas (atendimento_id, pessoa_id, papel, ordem) VALUES ($1,$2,'testemunha',1)`, ate, pes); err == nil {
		t.Error("papel fora da lista deveria ser recusado")
	}
	if _, err := e.pool.Exec(ctx, `UPDATE processos SET atendimento_origem_id='00000000-0000-0000-0000-000000000000' WHERE id=$1`, proc); err == nil {
		t.Error("atendimento de origem inexistente deveria violar a chave estrangeira")
	}
	if _, err := e.pool.Exec(ctx, `UPDATE processos SET atendimento_origem_id=$2 WHERE id=$1`, proc, ate); err != nil {
		t.Errorf("atendimento de origem válido: %v", err)
	}

	// tarefas: vínculo obrigatório, exclusivo; listas fechadas.
	tar := func(estado, prio, sub, origem string, a, p any) error {
		_, err := e.pool.Exec(ctx, `INSERT INTO tarefas (codigo, titulo, subtitulo, estado, prioridade, origem_tipo, atendimento_id, processo_id, created_by, updated_by)
			VALUES ('TAR. '||substr(md5(random()::text),1,7),'Tarefa',$1,$2,$3,$4,$5,$6,$7,$7)`, sub, estado, prio, origem, a, p, uid)
		return err
	}
	if err := tar("pendente", "Normal", "", "atendimento", nil, nil); err == nil {
		t.Error("tarefa sem vínculo deveria ser recusada")
	}
	if err := tar("pendente", "Normal", "", "atendimento", ate, proc); err == nil {
		t.Error("tarefa com os dois vínculos deveria ser recusada")
	}
	if err := tar("pendente", "Normal", "", "processo", ate, nil); err == nil {
		t.Error("origem_tipo incoerente com o vínculo deveria ser recusada")
	}
	if err := tar("feita", "Normal", "", "atendimento", ate, nil); err == nil {
		t.Error("estado fora da lista deveria ser recusado")
	}
	if err := tar("pendente", "Média", "", "atendimento", ate, nil); err == nil {
		t.Error("prioridade fora da lista deveria ser recusada")
	}
	if err := tar("pendente", "Normal", strings.Repeat("a", 81), "atendimento", ate, nil); err == nil {
		t.Error("subtítulo com mais de 80 caracteres deveria ser recusado")
	}
	if err := tar("a-confirmar", "Urgente", "Ligação", "atendimento", ate, nil); err != nil {
		t.Errorf("tarefa válida (atendimento): %v", err)
	}
	if err := tar("canceladas", "Baixa", "", "processo", nil, proc); err != nil {
		t.Errorf("tarefa válida (processo): %v", err)
	}

	// prazos aceitam linha ligada à tarefa e sem processo; um prazo vivo por tarefa.
	var tid string
	if err := e.pool.QueryRow(ctx, `SELECT id FROM tarefas LIMIT 1`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	prazo := func() error {
		_, err := e.pool.Exec(ctx, `INSERT INTO prazos (descricao, data_fatal, situacao, tarefa_id, created_by, updated_by)
			VALUES ('Prazo da tarefa','2026-12-01','a_conferir',$1,$2,$2)`, tid, uid)
		return err
	}
	if err := prazo(); err != nil {
		t.Errorf("prazo ligado à tarefa e sem processo: %v", err)
	}
	if err := prazo(); err == nil {
		t.Error("dois prazos vivos para a mesma tarefa deveriam ser recusados")
	}

	// documentos aceitam atendimento e tarefa como vínculo.
	if _, err := e.pool.Exec(ctx, `INSERT INTO documentos (nome_original, tipo, tamanho, sha256, arquivo, atendimento_id, tarefa_id, created_by, updated_by)
		VALUES ('a.pdf','pdf',10,'\x00','arq-9',$1,$2,$3,$3)`, ate, tid, uid); err != nil {
		t.Errorf("documento ligado a atendimento/tarefa: %v", err)
	}

	// listas dinâmicas da tarefa.
	if _, err := e.pool.Exec(ctx, `INSERT INTO tarefa_partes (tarefa_id, pessoa_id, papel, ordem) VALUES ($1,$2,'Autor',0)`, tid, pes); err != nil {
		t.Errorf("parte da tarefa: %v", err)
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO tarefa_responsaveis (tarefa_id, pessoa_id, ordem) VALUES ($1,$2,0)`, tid, pes); err != nil {
		t.Errorf("responsável da tarefa: %v", err)
	}

	// Papel do banco: escreve nas tabelas novas; não apaga a auditoria.
	for _, tb := range []string{"atendimentos", "atendimento_pessoas", "tarefas", "tarefa_partes", "tarefa_responsaveis"} {
		for _, priv := range []string{"INSERT", "UPDATE"} {
			var ok bool
			if err := e.pool.QueryRow(ctx, `SELECT has_table_privilege('painel_app', $1, $2)`, tb, priv).Scan(&ok); err != nil || !ok {
				t.Errorf("painel_app sem %s em %s", priv, tb)
			}
		}
	}
	var del bool
	if err := e.pool.QueryRow(ctx, `SELECT has_table_privilege('painel_app', 'audit_events', 'DELETE')`).Scan(&del); err != nil || del {
		t.Error("painel_app NÃO pode apagar da auditoria")
	}
}
