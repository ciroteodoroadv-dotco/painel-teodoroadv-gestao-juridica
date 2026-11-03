package processos

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/platform/webx"
)

// bump registra, na mesma transação, a alteração do processo (sem checar versão: ações rápidas da lista).
func (s *Service) bump(ctx context.Context, tx pgx.Tx, ac webx.Actor, id string) error {
	tag, err := tx.Exec(ctx, `UPDATE processos SET updated_at=now(), updated_by=$2, version=version+1
		WHERE id=$1 AND excluido_em IS NULL`, id, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return webx.ErrNotFound
	}
	return nil
}

func (s *Service) emTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func obrigatorio(campo, v string, max int) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", webx.Invalidf(campo + " é obrigatório.")
	}
	if utf8.RuneCountInString(v) > max {
		return "", webx.Invalidf(campo + " com texto longo demais.")
	}
	return v, nil
}

// MoverEstado muda o estado do processo (concorrência otimista). O motivo vai para a linha do tempo.
// O efeito sobre a tarefa de monitoramento (baixar/criar em cadeia) só passa a valer no Plano 6;
// até lá o servidor apenas registra a mudança.
func (s *Service) MoverEstado(ctx context.Context, ac webx.Actor, id string, version int, novo, motivo string) error {
	if !estadoValido(novo) {
		return webx.Invalidf("Escolha o novo estado do processo.")
	}
	motivo = strings.TrimSpace(motivo)
	if utf8.RuneCountInString(motivo) > 1000 {
		return webx.Invalidf("Motivo com texto longo demais.")
	}
	return s.emTx(ctx, func(tx pgx.Tx) error {
		var antigo string
		err := tx.QueryRow(ctx, `SELECT situacao FROM processos WHERE id=$1 AND excluido_em IS NULL FOR UPDATE`, id).Scan(&antigo)
		if err != nil {
			if err == pgx.ErrNoRows {
				return webx.ErrNotFound
			}
			return err
		}
		if antigo == novo {
			return webx.Invalidf("O processo já está nesse estado.")
		}
		tag, err := tx.Exec(ctx, `UPDATE processos SET situacao=$3, updated_at=now(), updated_by=$4, version=version+1
			WHERE id=$1 AND version=$2`, id, version, novo, ac.UserID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return webx.ErrConflict
		}
		texto := "Estado alterado de " + antigo + " para " + novo + "."
		if motivo != "" {
			texto += " Motivo: " + motivo
		}
		if _, err = tx.Exec(ctx, `INSERT INTO processo_andamentos (processo_id, data, o_que, created_by)
			VALUES ($1, (now() AT TIME ZONE 'America/Sao_Paulo')::date, $2, $3)`, id, texto, ac.UserID); err != nil {
			return err
		}
		return s.record(ctx, tx, ac, "processo.estado_movido", id, map[string]string{"de": antigo, "para": novo})
	})
}

// RegistrarAndamento acrescenta um item à linha do tempo do processo.
func (s *Service) RegistrarAndamento(ctx context.Context, ac webx.Actor, id string, data time.Time, oQue, documento string) error {
	oQue, err := obrigatorio("O que aconteceu", oQue, 4000)
	if err != nil {
		return err
	}
	documento = strings.TrimSpace(documento)
	if utf8.RuneCountInString(documento) > 300 {
		return webx.Invalidf("Documento relacionado com texto longo demais.")
	}
	return s.emTx(ctx, func(tx pgx.Tx) error {
		if err := s.bump(ctx, tx, ac, id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO processo_andamentos (processo_id, data, o_que, documento, created_by)
			VALUES ($1,$2,$3,$4,$5)`, id, data, oQue, documento, ac.UserID); err != nil {
			return err
		}
		return s.record(ctx, tx, ac, "processo.andamento_registrado", id, nil)
	})
}

// Andamento é uma linha da linha do tempo.
type Andamento struct {
	Data, OQue, Documento string
}

// Andamentos devolve a linha do tempo, da data mais recente para a mais antiga.
func (s *Service) Andamentos(ctx context.Context, id string) ([]Andamento, error) {
	rows, err := s.Pool.Query(ctx, `SELECT to_char(data,'YYYY-MM-DD'), o_que, documento FROM processo_andamentos
		WHERE processo_id=$1 ORDER BY data DESC, created_at DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Andamento
	for rows.Next() {
		var a Andamento
		if err := rows.Scan(&a.Data, &a.OQue, &a.Documento); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ItemFinanceiro é uma linha do financeiro do processo.
type ItemFinanceiro struct{ Item, Valor, Situacao string }

// Financeiro devolve os itens financeiros do processo, por ordem de cadastro.
func (s *Service) Financeiro(ctx context.Context, id string) ([]ItemFinanceiro, error) {
	rows, err := s.Pool.Query(ctx, `SELECT item, valor, situacao FROM processo_financeiro WHERE processo_id=$1 ORDER BY created_at, id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ItemFinanceiro
	for rows.Next() {
		var f ItemFinanceiro
		if err := rows.Scan(&f.Item, &f.Valor, &f.Situacao); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// AtualizarFinanceiro grava o item (honorários, custas…): atualiza se já existe um com o mesmo nome, senão cria.
func (s *Service) AtualizarFinanceiro(ctx context.Context, ac webx.Actor, id, item, valor, situacao string) error {
	item, err := obrigatorio("Item", item, 120)
	if err != nil {
		return err
	}
	valor, situacao = strings.TrimSpace(valor), strings.TrimSpace(situacao)
	if utf8.RuneCountInString(valor) > 60 || utf8.RuneCountInString(situacao) > 60 {
		return webx.Invalidf("Valor ou situação com texto longo demais.")
	}
	return s.emTx(ctx, func(tx pgx.Tx) error {
		if err := s.bump(ctx, tx, ac, id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO processo_financeiro (processo_id, item, valor, situacao, created_by)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (processo_id, lower(item)) DO UPDATE SET valor=EXCLUDED.valor, situacao=EXCLUDED.situacao`,
			id, item, valor, situacao, ac.UserID); err != nil {
			return err
		}
		return s.record(ctx, tx, ac, "processo.financeiro_atualizado", id, nil)
	})
}

// ProximaProvidencia grava o próximo passo do processo, com prazo (informado, nunca calculado) e responsável.
func (s *Service) ProximaProvidencia(ctx context.Context, ac webx.Actor, id, oQue string, ateQuando *time.Time, quem string) error {
	oQue, err := obrigatorio("O que fazer", oQue, 4000)
	if err != nil {
		return err
	}
	quem = strings.TrimSpace(quem)
	if utf8.RuneCountInString(quem) > 200 {
		return webx.Invalidf("Responsável com texto longo demais.")
	}
	return s.emTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE processos SET prox_o_que=$3, prox_ate_quando=$4, prox_quem=$5,
			updated_at=now(), updated_by=$2, version=version+1 WHERE id=$1 AND excluido_em IS NULL`,
			id, ac.UserID, oQue, ateQuando, quem)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return webx.ErrNotFound
		}
		return s.record(ctx, tx, ac, "processo.proxima_providencia", id, nil)
	})
}
