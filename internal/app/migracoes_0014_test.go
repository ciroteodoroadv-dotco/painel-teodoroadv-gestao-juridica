package app

import (
	"context"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
)

// Migração 0014 (Plano 7): calendário forense com fonte obrigatória; nada se apaga.
func TestMigracao0014(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	_, uid := e.user("m14@exemplo.test", auth.Advogado)
	t.Cleanup(func() { _, _ = e.pool.Exec(context.Background(), `TRUNCATE calendario_dias, calendarios CASCADE`) })
	exec := func(sql string, args ...any) error { _, err := e.pool.Exec(ctx, sql, args...); return err }
	var cal string
	if err := e.pool.QueryRow(ctx, `INSERT INTO calendarios (sigla, nome, ambito, created_by) VALUES ('TJTESTE','Tribunal de teste','estadual',$1) RETURNING id::text`, uid).Scan(&cal); err != nil {
		t.Fatalf("calendario: %v", err)
	}
	dia := func(data, tipo string, conta bool, fonte string, verificador any) error {
		return exec(`INSERT INTO calendario_dias (calendario_id, data, tipo, conta_como_nao_util, descricao, fonte_citacao, verificado_por, created_by)
			VALUES ($1,$2::date,$3,$4,'Dia de teste',$5,$6,$7)`, cal, data, tipo, conta, fonte, verificador, uid)
	}
	// Sem fonte não grava (vazia ou só espaços).
	for _, f := range []string{"", "   ", "ab"} {
		if err := dia("2026-03-03", "feriado", true, f, uid); err == nil {
			t.Errorf("fonte %q deveria ser recusada", f)
		}
	}
	if err := dia("2026-03-03", "feriado", true, "Portaria TJ-TESTE 1/2026", uid); err != nil {
		t.Fatalf("dia com fonte: %v", err)
	}
	// Unicidade (calendário, data, tipo) entre dias ativos.
	if err := dia("2026-03-03", "feriado", true, "Portaria TJ-TESTE 2/2026", uid); err == nil {
		t.Error("duplicidade deveria ser recusada")
	}
	// Ponto facultativo: contar como não útil só com quem verificou.
	if err := dia("2026-03-04", "ponto_facultativo", true, "Portaria TJ-TESTE 3/2026", nil); err == nil {
		t.Error("ponto facultativo como não útil sem verificador deveria ser recusado")
	}
	if err := dia("2026-03-04", "ponto_facultativo", false, "Portaria TJ-TESTE 3/2026", nil); err != nil {
		t.Errorf("ponto facultativo sem efeito é aceito: %v", err)
	}
	if err := dia("2026-03-05", "ponto_facultativo", true, "Portaria TJ-TESTE 3/2026", uid); err != nil {
		t.Errorf("ponto facultativo verificado: %v", err)
	}
	// Tipo inválido.
	if err := dia("2026-03-06", "folga", true, "Portaria TJ-TESTE 4/2026", uid); err == nil {
		t.Error("tipo inválido deveria ser recusado")
	}
	// Inativar exige motivo; depois do inativado o mesmo dia pode ser recadastrado corretamente.
	if err := exec(`UPDATE calendario_dias SET ativo=false WHERE data='2026-03-03'`); err == nil {
		t.Error("inativar sem motivo deveria ser recusado")
	}
	if err := exec(`UPDATE calendario_dias SET ativo=false, motivo_inativacao='data errada' WHERE data='2026-03-03'`); err != nil {
		t.Fatalf("inativar: %v", err)
	}
	if err := dia("2026-03-03", "feriado", true, "Portaria TJ-TESTE 5/2026", uid); err != nil {
		t.Errorf("recadastro após inativar: %v", err)
	}
	// Nada se apaga: nem o app (privilégio) nem o dono (gatilho).
	var del bool
	if err := e.pool.QueryRow(ctx, `SELECT has_table_privilege('painel_app','calendario_dias','DELETE') OR has_table_privilege('painel_app','calendarios','DELETE')`).Scan(&del); err != nil || del {
		t.Error("painel_app NÃO pode apagar do calendário")
	}
	if err := exec(`DELETE FROM calendario_dias WHERE data='2026-03-04'`); err == nil || !strings.Contains(err.Error(), "nada se apaga") {
		t.Errorf("DELETE deveria ser barrado pelo gatilho: %v", err)
	}
}
