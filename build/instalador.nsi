Unicode true
SetCompressor /SOLID lzma
!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "x64.nsh"
!include "LogicLib.nsh"

!ifndef VERSION
  !define VERSION "0.0.0"
!endif
!define APP "Clipvault"
!define EXE "Clipvault.exe"
!define REGAPP "Software\${APP}"
!define UNINST "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP}"

Name "${APP}"
OutFile "..\dist\Clipvault-Setup-${VERSION}.exe"
; Por defecto en C:\Program Files\Clipvault. Codex lo instala en C:\Program Files\Codex\Clipvault (/D=).
InstallDir "$PROGRAMFILES64\${APP}"
RequestExecutionLevel admin
; Nítido con la escala de pantalla de Windows (125 %, 150 %...).
ManifestDPIAware true
BrandingText "${APP} ${VERSION}"
VIProductVersion "${VERSION}.0"
VIAddVersionKey /LANG=1034 "ProductName" "${APP}"
VIAddVersionKey /LANG=1034 "FileDescription" "Instalador de ${APP}"
VIAddVersionKey /LANG=1034 "FileVersion" "${VERSION}"
VIAddVersionKey /LANG=1034 "ProductVersion" "${VERSION}"
VIAddVersionKey /LANG=1034 "LegalCopyright" "${APP}"

!define MUI_ICON "app.ico"
!define MUI_UNICON "app.ico"
!define MUI_WELCOMEFINISHPAGE_BITMAP "lateral.bmp"
!define MUI_WELCOMEFINISHPAGE_BITMAP_STRETCH AspectFitHeight
!define MUI_UNWELCOMEFINISHPAGE_BITMAP "lateral.bmp"
!define MUI_UNWELCOMEFINISHPAGE_BITMAP_STRETCH AspectFitHeight
!define MUI_ABORTWARNING

!define MUI_WELCOMEPAGE_TITLE "Instalar ${APP}"
!define MUI_WELCOMEPAGE_TEXT "Descargador de video y audio.$\r$\n$\r$\nSe instala con todo lo necesario para descargar. Tus descargas se guardarán en Videos\Clipvault (puedes cambiarlo desde la app)."
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES

; Casillas finales: abrir la app y crear el acceso directo del escritorio.
!define MUI_FINISHPAGE_RUN
!define MUI_FINISHPAGE_RUN_TEXT "Abrir ${APP}"
!define MUI_FINISHPAGE_RUN_FUNCTION AbrirApp
!define MUI_FINISHPAGE_SHOWREADME
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Crear acceso directo en el escritorio"
!define MUI_FINISHPAGE_SHOWREADME_FUNCTION CrearAccesoEscritorio
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "Spanish"

Function .onInit
  ${If} ${RunningX64}
    SetRegView 64
  ${EndIf}
  SetShellVarContext all
FunctionEnd

Function un.onInit
  ${If} ${RunningX64}
    SetRegView 64
  ${EndIf}
  SetShellVarContext all
FunctionEnd

; El instalador corre como administrador: la app se abre a través del Explorador
; para que arranque con los permisos normales del usuario.
Function AbrirApp
  Exec '"$WINDIR\explorer.exe" "$INSTDIR\${EXE}"'
FunctionEnd

; En el escritorio del usuario (no en el público), para que pueda borrarlo sin permisos.
Function CrearAccesoEscritorio
  SetShellVarContext current
  CreateShortcut "$DESKTOP\${APP}.lnk" "$INSTDIR\${EXE}"
  SetShellVarContext all
FunctionEnd

Section "Instalar"
  ; Cierra la app si estaba abierta para poder reemplazarla.
  nsExec::Exec 'taskkill /IM "${EXE}" /F'
  Sleep 500

  SetOutPath "$INSTDIR"
  File "..\dist\${EXE}"

  ; Componentes de descarga (motor, FFmpeg y Deno). Los usuarios pueden modificarlos para
  ; que la app actualice el motor sin pedir permisos de administrador.
  SetOutPath "$INSTDIR\componentes"
  File "..\dist\componentes\*.*"
  nsExec::Exec 'icacls "$INSTDIR\componentes" /grant *S-1-5-32-545:(OI)(CI)M /T /Q'
  SetOutPath "$INSTDIR"

  WriteUninstaller "$INSTDIR\Desinstalar.exe"
  CreateShortcut "$SMPROGRAMS\${APP}.lnk" "$INSTDIR\${EXE}"

  WriteRegStr HKLM "${REGAPP}" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "${UNINST}" "DisplayName" "${APP}"
  WriteRegStr HKLM "${UNINST}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "${UNINST}" "Publisher" "${APP}"
  WriteRegStr HKLM "${UNINST}" "DisplayIcon" "$INSTDIR\${EXE}"
  WriteRegStr HKLM "${UNINST}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKLM "${UNINST}" "UninstallString" '"$INSTDIR\Desinstalar.exe"'
  WriteRegStr HKLM "${UNINST}" "QuietUninstallString" '"$INSTDIR\Desinstalar.exe" /S'
  WriteRegDWORD HKLM "${UNINST}" "NoModify" 1
  WriteRegDWORD HKLM "${UNINST}" "NoRepair" 1
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKLM "${UNINST}" "EstimatedSize" "$0"
SectionEnd

Section "Uninstall"
  nsExec::Exec 'taskkill /IM "${EXE}" /F'
  Sleep 500
  Delete "$INSTDIR\${EXE}"
  Delete "$INSTDIR\Desinstalar.exe"
  RMDir /r "$INSTDIR\componentes"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\${APP}.lnk"
  DeleteRegKey HKLM "${UNINST}"
  DeleteRegKey HKLM "${REGAPP}"
  SetShellVarContext current
  Delete "$DESKTOP\${APP}.lnk"
  ; Borra la configuración; tus videos no se tocan.
  RMDir /r "$LOCALAPPDATA\${APP}"
SectionEnd
