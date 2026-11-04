package app

import (
	"context"
	"strings"
	"testing"

	"painelteodoro/internal/auth"
)

func TestMigracao0008(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	_, uid := e.user("m8@exemplo.test", auth.Advogado)
	t.Cleanup(func() { _, _ = e.pool.Exec(ctx, `TRUNCATE pessoas, processos CASCADE`) })

	pes := func(codigo, tipo, nome, razao string) error {
		_, err := e.pool.Exec(ctx, `INSERT INTO pessoas (codigo, tipo_pessoa, nome_completo, razao_social, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$5)`, codigo, tipo, nome, razao, uid)
		return err
	}
	if err := pes("PES. 0000001", "Física", "", ""); err == nil {
		t.Error("pessoa física sem nome completo deveria violar a regra")
	}
	if err := pes("PES. 0000002", "Jurídica", "Nome Qualquer", ""); err == nil {
		t.Error("pessoa jurídica sem razão social deveria violar a regra")
	}
	if err := pes("PES. 0000003", "Física", "MARIA EXEMPLO", ""); err != nil {
		t.Errorf("pessoa física válida: %v", err)
	}
	if err := pes("PES. 0000004", "Jurídica", "", "EMPRESA EXEMPLO LTDA"); err != nil {
		t.Errorf("pessoa jurídica válida: %v", err)
	}
	if err := pes("PES. 0000003", "Física", "OUTRA", ""); err == nil {
		t.Error("código repetido deveria ser recusado")
	}
	var exib string
	if err := e.pool.QueryRow(ctx, `SELECT nome_exibicao FROM pessoas WHERE codigo='PES. 0000004'`).Scan(&exib); err != nil || exib != "EMPRESA EXEMPLO LTDA" {
		t.Errorf("nome_exibicao da PJ = %q (%v)", exib, err)
	}

	proc := func(codigo, sit, cnj string) error {
		_, err := e.pool.Exec(ctx, `INSERT INTO processos (codigo, numero_cnj, situacao, created_by, updated_by) VALUES ($1,$2,$3,$4,$4)`,
			codigo, cnj, sit, uid)
		return err
	}
	if err := proc("PRO. 0000001", "suspenso", ""); err == nil {
		t.Error("situação 'suspenso' deveria ser recusada")
	}
	if err := proc("PRO. 0000002", "encerrado", ""); err == nil {
		t.Error("situação 'encerrado' deveria ser recusada")
	}
	for i, c := range []string{"PRO. 0000003", "PRO. 0000004"} {
		if err := proc(c, "ativo", ""); err != nil {
			t.Errorf("processo %d sem número deveria ser aceito: %v", i, err)
		}
	}
	if err := proc("PRO. 0000005", "extinto", "0000101-93.2025.8.26.0100"); err != nil {
		t.Errorf("processo com número: %v", err)
	}
	// Desde a 0012 o número não é mais único no banco: a regra de duplicidade (marcador + justificativa) vive no serviço.
	if err := proc("PRO. 0000006", "ativo", "0000101-93.2025.8.26.0100"); err != nil {
		t.Errorf("mesmo número é aceito pelo banco (a duplicidade é tratada pelo serviço): %v", err)
	}

	// Papel do banco: escreve nas tabelas novas, não apaga auditoria.
	for _, tb := range []string{"processo_andamentos", "processo_financeiro", "processo_responsaveis", "codigos_contador", "pessoas", "processos", "processo_partes"} {
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
	if !strings.HasPrefix("PES. 0000001", "PES.") {
		t.Fatal("sanidade")
	}
}
