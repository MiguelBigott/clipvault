# Historial de versiones

Todas las versiones siguen [Versionado Semántico](https://semver.org/lang/es/). Cada una tiene su
etiqueta `vX.Y.Z` y su página en [Releases](../../releases).

## Sin publicar

### Cambiado
- El instalador ahora instala en `C:\Program Files\Omnissiah` para todo el equipo (pide permisos de administrador). Al actualizar, quita la copia de la 1.0.0 en `%LOCALAPPDATA%\Programs` sin tocar la configuración ni los componentes.
- Las descargas van por defecto a `Videos\Descargas Omnissiah`. Quien tenía la carpeta por defecto de la 1.0.0 (`Descargas\Omnissiah`) pasa a la nueva automáticamente.
- Logo nuevo, vectorial: tecnosacerdote encapuchado dentro del engranaje del Mechanicus.

### Añadido
- Casilla al final del instalador para crear el acceso directo en el escritorio.
- `build/compilar.ps1`: compila el `.exe` (con icono y datos de versión) y el instalador en un solo paso.
- `build/generar_logo.py`: genera el logo y todos los iconos a partir de la fuente vectorial.
- CI en GitHub Actions y panel de actividad en el README.

## [1.0.0] - 2026-10-18

Primera versión.

- Descarga de video MP4 o solo audio MP3, con portada y metadatos.
- Cola de descargas con vista previa, progreso, cancelar y reintentar.
- Instalación automática de los componentes de descarga.
- Instalador para Windows.

[1.0.0]: https://github.com/MiguelBigott/Omnissiah/releases/tag/v1.0.0
