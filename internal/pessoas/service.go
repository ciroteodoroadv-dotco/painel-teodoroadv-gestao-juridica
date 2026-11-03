package pessoas

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/codigos"
	"painelteodoro/internal/platform/crypt"
	"painelteodoro/internal/platform/webx"
)

// Finalidades de cifra: cada campo sensível tem a sua subchave.
const (
	purposeDoc = "pessoas.documento"
	purposeRG  = "pessoas.rg"
	purposeRep = "pessoas.representante_cpf"
)

// Service concentra as regras do cadastro.
type Service struct {
	Pool     *pgxpool.Pool
	FieldKey []byte
	Papeis   PapeisLookup // opcional (nil em testes isolados)
}

func (s *Service) seal(purpose, valor string) ([]byte, error) {
	if valor == "" {
		return nil, nil
	}
	return crypt.Seal(crypt.Derive(s.FieldKey, purpose), []byte(valor))
}

func (s *Service) unseal(purpose string, enc []byte) string {
	if len(enc) == 0 {
		return ""
	}
	b, err := crypt.Open(crypt.Derive(s.FieldKey, purpose), enc)
	if err != nil {
		return ""
	}
	return string(b)
}

func (s *Service) record(ctx context.Context, tx pgx.Tx, ac webx.Actor, action, id string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: action, TargetType: "pessoa",
		TargetID: id, Result: audit.OK, RequestID: ac.RequestID})
}

// selado reúne os três campos cifrados e o índice cego do documento.
type selado struct{ doc, idx, rg, rep []byte }

func (s *Service) selar(n normalized) (x selado, err error) {
	if x.doc, err = s.seal(purposeDoc, n.doc); err != nil {
		return x, err
	}
	if n.doc != "" {
		x.idx = crypt.BlindIndex(s.FieldKey, purposeDoc, n.doc)
	}
	if x.rg, err = s.seal(purposeRG, n.rg); err != nil {
		return x, err
	}
	x.rep, err = s.seal(purposeRep, n.repCPF)
	return x, err
}

// Create cadastra uma pessoa e devolve o id e o código (PES. 0000001). Escrita e auditoria na mesma transação.
func (s *Service) Create(ctx context.Context, ac webx.Actor, in Input) (id, codigo string, err error) {
	n, err := in.normalize(in.TipoPessoa)
	if err != nil {
		return "", "", err
	}
	x, err := s.selar(n)
	if err != nil {
		return "", "", err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if codigo, err = codigos.Proximo(ctx, tx, "PES"); err != nil {
		return "", "", err
	}
	err = tx.QueryRow(ctx, `INSERT INTO pessoas (codigo, tipo_pessoa, nome_completo, documento_enc, documento_idx, rg_enc,
		estado_civil, profissao, nacionalidade, data_nascimento, razao_social, nome_fantasia, representante_nome,
		representante_cpf_enc, representante_cargo, logradouro, numero, complemento, bairro, cep, cidade, uf, email,
		telefone, observacoes, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$26)
		RETURNING id`,
		codigo, n.tipo, n.nomeCompleto, x.doc, x.idx, x.rg, n.estadoCivil, n.profissao, n.nacionalidade, n.nascimento,
		n.razao, n.fantasia, n.repNome, x.rep, n.repCargo, n.logradouro, n.numero, n.complemento, n.bairro, n.cep,
		n.cidade, n.uf, n.email, n.tel, n.obs, ac.UserID).Scan(&id)
	if webx.IsUniqueViolation(err) {
		return "", "", webx.Invalidf("Já existe uma pessoa com esse documento.")
	}
	if err == nil {
		err = s.record(ctx, tx, ac, "pessoa.criada", id)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return "", "", err
	}
	return id, codigo, nil
}

// Update altera com concorrência otimista (versão antiga → webx.ErrConflict). O tipo de pessoa não se troca.
func (s *Service) Update(ctx context.Context, ac webx.Actor, id string, version int, in Input) error {
	var tipo string
	err := s.Pool.QueryRow(ctx, `SELECT tipo_pessoa FROM pessoas WHERE id=$1 AND excluido_em IS NULL`, id).Scan(&tipo)
	if errors.Is(err, pgx.ErrNoRows) {
		return webx.ErrNotFound
	}
	if err != nil {
		return err
	}
	n, err := in.normalize(tipo)
	if err != nil {
		return err
	}
	x, err := s.selar(n)
	if err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE pessoas SET nome_completo=$3, documento_enc=$4, documento_idx=$5, rg_enc=$6,
		estado_civil=$7, profissao=$8, nacionalidade=$9, data_nascimento=$10, razao_social=$11, nome_fantasia=$12,
		representante_nome=$13, representante_cpf_enc=$14, representante_cargo=$15, logradouro=$16, numero=$17,
		complemento=$18, bairro=$19, cep=$20, cidade=$21, uf=$22, email=$23, telefone=$24, observacoes=$25,
		updated_at=now(), updated_by=$26, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL`,
		id, version, n.nomeCompleto, x.doc, x.idx, x.rg, n.estadoCivil, n.profissao, n.nacionalidade, n.nascimento,
		n.razao, n.fantasia, n.repNome, x.rep, n.repCargo, n.logradouro, n.numero, n.complemento, n.bairro, n.cep,
		n.cidade, n.uf, n.email, n.tel, n.obs, ac.UserID)
	if webx.IsUniqueViolation(err) {
		return webx.Invalidf("Já existe uma pessoa com esse documento.")
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return s.whyNoRows(ctx, id)
	}
	if err = s.record(ctx, tx, ac, "pessoa.alterada", id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) whyNoRows(ctx context.Context, id string) error {
	var one int
	err := s.Pool.QueryRow(ctx, `SELECT 1 FROM pessoas WHERE id=$1 AND excluido_em IS NULL`, id).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return webx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return webx.ErrConflict
}

// Delete marca a pessoa como excluída (sem apagar). Recusa se ainda aparecer em processo.
func (s *Service) Delete(ctx context.Context, ac webx.Actor, id string, version int) error {
	if s.Papeis != nil {
		m, err := s.Papeis.PapeisPorPessoa(ctx, []string{id})
		if err != nil {
			return err
		}
		if len(m[id]) > 0 {
			return webx.Invalidf("Esta pessoa é parte em processo(s). Retire-a dos processos antes de excluir.")
		}
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE pessoas SET excluido_em=now(), updated_at=now(), updated_by=$3, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL`, id, version, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return s.whyNoRows(ctx, id)
	}
	if err = s.record(ctx, tx, ac, "pessoa.excluida", id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
