// Package web embute os arquivos estáticos (CSS, futuramente htmx e ícones) no binário.
package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var assets embed.FS

// Static devolve o diretório static/ para servir em /static/.
func Static() fs.FS {
	sub, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err) // erro de programação: o diretório é embutido em tempo de compilação
	}
	return sub
}
