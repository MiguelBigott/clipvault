# Historial de versiones

Todas las versiones siguen [Versionado Semántico](https://semver.org/lang/es/). Cada una tiene su
etiqueta `vX.Y.Z` y su página en [Releases](../../releases).

## [1.0.0] - 2026-10-18

Primera versión como **Clipvault**. Antes la app se llamaba Omnissiah (hasta la 1.2.0); esta versión
parte de ese código.

### Cambiado
- Nuevo nombre y nuevo logo: la plumilla del copista sobre la línea roja de la rúbrica.
- Skin «Copista»: la app es un libro abierto, con la nueva descarga a la izquierda y la bóveda (la cola) a la derecha.
- Ventana nativa de Windows (WebView2) en lugar de una ventana de Edge: el icono de la barra de tareas se ve nítido en cualquier escala.
- Se instala en `C:\Program Files\Clipvault` y guarda las descargas en `Videos\Clipvault`.

### Añadido
- **Clip** (opcional, apagado por defecto): una línea de tiempo que se adapta a la duración del video pegado para descargar solo un tramo. Se eligen Desde y Hasta arrastrando los marcadores, con el teclado o escribiendo el tiempo (90, 1:30 o 1:02:15). El archivo se corta exacto y se llama «Título (clip 0m05s-0m12s)».
- Campo de enlace único: al pulsar Descargar el enlace pasa a la bóveda y el campo queda vacío para el siguiente.
- Forma parte de la suite [Codex](https://github.com/MiguelBigott/codex), que puede instalarla, actualizarla y abrirla.

[1.0.0]: https://github.com/MiguelBigott/clipvault/releases/tag/v1.0.0
