package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// claves lee web/i18n.js y devuelve las claves de cada idioma.
func claves(t *testing.T) map[string]map[string]bool {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("web", "i18n.js"))
	if err != nil {
		t.Fatal(err)
	}
	inicio := regexp.MustCompile(`(?m)^  (\w\w): \{$`)
	clave := regexp.MustCompile(`"([\w.-]+)":`)
	s := string(b)
	porIdioma := map[string]map[string]bool{}
	pos := inicio.FindAllStringSubmatchIndex(s, -1)
	for i, p := range pos {
		fin := len(s)
		if i+1 < len(pos) {
			fin = pos[i+1][0]
		}
		id := s[p[2]:p[3]]
		porIdioma[id] = map[string]bool{}
		for _, m := range clave.FindAllStringSubmatch(s[p[1]:fin], -1) {
			porIdioma[id][m[1]] = true
		}
	}
	return porIdioma
}

// Todos los idiomas tienen exactamente los mismos textos que el español.
func TestIdiomasCompletos(t *testing.T) {
	c := claves(t)
	for _, id := range idiomas {
		if c[id] == nil {
			t.Fatalf("falta el idioma %s en web/i18n.js", id)
		}
		for k := range c["es"] {
			if !c[id][k] {
				t.Errorf("%s: falta %q", id, k)
			}
		}
		for k := range c[id] {
			if !c["es"][k] {
				t.Errorf("%s: sobra %q (no está en español)", id, k)
			}
		}
		if _, ok := nativos[id]; !ok {
			t.Errorf("%s: faltan los textos de Go", id)
		}
	}
}

// Cada clave que manda Go y cada data-t del HTML existe en el diccionario.
func TestClavesUsadas(t *testing.T) {
	es := claves(t)["es"]
	usoGo := regexp.MustCompile(`msj\("(\w+)"`)
	archivos, _ := filepath.Glob("*.go")
	for _, f := range archivos {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, _ := os.ReadFile(f)
		for _, m := range usoGo.FindAllStringSubmatch(string(b), -1) {
			if !es[m[1]] {
				t.Errorf("%s usa la clave %q, que no está en web/i18n.js", f, m[1])
			}
		}
	}
	html, _ := os.ReadFile(filepath.Join("web", "index.html"))
	usoHTML := regexp.MustCompile(`data-t(?:-\w+)?="([\w.-]+)"`)
	for _, m := range usoHTML.FindAllStringSubmatch(string(html), -1) {
		if !es[m[1]] {
			t.Errorf("index.html usa la clave %q, que no está en web/i18n.js", m[1])
		}
	}
}

// El idioma de la app siempre es uno de los 8 (el de Codex, o el del PC).
func TestIdiomaApp(t *testing.T) {
	if got := idiomaApp(); !slices.Contains(idiomas, got) {
		t.Errorf("idiomaApp = %q", got)
	}
}

func TestComoMensaje(t *testing.T) {
	if m := comoMensaje(msj("faltaMotor")); m.Clave != "faltaMotor" {
		t.Errorf("clave = %q", m.Clave)
	}
	if m := comoMensaje(os.ErrNotExist); m.Clave != "errorDetalle" || m.Datos["detalle"] == "" {
		t.Errorf("error del sistema = %+v", m)
	}
	if m := msj("motorFallo").conCausa(msj("descargaCortada")); m.Causa == nil || m.Causa.Clave != "descargaCortada" {
		t.Errorf("causa = %+v", m.Causa)
	}
}
