// Clipvault — descargador de video y audio.
// Servidor local + interfaz web embebida en una ventana nativa (WebView2).
package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	webview2 "github.com/jchv/go-webview2"
)

const Version = "1.0.0"

// Puerto fijo: sirve también para detectar si la app ya está abierta.
const puerto = "127.0.0.1:47830"

//go:embed web
var webFS embed.FS

type App struct {
	mu       sync.Mutex
	cfg      Config
	cfgPath  string
	dataDir  string
	queue    *Queue
	comp     *Componentes
	lastPing time.Time
	pingSeen bool
}

type Config struct {
	Carpeta string `json:"carpeta"`
	Calidad string `json:"calidad"`
	Tipo    string `json:"tipo"`
}

func main() {
	dataDir := carpetaDatos()
	_ = os.MkdirAll(dataDir, 0o755)
	logFile, err := os.OpenFile(filepath.Join(dataDir, "clipvault.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err == nil {
		log.SetOutput(logFile)
	}

	ln, err := net.Listen("tcp", puerto)
	if err != nil {
		// Ya hay una instancia abierta: traemos su ventana al frente.
		traerAlFrente()
		return
	}

	app := &App{dataDir: dataDir, cfgPath: filepath.Join(dataDir, "config.json")}
	app.cargarConfig()
	app.comp = NuevosComponentes(app)
	app.queue = NuevaCola(app)

	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFS, "web")
	mux.Handle("/", http.FileServer(http.FS(sub)))
	app.rutas(mux)

	go func() {
		if err := http.Serve(ln, mux); err != nil {
			log.Println("servidor:", err)
		}
	}()

	url := "http://" + puerto + "/"
	if os.Getenv("CLIPVAULT_SIN_VENTANA") != "" {
		log.Println("modo prueba, escuchando en", url)
		select {}
	}

	ancho, alto := tamanoInicial()
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		DataPath:  filepath.Join(dataDir, "webview"),
		WindowOptions: webview2.WindowOptions{
			Title: tituloVentana, Width: ancho, Height: alto, Center: true, IconId: idIcono,
		},
	})
	if w == nil {
		mensaje("Clipvault necesita Microsoft Edge WebView2, que viene con Windows 11. " +
			"Instálalo desde https://developer.microsoft.com/microsoft-edge/webview2/ y vuelve a abrir Clipvault.")
		return
	}
	defer w.Destroy()
	prepararVentana(uintptr(w.Window()))
	w.SetSize(900, 620, webview2.HintMin)
	_ = w.Bind("caArrastrar", ventanaArrastrar)
	_ = w.Bind("caRedimensionar", ventanaRedimensionar)
	_ = w.Bind("caMinimizar", ventanaMinimizar)
	_ = w.Bind("caMaximizar", ventanaMaximizar)
	_ = w.Bind("caMaximizada", ventanaMaximizada)
	_ = w.Bind("caCerrar", ventanaCerrar)
	w.Navigate(url)
	w.Run()

	app.queue.CancelarTodo()
	time.Sleep(500 * time.Millisecond)
}

func (a *App) cargarConfig() {
	a.cfg = Config{Carpeta: carpetaDescargasPorDefecto(), Calidad: "max", Tipo: "video"}
	b, err := os.ReadFile(a.cfgPath)
	if err == nil {
		_ = json.Unmarshal(b, &a.cfg)
	}
	if a.cfg.Carpeta == "" {
		a.cfg.Carpeta = carpetaDescargasPorDefecto()
	}
}

func (a *App) guardarConfig() {
	b, _ := json.MarshalIndent(a.cfg, "", "  ")
	_ = os.WriteFile(a.cfgPath, b, 0o644)
}

func (a *App) config() Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg
}

// ---------------- API ----------------

func responder(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}

func fallar(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func leerJSON(r *http.Request, v any) error {
	if r.Method != http.MethodPost {
		return errors.New("método no permitido")
	}
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(v)
}

// Solo aceptamos peticiones de la propia interfaz (evita que otras webs usen la API).
func soloLocal(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && o != "http://"+puerto {
			fallar(w, http.StatusForbidden, "origen no permitido")
			return
		}
		if r.Method == http.MethodPost && r.Header.Get("X-Clipvault") != "1" {
			fallar(w, http.StatusForbidden, "falta cabecera")
			return
		}
		h(w, r)
	}
}

