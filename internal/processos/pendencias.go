package processos

// Pendencias lista, na ordem fixa, o que ainda não foi preenchido no processo. No B isto vinha de marcadores
// [VERIFICAR]/[FALTA] na ficha; no v2 é calculado dos campos (nada é inventado nem presumido).
func Pendencias(p Processo) []string {
	var out []string
	falta := func(vazio bool, rotulo string) {
		if vazio {
			out = append(out, rotulo)
		}
	}
	falta(p.Numero == "", "Número do processo (CNJ)")
	falta(p.Sistema == "", "Sistema")
	falta(p.Instancia == "", "Instância")
	falta(p.Classe == "", "Classe processual")
	falta(p.Materia == "", "Matéria / ramo do direito")
	falta(p.Comarca == "", "Comarca")
	falta(p.Juizo == "", "Juízo")
	falta(p.Distribuicao == "", "Distribuição")
	falta(p.Valor == "", "Valor da causa")
	temCliente := false
	for _, pt := range p.Partes {
		temCliente = temCliente || pt.Cliente
	}
	falta(!temCliente, "Pelo menos uma parte marcada como cliente")
	return out
}
