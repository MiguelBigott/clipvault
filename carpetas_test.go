package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Una carpeta con otras mayúsculas se renombra al nombre exacto, sin perder su contenido.
func TestCorregirMayusculas(t *testing.T) {
	base := t.TempDir()
	vieja := filepath.Join(base, "Clipvault")
	if err := os.Mkdir(vieja, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vieja, "video.mp4"), []byte("hola"), 0o644); err != nil {
		t.Fatal(err)
	}
	nueva := filepath.Join(base, "ClipVault")
	corregirMayusculas(nueva)

	entradas, _ := os.ReadDir(base)
	if len(entradas) != 1 || entradas[0].Name() != "ClipVault" {
		t.Fatalf("la carpeta debía llamarse ClipVault, quedó %v", entradas)
	}
	if b, err := os.ReadFile(filepath.Join(nueva, "video.mp4")); err != nil || string(b) != "hola" {
		t.Fatalf("se perdió el contenido: %v", err)
	}
	// Si ya tiene el nombre exacto, o no existe, no hace nada.
	corregirMayusculas(nueva)
	corregirMayusculas(filepath.Join(base, "NoExiste"))
	if entradas, _ := os.ReadDir(base); len(entradas) != 1 {
		t.Fatalf("no debía crear ni mover nada: %v", entradas)
	}
}
