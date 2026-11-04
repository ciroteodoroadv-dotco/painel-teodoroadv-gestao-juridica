package monitor

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode/utf8"

	"painelteodoro/internal/monitor/cnj"
	"painelteodoro/internal/monitor/datajud"
	"painelteodoro/internal/platform/webx"
)

// TextoAviso é dito na tela de detalhe de toda movimentação do DataJud: a fonte não traz o teor da intimação.
const TextoAviso = "Movimentação processual (DataJud). Esta fonte não fornece o teor da intimação."

// textoDoMovimento reúne TODOS os campos que o DataJud devolveu do movimento (E2): grau, órgão julgador, código e nome do
// movimento, complementos tabelados (nome, valor, descrição) e data e hora (horário de Brasília).
func textoDoMovimento(m datajud.Movimento, tribunal string) string {
	var b strings.Builder
	linha := func(rotulo, v string) {
		if strings.TrimSpace(v) != "" {
			b.WriteString(rotulo + ": " + v + "\n")
		}
	}
	linha("Fonte", "DataJud (CNJ)")
	linha("Tribunal", tribunal)
	linha("Grau", m.Grau)
	linha("Órgão julgador", m.OrgaoJulgador)
	linha("Código do movimento", m.Codigo)
	linha("Movimento", m.Nome)
	if !m.DataHora.IsZero() {
		linha("Data e hora", m.DataHora.In(webx.SP()).Format("02/01/2006 15:04")+" (Brasília)")
	} else {
		linha("Data e hora", m.DataBruta)
	}
	if len(m.Complementos) > 0 {
		b.WriteString("Complementos:\n")
		for _, c := range m.Complementos {
			item := strings.TrimSpace(c.Nome)
			if c.Valor != "" {
				item += ": " + c.Valor
			}
			if c.Descricao != "" {
				item += " (" + c.Descricao + ")"
			}
			if c.Codigo != "" {
				item += " [código " + c.Codigo + "]"
			}
			b.WriteString("- " + strings.TrimSpace(item) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func resumoDe(s string) string {
	if utf8.RuneCountInString(s) > 2000 {
		return string([]rune(s)[:2000])
	}
	return s
}

// hashDoMovimento identifica a intimação do movimento (processo + chave), estável entre rodadas.
func hashDoMovimento(processoID, chave string) string {
	h := sha256.Sum256([]byte("datajud\x1f" + processoID + "\x1f" + chave))
	return hex.EncodeToString(h[:])
}

func novaDoMovimento(p ProcessoRef, tribunal string, m datajud.Movimento, ator, reqID string) NovaIntimacao {
	num := cnj.Formatar(p.Numero)
	texto := textoDoMovimento(m, tribunal)
	n := NovaIntimacao{Hash: hashDoMovimento(p.ID, datajud.Chave(m)), Origem: origemDJ, Tribunal: tribunal, Orgao: m.OrgaoJulgador,
		ProcessoNumero: num, ProcessoID: p.ID, Tipo: m.Nome, Texto: texto, Resumo: resumoDe(texto),
		TituloAnalise:    "Analisar movimentação — " + corta(m.Nome, 120) + " — " + num,
		DescricaoAnalise: TextoAviso + "\nPrazo a apurar pelo advogado. O sistema não calculou prazo.",
		LinhaConsolidada: "DataJud: " + corta(m.Nome, 120) + " (" + m.Grau + ")", CriadoPor: p.CriadoPor, Ator: ator, RequestID: reqID}
	if !m.DataHora.IsZero() {
		d := webx.Hoje(m.DataHora)
		n.Data = &d
	}
	return n
}

func corta(s string, n int) string {
	if r := []rune(strings.TrimSpace(s)); len(r) > n {
		return string(r[:n])
	}
	return strings.TrimSpace(s)
}
