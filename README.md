<div align="center">

<img src="build/logo.png" alt="Logo de Omnissiah" width="88">

# Omnissiah

> Descargador de **video y audio** para Windows: pegas un enlace y obtienes el MP4 o el MP3 con su portada y metadatos, sin consola ni comandos.

[![Versión](https://img.shields.io/badge/versión-v1.2.0-c0161b)](CHANGELOG.md)
[![Licencia](https://img.shields.io/badge/licencia-privada-555)](#️-créditos)
[![CI](https://github.com/MiguelBigott/Omnissiah/actions/workflows/ci.yml/badge.svg)](https://github.com/MiguelBigott/Omnissiah/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Windows](https://img.shields.io/badge/Windows-10%20·%2011-0078D4)](#-instalación)
[![FFmpeg](https://img.shields.io/badge/motor-FFmpeg-007808?logo=ffmpeg&logoColor=white)](https://ffmpeg.org)
[![NSIS](https://img.shields.io/badge/instalador-NSIS-8a2be2)](https://nsis.sourceforge.io)

[Historial de versiones](CHANGELOG.md) · [Releases](../../releases) · [Descargar el instalador](../../releases/latest)

<img src="https://github.com/MiguelBigott/Omnissiah/raw/estadisticas/estadisticas.svg" alt="Actividad del repositorio en los últimos 30 días" width="100%">

<sub>Panel de actividad generado a diario por <a href=".github/workflows/estadisticas.yml">GitHub Actions</a>, sin servicios de terceros.</sub>

</div>

---

## 📖 Índice

- [Funciones](#-funciones)
- [Stack](#-stack)
- [Instalación](#-instalación)
- [Compilar](#️-compilar)
- [Seguridad](#-seguridad)
- [Versiones](#️-versiones)
- [Créditos](#️-créditos)

## ✨ Funciones

- **Video o audio**: MP4 o solo MP3 a máxima calidad, con la portada y los metadatos incrustados en el archivo.
- **Calidad a elección**: máxima disponible, hasta 1080p, 720p o 480p, o **compatible con todo** (H.264) para reproductores y editores exigentes.
- **Vista previa**: al pegar un enlace muestra el título y la miniatura antes de descargar; en las listas de reproducción indica cuántos videos tiene.
- **Cola de descargas**: varios enlaces a la vez (uno por línea), con barra de progreso, cancelar, reintentar y quitar las terminadas.
- **Todo incluido**: el instalador trae el motor de descarga, FFmpeg y Deno (el motor de JavaScript que algunos sitios exigen) en `C:\Program Files\Omnissiah\componentes`; no hay que bajar ni configurar nada. Botón para **actualizar el motor** cuando los sitios cambian y para **reparar los componentes** si alguno se borra.
- **Carpeta de destino**: `Videos\Descargas Omnissiah` por defecto, se cambia desde la app; botones para abrir la carpeta o mostrar el archivo descargado.
- **Ventana propia**: se abre como una aplicación independiente (Edge, Chrome o Brave en modo app) y no deja abrir dos copias a la vez.
- **Instalador**: en Archivos de programa, con acceso en el menú Inicio, casilla para el acceso directo del escritorio y desinstalador en la Configuración de Windows.

## 🧱 Stack

| Capa | Tecnología |
|---|---|
| Aplicación | [Go](https://go.dev): servidor local con la biblioteca estándar, sin dependencias externas |
| Interfaz | HTML, CSS y JavaScript embebidos en el ejecutable (`go:embed`), tipografía [Cinzel](https://fonts.google.com/specimen/Cinzel) |
| Ventana | Microsoft Edge, Chrome o Brave en modo aplicación |
| Descargas | Motor de descarga de código abierto, [FFmpeg](https://ffmpeg.org) y [Deno](https://deno.com) |
| Instalador | [NSIS](https://nsis.sourceforge.io) |
| Recursos del `.exe` | [go-winres](https://github.com/tc-hib/go-winres) (icono, manifiesto y datos de versión) |
| CI | GitHub Actions |

## 📦 Instalación

Descarga `Omnissiah-Setup-<versión>.exe` desde [Releases](../../releases/latest) y ejecútalo.

- Se instala en `C:\Program Files\Omnissiah` (pide permisos de administrador); la app se abre con los permisos normales del usuario.
- Las descargas van por defecto a `Videos\Descargas Omnissiah`.
- Todo queda en `C:\Program Files\Omnissiah`: la app y, en `componentes`, el motor de descarga, FFmpeg y Deno.
- La configuración se guarda en `%LOCALAPPDATA%\Omnissiah`.
- Al desinstalar se borran el programa, los componentes y la configuración; **tus videos no se tocan**.

## 🛠️ Compilar

Requisitos: [Go](https://go.dev/dl), [NSIS](https://nsis.sourceforge.io) y go-winres
(`go install github.com/tc-hib/go-winres@latest`).

```powershell
.\build\compilar.ps1
```

Genera `dist/Omnissiah.exe` (con icono y datos de versión), `dist/componentes/` y `dist/Omnissiah-Setup-<versión>.exe`.
La primera vez baja los componentes y los guarda en `build/componentes/` (fuera de git); para bajar versiones nuevas: `.\build\compilar.ps1 -ActualizarComponentes`.
La versión se toma de `const Version` en `main.go`.

El logo es vectorial: se define en `build/generar_logo.py`, que escribe `build/logo.svg` y regenera
`build/app.ico`, `build/icono512.png`, `build/logo.png`, `build/lateral.bmp` y `web/icono.ico`; los tamaños chicos de los iconos usan una versión simplificada del logo (requiere Python con Pillow y Microsoft Edge):

```powershell
python build\generar_logo.py
```

| Carpeta o archivo | Contenido |
|---|---|
| `main.go`, `cola.go`, `componentes.go` | Servidor, cola de descargas y gestión de los componentes de descarga |
| `plataforma_windows.go`, `plataforma_otros.go` | Código específico de cada sistema |
| `web/` | Interfaz de la app |
| `build/` | Logo, imágenes e instalador NSIS, más los scripts de compilación |
| `scripts/estadisticas.mjs` | Dibuja el panel de actividad del README |

## 🔐 Seguridad

- El servidor de la app escucha **solo en `127.0.0.1`**: no es accesible desde otros equipos de la red.
- La API rechaza peticiones con un origen distinto al de la propia app y las acciones exigen una cabecera propia, así que **una página web abierta en el navegador no puede usarla**. Los cuerpos de las peticiones están limitados a 1 MB.
- Los componentes que trae el instalador se bajan al compilar **solo desde sus repositorios oficiales** en GitHub y se verifican con la suma **SHA-256** que publica cada proyecto.
- La carpeta `componentes` es modificable por los usuarios del equipo para que la app actualice el motor sin pedir permisos de administrador; el resto de la instalación queda protegido.
- El instalador corre como administrador únicamente para escribir en Archivos de programa; la app nunca se ejecuta elevada.
- Los workflows de GitHub Actions tienen **permisos mínimos** y sus acciones están fijadas por SHA de commit.
- Este repositorio contiene **solo código**: no se versionan ejecutables, descargas ni datos del usuario.

## 🏷️ Versiones

El proyecto usa [Versionado Semántico](https://semver.org/lang/es/) (**MAYOR.MENOR.PARCHE**). Cada versión tiene su etiqueta `vX.Y.Z`, su entrada en el [CHANGELOG](CHANGELOG.md) y su página en [Releases](../../releases), con el instalador adjunto. La versión visible en la app se toma de `const Version` en `main.go`.

## 🖋️ Créditos

**Creado por Miguel Bigott** · Uso privado. Todos los derechos reservados.

Incluye [FFmpeg](https://ffmpeg.org) (GPL, licencia en `componentes/LICENSE-FFmpeg.txt`), [Deno](https://deno.com) (MIT) y la tipografía [Cinzel](https://fonts.google.com/specimen/Cinzel) (SIL Open Font License, ver `web/fonts/OFL-Cinzel.txt`). Inspirado en el universo de Warhammer 40,000; sin afiliación con Games Workshop.
