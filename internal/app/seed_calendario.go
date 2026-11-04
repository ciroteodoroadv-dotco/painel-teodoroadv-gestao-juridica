package app

import (
	"context"

	"painelteodoro/internal/platform/webx"
	"painelteodoro/internal/prazos/calendario"
)

// semearCalendarios cria, na demonstração, dois calendários FICTÍCIOS:
//   - TRF-DEMO (Justiça Federal): datas da Lei 5.010/1966 art. 62 como o plano descreve, marcado [VERIFICAR] e
//     com conta_como_nao_util=false — o texto da lei não foi conferido, então NÃO afeta cálculo algum;
//   - TRIBUNAL DE TESTE: dois dias com efeito (Quarta-feira de Cinzas e Corpus Christi de 2027), portaria fictícia.
func semearCalendarios(ctx context.Context, ac webx.Actor, cal *calendario.Service) error {
	jf, err := cal.CriarCalendario(ctx, ac, calendario.CalInput{Sigla: "TRF-DEMO", Nome: "Justiça Federal (TRF1–6) — demonstração [VERIFICAR]", Ambito: "federal"})
	if err != nil {
		return err
	}
	const lei = "Lei 5.010/1966, art. 62 [VERIFICAR: texto não conferido]"
	for _, d := range []struct{ data, desc string }{
		{"2027-02-08", "Segunda-feira de Carnaval"}, {"2027-02-09", "Terça-feira de Carnaval"},
		{"2027-03-24", "Quarta-feira da Semana Santa"}, {"2027-03-25", "Quinta-feira da Semana Santa"},
		{"2027-08-11", "Dia 11 de agosto"}, {"2027-12-08", "Dia 8 de dezembro"},
	} {
		if _, err := cal.CriarDia(ctx, ac, calendario.DiaInput{CalendarioID: jf, Data: d.data, Tipo: "feriado", Descricao: d.desc,
			FonteCitacao: lei, Observacao: "[VERIFICAR] dado de demonstração: não afeta o cálculo até o advogado validar."}, true); err != nil {
			return err
		}
	}
	tt, err := cal.CriarCalendario(ctx, ac, calendario.CalInput{Sigla: "TRIBUNAL DE TESTE", Nome: "Tribunal de teste (fictício)", Ambito: "outro"})
	if err != nil {
		return err
	}
	for _, d := range []struct{ data, desc string }{{"2027-02-10", "Quarta-feira de Cinzas (fictício)"}, {"2027-05-27", "Corpus Christi (fictício)"}} {
		if _, err := cal.CriarDia(ctx, ac, calendario.DiaInput{CalendarioID: tt, Data: d.data, Tipo: "feriado", Descricao: d.desc,
			FonteCitacao: "Portaria fictícia nº 1/2027 do Tribunal de Teste (dado de demonstração)", ContaNaoUtil: true}, true); err != nil {
			return err
		}
	}
	return nil
}
