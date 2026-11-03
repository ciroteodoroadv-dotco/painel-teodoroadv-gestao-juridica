package tarefas

// Estado é um dos 7 estados do fluxo de tarefas (CATEGORIAS_TAREFA e ROTULOS_ESTADO_TAREFA do B).
// Rotulo é o título do painel do acordeão e do select; Selo/Classe são o rótulo e a classe do selo (badge).
type Estado struct{ Chave, Rotulo, Selo, Classe string }

// Estados na ordem do B.
var Estados = []Estado{
	{"a-confirmar", "A confirmar", "A confirmar", "badge-warn"},
	{"pendente", "Pendente", "Pendente", "badge-warn"},
	{"em-execucao", "Em execução", "Em execução", "badge-neutral"},
	{"revisao", "Revisão", "Revisão", "badge-neutral"},
	{"concluidas-com-sucesso", "Concluída com sucesso", "Concluída", "badge-ok"},
	{"concluidas-sem-sucesso", "Concluída sem sucesso", "Concluída (sem êxito)", "badge-neutral"},
	{"canceladas", "Cancelada", "Cancelada", "badge-danger"},
}

// EstadoValido diz se a chave é um dos 7 estados.
func EstadoValido(chave string) bool {
	for _, e := range Estados {
		if e.Chave == chave {
			return true
		}
	}
	return false
}

// Aberto diz se o estado ainda corre (nem concluída nem cancelada).
func Aberto(chave string) bool {
	return chave != "concluidas-com-sucesso" && chave != "concluidas-sem-sucesso" && chave != "canceladas"
}

// motivoDeEncerramento diz o que acontece com o prazo fatal quando a tarefa entra no estado final.
func motivoDeEncerramento(chave string) string {
	if chave == "concluidas-com-sucesso" {
		return "cumprida"
	}
	return "cancelada"
}
