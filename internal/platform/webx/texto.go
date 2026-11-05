package webx

import "strings"

// PessoaRef é a referência curta a uma pessoa que os domínios trocam entre si (sem expor documento).
type PessoaRef struct{ ID, Codigo, Nome string }

var semAcento = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i", "ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u", "ç", "c", "ñ", "n")

// SemAcento devolve o texto em minúsculas e sem acento, para busca (a mesma ideia do normalize('NFD') do B).
func SemAcento(s string) string { return semAcento.Replace(strings.ToLower(s)) }

// Rotulo é o texto das opções de pessoa dos seletores do B: "NOME · PES. 0000001".
func (r PessoaRef) Rotulo() string { return strings.ToUpper(r.Nome) + " · " + r.Codigo }

// UUID diz se s tem o formato de um UUID (evita mandar lixo ao banco como identificador).
func UUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}
