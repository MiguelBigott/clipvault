"""Logo de Clipvault: la plumilla del copista apuntando hacia abajo (como una flecha de
descarga) sobre la línea roja de la rúbrica.

    python build/generar_logo.py

Escribe:
  web/logo.svg       logo para el encabezado de la app
  build/logo.png     logo para el README (256 px)
  build/app.ico      icono del .exe: versión reforzada en 16-32 px, normal desde 36 px
  build/lateral.bmp  imagen lateral del instalador (al doble de tamaño, para pantallas con escala)
Requiere Pillow y Microsoft Edge (renderiza el SVG).
"""
import os
import subprocess
import tempfile

from PIL import Image

BUILD = os.path.dirname(os.path.abspath(__file__))
RAIZ = os.path.dirname(BUILD)
EDGE = r"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe"

TINTA, BORDE, PERGAMINO, ORO, RUBRICA = "#16120e", "#3a3128", "#ece2cf", "#c9a45c", "#c8483a"


def svg(chico=False):
    """Logo en viewBox 512. `chico`: plumilla más grande y detalles más gruesos para 16-32 px."""
    if chico:
        pluma = "M256 70 C318 70 366 116 366 180 L366 262 L256 430 L146 262 L146 180 C146 116 194 70 256 70 Z"
        banda, grosor_banda = "M146 262 H366", 18
        agujero, r_agujero = (256, 206), 34
        ranura, grosor_ranura = "M256 240 V410", 20
        linea, grosor_linea = "M170 466 H342", 30
        borde = 0
    else:
        pluma = "M256 92 C306 92 344 128 344 180 L344 246 L256 404 L168 246 L168 180 C168 128 206 92 256 92 Z"
        banda, grosor_banda = "M168 246 H344", 12
        agujero, r_agujero = (256, 214), 24
        ranura, grosor_ranura = "M256 236 V390", 12
        linea, grosor_linea = "M184 432 H328", 22
        borde = 10
    return f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" width="512" height="512">
  <rect x="8" y="8" width="496" height="496" rx="112" fill="{TINTA}" stroke="{BORDE}" stroke-width="{borde}"/>
  <path d="{pluma}" fill="{PERGAMINO}"/>
  <path d="{banda}" stroke="{ORO}" stroke-width="{grosor_banda}"/>
  <circle cx="{agujero[0]}" cy="{agujero[1]}" r="{r_agujero}" fill="{TINTA}"/>
  <path d="{ranura}" stroke="{TINTA}" stroke-width="{grosor_ranura}" stroke-linecap="round"/>
  <path d="{linea}" stroke="{RUBRICA}" stroke-width="{grosor_linea}" stroke-linecap="round"/>
</svg>
'''


def renderizar(contenido_html, ancho, alto, destino):
    with tempfile.TemporaryDirectory() as perfil:
        html = os.path.join(perfil, "p.html")
        with open(html, "w", encoding="utf-8") as f:
            f.write(contenido_html)
        subprocess.run([EDGE, "--headless", "--disable-gpu", "--hide-scrollbars", "--allow-file-access-from-files",
                        f"--user-data-dir={perfil}", "--default-background-color=00000000",
                        f"--window-size={ancho},{alto}", "--virtual-time-budget=3000",
                        f"--screenshot={destino}", "file:///" + html.replace(os.sep, "/")],
                       check=True, capture_output=True, timeout=60)


def png_de_svg(texto_svg, lado, destino):
    with tempfile.TemporaryDirectory() as tmp:
        ruta = os.path.join(tmp, "logo.svg")
        with open(ruta, "w", encoding="utf-8") as f:
            f.write(texto_svg)
        renderizar(f'<html><body style="margin:0;background:transparent"><img src="file:///{ruta.replace(os.sep, "/")}" '
                   f'width="{lado}" height="{lado}" style="display:block"></body></html>', lado, lado, destino)
    im = Image.open(destino).convert("RGBA")
    im.load()
    return im


def lateral(logo_png, destino):
    fuente = os.path.join(RAIZ, "web", "fonts", "cormorant-garamond-latin-600-normal.woff2").replace(os.sep, "/")
    with tempfile.TemporaryDirectory() as tmp:
        png = os.path.join(tmp, "lateral.png")
        renderizar(f'''<html><head><style>
@font-face {{ font-family: Serif; src: url("file:///{fuente}"); font-weight: 600; }}
html, body {{ margin: 0; width: 328px; height: 628px; overflow: hidden; background: {TINTA}; }}
body {{ display: flex; flex-direction: column; align-items: center; color: {PERGAMINO}; }}
img {{ width: 200px; height: 200px; margin-top: 150px; }}
h1 {{ margin: 30px 0 0; font-family: Serif; font-weight: 600; font-size: 48px; }}
</style></head><body><img src="file:///{logo_png.replace(os.sep, "/")}"><h1>Clipvault</h1></body></html>''', 328, 628, png)
        Image.open(png).convert("RGB").save(destino, format="BMP")


def main():
    with open(os.path.join(RAIZ, "web", "logo.svg"), "w", encoding="utf-8", newline="\n") as f:
        f.write(svg())
    with tempfile.TemporaryDirectory() as tmp:
        normal = png_de_svg(svg(), 1024, os.path.join(tmp, "normal.png"))
        chico = png_de_svg(svg(chico=True), 1024, os.path.join(tmp, "chico.png"))
        normal.resize((256, 256), Image.LANCZOS).save(os.path.join(BUILD, "logo.png"))
        tam = [16, 20, 24, 30, 32, 36, 40, 48, 64, 128, 256]
        cuadros = [(chico if s <= 32 else normal).resize((s, s), Image.LANCZOS) for s in tam]
        cuadros[-1].save(os.path.join(BUILD, "app.ico"), format="ICO",
                         sizes=[(s, s) for s in tam], append_images=cuadros[:-1])
        logo_lat = os.path.join(tmp, "lat.png")
        normal.resize((400, 400), Image.LANCZOS).save(logo_lat)
        lateral(logo_lat, os.path.join(BUILD, "lateral.bmp"))
    print("logo generado")


if __name__ == "__main__":
    main()
