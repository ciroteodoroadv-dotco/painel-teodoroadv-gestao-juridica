package app

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/documentos"
	"painelteodoro/internal/pessoas"
	"painelteodoro/internal/platform/webx"
	"painelteodoro/internal/processos"
)

// pdfFicticio é um PDF mínimo e FICTÍCIO (só para provar que o documento é cifrado em disco e volta igual do backup).
var pdfFicticio = []byte("%PDF-1.4\n1 0 obj<</Type/Catalog>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF\nDocumento fictício de demonstração.\n")

// SeedDemoDocumentos acrescenta à demonstração UM documento fictício, cifrado na pasta de documentos, ligado ao primeiro processo.
// Recusa rodar se já houver documento. Só para desenvolvimento (o comando de linha exige PAINEL_ENV=dev).
func SeedDemoDocumentos(ctx context.Context, pool *pgxpool.Pool, fieldKey []byte, docsDir string, actorID string) error {
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM documentos`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return errors.New("já existem documentos cadastrados; a demonstração só roda sem documentos")
	}
	var processoID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM processos ORDER BY created_at, id LIMIT 1`).Scan(&processoID); err != nil {
		return errors.New("rode antes a demonstração de dados (seed-demo): não há processo para receber o documento")
	}
	fs, err := documentos.NewStore(docsDir, fieldKey)
	if err != nil {
		return err
	}
	pes := &pessoas.Service{Pool: pool, FieldKey: fieldKey}
	proc := &processos.Service{Pool: pool, Pessoas: pes}
	pes.Papeis = proc
	svc := &documentos.Service{Pool: pool, Store: fs, Processos: proc, Pessoas: pes}
	_, err = svc.Enviar(ctx, webx.Actor{UserID: actorID, RequestID: "seed-demo"}, documentos.Upload{
		Nome: "documento-exemplo.pdf", Descricao: "Documento fictício de demonstração (exemplo)", ProcessoID: processoID, Data: pdfFicticio})
	return err
}
