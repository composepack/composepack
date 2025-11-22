Param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:ProgramFiles\\ComposePack",
    [string]$Repo = $env:COMPOSEPACK_REPO
)

if (-not $Repo -or $Repo -eq "") {
    $Repo = "composepack/composepack"
}

if ($Version -eq "latest") {
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
        $Version = $response.tag_name
    } catch {
        Write-Error "Unable to determine latest release. Specify -Version."
        exit 1
    }
}

$arch = $env:PROCESSOR_ARCHITECTURE.ToLower()
switch ($arch) {
    "amd64" { $arch = "amd64" }
    "arm64" { $arch = "arm64" }
    default { Write-Error "Unsupported architecture: $arch"; exit 1 }
}

$asset = "composepack-windows-$arch.exe"
$downloadUrl = "https://github.com/$Repo/releases/download/$Version/$asset"
$tempDir = New-Item -ItemType Directory -Path ([System.IO.Path]::GetTempPath()) -Name ("composepack_" + [System.Guid]::NewGuid())
$assetPath = Join-Path $tempDir.FullName $asset

try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $assetPath -UseBasicParsing
} catch {
    Write-Error "Failed to download $downloadUrl"
    exit 1
}

New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Copy-Item -Path $assetPath -Destination (Join-Path $InstallDir "composepack.exe") -Force
Write-Output "composepack $Version installed to $InstallDir"
