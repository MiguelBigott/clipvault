"use strict";

const $ = (s) => document.querySelector(s);

async function api(ruta, datos) {
  const opciones = { headers: { "X-Clipvault": "1" } };
  if (datos !== undefined) {
    opciones.method = "POST";
    opciones.headers["Content-Type"] = "application/json";
    opciones.body = JSON.stringify(datos);
  }
  const r = await fetch(ruta, opciones);
  const json = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(json.error || "Error inesperado");
  return json;
}

let aviso_t;
function avisar(texto, esError) {
  const el = $("#aviso");
  el.textContent = texto;
  el.classList.toggle("error", !!esError);
  el.hidden = false;
  clearTimeout(aviso_t);
  aviso_t = setTimeout(() => (el.hidden = true), esError ? 7000 : 4000);
}

/* ---------------- Enlaces y vista previa ---------------- */

const vistas = new Map(); // url -> {estado: cargando|ok|error, datos, error}

// Un solo enlace por descarga: el campo es de una línea.
function urlsDelTexto() {
  const p = $("#enlaces").value.trim().split(/\s+/)[0] || "";
  return /^https?:\/\/\S+\.\S+/i.test(p) ? [p] : [];
}

function duracion(seg) {
  if (!seg) return "";
  seg = Math.round(seg);
  const h = Math.floor(seg / 3600), m = Math.floor((seg % 3600) / 60), s = seg % 60;
  const dd = (n) => String(n).padStart(2, "0");
  return h ? `${h}:${dd(m)}:${dd(s)}` : `${m}:${dd(s)}`;
}

function describirVista(v) {
  const partes = [];
  if (v.lista) partes.push(`Lista de ${v.lista} videos`);
  if (v.canal) partes.push(v.canal);
  if (v.duracion) partes.push(duracion(v.duracion));
  if (v.altura) partes.push(`hasta ${v.altura}p`);
  return partes.join(", ");
}

function pintarVistas() {
  const urls = urlsDelTexto();
  const cont = $("#vistas");
  cont.replaceChildren();
  const mostrar = urls.slice(0, 3);
  for (const u of mostrar) {
    const v = vistas.get(u);
    const caja = document.createElement("div");
    caja.className = "vista";
    if (!v || v.estado === "cargando") {
      caja.classList.add("cargando");
      caja.textContent = "Consultando el enlace…";
    } else if (v.estado === "error") {
      caja.classList.add("error");
      const t = document.createElement("p");
      t.className = "t";
      t.textContent = v.error;
      caja.append(t);
    } else {
      const img = document.createElement("img");
      img.alt = "";
      if (v.datos.miniatura) img.src = v.datos.miniatura;
      const txt = document.createElement("div");
      const t = document.createElement("p");
      t.className = "t";
      t.textContent = v.datos.titulo || u;
      const s = document.createElement("p");
      s.className = "s";
      s.textContent = describirVista(v.datos);
      txt.append(t, s);
      caja.append(img, txt);
    }
    cont.append(caja);
  }
  if (urls.length > mostrar.length) {
    const mas = document.createElement("p");
    mas.className = "mas";
    const n = urls.length - mostrar.length;
    mas.textContent = `y ${n} enlace${n === 1 ? "" : "s"} más`;
    cont.append(mas);
  }
  const listos = urls.filter((u) => vistas.get(u)?.estado !== "error");
  const btn = $("#btn-agregar");
  btn.disabled = listos.length === 0;
}

async function consultar(url) {
  if (vistas.has(url)) return;
  vistas.set(url, { estado: "cargando" });
  pintarVistas();
  try {
    const datos = await api("/api/vista", { url });
    vistas.set(url, { estado: "ok", datos });
  } catch (e) {
    vistas.set(url, { estado: "error", error: e.message });
  }
  pintarVistas();
}

let enlaces_t;
$("#enlaces").addEventListener("input", () => {
  pintarVistas();
  clearTimeout(enlaces_t);
  enlaces_t = setTimeout(() => urlsDelTexto().slice(0, 12).forEach(consultar), 450);
});

