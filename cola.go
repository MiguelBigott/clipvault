package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Item struct {
	ID        int      `json:"id"`
	URL       string   `json:"url"`
	Tipo      string   `json:"tipo"`    // video | audio
	Calidad   string   `json:"calidad"` // max | 1080 | 720 | 480 | compat
	Titulo    string   `json:"titulo"`
	Miniatura string   `json:"miniatura"`
	Estado    string   `json:"estado"`   // cola | descargando | procesando | listo | error | cancelado
	Fase      string   `json:"fase"`     // clave de web/i18n.js (fase.*)
	Progreso  float64  `json:"progreso"` // 0..1
	Velocidad string   `json:"velocidad"`
	Restante  string   `json:"restante"`
	Tamano    string   `json:"tamano"`
	Archivo   string   `json:"archivo"`
	Error     *Mensaje `json:"error"`
	Lista     int      `json:"lista"`
	Indice    int      `json:"indice"`
	// Clip: tramo en segundos; ClipHasta == 0 significa el video completo.
	ClipDesde float64 `json:"clipDesde"`
	ClipHasta float64 `json:"clipHasta"`

	cancel context.CancelFunc
}

type Queue struct {
	app   *App
	mu    sync.Mutex
	items []*Item
	next  int
	wake  chan struct{}
}

func NuevaCola(a *App) *Queue {
	q := &Queue{app: a, next: 1, wake: make(chan struct{}, 1)}
	go q.trabajador()
	return q
}

func (q *Queue) Agregar(url, tipo, calidad string, v Vista, desde, hasta float64) {
	q.mu.Lock()
	it := &Item{ID: q.next, URL: url, Tipo: tipo, Calidad: calidad, Estado: "cola",
		Titulo: v.Titulo, Miniatura: v.Miniatura, Lista: v.Lista, ClipDesde: desde, ClipHasta: hasta}
	q.next++
	q.items = append(q.items, it)
	q.mu.Unlock()
	q.despertar()
}

func (q *Queue) despertar() {
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

func (q *Queue) Snapshot() []Item {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]Item, len(q.items))
	for i, it := range q.items {
		out[i] = *it
		out[i].cancel = nil
	}
	return out
}

func (q *Queue) buscar(id int) *Item {
	for _, it := range q.items {
		if it.ID == id {
			return it
		}
	}
	return nil
}

func (q *Queue) Cancelar(id int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	it := q.buscar(id)
	if it == nil {
		return
	}
	switch it.Estado {
	case "cola":
		it.Estado = "cancelado"
	case "descargando", "procesando":
		it.Estado = "cancelado"
		if it.cancel != nil {
			it.cancel()
		}
	}
}

func (q *Queue) Reintentar(id int) {
	q.mu.Lock()
	it := q.buscar(id)
	if it != nil && (it.Estado == "error" || it.Estado == "cancelado") {
		it.Estado, it.Error, it.Progreso, it.Fase, it.Velocidad, it.Restante = "cola", nil, 0, "", "", ""
	}
	q.mu.Unlock()
	q.despertar()
}

func (q *Queue) LimpiarTerminados() {
	q.mu.Lock()
	defer q.mu.Unlock()
	var quedan []*Item
	for _, it := range q.items {
		if it.Estado == "cola" || it.Estado == "descargando" || it.Estado == "procesando" {
			quedan = append(quedan, it)
		}
	}
	q.items = quedan
}

func (q *Queue) CancelarTodo() {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, it := range q.items {
		if it.cancel != nil {
			it.cancel()
		}
	}
}

func (q *Queue) siguiente() *Item {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, it := range q.items {
		if it.Estado == "cola" {
			it.Estado = "descargando"
			it.Fase = "conectando"
			return it
		}
	}
	return nil
}

func (q *Queue) trabajador() {
	for {
		it := q.siguiente()
		if it == nil {
			<-q.wake
			continue
		}
		q.descargar(it)
	}
}

// actualizar aplica cambios a un item bajo el candado.
func (q *Queue) actualizar(it *Item, f func(*Item)) {
	q.mu.Lock()
	f(it)
	q.mu.Unlock()
}

