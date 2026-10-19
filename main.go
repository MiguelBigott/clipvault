// ClipVault — descargador de video y audio.
// Servidor local + interfaz web embebida en una ventana nativa (WebView2).
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	webview2 "github.com/jchv/go-webview2"
)

const Version = "1.3.1"

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
	corregirMayusculas(dataDir)
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
		mensaje(textosDe(idiomaApp()).FaltaWebView)
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
	// Desde la 1.2.0 la carpeta se llama como la app («ClipVault»): se corrige la ruta guardada
	// y las mayúsculas de la carpeta si ya existía.
	if def := carpetaDescargasPorDefecto(); strings.EqualFold(a.cfg.Carpeta, def) {
		a.cfg.Carpeta = def
		corregirMayusculas(def)
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

// fallar responde con un error que la interfaz muestra en su idioma.
func fallar(w http.ResponseWriter, status int, m *Mensaje) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]*Mensaje{"error": m})
}

func leerJSON(r *http.Request, v any) error {
	if r.Method != http.MethodPost {
		return msj("peticionInvalida")
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(v); err != nil {
		return msj("peticionInvalida")
	}
	return nil
}

// Solo aceptamos peticiones de la propia interfaz (evita que otras webs usen la API).
func soloLocal(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && o != "http://"+puerto {
			fallar(w, http.StatusForbidden, msj("accesoDenegado"))
			return
		}
		if r.Method == http.MethodPost && r.Header.Get("X-Clipvault") != "1" {
			fallar(w, http.StatusForbidden, msj("accesoDenegado"))
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
			"idioma":      idiomaApp(),
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
			fallar(w, http.StatusBadRequest, msj("urlVacia"))
			return
		}
		info, err := a.comp.VistaPrevia(r.Context(), req.URL)
		if err != nil {
			fallar(w, http.StatusUnprocessableEntity, comoMensaje(err))
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
			// Clip: tramo opcional en segundos (solo para un enlace que no sea una lista).
			Clip *struct {
				Desde float64 `json:"desde"`
				Hasta float64 `json:"hasta"`
			} `json:"clip"`
		}
		if err := leerJSON(r, &req); err != nil {
			fallar(w, http.StatusBadRequest, comoMensaje(err))
			return
		}
		if req.Tipo != "audio" {
			req.Tipo = "video"
		}
		var desde, hasta float64
		if req.Clip != nil {
			v := req.Vistas[firstURL(req.URLs)]
			valido := len(req.URLs) == 1 && v.Lista == 0 && req.Clip.Desde >= 0 && req.Clip.Hasta-req.Clip.Desde >= 1 &&
				(v.Duracion == 0 || req.Clip.Hasta <= v.Duracion+1)
			if !valido {
				fallar(w, http.StatusBadRequest, msj("clipInvalido"))
				return
			}
			desde, hasta = req.Clip.Desde, req.Clip.Hasta
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
			a.queue.Agregar(u, req.Tipo, req.Calidad, v, desde, hasta)
			n++
		}
		responder(w, map[string]int{"agregados": n})
	}))

	mux.HandleFunc("/api/cancelar", soloLocal(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int `json:"id"`
		}
		if err := leerJSON(r, &req); err != nil {
			fallar(w, http.StatusBadRequest, comoMensaje(err))
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
			fallar(w, http.StatusBadRequest, comoMensaje(err))
			return
		}
		a.queue.Reintentar(req.ID)
		responder(w, map[string]bool{"ok": true})
	}))

	mux.HandleFunc("/api/limpiar", soloLocal(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			fallar(w, http.StatusMethodNotAllowed, msj("peticionInvalida"))
			return
		}
		a.queue.LimpiarTerminados()
		responder(w, map[string]bool{"ok": true})
	}))

	mux.HandleFunc("/api/carpeta/elegir", soloLocal(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			fallar(w, http.StatusMethodNotAllowed, msj("peticionInvalida"))
			return
		}
		actual := a.config().Carpeta
		elegida, err := elegirCarpeta(textosDe(idiomaApp()).ElegirCarpeta, actual)
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
			fallar(w, http.StatusMethodNotAllowed, msj("peticionInvalida"))
			return
		}
		a.comp.InstalarEnSegundoPlano()
		responder(w, a.comp.Estado())
	}))

	mux.HandleFunc("/api/componentes/actualizar", soloLocal(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			fallar(w, http.StatusMethodNotAllowed, msj("peticionInvalida"))
			return
		}
		a.comp.ActualizarEnSegundoPlano()
		responder(w, a.comp.Estado())
	}))
}

func firstURL(urls []string) string {
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

func init() {
	log.SetFlags(log.LstdFlags)
	_ = fmt.Sprint
}
