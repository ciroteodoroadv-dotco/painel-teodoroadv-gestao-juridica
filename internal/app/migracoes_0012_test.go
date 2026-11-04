package app

import (
	"context"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/migrate"
)

func TestMigracao0012(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	_, uid := e.user("m12@exemplo.test", auth.Advogado)
	exec := func(sql string, args ...any) error { _, err := e.pool.Exec(ctx, sql, args...); return err }

	// Compromissos: cancelado exige motivo.
	comp := func(cancelado bool, motivo string) error {
		return exec(`INSERT INTO agenda_compromissos (tipo, descricao, data, hora, cancelado, cancelamento_motivo, cancelado_em, cancelado_por, created_by, updated_by)
			VALUES ('Aula','x','2026-11-10','10:00',$1,$2, CASE WHEN $1 THEN now() END, CASE WHEN $1 THEN $3::uuid END,$3,$3)`, cancelado, motivo, uid)
	}
	if err := comp(false, ""); err != nil {
		t.Errorf("compromisso ativo: %v", err)
	}
	if err := comp(true, "Aula remarcada"); err != nil {
		t.Errorf("compromisso cancelado com motivo: %v", err)
	}
	if err := comp(true, "  "); err == nil {
		t.Error("compromisso cancelado sem motivo deveria ser recusado")
	}
	// Diligências: mesmas colunas.
	dil := func(cod string, cancelado bool, motivo string) error {
		return exec(`INSERT INTO diligencias (codigo, titulo, data_limite, status, cancelado, cancelamento_motivo, created_by, updated_by)
			VALUES ($1,'D','2026-11-12', CASE WHEN $2 THEN 'Cancelada' ELSE 'Pendente' END,$2,$3,$4,$4)`, cod, cancelado, motivo, uid)
	}
	if err := dil("DIL. 0000001", true, "Perdeu o objeto"); err != nil {
		t.Errorf("diligência cancelada com motivo: %v", err)
	}
	if err := dil("DIL. 0000002", true, ""); err == nil {
		t.Error("diligência cancelada sem motivo deveria ser recusada")
	}
	// Intimações: 'cancelada' substitui 'descartada'; cancelar exige motivo.
	ins := func(hash, status, motivo string) error {
		return exec(`INSERT INTO intimacoes (hash, status, cancelamento_motivo, decidida_por) VALUES ($1,$2,$3, CASE WHEN $2='triagem' THEN NULL ELSE $4::uuid END)`, hash, status, motivo, uid)
	}
	if err := ins("a", "cancelada", "Duplicada"); err != nil {
		t.Errorf("intimação cancelada com motivo: %v", err)
	}
	if err := ins("b", "cancelada", ""); err == nil {
		t.Error("intimação cancelada sem motivo deveria ser recusada")
	}
	if err := ins("c", "descartada", "x"); err == nil {
		t.Error("'descartada' deixou de existir")
	}
	// Processo duplicado: justificativa obrigatória; número deixa de ser único.
	proc := func(cod, num string, dup bool, just string) error {
		return exec(`INSERT INTO processos (codigo, numero_cnj, situacao, duplicado, duplicado_justificativa, created_by, updated_by) VALUES ($1,$2,'ativo',$3,$4,$5,$5)`, cod, num, dup, just, uid)
	}
	if err := proc("PRO. 0000001", "0000101-25.2025.8.26.0100", false, ""); err != nil {
		t.Fatal(err)
	}
	if err := proc("PRO. 0000002", "0000101-25.2025.8.26.0100", true, "Reautuado após extinção do primeiro"); err != nil {
		t.Errorf("número repetido com marcador e justificativa: %v", err)
	}
	if err := proc("PRO. 0000003", "0000101-25.2025.8.26.0100", true, "curto"); err == nil {
		t.Error("duplicado com justificativa curta deveria ser recusado")
	}
	if err := proc("PRO. 0000004", "0000101-25.2025.8.26.0100", true, ""); err == nil {
		t.Error("duplicado sem justificativa deveria ser recusado")
	}
}

// A migração converte as intimações 'descartada' que já existirem (nada é apagado) e é reversível.
func TestMigracao0012PreservaIntimacoesDescartadas(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	_, uid := e.user("m12b@exemplo.test", auth.Advogado)
	for range 3 { // 0014, 0013 e 0012: volta ao esquema da 0011
		if err := migrate.Down(ctx, e.pool); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO intimacoes (hash, status, decidida_por) VALUES ('antiga','descartada',$1), ('nova','triagem',NULL)`, uid); err != nil {
		t.Fatalf("esquema da 0011: %v", err)
	}
	if err := migrate.Up(ctx, e.pool); err != nil {
		t.Fatal(err)
	}
	var st, motivo string
	if err := e.pool.QueryRow(ctx, `SELECT status, cancelamento_motivo FROM intimacoes WHERE hash='antiga'`).Scan(&st, &motivo); err != nil {
		t.Fatal(err)
	}
	if st != "cancelada" || !strings.Contains(motivo, "migração") {
		t.Errorf("descartada deveria virar cancelada com motivo explicando: %q %q", st, motivo)
	}
	if e.count(`SELECT count(*) FROM intimacoes`) != 2 {
		t.Error("nenhuma intimação pode ser perdida")
	}
}
