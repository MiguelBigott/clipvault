package main

// Textos e idioma. Los textos de la interfaz viven en web/i18n.js: Go manda una clave con sus
// datos y la interfaz la muestra en el idioma de Codex (el que se eligió en sus ajustes), o en
// el del PC si Codex está en automático o no está instalado.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/sys/windows"
)

// Mensaje es un texto para la interfaz: una clave de web/i18n.js, sus datos y, si lo hay, el
// error de fondo (Causa), que se traduce dentro de la frase. También sirve como error.
type Mensaje struct {
	Clave string            `json:"clave"`
	Datos map[string]string `json:"datos,omitempty"`
	Causa *Mensaje          `json:"causa,omitempty"`
}

func (m *Mensaje) Error() string { return m.Clave }

// msj arma un Mensaje; los datos van en pares nombre, valor.
func msj(clave string, datos ...string) *Mensaje {
	m := &Mensaje{Clave: clave}
	if len(datos) > 0 {
		m.Datos = map[string]string{}
		for i := 0; i+1 < len(datos); i += 2 {
			m.Datos[datos[i]] = datos[i+1]
		}
	}
	return m
}

// conCausa agrega el error de fondo, que se muestra traducido dentro del mensaje.
func (m *Mensaje) conCausa(err error) *Mensaje {
	m.Causa = comoMensaje(err)
	return m
}

// comoMensaje convierte cualquier error en un Mensaje: los que no vienen del diccionario
// (errores del sistema) se muestran tal cual.
func comoMensaje(err error) *Mensaje {
	var m *Mensaje
	if errors.As(err, &m) {
		return m
	}
	return msj("errorDetalle", "detalle", err.Error())
}

// idiomas que sabe mostrar la app: los mismos de Codex.
var idiomas = []string{"es", "en", "fr", "de", "ru", "pt", "zh", "ja"}

// idiomaSistema es el primero de los idiomas de Windows que la app sabe mostrar; si ninguno, inglés.
func idiomaSistema() string {
	lista, _ := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)
	for _, l := range lista {
		if base := strings.ToLower(strings.SplitN(l, "-", 2)[0]); slices.Contains(idiomas, base) {
			return base
		}
	}
	return "en"
}

// idiomaApp es el idioma elegido en Codex (%LOCALAPPDATA%\Codex\ajustes.json); en automático o
// sin Codex, el del PC. Se lee en cada consulta: si cambia en Codex, la app cambia sola.
func idiomaApp() string {
	if b, err := os.ReadFile(filepath.Join(os.Getenv("LOCALAPPDATA"), "Codex", "ajustes.json")); err == nil {
		var a struct {
			Idioma string `json:"idioma"`
		}
		if json.Unmarshal(b, &a) == nil && slices.Contains(idiomas, a.Idioma) {
			return a.Idioma
		}
	}
	return idiomaSistema()
}

// Textos que Go muestra por su cuenta, fuera de la interfaz.
type textosNativos struct{ FaltaWebView, ElegirCarpeta string }

var nativos = map[string]textosNativos{
	"es": {"ClipVault necesita Microsoft Edge WebView2, que viene con Windows 11. Instálalo desde https://developer.microsoft.com/microsoft-edge/webview2/ y vuelve a abrir ClipVault.", "Elige dónde guardar las descargas"},
	"en": {"ClipVault needs Microsoft Edge WebView2, which comes with Windows 11. Install it from https://developer.microsoft.com/microsoft-edge/webview2/ and open ClipVault again.", "Choose where to save downloads"},
	"fr": {"ClipVault a besoin de Microsoft Edge WebView2, inclus dans Windows 11. Installez-le depuis https://developer.microsoft.com/microsoft-edge/webview2/ puis rouvrez ClipVault.", "Choisissez où enregistrer les téléchargements"},
	"de": {"ClipVault benötigt Microsoft Edge WebView2, das in Windows 11 enthalten ist. Installiere es von https://developer.microsoft.com/microsoft-edge/webview2/ und öffne ClipVault erneut.", "Wähle, wo die Downloads gespeichert werden"},
	"ru": {"Для работы ClipVault нужен Microsoft Edge WebView2, который входит в Windows 11. Установите его с https://developer.microsoft.com/microsoft-edge/webview2/ и снова откройте ClipVault.", "Выберите, куда сохранять загрузки"},
	"pt": {"O ClipVault precisa do Microsoft Edge WebView2, que vem com o Windows 11. Instale-o em https://developer.microsoft.com/microsoft-edge/webview2/ e abra o ClipVault novamente.", "Escolha onde salvar os downloads"},
	"zh": {"ClipVault 需要 Microsoft Edge WebView2（Windows 11 已自带）。请从 https://developer.microsoft.com/microsoft-edge/webview2/ 安装后重新打开 ClipVault。", "选择下载内容的保存位置"},
	"ja": {"ClipVault には Microsoft Edge WebView2 が必要です（Windows 11 に付属しています）。https://developer.microsoft.com/microsoft-edge/webview2/ からインストールして、ClipVault をもう一度開いてください。", "ダウンロードの保存先を選択"},
}

func textosDe(idioma string) textosNativos {
	if t, ok := nativos[idioma]; ok {
		return t
	}
	return nativos["en"]
}
