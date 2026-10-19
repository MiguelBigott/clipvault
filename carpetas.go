package main

import (
	"os"
	"path/filepath"
	"strings"
)

// corregirMayusculas deja la carpeta con el nombre exacto de la app: si ya existe con otras
// mayúsculas («Clipvault» en vez de «ClipVault»), la renombra. Su contenido no se mueve.
func corregirMayusculas(ruta string) {
	padre, nombre := filepath.Dir(ruta), filepath.Base(ruta)
	entradas, err := os.ReadDir(padre)
	if err != nil {
		return
	}
	for _, e := range entradas {
		if e.IsDir() && e.Name() != nombre && strings.EqualFold(e.Name(), nombre) {
			_ = os.Rename(filepath.Join(padre, e.Name()), ruta)
			return
		}
	}
}
