package prazos

import (
	"fmt"
	"sync"
)

type Registry struct {
	mu         sync.RWMutex
	strategies map[RamoDireito]CalculadorStrategy
}

var defaultRegistry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		defaultRegistry = &Registry{
			strategies: make(map[RamoDireito]CalculadorStrategy),
		}
		defaultRegistry.Register(&CPCStrategy{})
		defaultRegistry.Register(&CPPStrategy{})
		defaultRegistry.Register(&CPStrategy{})
		defaultRegistry.Register(&CCStrategy{})
		defaultRegistry.Register(&CTNStrategy{})
		defaultRegistry.Register(&CTBStrategy{})
		defaultRegistry.Register(&LEFStrategy{})
		defaultRegistry.Register(&LEPStrategy{})
	})
	return defaultRegistry
}

func (r *Registry) Register(strategy CalculadorStrategy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies[strategy.Ramo()] = strategy
}

func (r *Registry) Get(ramo RamoDireito) (CalculadorStrategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if ramo == "" {
		ramo = RamoCPC
	}

	strategy, exists := r.strategies[ramo]
	if !exists {
		return nil, fmt.Errorf("motor de prazos não encontrado para o ramo: %s", ramo)
	}
	return strategy, nil
}

func CalcularPrazo(req RequisicaoPrazo) (ResultadoPrazo, error) {
	if req.QuantidadeDias <= 0 {
		return ResultadoPrazo{}, fmt.Errorf("a quantidade de dias deve ser maior que zero")
	}

	strategy, err := GetRegistry().Get(req.Ramo)
	if err != nil {
		return ResultadoPrazo{}, err
	}

	return strategy.Calcular(req)
}
