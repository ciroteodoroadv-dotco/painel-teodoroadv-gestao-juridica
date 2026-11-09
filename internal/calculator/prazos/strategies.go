package prazos

import (
	"fmt"
)

type CPCStrategy struct{}

func (s *CPCStrategy) Ramo() RamoDireito { return RamoCPC }
func (s *CPCStrategy) Calcular(req RequisicaoPrazo) (ResultadoPrazo, error) {
	obs := make([]string, 0)
	dataInicio := req.DataPublicacao.AddDate(0, 0, 1)

	for !IsDiaUtil(dataInicio, req.FeriadosCustomizados) {
		obs = append(obs, fmt.Sprintf("Início prorrogado: %s não é dia útil.", dataInicio.Format("02/01/2006")))
		dataInicio = dataInicio.AddDate(0, 0, 1)
	}

	cursor := dataInicio
	diasRestantes := req.QuantidadeDias

	for diasRestantes > 1 {
		cursor = cursor.AddDate(0, 0, 1)
		if IsDiaUtil(cursor, req.FeriadosCustomizados) {
			diasRestantes--
		}
	}

	for !IsDiaUtil(cursor, req.FeriadosCustomizados) {
		obs = append(obs, fmt.Sprintf("Vencimento prorrogado: %s não é dia útil.", cursor.Format("02/01/2006")))
		cursor = cursor.AddDate(0, 0, 1)
	}

	return ResultadoPrazo{
		Ramo:               RamoCPC,
		DataPublicacao:     req.DataPublicacao,
		DataInicioContagem: dataInicio,
		DataVencimento:     cursor,
		DiasContados:       req.QuantidadeDias,
		Observacoes:        obs,
	}, nil
}

type CPPStrategy struct{}

func (s *CPPStrategy) Ramo() RamoDireito { return RamoCPP }
func (s *CPPStrategy) Calcular(req RequisicaoPrazo) (ResultadoPrazo, error) {
	obs := []string{"Contagem em dias contínuos conforme Art. 798 do CPP."}
	dataInicio := req.DataPublicacao.AddDate(0, 0, 1)
	cursor := dataInicio.AddDate(0, 0, req.QuantidadeDias-1)

	for !IsDiaUtil(cursor, req.FeriadosCustomizados) {
		obs = append(obs, fmt.Sprintf("Vencimento prorrogado: %s não é dia útil.", cursor.Format("02/01/2006")))
		cursor = cursor.AddDate(0, 0, 1)
	}

	return ResultadoPrazo{
		Ramo:               RamoCPP,
		DataPublicacao:     req.DataPublicacao,
		DataInicioContagem: dataInicio,
		DataVencimento:     cursor,
		DiasContados:       req.QuantidadeDias,
		Observacoes:        obs,
	}, nil
}

type CPStrategy struct{}

func (s *CPStrategy) Ramo() RamoDireito { return RamoCP }
func (s *CPStrategy) Calcular(req RequisicaoPrazo) (ResultadoPrazo, error) {
	obs := []string{"Prazo penal material: Inclui-se o dia do começo (Art. 10 CP)."}
	dataInicio := req.DataPublicacao
	cursor := dataInicio.AddDate(0, 0, req.QuantidadeDias-1)

	return ResultadoPrazo{
		Ramo:               RamoCP,
		DataPublicacao:     req.DataPublicacao,
		DataInicioContagem: dataInicio,
		DataVencimento:     cursor,
		DiasContados:       req.QuantidadeDias,
		Observacoes:        obs,
	}, nil
}

type CCStrategy struct{}

func (s *CCStrategy) Ramo() RamoDireito { return RamoCC }
func (s *CCStrategy) Calcular(req RequisicaoPrazo) (ResultadoPrazo, error) {
	obs := []string{"Prazo civil material: Exclui o dia do início e inclui o do vencimento (Art. 132 CC)."}
	dataInicio := req.DataPublicacao.AddDate(0, 0, 1)
	cursor := dataInicio.AddDate(0, 0, req.QuantidadeDias-1)

	for !IsDiaUtil(cursor, req.FeriadosCustomizados) {
		obs = append(obs, fmt.Sprintf("Vencimento prorrogado: %s.", cursor.Format("02/01/2006")))
		cursor = cursor.AddDate(0, 0, 1)
	}

	return ResultadoPrazo{
		Ramo:               RamoCC,
		DataPublicacao:     req.DataPublicacao,
		DataInicioContagem: dataInicio,
		DataVencimento:     cursor,
		DiasContados:       req.QuantidadeDias,
		Observacoes:        obs,
	}, nil
}

