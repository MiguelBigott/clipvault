package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	apiYtdlp  = "https://api.github.com/repos/yt-dlp/yt-dlp/releases/latest"
	apiFfmpeg = "https://api.github.com/repos/yt-dlp/FFmpeg-Builds/releases/tags/latest"
)

type Componentes struct {
	app *App
	mu  sync.Mutex

	ytdlp, ffmpeg string // rutas encontradas
	versionYtdlp  string
	tarea         string // "", "instalando", "actualizando"
	tareaTexto    string // clave de web/i18n.js (tarea.*)
	tareaProgreso float64
	mensaje       *Mensaje
	mensajeError  bool
}

type EstadoComponentes struct {
	Ytdlp         string   `json:"ytdlp"`
	Ffmpeg        string   `json:"ffmpeg"`
	VersionYtdlp  string   `json:"versionYtdlp"`
	Listo         bool     `json:"listo"`
	Tarea         string   `json:"tarea"`
	TareaTexto    string   `json:"tareaTexto"`
	TareaProgreso float64  `json:"tareaProgreso"`
	Mensaje       *Mensaje `json:"mensaje"`
	MensajeError  bool     `json:"mensajeError"`
}

func NuevosComponentes(a *App) *Componentes {
	c := &Componentes{app: a}
	c.Refrescar()
	return c
}

// carpeta es el directorio escribible del usuario. Los componentes incluidos con la app
// permanecen en Program Files y se usan como respaldo.
func (c *Componentes) carpeta() string {
	return filepath.Join(c.app.dataDir, "componentes")
}

func (c *Componentes) directorios() []string {
	dirs := []string{c.carpeta()}
	if ex, err := os.Executable(); err == nil {
		instalados := filepath.Join(filepath.Dir(ex), "componentes")
		if !strings.EqualFold(filepath.Clean(instalados), filepath.Clean(dirs[0])) {
			dirs = append(dirs, instalados)
		}
	}
	return dirs
}

