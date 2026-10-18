//go:build windows

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const crearSinVentana = 0x08000000 // CREATE_NO_WINDOW

func exe(nombre string) string { return nombre + ".exe" }

func ocultar(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: crearSinVentana}
}

// comando crea un proceso sin ventana de consola, con salida en UTF-8.
func comando(ctx context.Context, nombre string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, nombre, args...)
	ocultar(cmd)
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8", "PYTHONUTF8=1")
	if strings.EqualFold(filepath.Base(nombre), "yt-dlp.exe") {
		cmd.Dir = filepath.Dir(nombre)
	}
	// El contexto se cancela con alCancelar (mata todo el árbol de procesos).
	cmd.Cancel = func() error { return matarArbol(cmd) }
	return cmd
}

func matarArbol(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	k := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
	ocultar(k)
	return k.Run()
}

func alCancelar(ctx context.Context, cmd *exec.Cmd) {}

func carpetaDatos() string {
	if d := os.Getenv("LOCALAPPDATA"); d != "" {
		return filepath.Join(d, "Omnissiah")
	}
	d, _ := os.UserConfigDir()
	return filepath.Join(d, "Omnissiah")
}

func carpetaDescargasPorDefecto() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Downloads", "Omnissiah")
}

func navegadorApp() string {
	var rutas []string
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		base := os.Getenv(env)
		if base == "" {
			continue
		}
		rutas = append(rutas,
			filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(base, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
		)
	}
	for _, r := range rutas {
		if _, err := os.Stat(r); err == nil {
			return r
		}
	}
	return ""
}

// abrirVentana muestra la interfaz como una ventana de aplicación independiente.
// Si esperar es true, regresa cuando el proceso de la ventana termina y devuelve
// true; devuelve false si no pudo seguir el proceso de la ventana.
func abrirVentana(url, dataDir string, esperar bool) bool {
	nav := navegadorApp()
	if nav == "" {
		c := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		_ = c.Start()
		return false
	}
	perfil := filepath.Join(dataDir, "ventana")
	cmd := exec.Command(nav,
		"--app="+url,
		"--user-data-dir="+perfil,
		"--window-size=1180,820",
		"--no-first-run", "--no-default-browser-check",
		"--disable-features=Translate,msEdgeSidebarV2,msHubApps,msUndersideButton",
		"--disable-sync", "--hide-crash-restore-bubble",
		"--disable-background-timer-throttling", "--disable-renderer-backgrounding",
		"--disable-backgrounding-occluded-windows",
	)
	if err := cmd.Start(); err != nil {
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
		return false
	}
	if !esperar {
		return false
	}
	t0 := time.Now()
	_ = cmd.Wait()
	// Si terminó enseguida, el navegador entregó la ventana a otro proceso.
	return time.Since(t0) > 5*time.Second
}

func powershell(script string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-STA", "-ExecutionPolicy", "Bypass", "-Command", script)
	ocultar(cmd)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func comillasPS(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func elegirCarpeta(titulo, inicial string) (string, error) {
	script := `[Console]::OutputEncoding=[Text.Encoding]::UTF8
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.Application]::EnableVisualStyles()
$f = New-Object System.Windows.Forms.Form -Property @{TopMost=$true; ShowInTaskbar=$false; Width=0; Height=0; StartPosition='CenterScreen'}
$d = New-Object System.Windows.Forms.FolderBrowserDialog
$d.Description = ` + comillasPS(titulo) + `
$d.ShowNewFolderButton = $true
$d.SelectedPath = ` + comillasPS(inicial) + `
if ($d.ShowDialog($f) -eq 'OK') { $d.SelectedPath }`
	return powershell(script)
}

func abrirCarpeta(ruta string) {
	_ = exec.Command("explorer.exe", ruta).Start()
}

func mostrarArchivo(ruta string) {
	c := exec.Command("explorer.exe")
	c.SysProcAttr = &syscall.SysProcAttr{CmdLine: `explorer.exe /select,"` + ruta + `"`}
	_ = c.Start()
}
