package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	urlYtdlp  = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
	urlFfmpeg = "https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip"
)

type Componentes struct {
	app *App
	mu  sync.Mutex

	ytdlp, ffmpeg string // rutas encontradas
	versionYtdlp  string
	tarea         string // "", "instalando", "actualizando"
	tareaTexto    string
	tareaProgreso float64
	mensaje       string
	mensajeError  bool
}

type EstadoComponentes struct {
	Ytdlp         string  `json:"ytdlp"`
	Ffmpeg        string  `json:"ffmpeg"`
	VersionYtdlp  string  `json:"versionYtdlp"`
	Listo         bool    `json:"listo"`
	Tarea         string  `json:"tarea"`
	TareaTexto    string  `json:"tareaTexto"`
	TareaProgreso float64 `json:"tareaProgreso"`
	Mensaje       string  `json:"mensaje"`
	MensajeError  bool    `json:"mensajeError"`
}

func NuevosComponentes(a *App) *Componentes {
	c := &Componentes{app: a}
	c.Refrescar()
	return c
}

// carpeta es donde viven los componentes: "componentes" junto al .exe de la app
// (C:\Program Files\Clipvault\componentes). El instalador los deja ahí.
func (c *Componentes) carpeta() string {
	ex, err := os.Executable()
	if err != nil {
		return filepath.Join(c.app.dataDir, "componentes")
	}
	return filepath.Join(filepath.Dir(ex), "componentes")
}

// argsMotor indica al motor de descarga dónde está Deno (su motor de JavaScript),
// que viene en la misma carpeta de componentes.
func (c *Componentes) argsMotor() []string {
	if d := buscar("deno", c.carpeta()); d != "" {
		return []string{"--js-runtimes", "deno:" + d}
	}
	return nil
}

func buscar(nombre, dir string) string {
	p := filepath.Join(dir, exe(nombre))
	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		return p
	}
	return ""
}

func (c *Componentes) Refrescar() {
	dir := c.carpeta()
	y := buscar("yt-dlp", dir)
	f := buscar("ffmpeg", dir)
	ver := ""
	if y != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		out, err := comando(ctx, y, "--version").Output()
		cancel()
		if err == nil {
			ver = strings.TrimSpace(string(out))
		}
	}
	c.mu.Lock()
	c.ytdlp, c.ffmpeg, c.versionYtdlp = y, f, ver
	c.mu.Unlock()
}

func (c *Componentes) Rutas() (string, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ytdlp, c.ffmpeg
}

func (c *Componentes) Estado() EstadoComponentes {
	c.mu.Lock()
	defer c.mu.Unlock()
	return EstadoComponentes{
		Ytdlp: c.ytdlp, Ffmpeg: c.ffmpeg, VersionYtdlp: c.versionYtdlp,
		Listo: c.ytdlp != "" && c.ffmpeg != "",
		Tarea: c.tarea, TareaTexto: c.tareaTexto, TareaProgreso: c.tareaProgreso,
		Mensaje: c.mensaje, MensajeError: c.mensajeError,
	}
}

func (c *Componentes) empezar(tarea, texto string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tarea != "" {
		return false
	}
	c.tarea, c.tareaTexto, c.tareaProgreso, c.mensaje, c.mensajeError = tarea, texto, 0, "", false
	return true
}

func (c *Componentes) avance(texto string, p float64) {
	c.mu.Lock()
	c.tareaTexto, c.tareaProgreso = texto, p
	c.mu.Unlock()
}

func (c *Componentes) terminar(msg string, esError bool) {
	c.mu.Lock()
	c.tarea, c.tareaTexto, c.tareaProgreso = "", "", 0
	c.mensaje, c.mensajeError = msg, esError
	c.mu.Unlock()
}

// ---------- Vista previa ----------

type Vista struct {
	Titulo    string  `json:"titulo"`
	Canal     string  `json:"canal"`
	Duracion  float64 `json:"duracion"`
	Miniatura string  `json:"miniatura"`
	Altura    int     `json:"altura"`
	Lista     int     `json:"lista"` // nº de videos si es lista de reproducción
}

func (c *Componentes) VistaPrevia(ctx context.Context, url string) (Vista, error) {
	y, _ := c.Rutas()
	if y == "" {
		return Vista{}, errors.New("Falta el motor de descarga. Instálalo desde el panel de componentes.")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	args := append([]string{"-J", "--flat-playlist", "--no-warnings"}, c.argsMotor()...)
	cmd := comando(ctx, y, append(args, "--", url)...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return Vista{}, errors.New(mensajeYtdlp(stderr.String(), "No se pudo leer ese enlace."))
	}
	var raw struct {
		Type       string  `json:"_type"`
		Title      string  `json:"title"`
		Uploader   string  `json:"uploader"`
		Channel    string  `json:"channel"`
		Duration   float64 `json:"duration"`
		Thumbnail  string  `json:"thumbnail"`
		Height     int     `json:"height"`
		Thumbnails []struct {
			URL   string `json:"url"`
			Width int    `json:"width"`
		} `json:"thumbnails"`
		Formats []struct {
			Height int `json:"height"`
		} `json:"formats"`
		Entries []json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return Vista{}, errors.New("El motor de descarga devolvió una respuesta que no se pudo leer.")
	}
	v := Vista{Titulo: raw.Title, Canal: raw.Channel, Duracion: raw.Duration, Miniatura: raw.Thumbnail, Altura: raw.Height}
	if v.Canal == "" {
		v.Canal = raw.Uploader
	}
	if v.Miniatura == "" {
		best := 0
		for _, t := range raw.Thumbnails {
			if t.Width >= best && strings.HasPrefix(t.URL, "http") {
				best, v.Miniatura = t.Width, t.URL
			}
		}
	}
	for _, f := range raw.Formats {
		if f.Height > v.Altura {
			v.Altura = f.Height
		}
	}
	if raw.Type == "playlist" {
		v.Lista = len(raw.Entries)
	}
	return v, nil
}

// mensajeYtdlp extrae la última línea ERROR de yt-dlp en un texto legible.
func mensajeYtdlp(stderr, porDefecto string) string {
	lineas := strings.Split(strings.TrimSpace(stderr), "\n")
	for i := len(lineas) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lineas[i])
		if strings.HasPrefix(l, "ERROR:") {
			l = strings.TrimSpace(strings.TrimPrefix(l, "ERROR:"))
			if strings.Contains(l, "is not a valid URL") || strings.Contains(l, "Unsupported URL") {
				return "Ese enlace no es de un sitio compatible. Revisa que esté completo."
			}
			if strings.Contains(l, "Private video") {
				return "El video es privado."
			}
			if strings.Contains(l, "Video unavailable") {
				return "El video no está disponible."
			}
			if strings.Contains(l, "Sign in to confirm") {
				return "El sitio pide iniciar sesión para ver este video."
			}
			if len(l) > 220 {
				l = l[:220] + "…"
			}
			return l
		}
	}
	return porDefecto
}