func formato(tipo, calidad string) []string {
	if tipo == "audio" {
		return []string{"-f", "ba/b", "-x", "--audio-format", "mp3", "--audio-quality", "0"}
	}
	f := "bv*+ba/b"
	switch calidad {
	case "1080", "720", "480":
		f = fmt.Sprintf("bv*[height<=%s]+ba/b[height<=%s]/bv*+ba/b", calidad, calidad)
	case "compat":
		f = "bv*[vcodec^=avc1]+ba[ext=m4a]/b[ext=mp4]/bv*+ba/b"
	}
	return []string{"-f", f, "--merge-output-format", "mp4"}
}

const marcaProgreso = "[omni]"

func (q *Queue) descargar(it *Item) {
	y, ff := q.app.comp.Rutas()
	if y == "" || ff == "" {
		q.actualizar(it, func(i *Item) {
			i.Estado, i.Error = "error", msj("faltanComponentes")
		})
		return
	}
	carpeta := q.app.config().Carpeta
	if err := os.MkdirAll(carpeta, 0o755); err != nil {
		q.actualizar(it, func(i *Item) { i.Estado, i.Error = "error", msj("noEscribe") })
		return
	}

	args := formato(it.Tipo, it.Calidad)
	args = append(args, q.app.comp.argsMotor()...)
	plantilla := "%(title)s.%(ext)s"
	if it.ClipHasta > 0 {
		// Solo el tramo elegido, cortado exacto en los tiempos (no en el fotograma clave más cercano).
		args = append(args, "--download-sections", fmt.Sprintf("*%.3f-%.3f", it.ClipDesde, it.ClipHasta),
			"--force-keyframes-at-cuts")
		plantilla = fmt.Sprintf("%%(title)s (clip %s-%s).%%(ext)s", tiempoArchivo(it.ClipDesde), tiempoArchivo(it.ClipHasta))
	}
	args = append(args,
		"--embed-thumbnail", "--embed-metadata",
		"--ffmpeg-location", filepath.Dir(ff),
		"-P", carpeta, "-o", plantilla,
		"--newline", "--no-colors", "--windows-filenames",
		"--progress-template", "download:"+marcaProgreso+"%(progress.status)s|%(progress.downloaded_bytes)s|%(progress.total_bytes)s|%(progress.total_bytes_estimate)s|%(progress.speed)s|%(progress.eta)s|%(info.vcodec)s|%(info.acodec)s|%(info.playlist_index)s|%(info.title)s",
		"--", it.URL,
	)

	ctx, cancel := context.WithCancel(context.Background())
	q.actualizar(it, func(i *Item) { i.cancel = cancel })
	defer cancel()

	cmd := comando(ctx, y, args...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		q.actualizar(it, func(i *Item) { i.Estado, i.Error = "error", msj("noIniciaMotor") })
		return
	}
	alCancelar(ctx, cmd)

	var errTexto strings.Builder
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		q.leerSalida(it, stdout)
	}()
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(stderr)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			l := sc.Text()
			errTexto.WriteString(l + "\n")
			q.procesarLinea(it, l)
		}
	}()
	wg.Wait()
	err := cmd.Wait()

	q.mu.Lock()
	defer q.mu.Unlock()
	it.cancel = nil
	if it.Estado == "cancelado" {
		return
	}
	if err != nil {
		it.Estado, it.Fase, it.Velocidad, it.Restante = "error", "", "", ""
		it.Error = mensajeYtdlp(errTexto.String(), msj("descargaFallo"))
		log.Println("descarga", it.URL, err, errTexto.String())
		return
	}
	it.Estado, it.Progreso, it.Fase, it.Velocidad, it.Restante = "listo", 1, "", "", ""
}

func (q *Queue) leerSalida(it *Item, r io.Reader) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		q.procesarLinea(it, sc.Text())
	}
}

