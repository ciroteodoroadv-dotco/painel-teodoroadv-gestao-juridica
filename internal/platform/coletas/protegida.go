package coletas

import (
	"context"
	"errors"
	"runtime"
	"time"
)

// ErrPanico indica que uma rodada de coleta entrou em pânico (erro de programação ou resposta inesperada da fonte).
var ErrPanico = errors.New("falha inesperada na rodada de coleta (pânico interno)")

// Protegida executa a rodada f convertendo pânico em erro. Sem isso, um pânico numa goroutine de coleta derruba o
// processo inteiro do painel e, com o reinício automático, vira um laço de quedas (auditoria 05/11, item 3.2).
// O valor do pânico não é repassado (pode conter dado da fonte); só o tipo de erro de execução, quando houver.
func Protegida[T any](f func() (T, error)) (r T, err error) {
	defer func() {
		if p := recover(); p != nil {
			var zero T
			r, err = zero, ErrPanico
			if re, ok := p.(runtime.Error); ok {
				err = errors.Join(ErrPanico, errors.New(re.Error()))
			}
		}
	}()
	return f()
}

// EmSegundoPlano roda a coleta pedida na tela DESACOPLADA da requisição (context.WithoutCancel) e espera por ela até
// "espera". Se terminar a tempo, devolve o resultado (pronto=true); senão a rodada segue sozinha e o resultado aparece
// no estado da coleta. Antes, a rodada morria no meio quando a conexão caía pelos limites do servidor e do Cloudflare
// (auditoria 05/11, item 1.5). Pânico vira erro (Protegida).
func EmSegundoPlano[T any](ctx context.Context, espera time.Duration, f func(context.Context) (T, error)) (r T, pronto bool, err error) {
	type saida struct {
		r   T
		err error
	}
	c := make(chan saida, 1)
	bg := context.WithoutCancel(ctx)
	go func() {
		r, err := Protegida(func() (T, error) { return f(bg) })
		c <- saida{r, err}
	}()
	t := time.NewTimer(espera)
	defer t.Stop()
	select {
	case s := <-c:
		return s.r, true, s.err
	case <-t.C:
		var zero T
		return zero, false, nil
	}
}
