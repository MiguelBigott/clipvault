# Omnissiah

Descargador de video y audio para Windows basado en [yt-dlp](https://github.com/yt-dlp/yt-dlp).
Servidor local en Go con una interfaz web embebida (`web/`) que se abre en una ventana de app.

*Alabado sea el Omnissiah.*

## Estructura

- `main.go`, `cola.go`, `componentes.go` — servidor, cola de descargas y gestión de componentes (yt-dlp, ffmpeg).
- `plataforma_windows.go` / `plataforma_otros.go` — código específico de plataforma.
- `web/` — interfaz (HTML, CSS, JS, fuentes Cinzel bajo OFL).
- `build/` — icono, manifiesto, recursos (`app.rc`) e instalador NSIS (`instalador.nsi`).

## Compilar

1. Compilar `build/app.rc` a `.syso` (p. ej. con `windres`) para incrustar icono y manifiesto.
2. `go build -ldflags "-H windowsgui" -o dist/Omnissiah.exe`
3. `makensis build/instalador.nsi` → `dist/Omnissiah-Setup-<versión>.exe`

## Versiones

Los instaladores se publican en [Releases](../../releases).
