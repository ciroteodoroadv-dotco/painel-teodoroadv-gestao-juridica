package prazos

import (
"time"
)

type TipoContagem string

const (
DiasUteis    TipoContagem = "DIAS_UTEIS"
DiasCorridos TipoContagem = "DIAS_CORRIDOS"
)

type RamoDireito string

const (
RamoCPC RamoDireito = "CPC"
RamoCPP RamoDireito = "CPP"
RamoCP  RamoDireito = "CP"
RamoCC  RamoDireito = "CC"
RamoLEF RamoDireito = "LEF"
RamoLEP RamoDireito = "LEP"
RamoCTN RamoDireito = "CTN"
RamoCTB RamoDireito = "CTB"
)

type RequisicaoPrazo struct {
DataPublicacao       time.Time
QuantidadeDias       int
Tipo                 TipoContagem
Ramo                 RamoDireito
FeriadosCustomizados []time.Time
}

type ResultadoPrazo struct {
Ramo               RamoDireito `json:"ramo"`
DataPublicacao     time.Time   `json:"data_publicacao"`
DataInicioContagem time.Time   `json:"data_inicio_contagem"`
DataVencimento     time.Time   `json:"data_vencimento"`
DiasContados       int         `json:"dias_contados"`
Observacoes        []string    `json:"observacoes"`
}

type CalculadorStrategy interface {
Calcular(req RequisicaoPrazo) (ResultadoPrazo, error)
Ramo() RamoDireito
}