// argsMotor indica al motor de descarga dónde está Deno.
func (c *Componentes) argsMotor() []string {
	for _, dir := range c.directorios() {
		if d := buscar("deno", dir); d != "" {
			return []string{"--js-runtimes", "deno:" + d}
		}
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
	var y, f string
	for _, dir := range c.directorios() {
		if y == "" {
			y = buscar("yt-dlp", dir)
		}
		if f == "" {
			f = buscar("ffmpeg", dir)
		}
	}
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
	c.tarea, c.tareaTexto, c.tareaProgreso, c.mensaje, c.mensajeError = tarea, texto, 0, nil, false
	return true
}

func (c *Componentes) avance(texto string, p float64) {
	c.mu.Lock()
	c.tareaTexto, c.tareaProgreso = texto, p
	c.mu.Unlock()
}

func (c *Componentes) terminar(msg *Mensaje, esError bool) {
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
		return Vista{}, msj("faltaMotor")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	args := append([]string{"-J", "--flat-playlist", "--no-warnings"}, c.argsMotor()...)
	cmd := comando(ctx, y, append(args, "--", url)...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return Vista{}, mensajeYtdlp(stderr.String(), msj("enlaceIlegible"))
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
		return Vista{}, msj("respuestaIlegible")
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

// mensajeYtdlp convierte la última línea ERROR de yt-dlp en un mensaje legible; las que no
// conoce se muestran tal cual (vienen en inglés del motor).
func mensajeYtdlp(stderr string, porDefecto *Mensaje) *Mensaje {
	lineas := strings.Split(strings.TrimSpace(stderr), "\n")
	for i := len(lineas) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lineas[i])
		if strings.HasPrefix(l, "ERROR:") {
			l = strings.TrimSpace(strings.TrimPrefix(l, "ERROR:"))
			if strings.Contains(l, "is not a valid URL") || strings.Contains(l, "Unsupported URL") {
				return msj("sitioNoCompatible")
			}
			if strings.Contains(l, "Private video") {
				return msj("videoPrivado")
			}
			if strings.Contains(l, "Video unavailable") {
				return msj("videoNoDisponible")
			}
			if strings.Contains(l, "Sign in to confirm") {
				return msj("pideSesion")
			}
			if len(l) > 220 {
				l = l[:220] + "…"
			}
			return msj("motorDijo", "detalle", l)
		}
	}
	return porDefecto
}

// ---------- Instalar / actualizar ----------

func (c *Componentes) InstalarEnSegundoPlano() {
	if !c.empezar("instalando", "preparando") {
		return
	}
	go func() {
		err := c.instalar()
		c.Refrescar()
		if err != nil {
			log.Println("instalar:", err)
			c.terminar(msj("componentesFallo").conCausa(err), true)
			return
		}
		c.terminar(msj("componentesListos"), false)
	}()
}

func (c *Componentes) instalar() error {
	dir := c.carpeta()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	y, f := c.Rutas()
	if y == "" {
		c.avance("motor", 0)
		if err := descargarVerificado(apiYtdlp, "yt-dlp.exe", filepath.Join(dir, "yt-dlp.exe"), 50<<20, func(p float64) {
			c.avance("motor", p)
		}); err != nil {
			return err
		}
	}
	if f == "" {
		zipPath := filepath.Join(dir, "ffmpeg.zip")
		c.avance("ffmpeg", 0)
		if err := descargarVerificado(apiFfmpeg, "ffmpeg-master-latest-win64-gpl.zip", zipPath, 500<<20, func(p float64) {
			c.avance("ffmpeg", p)
		}); err != nil {
			return err
		}
		c.avance("extrayendo", 1)
		err := extraerFfmpeg(zipPath, dir)
		_ = os.Remove(zipPath)
		if err != nil {
			return err
		}
	}
	return nil
}

type escritorProgreso struct {
	destino      io.Writer
	total, leido int64
	ultimo       time.Time
	progreso     func(float64)
}

func (e *escritorProgreso) Write(datos []byte) (int, error) {
	n, err := e.destino.Write(datos)
	e.leido += int64(n)
	if e.total > 0 && e.progreso != nil && time.Since(e.ultimo) >= 200*time.Millisecond {
		e.progreso(float64(e.leido) / float64(e.total))
		e.ultimo = time.Now()
	}
	return n, err
}

func descargarVerificado(apiURL, nombre, destino string, maxBytes int64, progreso func(float64)) error {
	cli := &http.Client{
		Timeout: 30 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme != "https" || len(via) >= 10 {
				return errors.New("redirección de descarga no válida")
			}
			return nil
		},
	}
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ClipVault")
	meta, err := cli.Do(req)
	if err != nil {
		return msj("sinConexionDescargas")
	}
	if meta.StatusCode != http.StatusOK {
		meta.Body.Close()
		return msj("servidorRespondio", "codigo", strconv.Itoa(meta.StatusCode))
	}
	var release struct {
		Assets []struct {
			Name   string `json:"name"`
			URL    string `json:"browser_download_url"`
			Digest string `json:"digest"`
			Size   int64  `json:"size"`
		} `json:"assets"`
	}
	decodeErr := json.NewDecoder(io.LimitReader(meta.Body, 2<<20)).Decode(&release)
	meta.Body.Close()
	if decodeErr != nil {
		return decodeErr
	}
	var assetURL, expected string
	var expectedSize int64
	for _, asset := range release.Assets {
		if asset.Name == nombre {
			assetURL, expected, expectedSize = asset.URL, asset.Digest, asset.Size
			break
		}
	}
	asset, err := url.Parse(assetURL)
	if err != nil || asset.Scheme != "https" || asset.Host != "github.com" ||
		!strings.HasPrefix(expected, "sha256:") || len(expected) != len("sha256:")+64 ||
		expectedSize <= 0 || expectedSize > maxBytes {
		return errors.New("release sin checksum SHA-256 válido o fuera del tamaño permitido")
	}
	expected = strings.TrimPrefix(expected, "sha256:")
	if _, err := hex.DecodeString(expected); err != nil {
		return errors.New("checksum SHA-256 inválido")
	}
	downloadReq, err := http.NewRequest(http.MethodGet, assetURL, nil)
	if err != nil {
		return err
	}
	resp, err := cli.Do(downloadReq)
	if err != nil {
		return msj("sinConexionDescargas")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Request.URL.Scheme != "https" {
		return msj("servidorRespondio", "codigo", strconv.Itoa(resp.StatusCode))
	}
	if resp.ContentLength > maxBytes || (resp.ContentLength >= 0 && resp.ContentLength != expectedSize) {
		return errors.New("tamaño de descarga inesperado")
	}
	if err := os.MkdirAll(filepath.Dir(destino), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(destino), ".clipvault-*.part")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	hash := sha256.New()
	observador := &escritorProgreso{destino: io.MultiWriter(tmp, hash), total: expectedSize, ultimo: time.Now(), progreso: progreso}
	total, copyErr := io.Copy(observador, io.LimitReader(resp.Body, maxBytes+1))
	if copyErr == nil && total > maxBytes {
		copyErr = errors.New("descarga supera el tamaño permitido")
	}
	if copyErr == nil && total != expectedSize {
		copyErr = errors.New("descarga incompleta")
	}
	if copyErr == nil && !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), expected) {
		copyErr = errors.New("el checksum SHA-256 no coincide")
	}
	if copyErr == nil {
		copyErr = tmp.Sync()
	}
	closeErr := tmp.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return copyErr
	}
	progreso(1)
	backup := destino + ".previous"
	_ = os.Remove(backup)
	if _, err := os.Stat(destino); err == nil {
		if err := os.Rename(destino, backup); err != nil {
			return err
		}
	}
	if err := os.Rename(tmpPath, destino); err != nil {
		_ = os.Rename(backup, destino)
		return err
	}
	_ = os.Remove(backup)
	return nil
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
		if f.UncompressedSize64 > 300<<20 {
			return errors.New("ejecutable ffmpeg supera el tamaño permitido")
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
		return msj("zipSinFfmpeg")
	}
	return nil
}

func (c *Componentes) ActualizarEnSegundoPlano() {
	if !c.empezar("actualizando", "rito") {
		return
	}
	go func() {
		antes := c.Estado().VersionYtdlp
		y, _ := c.Rutas()
		if y == "" {
			c.terminar(msj("faltaMotor"), true)
			return
		}
		destino := filepath.Join(c.carpeta(), "yt-dlp.exe")
		c.avance("reciente", 0)
		err := descargarVerificado(apiYtdlp, "yt-dlp.exe", destino, 50<<20, func(p float64) {
			c.avance("reciente", p)
		})
		c.Refrescar()
		if err != nil {
			c.terminar(msj("motorFallo").conCausa(err), true)
			return
		}
		despues := c.Estado().VersionYtdlp
		if despues == antes {
			c.terminar(msj("motorAlDia", "version", despues), false)
		} else {
			c.terminar(msj("motorRenovado", "version", despues), false)
		}
	}()
}
