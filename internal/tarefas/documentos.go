package tarefas

import (
	"context"
	"io"
	"strings"

	"painelteodoro/internal/platform/webx"
)

// AnexarDocumento guarda o PDF do caso (campo "Documentos do caso" da edição) no domínio Documentos, ligado à
// tarefa e ao vínculo dela. Só PDF: o domínio Documentos confere o tipo pelo CONTEÚDO, não pela extensão.
func (s *Service) AnexarDocumento(ctx context.Context, ac webx.Actor, id, nome string, conteudo io.Reader) error {
	t, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(nome)), ".pdf") {
		return webx.Invalidf("Envie um arquivo PDF.")
	}
	return s.Docs.Anexar(ctx, ac, t.ID, t.OrigemTipo, t.OrigemID, nome, conteudo)
}
