# Compila ClipVault.exe y el instalador en dist\.
# Requiere: Go, NSIS (makensis) y go-winres (go install github.com/tc-hib/go-winres@latest).
#
# Los componentes de descarga se bajan de sus fuentes oficiales, se verifican con
# SHA-256 y se guardan en build\componentes (fuera de git). Para bajar versiones nuevas:
#   .\build\compilar.ps1 -ActualizarComponentes
param([switch]$ActualizarComponentes)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$raiz = Split-Path $PSScriptRoot -Parent
Set-Location $raiz

$version = (Select-String -Path main.go -Pattern 'const Version = "(.+)"').Matches[0].Groups[1].Value
$goExe = (Get-Command go -ErrorAction SilentlyContinue).Source
if (-not $goExe) { $goExe = 'C:\Program Files\Go\bin\go.exe' }
$winres = Join-Path (& $goExe env GOPATH) 'bin\go-winres.exe'
$makensis = 'C:\Program Files (x86)\NSIS\makensis.exe'

Write-Host "ClipVault $version"

# ---------- Componentes de descarga ----------
$motorUrl = 'https://github.com/yt-dlp/yt-dlp/releases/latest/download'
$ffmpegUrl = 'https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest'
# Versión "shared": ejecutables pequeños que comparten sus DLL (la mitad de tamaño que la estática).
$ffmpegZip = 'ffmpeg-master-latest-win64-gpl-shared.zip'
$cache = Join-Path $PSScriptRoot 'componentes'

function Get-SumaOficial($url, $archivo) {
    $contenido = (Invoke-WebRequest -UseBasicParsing $url).Content
    if ($contenido -is [byte[]]) { $contenido = [Text.Encoding]::UTF8.GetString($contenido) }
    $linea = $contenido -split "`n" |
        Where-Object { $_ -match "\s\*?$([regex]::Escape($archivo))\s*$" } | Select-Object -First 1
    if (-not $linea) { throw "No se encontró la suma SHA-256 de $archivo en $url" }
    ($linea -split '\s+')[0].ToLower()
}

function Get-Verificado($url, $destino, $suma) {
    Invoke-WebRequest -UseBasicParsing $url -OutFile $destino
    $real = (Get-FileHash -Algorithm SHA256 $destino).Hash.ToLower()
    if ($real -ne $suma) { Remove-Item $destino; throw "La suma SHA-256 de $(Split-Path $destino -Leaf) no coincide" }
}

# Deno: motor de JavaScript que el motor de descarga necesita para algunos sitios.
$denoUrl = 'https://github.com/denoland/deno/releases/latest/download'
$denoZip = 'deno-x86_64-pc-windows-msvc.zip'

$necesarios = 'yt-dlp.exe', 'ffmpeg.exe', 'ffprobe.exe', 'LICENSE-FFmpeg.txt', 'deno.exe'
$faltan = @($necesarios | Where-Object { -not (Test-Path (Join-Path $cache $_)) })
if (-not (Test-Path (Join-Path $cache 'avcodec-*.dll'))) { $faltan += 'DLL de FFmpeg' }
if ($ActualizarComponentes -or $faltan) {
    Write-Host 'Descargando componentes de descarga...'
    $tmp = Join-Path ([IO.Path]::GetTempPath()) "clipvault-comp-$([guid]::NewGuid())"
    New-Item -ItemType Directory $tmp | Out-Null
    try {
        Get-Verificado "$motorUrl/yt-dlp.exe" (Join-Path $tmp 'yt-dlp.exe') (Get-SumaOficial "$motorUrl/SHA2-256SUMS" 'yt-dlp.exe')
        Get-Verificado "$ffmpegUrl/$ffmpegZip" (Join-Path $tmp $ffmpegZip) (Get-SumaOficial "$ffmpegUrl/checksums.sha256" $ffmpegZip)
        # Deno publica su suma con el formato de Get-FileHash ("Hash : ...").
        $textoDeno = (Invoke-WebRequest -UseBasicParsing "$denoUrl/$denoZip.sha256sum").Content
        if ($textoDeno -is [byte[]]) { $textoDeno = [Text.Encoding]::UTF8.GetString($textoDeno) }
        $sumaDeno = ([regex]::Match($textoDeno, 'Hash\s*:\s*([0-9A-Fa-f]{64})')).Groups[1].Value.ToLower()
        if (-not $sumaDeno) { throw "No se encontró la suma SHA-256 de $denoZip" }
        Get-Verificado "$denoUrl/$denoZip" (Join-Path $tmp $denoZip) $sumaDeno
        Expand-Archive (Join-Path $tmp $denoZip) (Join-Path $tmp 'deno')
        Expand-Archive (Join-Path $tmp $ffmpegZip) (Join-Path $tmp 'ffmpeg')
        $base = Get-ChildItem (Join-Path $tmp 'ffmpeg') -Directory | Select-Object -First 1
        if (Test-Path $cache) { Remove-Item -Recurse -Force $cache }
        New-Item -ItemType Directory $cache | Out-Null
        Copy-Item (Join-Path $tmp 'yt-dlp.exe') $cache
        $bin = Join-Path $base.FullName 'bin'
        Copy-Item (Join-Path $bin 'ffmpeg.exe'), (Join-Path $bin 'ffprobe.exe') $cache
        Copy-Item (Join-Path $bin '*.dll') $cache
        Copy-Item (Join-Path $tmp 'deno\deno.exe') $cache
        Copy-Item (Join-Path $base.FullName 'LICENSE.txt') (Join-Path $cache 'LICENSE-FFmpeg.txt') -Force
    } finally {
        Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
    }
}
$verMotor = (& (Join-Path $cache 'yt-dlp.exe') --version).Trim()
$verDeno = ((& (Join-Path $cache 'deno.exe') --version) | Select-Object -First 1).Trim()
Write-Host "Componentes: motor $verMotor + FFmpeg + $verDeno"