func (a *App) rutas(mux *http.ServeMux) {
	mux.HandleFunc("/api/ping", soloLocal(func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		a.lastPing = time.Now()
		a.pingSeen = true
		a.mu.Unlock()
		responder(w, map[string]bool{"ok": true})
	}))

	mux.HandleFunc("/api/estado", soloLocal(func(w http.ResponseWriter, r *http.Request) {
		cfg := a.config()
		responder(w, map[string]any{
			"version":     Version,
			"config":      cfg,
			"componentes": a.comp.Estado(),
			"cola":        a.queue.Snapshot(),
		})
	}))

	mux.HandleFunc("/api/vista", soloLocal(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL string `json:"url"`
		}
		if err := leerJSON(r, &req); err != nil || req.URL == "" {
			fallar(w, http.StatusBadRequest, "URL vacía")
			return
		}
		info, err := a.comp.VistaPrevia(r.Context(), req.URL)
		if err != nil {
			fallar(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		responder(w, info)
	}))

	mux.HandleFunc("/api/agregar", soloLocal(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URLs    []string         `json:"urls"`
			Tipo    string           `json:"tipo"`
			Calidad string           `json:"calidad"`
			Vistas  map[string]Vista `json:"vistas"`
		}
		if err := leerJSON(r, &req); err != nil {
			fallar(w, http.StatusBadRequest, err.Error())
			return
		}
		if req.Tipo != "audio" {
			req.Tipo = "video"
		}
		a.mu.Lock()
		a.cfg.Tipo, a.cfg.Calidad = req.Tipo, req.Calidad
		a.guardarConfig()
		a.mu.Unlock()
		n := 0
		for _, u := range req.URLs {
			if u == "" {
				continue
			}
			v := req.Vistas[u]
			a.queue.Agregar(u, req.Tipo, req.Calidad, v)
			n++
		}
		responder(w, map[string]int{"agregados": n})
	}))

	mux.HandleFunc("/api/cancelar", soloLocal(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int `json:"id"`
		}
		if err := leerJSON(r, &req); err != nil {
			fallar(w, http.StatusBadRequest, err.Error())
			return
		}
		a.queue.Cancelar(req.ID)
		responder(w, map[string]bool{"ok": true})
	}))

	mux.HandleFunc("/api/reintentar", soloLocal(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int `json:"id"`
		}
		if err := leerJSON(r, &req); err != nil {
			fallar(w, http.StatusBadRequest, err.Error())
			return
		}
		a.queue.Reintentar(req.ID)
		responder(w, map[string]bool{"ok": true})
	}))

	mux.HandleFunc("/api/limpiar", soloLocal(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			fallar(w, http.StatusMethodNotAllowed, "método no permitido")
			return
		}
		a.queue.LimpiarTerminados()
		responder(w, map[string]bool{"ok": true})
	}))

	mux.HandleFunc("/api/carpeta/elegir", soloLocal(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			fallar(w, http.StatusMethodNotAllowed, "método no permitido")
			return
		}
		actual := a.config().Carpeta
		elegida, err := elegirCarpeta("Elige dónde guardar las descargas", actual)
		if err != nil || elegida == "" {
			responder(w, map[string]any{"carpeta": actual, "cambiada": false})
			return
		}
		a.mu.Lock()
		a.cfg.Carpeta = elegida
		a.guardarConfig()
		a.mu.Unlock()
		responder(w, map[string]any{"carpeta": elegida, "cambiada": true})
	}))

	mux.HandleFunc("/api/carpeta/abrir", soloLocal(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Archivo string `json:"archivo"`
		}
		_ = leerJSON(r, &req)
		if req.Archivo != "" {
			if _, err := os.Stat(req.Archivo); err == nil {
				mostrarArchivo(req.Archivo)
				responder(w, map[string]bool{"ok": true})
				return
			}
		}
		c := a.config().Carpeta
		_ = os.MkdirAll(c, 0o755)
		abrirCarpeta(c)
		responder(w, map[string]bool{"ok": true})
	}))

	mux.HandleFunc("/api/componentes/instalar", soloLocal(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			fallar(w, http.StatusMethodNotAllowed, "método no permitido")
			return
		}
		a.comp.InstalarEnSegundoPlano()
		responder(w, a.comp.Estado())
	}))

	mux.HandleFunc("/api/componentes/actualizar", soloLocal(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			fallar(w, http.StatusMethodNotAllowed, "método no permitido")
			return
		}
		a.comp.ActualizarEnSegundoPlano()
		responder(w, a.comp.Estado())
	}))
}

func init() {
	log.SetFlags(log.LstdFlags)
	_ = fmt.Sprint
}