$("#enlaces").addEventListener("keydown", (e) => {
  if (e.key === "Enter") {
    e.preventDefault();
    agregar();
  }
});

$("#btn-pegar").addEventListener("click", async () => {
  try {
    const t = await navigator.clipboard.readText();
    if (!t) return;
    const campo = $("#enlaces");
    campo.value = t.trim();
    campo.dispatchEvent(new Event("input"));
  } catch {
    avisar("No pude leer el portapapeles. Pega con Ctrl+V en el cuadro.", true);
    $("#enlaces").focus();

/* ---------------- Ventana (solo dentro de la app, no en un navegador) ---------------- */

if (typeof window.caArrastrar === "function") {
  $("#controles").hidden = false;
  const barra = $("#barra");
  const interactivo = "button, a, input, select, textarea";
  barra.addEventListener("mousedown", (ev) => {
    if (ev.button !== 0 || ev.detail > 1 || ev.target.closest(interactivo)) return;
    window.caArrastrar();
  });
  const actualizarMaximizada = async () => {
    const max = await window.caMaximizada();
    document.body.classList.toggle("maximizada", max);
    $("#btn-max").setAttribute("aria-label", max ? "Restaurar" : "Maximizar");
  };
  barra.addEventListener("dblclick", (ev) => {
    if (!ev.target.closest(interactivo)) window.caMaximizar().then(actualizarMaximizada);
  });
  document.querySelectorAll(".redim").forEach((r) => r.addEventListener("mousedown", (ev) => {
    if (ev.button === 0) window.caRedimensionar(r.dataset.borde);
  }));
  $("#btn-min").addEventListener("click", () => window.caMinimizar());
  $("#btn-max").addEventListener("click", () => window.caMaximizar().then(actualizarMaximizada));
  $("#btn-cerrar").addEventListener("click", () => window.caCerrar());
  window.addEventListener("resize", actualizarMaximizada);
  actualizarMaximizada();
} else {
  document.querySelectorAll(".redim").forEach((r) => r.remove());
}
  }
});

function tipoElegido() {
  return document.querySelector('input[name="tipo"]:checked').value;
}

function actualizarCalidad() {
  $("#calidad-caja").hidden = tipoElegido() === "audio";
}
document.querySelectorAll('input[name="tipo"]').forEach((r) => r.addEventListener("change", actualizarCalidad));

async function agregar() {
  const urls = urlsDelTexto().filter((u) => vistas.get(u)?.estado !== "error");
  if (!urls.length) return;
  const datosVistas = {};
  for (const u of urls) {
    const v = vistas.get(u);
    if (v?.estado === "ok") datosVistas[u] = v.datos;
  }
  try {
    const r = await api("/api/agregar", {
      urls, tipo: tipoElegido(), calidad: $("#calidad").value, vistas: datosVistas,
    });
    $("#enlaces").value = "";
    pintarVistas();
    avisar("Añadido a la bóveda.");
    $("#enlaces").focus();
    refrescar();
  } catch (e) {
    avisar(e.message, true);
  }
}
$("#btn-agregar").addEventListener("click", agregar);

/* ---------------- Cola ---------------- */

const etiquetaCalidad = {
  max: "Máxima", 1080: "1080p", 720: "720p", 480: "480p", compat: "H.264",
};

function detalleItem(it) {
  const tipo = it.tipo === "audio" ? "MP3" : `MP4 ${etiquetaCalidad[it.calidad] || ""}`.trim();
  switch (it.estado) {
    case "cola": return `${tipo}, en espera`;
    case "descargando": {
      const p = [it.fase];
      if (it.lista && it.indice) p.push(`video ${it.indice} de ${it.lista}`);
      if (it.progreso) p.push(`${Math.floor(it.progreso * 100)} %`);
      if (it.tamano) p.push(it.tamano);
      if (it.velocidad) p.push(it.velocidad);
      if (it.restante) p.push(`quedan ${it.restante}`);
      return p.join(", ");
    }
    case "procesando": return `${it.fase}…`;
    case "listo": return `${tipo}, completado`;
    case "cancelado": return "Cancelada";
    case "error": return it.error || "La descarga falló";
  }
  return "";
}

