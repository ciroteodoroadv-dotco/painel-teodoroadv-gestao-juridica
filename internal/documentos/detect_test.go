package documentos

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func zipWith(names ...string) []byte {
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	for _, n := range names {
		w, _ := zw.Create(n)
		_, _ = w.Write([]byte("x"))
	}
	_ = zw.Close()
	return b.Bytes()
}

func TestDetect(t *testing.T) {
	ok := map[string][]byte{
		"pdf":  []byte("%PDF-1.7 ..."),
		"png":  []byte("\x89PNG\r\n\x1a\nxxxx"),
		"jpeg": {0xFF, 0xD8, 0xFF, 0xE1, 0},
		"docx": zipWith("[Content_Types].xml", "word/document.xml"),
	}
	for want, data := range ok {
		if got, err := Detect(data); err != nil || got != want {
			t.Errorf("Detect(%s) = %q, %v", want, got, err)
		}
	}
	for nome, data := range map[string][]byte{
		"executável":      []byte("MZ\x90\x00"),
		"script":          []byte("#!/bin/sh\nrm -rf /"),
		"html":            []byte("<html><script>alert(1)</script>"),
		"zip comum":       zipWith("a.txt"),
		"docx incompleto": zipWith("[Content_Types].xml"),
		"vazio":           nil,
		"pdf sem versão":  []byte("%PDF"),
	} {
		if _, err := Detect(data); err == nil {
			t.Errorf("%s deveria ser recusado", nome)
		}
	}
}

func TestExtensaoEConteudo(t *testing.T) {
	if !ExtensionMatches("A.PDF", "pdf") || !ExtensionMatches("f.jpeg", "jpeg") || !ExtensionMatches("f.JPG", "jpeg") {
		t.Error("extensões válidas recusadas")
	}
	if ExtensionMatches("a.pdf", "png") || ExtensionMatches("semextensao", "pdf") || ExtensionMatches("a.pdf.exe", "pdf") {
		t.Error("extensões que não combinam foram aceitas")
	}
	if MimeOf("pdf") != "application/pdf" || MimeOf("???") != "application/octet-stream" {
		t.Error("MIME inesperado")
	}
}

func TestSafeName(t *testing.T) {
	casos := map[string]string{
		"../../etc/passwd.pdf":     "passwd.pdf",
		`C:\Users\x\Contrato.docx`: "Contrato.docx",
		"a\x00b\r\n.pdf":           "ab.pdf",
		`com"aspas.pdf`:            "comaspas.pdf",
		"  espaço.pdf  ":           "espaço.pdf",
	}
	for in, want := range casos {
		if got := SafeName(in); got != want {
			t.Errorf("SafeName(%q) = %q, esperado %q", in, got, want)
		}
	}
	longo := SafeName(strings.Repeat("a", 400) + ".pdf")
	if len([]rune(longo)) != 255 || !strings.HasSuffix(longo, ".pdf") {
		t.Errorf("nome longo deveria ser cortado em 255 preservando a extensão: %d", len([]rune(longo)))
	}
}

func TestStoreNomesInvalidos(t *testing.T) {
	s, err := NewStore(t.TempDir(), bytes.Repeat([]byte{1}, 32))
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"../x", "a/b", "", strings.Repeat("g", 64), strings.Repeat("a", 63)} {
		if _, err := s.Get(n); err == nil {
			t.Errorf("nome %q deveria ser recusado", n)
		}
	}
	name, err := s.Put([]byte("conteudo"))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.Get(name); err != nil || string(got) != "conteudo" {
		t.Errorf("ida e volta: %q %v", got, err)
	}
}