// ---------- Instalar / actualizar ----------

func (c *Componentes) InstalarEnSegundoPlano() {
	if !c.empezar("instalando", "Preparando la descarga de componentes") {
		return
	}
	go func() {
		err := c.instalar()
		c.Refrescar()
		if err != nil {
			log.Println("instalar:", err)
			c.terminar("No se pudieron instalar los componentes: "+err.Error(), true)
			return
		}
		c.terminar("Componentes instalados. El espíritu-máquina está listo.", false)
	}()
}

func (c *Componentes) instalar() error {
	dir := c.carpeta()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	y, f := c.Rutas()
	if y == "" {
		c.avance("Descargando el motor", 0)
		if err := descargarArchivo(urlYtdlp, filepath.Join(dir, "yt-dlp.exe"), func(p float64) {
			c.avance("Descargando el motor", p)
		}); err != nil {
			return fmt.Errorf("motor de descarga: %w", err)
		}
	}
	if f == "" {
		zipPath := filepath.Join(dir, "ffmpeg.zip")
		c.avance("Descargando ffmpeg (unos 100 MB)", 0)
		if err := descargarArchivo(urlFfmpeg, zipPath, func(p float64) {
			c.avance("Descargando ffmpeg (unos 100 MB)", p)
		}); err != nil {
			return fmt.Errorf("ffmpeg: %w", err)
		}
		c.avance("Extrayendo ffmpeg", 1)
		err := extraerFfmpeg(zipPath, dir)
		_ = os.Remove(zipPath)
		if err != nil {
			return fmt.Errorf("ffmpeg: %w", err)
		}
	}
	return nil
}

func descargarArchivo(url, destino string, progreso func(float64)) error {
	cli := &http.Client{Timeout: 30 * time.Minute}
	resp, err := cli.Get(url)
	if err != nil {
		return errors.New("sin conexión con el servidor de descargas")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("el servidor respondió %d", resp.StatusCode)
	}
	tmp := destino + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	total := resp.ContentLength
	var leido int64
	buf := make([]byte, 256*1024)
	ultimo := time.Now()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				out.Close()
				return werr
			}
			leido += int64(n)
			if total > 0 && time.Since(ultimo) > 200*time.Millisecond {
				progreso(float64(leido) / float64(total))
				ultimo = time.Now()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			out.Close()
			return errors.New("la descarga se cortó")
		}
	}
	out.Close()
	progreso(1)
	_ = os.Remove(destino)
	return os.Rename(tmp, destino)
}

func extraerFfmpeg(zipPath, dir string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	encontrados := 0
	for _, f := range zr.File {
		base := filepath.Base(f.Name)
		if !strings.Contains(f.Name, "/bin/") || (base != "ffmpeg.exe" && base != "ffprobe.exe") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(filepath.Join(dir, base))
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
		encontrados++
	}
	if encontrados < 2 {
		return errors.New("el archivo descargado no traía ffmpeg.exe y ffprobe.exe")
	}
	return nil
}

func (c *Componentes) ActualizarEnSegundoPlano() {
	if !c.empezar("actualizando", "Ejecutando el Rito de Actualización") {
		return
	}
	go func() {
		antes := c.Estado().VersionYtdlp
		y, _ := c.Rutas()
		var err error
		if y == "" {
			err = errors.New("falta el motor de descarga")
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			var out strings.Builder
			cmd := comando(ctx, y, "-U")
			cmd.Stdout, cmd.Stderr = &out, &out
			err = cmd.Run()
			cancel()
			log.Println("yt-dlp -U:", out.String())
			if err != nil {
				// Algunas variantes de yt-dlp no se actualizan solas: bajamos el .exe oficial.
				c.avance("Descargando la versión más reciente", 0)
				err = descargarArchivo(urlYtdlp, y, func(p float64) {
					c.avance("Descargando la versión más reciente", p)
				})
			}
		}
		c.Refrescar()
		if err != nil {
			c.terminar("No se pudo actualizar el motor: "+err.Error(), true)
			return
		}
		despues := c.Estado().VersionYtdlp
		if despues == antes {
			c.terminar("El motor ya estaba en la versión más reciente ("+despues+").", false)
		} else {
			c.terminar("El espíritu-máquina ha sido renovado (versión "+despues+").", false)
		}
	}()
}