type CTNStrategy struct{}

func (s *CTNStrategy) Ramo() RamoDireito { return RamoCTN }
func (s *CTNStrategy) Calcular(req RequisicaoPrazo) (ResultadoPrazo, error) {
	obs := []string{"Prazo tributário: Dias contínuos conforme Art. 210 do CTN."}
	dataInicio := req.DataPublicacao.AddDate(0, 0, 1)

	for !IsDiaUtil(dataInicio, req.FeriadosCustomizados) {
		obs = append(obs, fmt.Sprintf("Início prorrogado: %s sem expediente bancário/fiscal.", dataInicio.Format("02/01/2006")))
		dataInicio = dataInicio.AddDate(0, 0, 1)
	}

	cursor := dataInicio.AddDate(0, 0, req.QuantidadeDias-1)

	for !IsDiaUtil(cursor, req.FeriadosCustomizados) {
		obs = append(obs, fmt.Sprintf("Vencimento prorrogado: %s sem expediente bancário/fiscal.", cursor.Format("02/01/2006")))
		cursor = cursor.AddDate(0, 0, 1)
	}

	return ResultadoPrazo{
		Ramo:               RamoCTN,
		DataPublicacao:     req.DataPublicacao,
		DataInicioContagem: dataInicio,
		DataVencimento:     cursor,
		DiasContados:       req.QuantidadeDias,
		Observacoes:        obs,
	}, nil
}

type CTBStrategy struct{}

func (s *CTBStrategy) Ramo() RamoDireito { return RamoCTB }
func (s *CTBStrategy) Calcular(req RequisicaoPrazo) (ResultadoPrazo, error) {
	obs := []string{"Prazo administrativo de trânsito (CTB)."}
	dataInicio := req.DataPublicacao.AddDate(0, 0, 1)
	cursor := dataInicio.AddDate(0, 0, req.QuantidadeDias-1)

	for !IsDiaUtil(cursor, req.FeriadosCustomizados) {
		obs = append(obs, fmt.Sprintf("Vencimento prorrogado para o primeiro dia útil: %s.", cursor.Format("02/01/2006")))
		cursor = cursor.AddDate(0, 0, 1)
	}

	return ResultadoPrazo{
		Ramo:               RamoCTB,
		DataPublicacao:     req.DataPublicacao,
		DataInicioContagem: dataInicio,
		DataVencimento:     cursor,
		DiasContados:       req.QuantidadeDias,
		Observacoes:        obs,
	}, nil
}

type LEFStrategy struct{}

func (s *LEFStrategy) Ramo() RamoDireito { return RamoLEF }
func (s *LEFStrategy) Calcular(req RequisicaoPrazo) (ResultadoPrazo, error) {
	cpc := &CPCStrategy{}
	res, err := cpc.Calcular(req)
	if err != nil {
		return ResultadoPrazo{}, err
	}
	res.Ramo = RamoLEF
	res.Observacoes = append([]string{"Execução Fiscal (Lei 6.830/80 c/c CPC)."}, res.Observacoes...)
	return res, nil
}

type LEPStrategy struct{}

func (s *LEPStrategy) Ramo() RamoDireito { return RamoLEP }
func (s *LEPStrategy) Calcular(req RequisicaoPrazo) (ResultadoPrazo, error) {
	cpp := &CPPStrategy{}
	res, err := cpp.Calcular(req)
	if err != nil {
		return ResultadoPrazo{}, err
	}
	res.Ramo = RamoLEP
	res.Observacoes = append([]string{"Execução Penal (Lei 7.210/84 c/c CPP)."}, res.Observacoes...)
	return res, nil
}
