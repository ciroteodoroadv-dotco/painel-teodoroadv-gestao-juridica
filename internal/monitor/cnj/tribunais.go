package cnj

import (
	"errors"
	"fmt"
	"strconv"
)

// Siglas das UFs na ordem dos códigos TR 01..27 da Justiça Estadual e da Eleitoral (Res. CNJ 65, Anexo VI).
// DF aparece como "dft" nos aliases do DataJud (conferir no teste manual: o wiki não pôde ser lido).
var ufsPorTR = [...]string{"", "ac", "al", "ap", "am", "ba", "ce", "dft", "es", "go", "ma", "mt", "ms", "mg", "pa", "pb", "pr", "pe", "pi",
	"rj", "rn", "rs", "ro", "rr", "sc", "se", "sp", "to"}

var tjmPorTR = map[int]string{13: "tjmmg", 21: "tjmrs", 26: "tjmsp"}

// aliasPorJTR recebe os 20 dígitos e olha J (posição 13) e TR (14-15).
func aliasPorJTR(d string) (string, error) {
	if len(d) != 20 {
		return "", errors.New("número com tamanho inválido")
	}
	j := int(d[13] - '0')
	tr, _ := strconv.Atoi(d[14:16])
	switch j {
	case 1:
		return "", errors.New("o STF não tem endpoint na API pública do DataJud")
	case 2:
		return "", errors.New("o CNJ não tem endpoint na API pública do DataJud")
	case 3:
		if tr == 0 {
			return "stj", nil
		}
	case 4:
		if tr >= 1 && tr <= 6 {
			return "trf" + strconv.Itoa(tr), nil
		}
	case 5:
		if tr == 0 {
			return "tst", nil
		}
		if tr <= 24 {
			return "trt" + strconv.Itoa(tr), nil
		}
	case 6:
		if tr == 0 {
			return "tse", nil
		}
		if tr <= 27 {
			return "tre-" + ufsPorTR[tr], nil
		}
	case 7:
		if tr == 0 {
			return "stm", nil
		}
	case 8:
		if tr >= 1 && tr <= 27 {
			return "tj" + ufsPorTR[tr], nil
		}
	case 9:
		if a, ok := tjmPorTR[tr]; ok {
			return a, nil
		}
	default:
		return "", fmt.Errorf("segmento da justiça (J=%d) inexistente", j)
	}
	return "", fmt.Errorf("tribunal (J=%d, TR=%02d) fora da faixa conhecida do DataJud", j, tr)
}
