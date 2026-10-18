"""Genera el logo de Omnissiah.

    python build/generar_logo.py                 # genera los iconos con la variante por defecto
    python build/generar_logo.py --vista rojo X  # solo renderiza una vista previa en X.png

1. Escribe build/logo.svg (la fuente vectorial).
2. Lo renderiza con Microsoft Edge en modo headless.
3. Produce build/icono512.png, build/app.ico, web/icono.ico, build/logo.png
   (sin fondo, para el README) y build/lateral.bmp (imagen lateral del instalador).
Requiere Pillow (pip install pillow) y Microsoft Edge.
"""
import math
import os
import subprocess
import sys
import tempfile

from PIL import Image

BUILD = os.path.dirname(os.path.abspath(__file__))
RAIZ = os.path.dirname(BUILD)
EDGE = r"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe"
CX = CY = 256
VARIANTE = "acero"  # color del engranaje: "acero" o "rojo"


def poligono(pts):
    return "M " + " L ".join(f"{x:.1f} {y:.1f}" for x, y in pts) + " Z"


def engranaje(cx, cy, dientes, r_cuerpo, perfil, arco=0):
    """Contorno de un engranaje: `perfil` son pares (fracción del paso, radio)."""
    pts = []
    paso = 2 * math.pi / dientes
    for i in range(dientes):
        a0 = paso * i - math.pi / 2
        for frac, r in perfil:
            a = a0 + frac * paso
            pts.append((cx + r * math.cos(a), cy + r * math.sin(a)))
        for k in range(1, arco):
            a = a0 + (0.5 + 0.5 * k / arco) * paso
            pts.append((cx + r_cuerpo * math.cos(a), cy + r_cuerpo * math.sin(a)))
    return poligono(pts)


def circulo(cx, cy, r):
    return f"M {cx} {cy - r} A {r} {r} 0 1 0 {cx} {cy + r} A {r} {r} 0 1 0 {cx} {cy - r} Z"


R_DIENTE, R_CUERPO, R_INTERIOR = 216, 186, 152
RUEDA = engranaje(CX, CY, 12, R_CUERPO,
                  ((0, R_CUERPO), (0.12, R_DIENTE), (0.38, R_DIENTE), (0.5, R_CUERPO)), arco=6)

# ---- Capucha y su ribete dentado -------------------------------------------
CAPUCHA = ("M 116 414 C 122 350 128 290 134 230 C 142 134 198 94 256 94 "
           "C 314 94 370 134 378 230 C 384 290 390 350 396 414 Q 256 434 116 414 Z")
ABERTURA_R = 92
ABERTURA_CY = 250
ABERTURA_BORDE = (f"M {CX - ABERTURA_R} 398 L {CX - ABERTURA_R} {ABERTURA_CY} "
                  f"A {ABERTURA_R} {ABERTURA_R} 0 0 1 {CX + ABERTURA_R} {ABERTURA_CY} "
                  f"L {CX + ABERTURA_R} 398")
ABERTURA = ABERTURA_BORDE + " Z"


def dientes_ribete():
    """Almenas blancas del ribete, apuntando hacia la tela roja."""
    t = 16
    banda = 6  # mitad del grosor del ribete
    cuadros = []
    n = 6  # arco superior
    for i in range(n + 1):
        a = math.pi + math.pi * i / n
        r = ABERTURA_R + banda + t / 2 - 2
        x, y = CX + r * math.cos(a), ABERTURA_CY + r * math.sin(a)
        ang = math.degrees(a) + 90
        cuadros.append(f'<rect x="{x - t / 2:.1f}" y="{y - t / 2:.1f}" width="{t}" height="{t}" '
                       f'transform="rotate({ang:.1f} {x:.1f} {y:.1f})"/>')
    y = ABERTURA_CY + 22  # laterales
    while y < 350:
        for x in (CX - ABERTURA_R - banda - t + 2, CX + ABERTURA_R + banda - 2):
            cuadros.append(f'<rect x="{x:.1f}" y="{y:.1f}" width="{t}" height="{t}"/>')
        y += 32
    return "".join(cuadros)