function boton(texto, fn) {
  const b = document.createElement("button");
  b.type = "button";
  b.className = "btn btn-chico";
  b.textContent = texto;
  b.addEventListener("click", fn);
  return b;
}

const filas = new Map(); // id -> li

function pintarCola(cola) {
  const lista = $("#lista");
  const vivos = new Set();
  cola.forEach((it, i) => {
    vivos.add(it.id);
    let li = filas.get(it.id);
    if (!li) {
      li = $("#tpl-item").content.firstElementChild.cloneNode(true);
      filas.set(it.id, li);
    }
    if (lista.children[i] !== li) lista.insertBefore(li, lista.children[i] || null);

    li.className = `item ${it.estado}`;
    const img = li.querySelector(".mini img");
    const src = it.miniatura || "";
    if ((img.getAttribute("src") || "") !== src) {
      if (src) img.src = src;
      else img.removeAttribute("src");
    }
    li.querySelector(".nombre").textContent = it.titulo || it.url;
    li.querySelector(".nombre").title = it.titulo || it.url;
    li.querySelector(".detalle").textContent = detalleItem(it);
    li.querySelector(".medidor span").style.width = `${Math.round((it.progreso || 0) * 100)}%`;

    const firma = `${it.estado}|${it.archivo ? 1 : 0}`;
    const acc = li.querySelector(".acciones");
    if (acc.dataset.firma !== firma) {
      acc.dataset.firma = firma;
      acc.replaceChildren();
      if (["cola", "descargando", "procesando"].includes(it.estado)) {
        acc.append(boton("Cancelar", () => api("/api/cancelar", { id: it.id }).then(refrescar)));
      } else if (it.estado === "error" || it.estado === "cancelado") {
        acc.append(boton("Reintentar", () => api("/api/reintentar", { id: it.id }).then(refrescar)));
      } else if (it.estado === "listo") {
        acc.append(boton(it.archivo ? "Mostrar archivo" : "Abrir carpeta",
          () => api("/api/carpeta/abrir", { archivo: it.archivo || "" })));
      }
    }
  });
  for (const [id, li] of filas) {
    if (!vivos.has(id)) { li.remove(); filas.delete(id); }
  }
  $("#vacio").hidden = cola.length > 0;
  $("#btn-limpiar").hidden = !cola.some((it) => ["listo", "error", "cancelado"].includes(it.estado));
  const pendientes = cola.filter((it) => ["cola", "descargando", "procesando"].includes(it.estado)).length;
  document.title = pendientes ? `Clipvault (${pendientes})` : "Clipvault";
}

$("#btn-limpiar").addEventListener("click", () => api("/api/limpiar", {}).then(refrescar));

/* ---------------- Componentes y carpeta ---------------- */

let ultimoMensaje = "";

function pintarComponentes(c) {
  const vy = $("#ver-ytdlp"), vf = $("#ver-ffmpeg");
  vy.textContent = c.ytdlp ? (c.versionYtdlp || "instalado") : "no encontrado";
  vy.classList.toggle("falta", !c.ytdlp);
  vy.title = c.ytdlp || "";
  vf.textContent = c.ffmpeg ? "instalado" : "no encontrado";
  vf.classList.toggle("falta", !c.ffmpeg);
  vf.title = c.ffmpeg || "";

  const ocupado = !!c.tarea;
  $("#tarea").hidden = !ocupado;
  if (ocupado) {
    $("#tarea-texto").textContent = c.tareaTexto + "…";
    $("#tarea-barra").style.width = `${Math.round(c.tareaProgreso * 100)}%`;
  }
  $("#motor-faltan").hidden = c.listo || ocupado;
  if (!c.listo) {
    const faltan = [!c.ytdlp && "el motor de descarga", !c.ffmpeg && "FFmpeg"].filter(Boolean);
    const peso = c.ffmpeg ? "unos 20 MB" : "unos 100 MB";
    $("#faltan-texto").textContent =
      `${faltan.length > 1 ? "Faltan" : "Falta"} ${faltan.join(" y ")} para descargar. Usa «Reparar componentes» para volver a bajar ${faltan.length > 1 ? "los dos" : "lo que falta"} a la carpeta de Clipvault (${peso}).`;
  }
  $("#motor-listo").hidden = !c.ytdlp || ocupado;

  if (c.mensaje && c.mensaje !== ultimoMensaje) avisar(c.mensaje, c.mensajeError);
  ultimoMensaje = c.mensaje;
}

