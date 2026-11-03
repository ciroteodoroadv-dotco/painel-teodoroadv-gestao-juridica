package tarefas

// Prioridades na ordem do B (OPCOES.prioridade); "Normal" é o padrão.
var Prioridades = []string{"Urgente", "Alta", "Normal", "Baixa"}

// PapeisParte é a lista de papéis das Partes (PAPEIS_PROCESSUAIS do B); "Outro" revela um campo de texto.
var PapeisParte = []string{"Autor", "Réu", "Exequente", "Executado", "Embargante", "Terceiro interessado", "Assistente", "Outro"}

// Subtitulos são as sugestões do campo Subtítulo (datalist do B).
var Subtitulos = []string{"Ligação", "Reunião", "Aula", "Análise de decisão", "Monitoramento"}

// PrioridadeClasse devolve a classe do selo de prioridade (prioridadeBadgeClass do B).
func PrioridadeClasse(p string) string {
	switch p {
	case "Urgente":
		return "prioridade-alta"
	case "Alta":
		return "prioridade-media"
	case "Baixa":
		return "prioridade-baixa"
	}
	return "prioridade-normal"
}

// PrioridadeCls é a classe do selo de prioridade desta tarefa.
func (t Tarefa) PrioridadeCls() string { return PrioridadeClasse(t.Prioridade) }

func prioridadeValida(p string) bool {
	for _, x := range Prioridades {
		if x == p {
			return true
		}
	}
	return false
}
