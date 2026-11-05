package processos

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/webx"
)

// msgJustificativa é a recusa de uma duplicata sem justificativa (mínimo de 10 caracteres úteis, como a restrição do banco).
const msgJustificativa = "Informe a justificativa da duplicidade."

// DuplicadoView é o bloco "PROCESSO DUPLICADO" do formulário: o processo que já tem o número e a justificativa.
type DuplicadoView struct{ Codigo, Cliente, Justificativa string }

type duplicata struct{ id, codigo string }

// duplicataDe procura o processo MAIS ANTIGO, não excluído, com o mesmo número CNJ (já normalizado), ignorando o próprio.
// Número vazio nunca é duplicata.
func (s *Service) duplicataDe(ctx context.Context, numero, ignorar string) (*duplicata, error) {
	if numero == "" {
		return nil, nil
	}
	var d duplicata
	err := s.Pool.QueryRow(ctx, `SELECT id::text, codigo FROM processos WHERE numero_cnj=$1 AND excluido_em IS NULL
		AND ($2 = '' OR id <> $2::uuid) ORDER BY created_at, id LIMIT 1`, numero, ignorar).Scan(&d.id, &d.codigo)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &d, err
}

// decideDuplicidade aplica a regra do número repetido a n, ANTES de gravar. atual é o estado gravado (nil na criação).
// Duplicata nova exige justificativa; duplicata já marcada com o mesmo número mantém marcador e justificativa;
// número livre (ou processo que é o original) limpa o marcador. Devolve se a justificativa foi dada ou alterada.
func (s *Service) decideDuplicidade(ctx context.Context, n *normalized, in Input, id string, atualNumero string, atualDup bool, atualJust string) (justificou bool, err error) {
	dup, err := s.duplicataDe(ctx, n.numero, id)
	if err != nil || dup == nil {
		return false, err
	}
	just := strings.TrimSpace(in.DuplicadoJustificativa)
	mesmoNumero := id != "" && atualNumero == n.numero
	switch {
	case mesmoNumero && !atualDup: // é o original (ou o mais antigo): não vira duplicata ao ser editado
		return false, nil
	case mesmoNumero && atualDup && just == "":
		just = atualJust
	}
	if utf8.RuneCountInString(just) < 10 {
		return false, webx.Invalidf(msgJustificativa)
	}
	n.duplicado, n.dupJust, n.dupDe = true, just, dup.id
	return !mesmoNumero || !atualDup || just != atualJust, nil
}

// duplicadoView monta o bloco para um processo já marcado (mostra o original e a justificativa gravada).
func (s *Service) duplicadoView(ctx context.Context, numero, ignorar, just string) (*DuplicadoView, error) {
	dup, err := s.duplicataDe(ctx, numero, ignorar)
	if err != nil || dup == nil {
		return nil, err
	}
	v := &DuplicadoView{Codigo: dup.codigo, Justificativa: just}
	if m, err := s.ResumosVinculo(ctx, []string{dup.id}); err == nil {
		v.Cliente = strings.ToUpper(m[dup.id][1])
	}
	return v, nil
}

// verificaDuplicado responde, ao sair do campo do número, com o bloco do formulário (vazio se não houver duplicidade).
// É POST para o número do processo não ir na URL.
func (h *Handler) verificaDuplicado(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	numero, err := br.NormalizeCNJ(strings.TrimSpace(r.PostFormValue("numero")))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err != nil {
		return
	}
	ignorar := strings.TrimSpace(r.PostFormValue("ignorar"))
	if ignorar != "" && !webx.UUID(ignorar) {
		ignorar = ""
	}
	v, err := h.S.duplicadoView(r.Context(), numero, ignorar, "")
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	if v == nil {
		return
	}
	if err := fragmento("duplicado.html").ExecuteTemplate(w, "duplicado", v); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}
