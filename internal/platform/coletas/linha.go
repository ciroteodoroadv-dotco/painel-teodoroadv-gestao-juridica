package coletas

import "time"

// Rótulos e estados do rodapé, iguais aos de 16-sync-status.js do painel B.
var (
	rotulos = map[string]string{"intimacoes": "DJEN", "monitor": "DataJud", "monitor-segredos": "DJEN por processo sigiloso"}
	estados = map[string]string{"sucesso": "última coleta concluída", "parcial": "coleta parcial", "falha": "erro",
		"bloqueado": "bloqueado", "nao_sincronizado": "sem tentativa registrada"}
)

func horario(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "não registrado"
	}
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.FixedZone("BRT", -3*3600)
	}
	return t.In(loc).Format("02/01/2006, 15:04:05") + " (Brasília)"
}

// Linha é o texto de uma coleta no rodapé "Estado das coletas". erro=true quando o estado não é sucesso (classe sync-error).
func (e Estado) Linha() (texto string, erro bool) {
	rotulo, ok := rotulos[e.Modulo]
	if !ok {
		rotulo = e.Modulo
	}
	estado, ok := estados[e.Status]
	if !ok {
		estado = "estado desconhecido"
	}
	return rotulo + ": " + estado + ". Tentativa: " + horario(e.UltimaTentativa) + ". Sucesso: " + horario(e.UltimoSucesso) + ". " + e.Mensagem,
		e.Status != "sucesso"
}
