//go:build !windows

package main

// Versión mínima para probar la app en Linux durante el desarrollo.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func exe(nombre string) string { return nombre }

func comando(ctx context.Context, nombre string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, nombre, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}
	return cmd
}

func alCancelar(ctx context.Context, cmd *exec.Cmd) {}

func carpetaDatos() string {
	if d := os.Getenv("OMNISSIAH_DATOS"); d != "" {
		return d
	}
	d, _ := os.UserConfigDir()
	return filepath.Join(d, "Omnissiah")
}

func carpetaDescargasPorDefecto() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Videos", "Descargas Omnissiah")
}

func carpetaDescargasAntigua() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Downloads", "Omnissiah")
}

func abrirVentana(url, dataDir string, esperar bool) bool { return false }

func elegirCarpeta(titulo, inicial string) (string, error) {
	return os.Getenv("OMNISSIAH_CARPETA_PRUEBA"), nil
}

func abrirCarpeta(ruta string)   {}
func mostrarArchivo(ruta string) {}