# Licencias del motor de descarga y de Deno, de la misma versión que sus ejecutables.
$licencias = @{
    'LICENSE-yt-dlp.txt'                = "https://raw.githubusercontent.com/yt-dlp/yt-dlp/$verMotor/LICENSE"
    'THIRD_PARTY_LICENSES-yt-dlp.txt'   = "https://raw.githubusercontent.com/yt-dlp/yt-dlp/$verMotor/THIRD_PARTY_LICENSES.txt"
    'LICENSE-Deno.txt'                  = "https://raw.githubusercontent.com/denoland/deno/v$(($verDeno -split ' ')[1])/LICENSE.md"
}
foreach ($par in $licencias.GetEnumerator()) {
    $archivo = Join-Path $cache $par.Key
    if (-not (Test-Path $archivo)) { Invoke-WebRequest -UseBasicParsing $par.Value -OutFile $archivo }
}

# ---------- Ejecutable ----------
# Icono, manifiesto y datos de versión dentro del .exe
& $winres simply --arch amd64 --out rsrc --manifest gui `
    --icon build\app.ico `
    --product-name ClipVault --file-description 'ClipVault - descargador de video y audio' `
    --product-version $version --file-version $version `
    --original-filename ClipVault.exe --copyright 'Copyright (c) 2026 Miguel Bigott'
if ($LASTEXITCODE) { throw 'go-winres falló' }

New-Item -ItemType Directory -Force dist | Out-Null
& $goExe build -trimpath -ldflags '-H windowsgui -s -w' -o dist\ClipVault.exe .
if ($LASTEXITCODE) { throw 'go build falló' }

# La app busca los componentes en la carpeta "componentes" junto a su .exe.
$destino = Join-Path $raiz 'dist\componentes'
if (Test-Path $destino) { Remove-Item -Recurse -Force $destino }
New-Item -ItemType Directory $destino | Out-Null
Copy-Item (Join-Path $cache '*') $destino
# Dónde conseguir el código fuente de FFmpeg (lo pide su licencia GPL).
Copy-Item build\FUENTES-FFmpeg.txt $destino

# ---------- Licencias ----------
# LICENSE.txt es la licencia MIT de la app. LICENCIAS-TERCEROS.txt junta las de todo lo que va
# dentro del .exe (Go, sus módulos y las tipografías), que deben acompañar al programa.
Copy-Item LICENSE (Join-Path $raiz 'dist\LICENSE.txt')
$terceros = @("Licencias de terceros incluidas en ClipVault.exe. Los componentes de la carpeta «componentes» (FFmpeg, Deno y el motor de descarga) traen las suyas allí.")
$terceros += "==== $(& $goExe env GOVERSION): biblioteca estándar y runtime de Go ====`n" + (Get-Content -Raw (Join-Path (& $goExe env GOROOT) 'LICENSE'))
$modulos = & $goExe list -deps -f '{{with .Module}}{{if not .Main}}{{.Path}}@{{.Version}}|{{.Dir}}{{end}}{{end}}' . | Sort-Object -Unique
if ($LASTEXITCODE) { throw 'go list falló' }
foreach ($m in $modulos) {
    $nombre, $dir = $m -split '\|', 2
    Get-ChildItem $dir -Recurse -Depth 3 -File | Where-Object { $_.Name -match '^(LICEN[CS]E|COPYING|NOTICE)' } | ForEach-Object {
        $terceros += "==== $nombre ($($_.FullName.Substring($dir.Length + 1))) ====`n" + (Get-Content -Raw $_.FullName)
    }
}
Get-ChildItem web\fonts -Filter 'OFL-*.txt' | ForEach-Object {
    $terceros += "==== Tipografía $($_.BaseName.Substring(4)) ====`n" + (Get-Content -Raw $_.FullName)
}
Set-Content (Join-Path $raiz 'dist\LICENCIAS-TERCEROS.txt') ($terceros -join "`n`n") -Encoding utf8

# ---------- Instalador ----------
# El .nsi está en UTF-8 sin BOM: sin /INPUTCHARSET, makensis lo lee como ANSI y rompe las tildes.
& $makensis /V2 /INPUTCHARSET UTF8 "/DVERSION=$version" build\instalador.nsi
if ($LASTEXITCODE) { throw 'makensis falló' }

Get-ChildItem dist -File | Format-Table Name, @{n = 'MB'; e = { [math]::Round($_.Length / 1MB, 2) } }
