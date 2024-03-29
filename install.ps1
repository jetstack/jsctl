#!/usr/bin/env pwsh

$ErrorActionPreference = 'Stop'

$JsctlInstall = $env:JSCTL_INSTALL
$BinDir = if ($JsctlInstall) {
  "${JsctlInstall}\bin"
} else {
  "${Home}\.jsctl\bin"
}

$JsctlTar = "$BinDir\jsctl.tar.gz"
$JsctlExe = "$BinDir\jsctl.exe"

$Target = if ($ENV:OS -eq "Windows_NT") {
    $arch = if ($ENV:PROCESSOR_ARCHITEW6432) {
        $ENV:PROCESSOR_ARCHITEW6432
    } else {
        $ENV:PROCESSOR_ARCHITECTURE
    }

    switch ($arch) {
        "AMD64" { "jsctl-windows-amd64" }
        "ARM64" { "jsctl-windows-arm64" }
        default { throw "Error: Unsupported windows achitecture: ${arch}" }
    }
} else {
    throw "Error: Unsupported operating system, use the install.sh script instead."
}

$DownloadUrl = "https://github.com/jetstack/jsctl/releases/latest/download/${Target}.tar.gz"

$CurrentInstall = (Get-Command jsctl -ErrorAction SilentlyContinue).Definition
if (!(Test-Path $JsctlExe) -and $CurrentInstall -and ($CurrentInstall -ne $JsctlExe)) {
    throw "Error: tried to install jsctl to '$JsctlExe', but is already installed at '$CurrentInstall'."
}

if (!(Test-Path $BinDir)) {
  New-Item $BinDir -ItemType Directory | Out-Null
}

curl.exe --fail --location --progress-bar --output $JsctlTar $DownloadUrl

tar.exe -x -C $BinDir -f $JsctlTar "$Target/jsctl.exe"

Move-Item -Path "$BinDir\$Target\jsctl.exe" -Destination $JsctlExe -Force

Remove-Item "$BinDir\$Target\"
Remove-Item $JsctlTar

$User = [System.EnvironmentVariableTarget]::User
$Path = [System.Environment]::GetEnvironmentVariable('Path', $User)
if (!(";${Path};".ToLower() -like "*;${BinDir};*".ToLower())) {
    [System.Environment]::SetEnvironmentVariable('Path', "${Path};${BinDir}", $User)
    $Env:Path += ";${BinDir}"
}

Write-Output "jsctl was installed successfully to ${JsctlExe}"
Write-Output "Run 'jsctl --help' to get started"
Write-Output "Checkout https://platform.jetstack.io/documentation for more information"
