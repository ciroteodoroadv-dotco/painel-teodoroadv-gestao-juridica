// Package documentos recebe, guarda e entrega arquivos (PDF, PNG, JPEG, DOCX).
// O tipo é decidido pelo CONTEÚDO (bytes mágicos), nunca pelo nome ou pelo que o navegador declara.
// O arquivo fica cifrado (AES-256-GCM) num volume privado, com nome aleatório, fora de qualquer pasta pública.
package documentos

import (
	"archive/zip"
	"bytes"
	"errors"
	"path/filepath"
	"strings"
)

// MaxArquivo é o limite de um upload (25 MiB).
const MaxArquivo = 25 << 20

const (
	maxEntradasZip = 10000
	maxZipDescomp  = 500 << 20 // soma declarada; o arquivo nunca é descompactado pelo sistema
)

// info descreve cada tipo aceito.
var info = map[string]struct {
	Mime string
	Exts []string
}{
	"pdf":  {"application/pdf", []string{".pdf"}},
	"png":  {"image/png", []string{".png"}},
	"jpeg": {"image/jpeg", []string{".jpg", ".jpeg"}},
	"docx": {"application/vnd.openxmlformats-officedocument.wordprocessingml.document", []string{".docx"}},
}

// Detect devolve o tipo pelo conteúdo ou erro se não for um dos aceitos.
func Detect(data []byte) (string, error) {
	switch {
	case bytes.HasPrefix(data, []byte("%PDF-")):
		return "pdf", nil
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		return "png", nil
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return "jpeg", nil
	case bytes.HasPrefix(data, []byte("PK\x03\x04")):
		if isDocx(data) {
			return "docx", nil
		}
		return "", errors.New("arquivo compactado que não é um documento Word (.docx) válido")
	}
	return "", errors.New("tipo de arquivo não aceito (use PDF, PNG, JPEG ou DOCX)")
}

// isDocx confere a estrutura (sem descompactar): precisa ter [Content_Types].xml e word/document.xml.
func isDocx(data []byte) bool {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(zr.File) == 0 || len(zr.File) > maxEntradasZip {
		return false
	}
	var total uint64
	var ct, doc bool
	for _, f := range zr.File {
		total += f.UncompressedSize64
		switch f.Name {
		case "[Content_Types].xml":
			ct = true
		case "word/document.xml":
			doc = true
		}
	}
	return ct && doc && total <= maxZipDescomp
}

// ExtensionMatches diz se a extensão do nome combina com o tipo detectado.
func ExtensionMatches(name, tipo string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	for _, e := range info[tipo].Exts {
		if ext == e {
			return true
		}
	}
	return false
}

// MimeOf devolve o tipo MIME seguro para a resposta de download.
func MimeOf(tipo string) string {
	if m := info[tipo].Mime; m != "" {
		return m
	}
	return "application/octet-stream"
}

// SafeName deixa só o nome do arquivo (sem pastas), sem caracteres de controle e com até 255 caracteres.
func SafeName(n string) string {
	n = strings.ReplaceAll(n, "\\", "/")
	if i := strings.LastIndex(n, "/"); i >= 0 {
		n = n[i+1:]
	}
	n = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r == '"' {
			return -1
		}
		return r
	}, n)
	n = strings.TrimSpace(n)
	if r := []rune(n); len(r) > 255 {
		ext := filepath.Ext(n)
		n = string(r[:255-len([]rune(ext))]) + ext
	}
	return n
}
