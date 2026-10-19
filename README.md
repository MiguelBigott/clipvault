<div align="center">

<img src="build/logo.png" alt="Logo de ClipVault" width="88">

# ClipVault

> Descargador de **video y audio** para Windows: pegas un enlace y obtienes el MP4 o el MP3 con su portada y metadatos, sin consola ni comandos.

[![Versión](https://img.shields.io/badge/versión-v1.3.0-c8483a)](CHANGELOG.md)
[![CI](https://github.com/MiguelBigott/clipvault/actions/workflows/ci.yml/badge.svg)](https://github.com/MiguelBigott/clipvault/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Windows](https://img.shields.io/badge/Windows-10%20·%2011-0078D4)](#-instalación)
[![Licencia](https://img.shields.io/badge/licencia-MIT-3f9d6e)](LICENSE)
[![FFmpeg](https://img.shields.io/badge/motor-FFmpeg-007808?logo=ffmpeg&logoColor=white)](https://ffmpeg.org)
[![Codex](https://img.shields.io/badge/suite-Codex-c9a45c)](https://github.com/MiguelBigott/codex)

[Historial de versiones](CHANGELOG.md) · [Releases](../../releases) · [Descargar el instalador](../../releases/latest)

<img src="https://github.com/MiguelBigott/clipvault/raw/estadisticas/estadisticas.svg" alt="Actividad del repositorio en los últimos 30 días" width="100%">

<sub>Panel de actividad generado a diario por <a href=".github/workflows/estadisticas.yml">GitHub Actions</a>, sin servicios de terceros.</sub>

</div>

---

## 📖 Índice

- [Funciones](#-funciones)
- [Stack](#-stack)
- [Instalación](#-instalación)
- [Compilar](#️-compilar)
- [Seguridad](#-seguridad)
- [Licencia](#️-licencia)
- [Créditos](#️-créditos)

## ✨ Funciones

- **Video o audio**: MP4 o solo MP3 a máxima calidad, con la portada y los metadatos incrustados en el archivo.
- **Un enlace a la vez**: pegas el enlace, pulsas Descargar y el campo queda libre para el siguiente.
- **Clip**: opcional y apagado por defecto. Al activarlo aparece una línea de tiempo con la duración del video pegado; arrastras los marcadores (o escribes Desde y Hasta) y se descarga solo ese tramo, cortado exacto.
- **Calidad a elección**: máxima disponible, hasta 1080p, 720p o 480p, o **compatible con todo** (H.264).
- **Vista previa**: al pegar un enlace muestra el título, la miniatura, el autor y la duración antes de descargar; en las listas de reproducción indica cuántos videos tiene.
- **La bóveda**: cola de descargas con progreso, velocidad y tiempo restante; cancelar, reintentar y quitar las terminadas.
- **Todo incluido**: el instalador trae el motor de descarga, FFmpeg y Deno; un botón actualiza el motor y otro repara los componentes si alguno se borra.
- **Ventana propia**: ventana nativa de Windows (WebView2) con su icono nítido en la barra de tareas, sin barra de título del sistema.
- **Parte de [Codex](https://github.com/MiguelBigott/codex)**: el launcher de la suite la instala, la actualiza y la abre.
- **En tu idioma**: usa el idioma elegido en [Codex](https://github.com/MiguelBigott/codex) (8 idiomas) o, sin Codex, el del PC.

## 🧱 Stack

| Capa | Tecnología |
|---|---|
| Aplicación | [Go](https://go.dev): servidor local con la biblioteca estándar |
| Ventana | [WebView2](https://developer.microsoft.com/microsoft-edge/webview2/) con [go-webview2](https://github.com/jchv/go-webview2) |
| Interfaz | HTML, CSS y JavaScript embebidos (`go:embed`), skin «Cofre» con Inter y Space Mono |
| Descargas | Motor de descarga de código abierto, [FFmpeg](https://ffmpeg.org) y [Deno](https://deno.com) |
| Instalador | [NSIS](https://nsis.sourceforge.io) |
| CI | GitHub Actions |

## 📦 Instalación

Descarga `ClipVault-Setup-<versión>.exe` desde [Releases](../../releases/latest), o instálalo desde [Codex](https://github.com/MiguelBigott/codex).

- Se instala en `C:\Program Files\ClipVault` (desde Codex, en `C:\Program Files\Codex\ClipVault`) con el motor de descarga, FFmpeg y Deno en `componentes`.
- Las descargas van por defecto a `Videos\ClipVault`; se puede cambiar desde la app.
- La configuración se guarda en `%LOCALAPPDATA%\ClipVault`.
- Al desinstalar se borran el programa, los componentes y la configuración; **tus videos no se tocan**.

## 🛠️ Compilar

Requisitos: [Go](https://go.dev/dl), [NSIS](https://nsis.sourceforge.io) y go-winres
(`go install github.com/tc-hib/go-winres@latest`).

```powershell
.\build\compilar.ps1
```

Genera `dist/ClipVault.exe`, `dist/componentes/` y `dist/ClipVault-Setup-<versión>.exe`. La primera vez baja
los componentes de sus repositorios oficiales, los verifica con SHA-256 y los guarda en `build/componentes/`
(fuera de git); para bajar versiones nuevas: `.\build\compilar.ps1 -ActualizarComponentes`.

El logo «Cofre» (un cofre de pergamino con correas y cerradura de oro, en el estilo del logo de Codex) se define en `build/generar_logo.py`, que escribe `web/logo.svg`,
`build/logo.png`, `build/app.ico` (con una versión reforzada para 16-32 px) y `build/lateral.bmp`:

```powershell
python build\generar_logo.py
```

Para revisar la interfaz en un navegador, sin ventana, ejecuta la app con `CLIPVAULT_SIN_VENTANA=1` y abre
`http://127.0.0.1:47830`.

## 🔐 Seguridad

- El servidor de la app escucha **solo en `127.0.0.1`**: no es accesible desde otros equipos.
- La API rechaza peticiones con un origen distinto al de la propia app y las acciones exigen una cabecera propia, así que una página web abierta en el navegador no puede usarla.
- Los componentes que trae el instalador se bajan al compilar **solo desde sus repositorios oficiales** y se verifican con la suma **SHA-256** que publica cada proyecto.
- La carpeta `componentes` es modificable por los usuarios del equipo para actualizar el motor sin permisos de administrador; el resto de la instalación queda protegido.

## ⚖️ Licencia

**Creado por [Miguel Bigott](https://github.com/MiguelBigott)** y publicado con licencia [MIT](LICENSE): cualquiera puede usarlo, copiarlo, modificarlo y compartirlo libremente, también en sus propios proyectos, siempre que mantenga el aviso de autor. Se entrega tal cual, sin garantías.

Los componentes de terceros conservan sus licencias: FFmpeg (GPL 3, con dónde conseguir su código en `componentes/FUENTES-FFmpeg.txt`), Deno (MIT) y el motor de descarga (dominio público) traen las suyas en `componentes/`, y lo que va dentro de ClipVault.exe (Go, go-webview2 y las tipografías) está en `LICENCIAS-TERCEROS.txt`, junto a la app.

## 🖋️ Créditos

**Creado por Miguel Bigott.** Antes se llamaba Omnissiah (hasta la 1.2.0).

Incluye [FFmpeg](https://ffmpeg.org) (GPL, licencia en `componentes/LICENSE-FFmpeg.txt`), [Deno](https://deno.com) (MIT) y las tipografías [Inter](https://fonts.google.com/specimen/Inter) y [Space Mono](https://fonts.google.com/specimen/Space+Mono) (SIL Open Font License, ver `web/fonts/`).
