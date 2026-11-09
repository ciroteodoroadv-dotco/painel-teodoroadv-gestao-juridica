package prazos

import (
"testing"
"time"
)

func TestCalcularPrazoMultidisciplinar(t *testing.T) {
pub := time.Date(2026, time.March, 6, 0, 0, 0, 0, time.UTC)

t.Run("CPC - Dias Uteis", func(t *testing.T) {
res, err := CalcularPrazo(RequisicaoPrazo{DataPublicacao: pub, QuantidadeDias: 5, Ramo: RamoCPC})
if err != nil {
t.Fatalf("Erro no CPC: %v", err)
}
esperado := time.Date(2026, time.March, 13, 0, 0, 0, 0, time.UTC)
if !res.DataVencimento.Equal(esperado) {
t.Errorf("CPC incorreto. Esperado: %s, Obtido: %s", esperado.Format("02/01/2006"), res.DataVencimento.Format("02/01/2006"))
}
})

t.Run("CPP - Dias Corridos com Prorrogacao", func(t *testing.T) {
res, err := CalcularPrazo(RequisicaoPrazo{DataPublicacao: pub, QuantidadeDias: 2, Ramo: RamoCPP})
if err != nil {
t.Fatalf("Erro no CPP: %v", err)
}
esperado := time.Date(2026, time.March, 9, 0, 0, 0, 0, time.UTC)
if !res.DataVencimento.Equal(esperado) {
t.Errorf("CPP incorreto. Esperado: %s, Obtido: %s", esperado.Format("02/01/2006"), res.DataVencimento.Format("02/01/2006"))
}
})

t.Run("CP - Penal Material", func(t *testing.T) {
res, err := CalcularPrazo(RequisicaoPrazo{DataPublicacao: pub, QuantidadeDias: 5, Ramo: RamoCP})
if err != nil {
t.Fatalf("Erro no CP: %v", err)
}
esperado := time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC)
if !res.DataVencimento.Equal(esperado) {
t.Errorf("CP incorreto. Esperado: %s, Obtido: %s", esperado.Format("02/01/2006"), res.DataVencimento.Format("02/01/2006"))
}
})
}
