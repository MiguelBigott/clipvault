Unicode true
SetCompressor /SOLID lzma
!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "x64.nsh"
!include "LogicLib.nsh"

!define APP "Omnissiah"
!define VERSION "1.0.0"
!define REGAPP "Software\${APP}"
!define UNINST "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP}"

Name "${APP}"
OutFile "..\dist\Omnissiah-Setup-${VERSION}.exe"
; Instalación para todo el equipo en C:\Program Files\Omnissiah.
InstallDir "$PROGRAMFILES64\${APP}"
InstallDirRegKey HKLM "${REGAPP}" "InstallDir"
RequestExecutionLevel admin
BrandingText "${APP} ${VERSION}"
VIProductVersion "1.0.0.0"
VIAddVersionKey /LANG=1034 "ProductName" "${APP}"
VIAddVersionKey /LANG=1034 "FileDescription" "Instalador de ${APP}"
VIAddVersionKey /LANG=1034 "FileVersion" "${VERSION}"
VIAddVersionKey /LANG=1034 "ProductVersion" "${VERSION}"
VIAddVersionKey /LANG=1034 "LegalCopyright" "${APP}"

!define MUI_ICON "app.ico"
!define MUI_UNICON "app.ico"
!define MUI_WELCOMEFINISHPAGE_BITMAP "lateral.bmp"
!define MUI_UNWELCOMEFINISHPAGE_BITMAP "lateral.bmp"
!define MUI_ABORTWARNING

!define MUI_WELCOMEPAGE_TITLE "Instalar ${APP}"
!define MUI_WELCOMEPAGE_TEXT "Descargador de video y audio.$\r$\n$\r$\nSe instala en Archivos de programa. Tus descargas se guardarán en Videos\Descargas Omnissiah (puedes cambiarlo desde la app).$\r$\n$\r$\nAlabado sea el Omnissiah."
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
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

; El instalador corre como administrador: abrimos la app a través del Explorador
; para que arranque con los permisos normales del usuario.
Function AbrirApp
  Exec '"$WINDIR\explorer.exe" "$INSTDIR\Omnissiah.exe"'
FunctionEnd

; En el escritorio del usuario (no en el público), para que pueda borrarlo sin permisos.
Function CrearAccesoEscritorio
  SetShellVarContext current
  CreateShortcut "$DESKTOP\${APP}.lnk" "$INSTDIR\Omnissiah.exe"
  SetShellVarContext all
FunctionEnd

; La 1.0.0 se instalaba solo para el usuario en %LOCALAPPDATA%\Programs.
; Quitamos esa copia sin tocar la configuración ni los componentes descargados.
Function QuitarVersionUsuario
  ReadRegStr $0 HKCU "${REGAPP}" "InstallDir"
  ${If} $0 == ""
    Return
  ${EndIf}
  ${If} $0 == $INSTDIR
    Return
  ${EndIf}
  Delete "$0\Omnissiah.exe"
  Delete "$0\Desinstalar.exe"
  RMDir "$0"
  SetShellVarContext current
  Delete "$SMPROGRAMS\${APP}.lnk"
  Delete "$DESKTOP\${APP}.lnk"
  SetShellVarContext all
  DeleteRegKey HKCU "${UNINST}"
  DeleteRegKey HKCU "${REGAPP}"
FunctionEnd

Section "Instalar"
  ; Cierra la app si estaba abierta para poder reemplazarla.
  nsExec::Exec 'taskkill /IM Omnissiah.exe /F'
  Sleep 400
  Call QuitarVersionUsuario

  SetOutPath "$INSTDIR"
  File "..\dist\Omnissiah.exe"
  WriteUninstaller "$INSTDIR\Desinstalar.exe"

  CreateShortcut "$SMPROGRAMS\${APP}.lnk" "$INSTDIR\Omnissiah.exe"

  WriteRegStr HKLM "${REGAPP}" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "${UNINST}" "DisplayName" "${APP}"
  WriteRegStr HKLM "${UNINST}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "${UNINST}" "Publisher" "${APP}"
  WriteRegStr HKLM "${UNINST}" "DisplayIcon" "$INSTDIR\Omnissiah.exe"
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
  nsExec::Exec 'taskkill /IM Omnissiah.exe /F'
  Sleep 400
  Delete "$INSTDIR\Omnissiah.exe"
  Delete "$INSTDIR\Desinstalar.exe"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\${APP}.lnk"
  DeleteRegKey HKLM "${UNINST}"
  DeleteRegKey HKLM "${REGAPP}"
  SetShellVarContext current
  Delete "$DESKTOP\${APP}.lnk"
  ; Borra la configuración y los componentes descargados; tus videos no se tocan.
  RMDir /r "$LOCALAPPDATA\${APP}"
SectionEnd