func (q *Queue) procesarLinea(it *Item, l string) {
	l = strings.TrimRight(l, "\r")
	if strings.HasPrefix(l, marcaProgreso) {
		q.procesarProgreso(it, strings.TrimPrefix(l, marcaProgreso))
		return
	}
	if ruta := rutaDe(l); ruta != "" {
		q.actualizar(it, func(i *Item) { i.Archivo = ruta })
	}
	fase := ""
	switch {
	case strings.HasPrefix(l, "[Merger]"):
		fase = "uniendo"
	case strings.HasPrefix(l, "[ExtractAudio]"):
		fase = "mp3"
	case strings.HasPrefix(l, "[EmbedThumbnail]"):
		fase = "portada"
	case strings.HasPrefix(l, "[Metadata]"):
		fase = "metadatos"
	case strings.HasPrefix(l, "[FixupM3u8]"), strings.HasPrefix(l, "[VideoConvertor]"), strings.HasPrefix(l, "[VideoRemuxer]"):
		fase = "procesando"
	}
	if fase != "" {
		q.actualizar(it, func(i *Item) {
			if i.Estado != "cancelado" {
				i.Estado, i.Fase, i.Velocidad, i.Restante, i.Progreso = "procesando", fase, "", "", 1
			}
		})
	}
}

// rutaDe reconoce las líneas de yt-dlp que nombran el archivo resultante.
func rutaDe(l string) string {
	quitarComillas := func(s string) string {
		s = strings.TrimSpace(s)
		return strings.Trim(s, "\"")
	}
	switch {
	case strings.HasPrefix(l, "[Merger] Merging formats into "):
		return quitarComillas(strings.TrimPrefix(l, "[Merger] Merging formats into "))
	case strings.HasPrefix(l, "[ExtractAudio] Destination: "):
		return quitarComillas(strings.TrimPrefix(l, "[ExtractAudio] Destination: "))
	case strings.HasPrefix(l, "[download] Destination: "):
		return quitarComillas(strings.TrimPrefix(l, "[download] Destination: "))
	case strings.HasPrefix(l, "[download] ") && strings.HasSuffix(l, " has already been downloaded"):
		return quitarComillas(strings.TrimSuffix(strings.TrimPrefix(l, "[download] "), " has already been downloaded"))
	case strings.HasPrefix(l, "[VideoRemuxer] Remuxing video from ") && strings.Contains(l, "; Destination: "):
		return quitarComillas(l[strings.Index(l, "; Destination: ")+len("; Destination: "):])
	}
	return ""
}

func (q *Queue) procesarProgreso(it *Item, datos string) {
	campos := strings.SplitN(datos, "|", 10)
	if len(campos) < 10 {
		return
	}
	bajado := num(campos[1])
	total := num(campos[2])
	if total <= 0 {
		total = num(campos[3])
	}
	vel := num(campos[4])
	eta := num(campos[5])
	vcodec, acodec := campos[6], campos[7]
	idx, _ := strconv.Atoi(strings.TrimSpace(campos[8]))
	titulo := campos[9]
	fase := "video"
	if it.Tipo == "audio" || (vcodec == "none" && acodec != "none") {
		fase = "audio"
	}
	q.actualizar(it, func(i *Item) {
		if i.Estado == "cancelado" {
			return
		}
		i.Estado, i.Fase = "descargando", fase
		if total > 0 {
			i.Progreso = bajado / total
			i.Tamano = bytesLegibles(total)
		}
		i.Velocidad, i.Restante = "", ""
		if vel > 0 {
			i.Velocidad = bytesLegibles(vel) + "/s"
		}
		if eta > 0 {
			i.Restante = duracionLegible(eta)
		}
		if i.Lista == 0 && titulo != "" && titulo != "NA" {
			i.Titulo = titulo
		}
		if idx > 0 {
			i.Indice = idx
		}
	})
}

// tiempoArchivo escribe un tiempo apto para nombres de archivo: 12m00s, 1h02m15s.
func tiempoArchivo(seg float64) string {
	s := int(seg + .5)
	if s >= 3600 {
		return fmt.Sprintf("%dh%02dm%02ds", s/3600, s%3600/60, s%60)
	}
	return fmt.Sprintf("%dm%02ds", s/60, s%60)
}

func num(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return f
}

func bytesLegibles(b float64) string {
	unidades := []string{"B", "KB", "MB", "GB"}
	i := 0
	for b >= 1024 && i < len(unidades)-1 {
		b /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", b, unidades[i])
	}
	return fmt.Sprintf("%.1f %s", b, unidades[i])
}

func duracionLegible(seg float64) string {
	d := time.Duration(seg) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