function pintarConfig(cfg, primera) {
  const r = $("#ruta-destino");
  r.textContent = cfg.carpeta;
  r.title = cfg.carpeta;
  if (primera) {
    const radio = document.querySelector(`input[name="tipo"][value="${cfg.tipo}"]`);
    if (radio) radio.checked = true;
    if (cfg.calidad && [...$("#calidad").options].some((o) => o.value === cfg.calidad)) {
      $("#calidad").value = cfg.calidad;
    }
    actualizarCalidad();
  }
}

$("#btn-cambiar-carpeta").addEventListener("click", async (e) => {
  e.currentTarget.disabled = true;
  try {
    const r = await api("/api/carpeta/elegir", {});
    $("#ruta-destino").textContent = r.carpeta;
    if (r.cambiada) avisar("Las descargas se guardarán en la nueva carpeta.");
  } finally {
    e.currentTarget.disabled = false;
  }
});
$("#btn-abrir-carpeta").addEventListener("click", () => api("/api/carpeta/abrir", {}));
$("#btn-instalar").addEventListener("click", () => api("/api/componentes/instalar", {}).then(refrescar));
$("#btn-actualizar").addEventListener("click", () => api("/api/componentes/actualizar", {}).then(refrescar));

/* ---------------- Ciclo ---------------- */

let primera = true;
async function refrescar() {
  try {
    const e = await api("/api/estado");
    pintarConfig(e.config, primera);
    pintarComponentes(e.componentes);
    pintarCola(e.cola);
    if (primera) $("#version").textContent = `Clipvault ${e.version} · de la biblioteca Codex`;
    primera = false;
  } catch {
    /* el servidor puede estar cerrándose */
  }
}

refrescar();
setInterval(refrescar, 700);
const latir = () => api("/api/ping").catch(() => {});
latir();
setInterval(latir, 5000);
$("#enlaces").focus();

/* ---------------- Ventana (solo dentro de la app, no en un navegador) ---------------- */

if (typeof window.caArrastrar === "function") {
  $("#controles").hidden = false;
  const barra = $("#barra");
  const interactivo = "button, a, input, select, textarea";
  barra.addEventListener("mousedown", (ev) => {
    if (ev.button !== 0 || ev.detail > 1 || ev.target.closest(interactivo)) return;
    window.caArrastrar();
  });
  const actualizarMaximizada = async () => {
    const max = await window.caMaximizada();
    document.body.classList.toggle("maximizada", max);
    $("#btn-max").setAttribute("aria-label", max ? "Restaurar" : "Maximizar");
  };
  barra.addEventListener("dblclick", (ev) => {
    if (!ev.target.closest(interactivo)) window.caMaximizar().then(actualizarMaximizada);
  });
  document.querySelectorAll(".redim").forEach((r) => r.addEventListener("mousedown", (ev) => {
    if (ev.button === 0) window.caRedimensionar(r.dataset.borde);
  }));
  $("#btn-min").addEventListener("click", () => window.caMinimizar());
  $("#btn-max").addEventListener("click", () => window.caMaximizar().then(actualizarMaximizada));
  $("#btn-cerrar").addEventListener("click", () => window.caCerrar());
  window.addEventListener("resize", actualizarMaximizada);
  actualizarMaximizada();
} else {
  document.querySelectorAll(".redim").forEach((r) => r.remove());
}
