package app

import (
	"context"
	"testing"

	"painelteodoro/internal/auth"
)

// Migração 0013 (Plano 6): monitor DataJud, movimentos, sigilosos e as marcas das tarefas automáticas.
func TestMigracao0013(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	_, uid := e.user("m13@exemplo.test", auth.Advogado)
	exec := func(sql string, args ...any) error { _, err := e.pool.Exec(ctx, sql, args...); return err }
	proc := func(cod, num string) string {
		var id string
		if err := e.pool.QueryRow(ctx, `INSERT INTO processos (codigo, numero_cnj, situacao, created_by, updated_by) VALUES ($1,$2,'ativo',$3,$3) RETURNING id::text`, cod, num, uid).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	p1, p2 := proc("PRO. 0000001", "0000101-25.2025.8.26.0100"), proc("PRO. 0000002", "0000102-25.2025.8.26.0100")
	tarefa := func(cod, p, estado string, monit bool) error {
		return exec(`INSERT INTO tarefas (codigo, titulo, estado, origem_tipo, processo_id, monitoramento, created_by, updated_by) VALUES ($1,'Monitoramento',$2,'processo',$3,$4,$5,$5)`, cod, estado, p, monit, uid)
	}
	// Uma tarefa de monitoramento ABERTA por processo; baixada não conta.
	if err := tarefa("TAR. 0000001", p1, "pendente", true); err != nil {
		t.Fatalf("primeira aberta: %v", err)
	}
	if err := tarefa("TAR. 0000002", p1, "pendente", true); err == nil {
		t.Error("duas tarefas de monitoramento abertas para o mesmo processo deveriam ser recusadas")
	}
	if err := exec(`UPDATE tarefas SET estado='concluidas-com-sucesso' WHERE codigo='TAR. 0000001'`); err != nil {
		t.Fatal(err)
	}
	if err := tarefa("TAR. 0000003", p1, "pendente", true); err != nil {
		t.Errorf("uma baixada + uma aberta deveria ser aceito: %v", err)
	}
	if err := tarefa("TAR. 0000004", p2, "pendente", false); err != nil {
		t.Errorf("tarefa comum não é limitada: %v", err)
	}
	// Tarefa consolidada: uma por processo e por dia.
	cons := func(cod, p, dia string) error {
		return exec(`INSERT INTO tarefas (codigo, titulo, estado, origem_tipo, processo_id, consolidada_dia, prazo_interno, prazo_fatal, created_by, updated_by)
			VALUES ($1,'Consolidada','a-confirmar','processo',$2,$3::date,true,$3::date,$4,$4)`, cod, p, dia, uid)
	}
	if err := cons("TAR. 0000005", p1, "2026-11-04"); err != nil {
		t.Fatalf("consolidada: %v", err)
	}
	if err := cons("TAR. 0000006", p1, "2026-11-04"); err == nil {
		t.Error("duas consolidadas do mesmo processo e dia deveriam ser recusadas")
	}
	if err := cons("TAR. 0000007", p1, "2026-11-05"); err != nil {
		t.Errorf("outro dia: %v", err)
	}
	// Monitor por processo: status restrito.
	if err := exec(`INSERT INTO monitor_processos (processo_id, tribunal_alias, status) VALUES ($1,'tjsp','sucesso')`, p1); err != nil {
		t.Fatalf("monitor_processos: %v", err)
	}
	if err := exec(`INSERT INTO monitor_processos (processo_id, status) VALUES ($1,'otimo')`, p2); err == nil {
		t.Error("status inválido deveria ser recusado")
	}
	// Movimentos: único por (processo, fonte, chave); fonte restrita; complemento limitado; base e vínculo com a intimação.
	mov := func(chave, fonte string) error {
		return exec(`INSERT INTO processo_movimentos (processo_id, fonte, grau, chave, data, codigo, nome, base) VALUES ($1,$2,'G1',$3,'2026-11-01','26','Distribuição',true)`, p1, fonte, chave)
	}
	if err := mov("k1", "DataJud"); err != nil {
		t.Fatalf("movimento: %v", err)
	}
	if err := mov("k1", "DataJud"); err == nil {
		t.Error("movimento duplicado deveria ser recusado")
	}
	if err := mov("k1", "Outra"); err == nil {
		t.Error("fonte inválida deveria ser recusada")
	}
	if err := exec(`INSERT INTO processo_movimentos (processo_id, fonte, chave, nome, complemento) VALUES ($1,'DataJud','k2','x',repeat('a',501))`, p1); err == nil {
		t.Error("complemento acima de 500 caracteres deveria ser recusado")
	}
	var iid string
	if err := e.pool.QueryRow(ctx, `INSERT INTO intimacoes (hash, origem, texto) VALUES ('h13','DataJud',repeat('é',5000)) RETURNING id::text`).Scan(&iid); err != nil {
		t.Fatalf("intimação DataJud sem OAB e com texto longo: %v", err)
	}
	if err := exec(`UPDATE processo_movimentos SET intimacao_id=$1::uuid, base=false WHERE chave='k1'`, iid); err != nil {
		t.Errorf("vínculo movimento-intimação: %v", err)
	}
	if err := exec(`INSERT INTO intimacoes (hash, origem) VALUES ('h13b','Outra')`); err == nil {
		t.Error("origem inválida deveria ser recusada")
	}
	if err := exec(`INSERT INTO intimacoes (hash, origem) VALUES ('h13c','DJEN (sigiloso)')`); err != nil {
		t.Errorf("origem 'DJEN (sigiloso)': %v", err)
	}
	// Sigilosos: processo existente, um por processo.
	var oab string
	if err := e.pool.QueryRow(ctx, `INSERT INTO oab_monitoradas (advogado_nome, numero_oab, uf, autorizacao_base, autorizacao_em, created_by, updated_by)
		VALUES ('Adv Teste','123456','SP','Distribuição','2026-11-01',$1,$1) RETURNING id::text`, uid).Scan(&oab); err != nil {
		t.Fatal(err)
	}
	sig := func(p string) error {
		return exec(`INSERT INTO monitor_sigilosos (processo_id, oab_id, created_by, updated_by) VALUES ($1::uuid,$2::uuid,$3,$3)`, p, oab, uid)
	}
	if err := sig("00000000-0000-0000-0000-000000000000"); err == nil {
		t.Error("sigiloso exige processo existente")
	}
	if err := sig(p1); err != nil {
		t.Fatalf("sigiloso: %v", err)
	}
	if err := sig(p1); err == nil {
		t.Error("um sigiloso por processo")
	}
}
