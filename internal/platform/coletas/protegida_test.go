package coletas

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestProtegidaConvertePanicoEmErro(t *testing.T) {
	_, err := Protegida(func() (int, error) {
		var v []int
		i := len(v) + 1
		return v[i], nil // pânico de execução (índice fora do limite)
	})
	if !errors.Is(err, ErrPanico) {
		t.Fatalf("erro = %v, esperado ErrPanico", err)
	}
	_, err = Protegida(func() (int, error) { panic("dado sensível da fonte") })
	if !errors.Is(err, ErrPanico) || err.Error() != ErrPanico.Error() {
		t.Fatalf("pânico com valor arbitrário não pode repassar o valor: %v", err)
	}
	if n, err := Protegida(func() (int, error) { return 7, nil }); n != 7 || err != nil {
		t.Fatalf("sem pânico = %d %v", n, err)
	}
}

// Auditoria 05/11, item 1.5: a rodada pedida na tela não morre quando a requisição é cancelada.
func TestEmSegundoPlanoSobreviveAoCancelamentoDoPedido(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fim := make(chan error, 1)
	_, pronto, err := EmSegundoPlano(ctx, 10*time.Millisecond, func(c context.Context) (int, error) {
		time.Sleep(50 * time.Millisecond)
		fim <- c.Err()
		return 1, nil
	})
	if pronto || err != nil {
		t.Fatalf("rodada lenta: pronto=%v err=%v, esperado em andamento", pronto, err)
	}
	cancel() // a conexão do navegador caiu
	if e := <-fim; e != nil {
		t.Fatalf("a rodada recebeu contexto cancelado: %v", e)
	}
	n, pronto, err := EmSegundoPlano(context.Background(), time.Second, func(context.Context) (int, error) { return 7, nil })
	if !pronto || err != nil || n != 7 {
		t.Fatalf("rodada rápida = %d %v %v", n, pronto, err)
	}
}
