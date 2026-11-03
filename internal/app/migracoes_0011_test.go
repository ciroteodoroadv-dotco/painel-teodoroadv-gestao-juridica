package app

import (
	"context"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
)

func TestMigracao0011(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	_, uid := e.user("m11@exemplo.test", auth.Advogado)
	exec := func(sql string, args ...any) error { _, err := e.pool.Exec(ctx, sql, args...); return err }
	var oab string
	if err := e.pool.QueryRow(ctx, `INSERT INTO oab_monitoradas (advogado_nome, numero_oab, uf, autorizacao_base, autorizacao_em, created_by, updated_by)
		VALUES ('Advogado Fictício','123456','SP','Distribuição dos processos em nome do advogado','2026-11-01',$1,$1) RETURNING id::text`, uid).Scan(&oab); err != nil {
		t.Fatalf("OAB válida: %v", err)
	}
	if err := exec(`INSERT INTO oab_monitoradas (advogado_nome, numero_oab, uf, autorizacao_base, created_by, updated_by) VALUES ('Sem Data','999','SP','Base qualquer',$1,$1)`, uid); err == nil {
		t.Error("OAB sem data da autorização deveria ser recusada")
	}
	if err := exec(`INSERT INTO oab_monitoradas (advogado_nome, numero_oab, uf, autorizacao_base, autorizacao_em, created_by, updated_by) VALUES ('Sem Base','998','SP','  ','2026-11-01',$1,$1)`, uid); err == nil {
		t.Error("OAB sem a base da autorização deveria ser recusada")
	}
	if err := exec(`INSERT INTO oab_monitoradas (advogado_nome, numero_oab, uf, autorizacao_base, autorizacao_em, created_by, updated_by) VALUES ('Repetida','123456','SP','Base','2026-11-01',$1,$1)`, uid); err == nil {
		t.Error("OAB repetida (número+UF) deveria ser recusada")
	}
	ins := func(hash, status, resumo, link string) error {
		return exec(`INSERT INTO intimacoes (hash, oab_id, status, resumo, link, decidida_por) VALUES ($1,$2,$3,$4,$5, CASE WHEN $3='triagem' THEN NULL ELSE $6::uuid END)`,
			hash, oab, status, resumo, link, uid)
	}
	for i, st := range []string{"triagem", "confirmada", "descartada"} {
		if err := ins("h"+string(rune('1'+i)), st, "resumo", "https://exemplo.test/a"); err != nil {
			t.Errorf("status %q deveria ser aceito: %v", st, err)
		}
	}
	if err := ins("h4", "em_aberto", "x", ""); err == nil {
		t.Error("status fora da lista deveria ser recusado")
	}
	if err := ins("h1", "triagem", "x", ""); err == nil {
		t.Error("hash repetido deveria ser recusado")
	}
	if err := ins("h5", "triagem", strings.Repeat("ã", 2000), ""); err != nil {
		t.Errorf("resumo de 2000 caracteres (4000 bytes) deveria ser aceito: %v", err)
	}
	if err := ins("h6", "triagem", strings.Repeat("ã", 2001), ""); err == nil {
		t.Error("resumo de 2001 caracteres deveria ser recusado")
	}
	if err := ins("h7", "triagem", "x", "http://inseguro.test"); err == nil {
		t.Error("link http:// deveria ser recusado")
	}

	// Tarefa com origem 'intimacao' exige intimacao_id e nenhum outro vínculo.
	var intim string
	if err := e.pool.QueryRow(ctx, `SELECT id::text FROM intimacoes WHERE hash='h1'`).Scan(&intim); err != nil {
		t.Fatal(err)
	}
	tar := func(cod, origem string, id any) error {
		return exec(`INSERT INTO tarefas (codigo, titulo, estado, origem_tipo, intimacao_id, created_by, updated_by) VALUES ($1,'Tarefa de teste','a-confirmar',$2,$3,$4,$4)`, cod, origem, id, uid)
	}
	if err := tar("TAR. 9000001", "intimacao", intim); err != nil {
		t.Errorf("tarefa de intimação: %v", err)
	}
	if err := tar("TAR. 9000002", "intimacao", intim); err == nil {
		t.Error("duas tarefas para a mesma intimação deveriam ser recusadas")
	}
	if err := tar("TAR. 9000003", "intimacao", nil); err == nil {
		t.Error("origem intimacao sem intimacao_id deveria ser recusada")
	}
	if err := tar("TAR. 9000004", "atendimento", intim); err == nil {
		t.Error("origem atendimento sem atendimento_id deveria ser recusada")
	}

	// coletas: 5 status.
	for _, st := range []string{"sucesso", "parcial", "falha", "bloqueado", "nao_sincronizado"} {
		if err := exec(`INSERT INTO coletas (modulo, origem, status) VALUES ($1,'DJEN',$2)`, "m-"+st, st); err != nil {
			t.Errorf("status de coleta %q: %v", st, err)
		}
	}
	if err := exec(`INSERT INTO coletas (modulo, origem, status) VALUES ('m-x','DJEN','ok')`); err == nil {
		t.Error("status de coleta fora da lista deveria ser recusado")
	}
	// painel_app não apaga a auditoria (continua valendo com as tabelas novas).
	if e.count(`SELECT count(*) FROM information_schema.role_table_grants WHERE grantee='painel_app' AND table_name='audit_events' AND privilege_type IN ('UPDATE','DELETE')`) != 0 {
		t.Error("painel_app não pode alterar nem apagar auditoria")
	}
	if e.count(`SELECT count(*) FROM information_schema.role_table_grants WHERE grantee='painel_app' AND table_name='intimacoes' AND privilege_type='INSERT'`) != 1 {
		t.Error("painel_app deveria poder gravar intimações")
	}
}
