package auth

import (
	_ "embed"
	"strings"
	"sync"
)

// senhasComuns é a lista de senhas notoriamente vazadas com 12+ caracteres (procedência em
// senhas_comuns.LEIAME.txt). A política já exige 12+, então só essas importam.
//
//go:embed senhas_comuns.txt
var senhasComuns string

var (
	commonOnce sync.Once
	commonSet  map[string]struct{}
)

func isCommonPassword(pw string) bool {
	commonOnce.Do(func() {
		lines := strings.Split(senhasComuns, "\n")
		commonSet = make(map[string]struct{}, len(lines))
		for _, l := range lines {
			commonSet[strings.TrimSpace(l)] = struct{}{}
		}
	})
	_, ok := commonSet[strings.TrimSpace(strings.ToLower(pw))]
	return ok
}
