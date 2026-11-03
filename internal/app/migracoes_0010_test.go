package app

import (
	"context"
	"testing"

	"painelteodoro/internal/auth"
)

func TestMigracao0010(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	_, uid := e.user("m10@exemplo.test", auth.Advogado)
	t.Cleanup(func() {
		_, _ = e.pool.Exec(ctx, `TRUNCATE agenda_compromissos, audiencias, diligencias`)
	})
	exec := func(sql string, args ...any) error { _, err := e.pool.Exec(ctx, sql, args...); return err }

	// Tabelas antigas (eventos e participantes) deixam de existir.
	for _, tb := range []string{"agenda_eventos", "agenda_participantes"} {
		if e.count(`SELECT count(*) FROM information_schema.tables WHERE table_name=$1`, tb) != 0 {
			t.Errorf("a tabela %s deveria ter sido substituída", tb)
		}
	}

	// agenda_compromissos: Reunião, Ligação e Aula (Diligência tem ficha própria).
	comp := func(tipo string) error {
		return exec(`INSERT INTO agenda_compromissos (tipo, descricao, data, hora, created_by, updated_by) VALUES ($1,'Compromisso de teste','2026-11-10','14:00',$2,$2)`, tipo, uid)
	}
	for _, tipo := range []string{"Reunião", "Ligação", "Aula"} {
		if err := comp(tipo); err != nil {
			t.Errorf("compromisso %q deveria ser aceito: %v", tipo, err)
		}
	}
	for _, tipo := range []string{"Diligência", "Audiência", "Prazo Fatal", "aula"} {
		if err := comp(tipo); err == nil {
			t.Errorf("tipo %q deveria ser recusado em compromissos", tipo)
		}
	}
	if err := exec(`INSERT INTO agenda_compromissos (tipo, descricao, data, created_by, updated_by) VALUES ('Aula','x','2026-11-10',$1,$1)`, uid); err == nil {
		t.Error("compromisso sem hora deveria ser recusado")
	}

	// audiencias: 6 tipos de ato; cancelada exige motivo.
	aud := func(codigo, tipo string, cancelada bool, motivo string) error {
		return exec(`INSERT INTO audiencias (codigo, tipo_ato, titulo, data, hora, cancelada, cancelamento_motivo, created_by, updated_by)
			VALUES ($1,$2,'Audiência de teste','2026-11-10','10:00',$3,$4,$5,$5)`, codigo, tipo, cancelada, motivo, uid)
	}
	for i, tipo := range []string{"audiencia-ad-monitoria", "conciliacao-e-mediacao", "inicial-justica-do-trabalho",
		"instrucao-e-julgamento", "sustentacao-oral", "una-justica-do-trabalho"} {
		if err := aud("AUD. 000000"+string(rune('1'+i)), tipo, false, ""); err != nil {
			t.Errorf("tipo de ato %q deveria ser aceito: %v", tipo, err)
		}
	}
	if err := aud("AUD. 0000009", "outro-ato", false, ""); err == nil {
		t.Error("tipo de ato fora dos 6 deveria ser recusado")
	}
	if err := aud("AUD. 0000001", "sustentacao-oral", false, ""); err == nil {
		t.Error("código repetido deveria ser recusado")
	}
	if err := aud("AUD. 0000008", "sustentacao-oral", true, ""); err == nil {
		t.Error("audiência cancelada sem motivo deveria ser recusada")
	}
	if err := aud("AUD. 0000008", "sustentacao-oral", true, "Remarcada pelo juízo"); err != nil {
		t.Errorf("audiência cancelada com motivo: %v", err)
	}

	// diligencias: data-limite obrigatória; status do B; hora opcional.
	dil := func(codigo, status string, data any) error {
		return exec(`INSERT INTO diligencias (codigo, titulo, data_limite, status, created_by, updated_by) VALUES ($1,'Diligência de teste',$2,$3,$4,$4)`, codigo, data, status, uid)
	}
	if err := dil("DIL. 0000001", "Pendente", "2026-11-12"); err != nil {
		t.Errorf("diligência válida (hora vazia): %v", err)
	}
	if err := dil("DIL. 0000002", "Concluída", "2026-11-12"); err != nil {
		t.Errorf("status Concluída: %v", err)
	}
	if err := dil("DIL. 0000003", "Pendente", nil); err == nil {
		t.Error("diligência sem data-limite deveria ser recusada")
	}
	if err := dil("DIL. 0000004", "Em curso", "2026-11-12"); err == nil {
		t.Error("status fora da lista deveria ser recusado")
	}
}