PLIEGUES = """
      <path d="M 196 118 C 186 150 180 184 178 214"/>
      <path d="M 232 102 C 226 126 226 146 230 162"/>
      <path d="M 284 102 C 290 124 290 144 286 162"/>
      <path d="M 318 118 C 330 150 336 184 336 214"/>
      <path d="M 146 250 C 142 300 138 350 134 410"/>
      <path d="M 366 250 C 370 300 374 350 378 410"/>
"""

ROSTRO = ("M 180 230 C 182 196 214 180 256 180 C 298 180 330 196 332 230 "
          "L 326 302 C 320 334 294 350 256 352 C 218 350 192 334 186 302 Z")


# Tres lentes pequeñas en triángulo equilátero (vértice arriba)
TRI_X, TRI_Y, TRI_R = 293, 266, 22
TRIANGULO = [(TRI_X + TRI_R * math.cos(math.radians(a)), TRI_Y - TRI_R * math.sin(math.radians(a)))
             for a in (90, 210, 330)]
LENTES_ARO = "".join(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="14.5"/>' for x, y in TRIANGULO)
LENTES = "".join(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="9.5"/>' for x, y in TRIANGULO)

# En la versión chica las lentes son más grandes y están más separadas.
TRIANGULO_CHICO = [(291 + 24 * math.cos(math.radians(a)), 268 - 24 * math.sin(math.radians(a)))
                   for a in (90, 210, 330)]
LENTES_ARO_CHICO = "".join(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="18"/>' for x, y in TRIANGULO_CHICO)
LENTES_CHICO = "".join(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="12"/>' for x, y in TRIANGULO_CHICO)

RESPIRADOR = """<rect x="164" y="352" width="184" height="64" rx="12" fill="#3f424c"/>
    <rect x="166" y="344" width="26" height="74" rx="9" fill="url(#cilindro)"/>
    <rect x="196" y="338" width="28" height="82" rx="9" fill="url(#cilindro)"/>
    <rect x="288" y="338" width="28" height="82" rx="9" fill="url(#cilindro)"/>
    <rect x="320" y="344" width="26" height="74" rx="9" fill="url(#cilindro)"/>
    <rect x="226" y="324" width="60" height="96" rx="10" fill="#2c2e36"/>
    <g fill="url(#costilla)">
      <rect x="230" y="330" width="52" height="16" rx="7"/><rect x="230" y="350" width="52" height="16" rx="7"/>
      <rect x="230" y="370" width="52" height="16" rx="7"/><rect x="230" y="390" width="52" height="16" rx="7"/>
    </g>"""
RESPIRADOR_CHICO = """<rect x="164" y="350" width="184" height="70" rx="14" fill="#3f424c"/>
    <rect x="170" y="340" width="48" height="84" rx="14" fill="url(#cilindro)"/>
    <rect x="294" y="340" width="48" height="84" rx="14" fill="url(#cilindro)"/>
    <rect x="224" y="322" width="64" height="100" rx="12" fill="#2c2e36"/>
    <g fill="url(#costilla)">
      <rect x="230" y="330" width="52" height="22" rx="9"/><rect x="230" y="358" width="52" height="22" rx="9"/>
      <rect x="230" y="386" width="52" height="22" rx="9"/>
    </g>"""

# La cabeza se dibuja a escala y se recorta con el círculo interior del engranaje.
R_RECORTE = R_INTERIOR - 9
CABEZA_ESCALA = 0.82
CABEZA_ARRIBA = 122  # dónde queda la punta de la capucha (y=94 en el dibujo original)


# Versión simplificada para tamaños de 48 px o menos (barra de tareas, título de ventana):
# engranaje de 8 dientes gruesos, disco y cabeza más grandes, sin piezas finas.
R_DIENTE_CHICO, R_CUERPO_CHICO, R_INTERIOR_CHICO = 240, 206, 164
RUEDA_CHICA = engranaje(CX, CY, 8, R_CUERPO_CHICO,
                        ((0, R_CUERPO_CHICO), (0.13, R_DIENTE_CHICO), (0.40, R_DIENTE_CHICO), (0.53, R_CUERPO_CHICO)),
                        arco=6)
TAMANOS_CHICOS = (16, 20, 24, 30, 32, 36, 40, 48)


def svg(variante, fondo=True, chico=False):
    """Logo completo, o con `chico=True` la versión simplificada para iconos pequeños."""
    detalle = not chico
    rueda_d, r_interior = (RUEDA_CHICA, R_INTERIOR_CHICO) if chico else (RUEDA, R_INTERIOR)
    r_recorte = r_interior - (14 if chico else 9)
    escala, arriba = (0.96, 98) if chico else (CABEZA_ESCALA, CABEZA_ARRIBA)
    sombra = '' if chico else 'filter="url(#sombra)"'
    if variante == "rojo":
        rueda = ('<stop offset="0" stop-color="#e0453a"/><stop offset=".5" stop-color="#b3161a"/>'
                 '<stop offset="1" stop-color="#6a0a0b"/>')
        brillo_rueda, disco, aro = "#ff8a70", "none", 0
    else:
        rueda = ('<stop offset="0" stop-color="#e4e6ec"/><stop offset=".5" stop-color="#a3a7b3"/>'
                 '<stop offset="1" stop-color="#5a5e6b"/>')
        brillo_rueda, disco, aro = "#ffffff", "url(#disco)", 1
    return f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" width="512" height="512">
  <!-- Omnissiah: tecnosacerdote encapuchado dentro del engranaje. Generado por build/generar_logo.py -->
  <defs>
    <radialGradient id="fondo" cx="50%" cy="42%" r="70%">
      <stop offset="0" stop-color="#5a0b08"/>
      <stop offset=".55" stop-color="#380505"/>
      <stop offset="1" stop-color="#1f0202"/>
    </radialGradient>
    <radialGradient id="disco" cx="50%" cy="40%" r="60%">
      <stop offset="0" stop-color="#241010"/>
      <stop offset="1" stop-color="#0c0405"/>
    </radialGradient>
    <linearGradient id="rueda" x1="0" y1="0" x2="0" y2="1">{rueda}</linearGradient>
    <linearGradient id="tela" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="#e8352c"/>
      <stop offset=".55" stop-color="#c0161b"/>
      <stop offset="1" stop-color="#7c0b0e"/>
    </linearGradient>
    <linearGradient id="tela-lado" x1="0" y1="0" x2="1" y2="0">
      <stop offset="0" stop-color="#3a0204" stop-opacity=".55"/>
      <stop offset=".25" stop-color="#3a0204" stop-opacity="0"/>
      <stop offset=".75" stop-color="#3a0204" stop-opacity="0"/>
      <stop offset="1" stop-color="#3a0204" stop-opacity=".55"/>
    </linearGradient>
    <radialGradient id="mascara" cx="42%" cy="30%" r="75%">
      <stop offset="0" stop-color="#9ea2ad"/>
      <stop offset=".55" stop-color="#666a76"/>
      <stop offset="1" stop-color="#34363f"/>
    </radialGradient>
    <linearGradient id="metal" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="#d9dce4"/>
      <stop offset=".45" stop-color="#959aa6"/>
      <stop offset="1" stop-color="#50535f"/>
    </linearGradient>
    <linearGradient id="cilindro" x1="0" y1="0" x2="1" y2="0">
      <stop offset="0" stop-color="#4a4d58"/>
      <stop offset=".4" stop-color="#c4c7d0"/>
      <stop offset=".7" stop-color="#8d919d"/>
      <stop offset="1" stop-color="#3f424c"/>
    </linearGradient>
    <linearGradient id="costilla" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="#a6aab5"/>
      <stop offset=".5" stop-color="#5c606c"/>
      <stop offset="1" stop-color="#2f3139"/>
    </linearGradient>
    <linearGradient id="tubo" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#c3c27a"/>
      <stop offset=".5" stop-color="#9a9850"/>
      <stop offset="1" stop-color="#5e5c2c"/>
    </linearGradient>
    <radialGradient id="lente" cx="40%" cy="35%" r="65%">
      <stop offset="0" stop-color="#ffffff"/>
      <stop offset=".3" stop-color="#a8e8ff"/>
      <stop offset=".75" stop-color="#3fa9ee"/>
      <stop offset="1" stop-color="#1b5f9e"/>
    </radialGradient>
    <radialGradient id="halo-azul"><stop offset="0" stop-color="#7fd4ff" stop-opacity=".6"/><stop offset="1" stop-color="#7fd4ff" stop-opacity="0"/></radialGradient>
    <radialGradient id="bola" cx="38%" cy="35%" r="65%">
      <stop offset="0" stop-color="#b9bcc6"/><stop offset="1" stop-color="#3b3d46"/>
    </radialGradient>
    <clipPath id="recorte-disco"><circle cx="{CX}" cy="{CY}" r="{r_recorte}"/></clipPath>
    <clipPath id="recorte-capucha"><path d="{CAPUCHA}"/></clipPath>
    <filter id="sombra" x="-20%" y="-20%" width="140%" height="140%">
      <feDropShadow dx="0" dy="8" stdDeviation="9" flood-color="#0a0000" flood-opacity=".65"/>
    </filter>
  </defs>

  {'<rect width="512" height="512" rx="92" fill="url(#fondo)"/>' if fondo else ''}

  <!-- Engranaje -->
  <path d="{rueda_d} {circulo(CX, CY, r_interior)}" fill="url(#rueda)" fill-rule="evenodd" {sombra}/>
  <circle cx="{CX}" cy="{CY}" r="{r_interior}" fill="{disco}"/>
  {f'<circle cx="{CX}" cy="{CY}" r="{R_CUERPO - 6}" fill="none" stroke="{brillo_rueda}" stroke-width="2" opacity=".3"/>' if detalle else ''}

  <g clip-path="url(#recorte-disco)">
  <g {sombra} transform="translate({CX} {arriba}) scale({escala}) translate({-CX} -94)">
    <!-- Capucha -->
    <path d="{CAPUCHA}" fill="url(#tela)"/>
    {f"""<path d="{CAPUCHA}" fill="url(#tela-lado)"/>
    <g clip-path="url(#recorte-capucha)" fill="none" stroke="#6e0a0d" stroke-width="4" stroke-linecap="round" opacity=".7">{PLIEGUES}</g>
    <path d="{CAPUCHA}" fill="none" stroke="#2a0203" stroke-width="4"/>""" if detalle else ''}
    <path d="{ABERTURA}" fill="#0b0405"/>
    <path d="{ABERTURA_BORDE}" fill="none" stroke="#f4f1ec" stroke-width="{12 if detalle else 22}"/>
    {f'<g fill="#f4f1ec">{dientes_ribete()}</g>' if detalle else ''}
{f"""
    <!-- Óptico lateral -->
    <rect x="140" y="222" width="11" height="40" rx="3" fill="#3b3d46" transform="rotate(-20 145 222)"/>
    <path d="{engranaje(142, 206, 6, 26, ((0, 26), (0.5, 26)))}" fill="url(#metal)" stroke="#2b2d34" stroke-width="3"/>
    <circle cx="142" cy="206" r="15" fill="#23252c"/>
    <circle cx="142" cy="206" r="11" fill="url(#lente)"/>

    <!-- Perno -->
    <g transform="rotate(35 352 146)">
      <rect x="347" y="114" width="10" height="8" rx="2" fill="#c9ccd4"/>
      <rect x="345" y="122" width="14" height="6" rx="2" fill="#7d818d"/>
      <rect x="347" y="128" width="10" height="6" rx="2" fill="#c9ccd4"/>
    </g>
    <circle cx="346" cy="152" r="15" fill="url(#bola)"/>""" if detalle else ''}

    <!-- Rostro -->
    <path d="{ROSTRO}" fill="url(#mascara)"/>
    <path d="M 212 184 L 300 184 L 294 214 L 256 242 L 218 214 Z" fill="#2c2e36"/>
    {'<rect x="240" y="182" width="32" height="15" rx="4" fill="#757985"/>' if detalle else ''}
    <!-- Ojo grande -->
    {'<circle cx="222" cy="264" r="46" fill="url(#halo-azul)"/>' if detalle else ''}
    <circle cx="222" cy="264" r="{33 if detalle else 38}" fill="#2a2c33"/>
    <circle cx="222" cy="264" r="{26 if detalle else 31}" fill="url(#lente)"/>
    <!-- Lentes pequeñas -->
    <g fill="#2a2c33">{LENTES_ARO if detalle else LENTES_ARO_CHICO}</g>
    <g fill="url(#lente)">{LENTES if detalle else LENTES_CHICO}</g>
{f"""
    <!-- Tubo -->
    <path d="M 372 214 C 418 228 422 318 348 380" fill="none" stroke="#3a3a1c" stroke-width="20" stroke-linecap="round"/>
    <path d="M 372 214 C 418 228 422 318 348 380" fill="none" stroke="url(#tubo)" stroke-width="15" stroke-linecap="round"/>
    <path d="M 382 222 C 410 240 412 304 362 364" fill="none" stroke="#e6e5b0" stroke-width="3" opacity=".5" stroke-linecap="round"/>""" if detalle else ''}

    <!-- Respirador -->
    {RESPIRADOR if detalle else RESPIRADOR_CHICO}
  </g>
  </g>
  <circle cx="{CX}" cy="{CY}" r="{r_interior - 7}" fill="none" stroke="#c0161b" stroke-width="{6 if detalle else 12}" opacity="{aro}"/>
  <circle cx="{CX}" cy="{CY}" r="{r_interior}" fill="none" stroke="#2a0203" stroke-width="{5 if detalle else 8}"/>
</svg>
'''


def renderizar(svg_path, png_path, lado):
    """Renderiza el SVG a `lado` px con fondo transparente usando Edge headless."""
    with tempfile.TemporaryDirectory() as perfil:
        html = os.path.join(perfil, "logo.html")
        with open(html, "w", encoding="utf-8") as f:
            f.write(f'<html><body style="margin:0;background:transparent">'
                    f'<img src="file:///{svg_path.replace(os.sep, "/")}" width="{lado}" height="{lado}" style="display:block">'
                    f'</body></html>')
        subprocess.run([EDGE, "--headless", "--disable-gpu", "--hide-scrollbars",
                        f"--user-data-dir={perfil}", "--default-background-color=00000000",
                        f"--window-size={lado},{lado}", f"--screenshot={png_path}",
                        "file:///" + html.replace(os.sep, "/")],
                       check=True, capture_output=True, timeout=60)


def generar_lateral(logo_png, destino):
    """Imagen lateral del instalador al doble de su tamaño (164x314) para pantallas con escala."""
    ancho, alto = 328, 628
    fuente = os.path.join(RAIZ, "web", "fonts", "cinzel-latin-700-normal.woff2").replace(os.sep, "/")
    with tempfile.TemporaryDirectory() as perfil:
        html = os.path.join(perfil, "lateral.html")
        png = os.path.join(perfil, "lateral.png")
        with open(html, "w", encoding="utf-8") as f:
            f.write(f'''<html><head><style>
@font-face {{ font-family: Cinzel; src: url("file:///{fuente}") format("woff2"); font-weight: 700; }}
html, body {{ margin: 0; width: {ancho}px; height: {alto}px; overflow: hidden; }}
body {{ background: radial-gradient(ellipse 90% 60% at 50% 34%, #5a0b08 0%, #380505 55%, #1a0202 100%);
       display: flex; flex-direction: column; align-items: center; font-family: Cinzel, serif; }}
img {{ width: 248px; height: 248px; margin-top: 112px; }}
h1 {{ margin: 34px 0 0; font-size: 38px; font-weight: 700; letter-spacing: 1px; color: #ebe0c8; }}
p {{ margin: 14px 0 0; font-size: 15px; font-weight: 700; letter-spacing: 1px; color: #a39484; }}
</style></head><body>
<img src="file:///{logo_png.replace(os.sep, "/")}"><h1>Omnissiah</h1><p>Alabado sea el Omnissiah</p>
</body></html>''')
        subprocess.run([EDGE, "--headless", "--disable-gpu", "--hide-scrollbars", "--allow-file-access-from-files",
                        f"--user-data-dir={perfil}", f"--window-size={ancho},{alto}", "--virtual-time-budget=3000",
                        f"--screenshot={png}", "file:///" + html.replace(os.sep, "/")],
                       check=True, capture_output=True, timeout=60)
        Image.open(png).convert("RGB").save(destino, format="BMP")


def vista(variante, destino):
    with tempfile.TemporaryDirectory() as tmp:
        ruta = os.path.join(tmp, "logo.svg")
        with open(ruta, "w", encoding="utf-8") as f:
            f.write(svg(variante))
        renderizar(ruta, destino, 1024)
        im = Image.open(destino).convert("RGBA")
        im.resize((512, 512), Image.LANCZOS).save(destino)


def main():
    if len(sys.argv) == 4 and sys.argv[1] == "--vista":
        vista(sys.argv[2], sys.argv[3])
        return
    svg_path = os.path.join(BUILD, "logo.svg")
    with open(svg_path, "w", encoding="utf-8", newline="\n") as f:
        f.write(svg(VARIANTE))

    with tempfile.TemporaryDirectory() as tmp:
        grande = os.path.join(tmp, "logo1024.png")
        renderizar(svg_path, grande, 1024)
        base = Image.open(grande).convert("RGBA")
        base.load()

        chico_svg = os.path.join(tmp, "logo-chico.svg")
        with open(chico_svg, "w", encoding="utf-8") as f:
            f.write(svg(VARIANTE, chico=True))
        chico_png = os.path.join(tmp, "logo-chico1024.png")
        renderizar(chico_svg, chico_png, 1024)
        chico = Image.open(chico_png).convert("RGBA")
        chico.load()

    base.resize((512, 512), Image.LANCZOS).save(os.path.join(BUILD, "icono512.png"))

    # Iconos: versión simplificada en los tamaños chicos que usa Windows (barra de tareas a
    # 100-200 % de escala) y el logo completo en los grandes.
    tam = list(TAMANOS_CHICOS) + [64, 128, 256]
    cuadros = [(chico if s in TAMANOS_CHICOS else base).resize((s, s), Image.LANCZOS) for s in tam]
    for destino in (os.path.join(BUILD, "app.ico"), os.path.join(RAIZ, "web", "icono.ico")):
        cuadros[-1].save(destino, format="ICO", sizes=[(s, s) for s in tam], append_images=cuadros[:-1])

    # Versión sin fondo para la cabecera del README
    with tempfile.TemporaryDirectory() as tmp:
        ruta = os.path.join(tmp, "logo.svg")
        with open(ruta, "w", encoding="utf-8") as f:
            f.write(svg(VARIANTE, fondo=False))
        grande = os.path.join(tmp, "logo1024.png")
        renderizar(ruta, grande, 1024)
        sin_fondo = Image.open(grande).convert("RGBA")
        sin_fondo.resize((256, 256), Image.LANCZOS).save(os.path.join(BUILD, "logo.png"))
        lateral_logo = os.path.join(tmp, "logo-lateral.png")
        sin_fondo.resize((496, 496), Image.LANCZOS).save(lateral_logo)
        generar_lateral(lateral_logo, os.path.join(BUILD, "lateral.bmp"))
    print("logo generado:", svg_path)


if __name__ == "__main__":
    main()
