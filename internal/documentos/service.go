package documentos

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/webx"
)

// Opcao é valor/rótulo de lista suspensa.
type Opcao = webx.Opcao

// ProcessoLookup e PessoaLookup são os contratos com os outros domínios.
type ProcessoLookup interface {
	Existe(ctx context.Context, id string) (bool, error)
	Resumos(ctx context.Context, ids []string) (map[string]string, error)
	Buscar(ctx context.Context, q string, limit int) ([]Opcao, error)
}

// PessoaLookup é o que este domínio precisa do cadastro de pessoas.
type PessoaLookup interface {
	Names(ctx context.Context, ids []string) (map[string]string, error)
	Exist(ctx context.Context, ids []string) (map[string]bool, error)
	Buscar(ctx context.Context, q string, limit int) ([]Opcao, error)
}

// Doc é a visão de leitura.
type Doc struct {
	ID, Nome, Tipo, Descricao  string
	Tamanho                    int64
	ProcessoID, ProcessoNumero string
	PessoaID, PessoaNome       string
	CriadoEm                   time.Time
	Version                    int
	arquivo                    string
}

// Upload é o pedido de envio.
type Upload struct {
	Nome, Descricao, ProcessoID, PessoaID string
	Data                                  []byte
}

// Service concentra as regras. Só toca nas próprias tabelas e no volume de arquivos.
type Service struct {
	Pool      *pgxpool.Pool
	Store     *Store
	Processos ProcessoLookup
	Pessoas   PessoaLookup
}

func (s *Service) record(ctx context.Context, tx pgx.Tx, ac webx.Actor, action, id string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: action, TargetType: "documento",
		TargetID: id, Result: audit.OK, RequestID: ac.RequestID})
}

// Enviar valida (tamanho, tipo pelo conteúdo, extensão, vínculo), grava o arquivo cifrado e o registro.
// Se o banco falhar, o arquivo é removido: não sobra arquivo sem registro.
func (s *Service) Enviar(ctx context.Context, ac webx.Actor, u Upload) (string, error) {
	nome := SafeName(u.Nome)
	switch {
	case nome == "":
		return "", webx.Invalidf("Escolha um arquivo.")
	case len(u.Data) == 0:
		return "", webx.Invalidf("O arquivo está vazio.")
	case len(u.Data) > MaxArquivo:
		return "", webx.Invalidf("Arquivo grande demais (o limite é 25 MiB).")
	case u.ProcessoID == "" && u.PessoaID == "":
		return "", webx.Invalidf("Vincule o documento a um processo ou a uma pessoa.")
	case len([]rune(u.Descricao)) > 500:
		return "", webx.Invalidf("Descrição com mais de 500 caracteres.")
	}
	tipo, err := Detect(u.Data)
	if err != nil {
		return "", webx.Invalidf(capitalize(err.Error()) + ".")
	}
	if !ExtensionMatches(nome, tipo) {
		return "", webx.Invalidf("A extensão do nome não confere com o conteúdo do arquivo.")
	}
	if u.ProcessoID != "" {
		if ok, err := s.Processos.Existe(ctx, u.ProcessoID); err != nil {
			return "", err
		} else if !ok {
			return "", webx.Invalidf("Processo não encontrado.")
		}
	}
	if u.PessoaID != "" {
		if ok, err := s.Pessoas.Exist(ctx, []string{u.PessoaID}); err != nil {
			return "", err
		} else if !ok[u.PessoaID] {
			return "", webx.Invalidf("Pessoa não encontrada.")
		}
	}
	arq, err := s.Store.Put(u.Data)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(u.Data)
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.Store.Remove(arq)
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO documentos (nome_original, tipo, tamanho, sha256, arquivo, descricao, processo_id, pessoa_id, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9) RETURNING id`, nome, tipo, len(u.Data), sum[:], arq, u.Descricao,
		webx.Nullable(u.ProcessoID), webx.Nullable(u.PessoaID), ac.UserID).Scan(&id)
	if err == nil {
		err = s.record(ctx, tx, ac, "documento.enviado", id)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		s.Store.Remove(arq)
		return "", err
	}
	return id, nil
}

// Baixar devolve o conteúdo decifrado. A leitura é auditada ANTES da entrega: sem auditoria, sem arquivo.
func (s *Service) Baixar(ctx context.Context, ac webx.Actor, id string) (*Doc, []byte, error) {
	d, err := s.get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if err := audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ac.UserID, Action: "documento.lido", TargetType: "documento",
		TargetID: id, Result: audit.OK, RequestID: ac.RequestID}); err != nil {
		return nil, nil, err
	}
	data, err := s.Store.Get(d.arquivo)
	if err != nil {
		return nil, nil, err
	}
	sum := sha256.Sum256(data)
	var want []byte
	if err := s.Pool.QueryRow(ctx, `SELECT sha256 FROM documentos WHERE id=$1`, id).Scan(&want); err != nil || string(want) != string(sum[:]) {
		return nil, nil, errors.New("integridade do arquivo não confere")
	}
	return d, data, nil
}

// Excluir oculta o documento (o arquivo permanece no volume até um expurgo decidido pelo escritório).
func (s *Service) Excluir(ctx context.Context, ac webx.Actor, id string, version int) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE documentos SET excluido_em=now(), updated_at=now(), updated_by=$3, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL`, id, version, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var one int
		e := s.Pool.QueryRow(ctx, `SELECT 1 FROM documentos WHERE id=$1 AND excluido_em IS NULL`, id).Scan(&one)
		if errors.Is(e, pgx.ErrNoRows) {
			return webx.ErrNotFound
		}
		if e != nil {
			return e
		}
		return webx.ErrConflict
	}
	if err = s.record(ctx, tx, ac, "documento.excluido", id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func capitalize(s string) string {
	if r := []rune(s); len(r) > 0 && r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 32
		return string(r)
	}
	return s
}
