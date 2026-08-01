param(
    [ValidatePattern('^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$')]
    [string]$Version = '3.1.0'
)

$ErrorActionPreference = 'Stop'
$projectRoot = $PSScriptRoot
$distPath = Join-Path $projectRoot 'dist'
$stagingPath = $null
$previousPath = $null
$preserveStaging = $false
$savedEnvironment = @{}
foreach ($name in @('CGO_ENABLED', 'GOOS', 'GOARCH')) {
    $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}

Push-Location $projectRoot
try {
    go mod verify
    if ($LASTEXITCODE -ne 0) { throw "go mod verify failed with exit code $LASTEXITCODE" }
    go test -mod=readonly ./...
    if ($LASTEXITCODE -ne 0) { throw "go test failed with exit code $LASTEXITCODE" }
    go vet -mod=readonly ./...
    if ($LASTEXITCODE -ne 0) { throw "go vet failed with exit code $LASTEXITCODE" }

    New-Item -ItemType Directory -Force -Path $distPath | Out-Null
    $stagingPath = Join-Path $distPath ('.build-' + [Guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $stagingPath | Out-Null
    $env:CGO_ENABLED = '0'
    $env:GOOS = 'windows'

    $architectures = @('amd64', 'arm64', '386')
    foreach ($architecture in $architectures) {
        $env:GOARCH = $architecture
        $outputPath = Join-Path $stagingPath "GofMan3-Optimizer-$architecture.exe"
        go build -mod=readonly -trimpath -ldflags "-s -w -X main.version=$Version" -o $outputPath .
        if ($LASTEXITCODE -ne 0) { throw "go build $architecture failed with exit code $LASTEXITCODE" }
    }

    $hashByName = @{}
    $hashLines = foreach ($architecture in $architectures) {
        $name = "GofMan3-Optimizer-$architecture.exe"
        $artifactPath = Join-Path $stagingPath $name
        if (-not (Test-Path -LiteralPath $artifactPath -PathType Leaf)) { throw "Missing artifact: $name" }
        $hash = (Get-FileHash -LiteralPath $artifactPath -Algorithm SHA256).Hash.ToLowerInvariant()
        $hashByName[$name] = $hash
        "$hash  $name"
    }
    $hashPath = Join-Path $stagingPath 'SHA256SUMS.txt'
    [IO.File]::WriteAllLines($hashPath, [string[]]$hashLines, [Text.UTF8Encoding]::new($false))
    $artifactNames = @($architectures | ForEach-Object { "GofMan3-Optimizer-$_.exe" }) + 'SHA256SUMS.txt'
    $previousPath = Join-Path $stagingPath 'previous'
    New-Item -ItemType Directory -Path $previousPath | Out-Null
    $movedPrevious = @()
    $movedNew = @()
    try {
        foreach ($name in $artifactNames) {
            $destination = Join-Path $distPath $name
            if (Test-Path -LiteralPath $destination -PathType Leaf) {
                Move-Item -LiteralPath $destination -Destination (Join-Path $previousPath $name)
                $movedPrevious += $name
            }
        }
        foreach ($name in $artifactNames) {
            Move-Item -LiteralPath (Join-Path $stagingPath $name) -Destination (Join-Path $distPath $name)
            $movedNew += $name
        }
        foreach ($name in $hashByName.Keys) {
            $publishedHash = (Get-FileHash -LiteralPath (Join-Path $distPath $name) -Algorithm SHA256).Hash.ToLowerInvariant()
            if ($publishedHash -ne $hashByName[$name]) { throw "Published hash mismatch: $name" }
        }
    }
    catch {
        $publicationError = $_
        $rollbackProblems = @()
        foreach ($name in $movedNew) {
            try { Remove-Item -LiteralPath (Join-Path $distPath $name) -Force }
            catch { $rollbackProblems += $_.Exception.Message }
        }
        foreach ($name in $movedPrevious) {
            try { Move-Item -LiteralPath (Join-Path $previousPath $name) -Destination (Join-Path $distPath $name) -Force }
            catch { $rollbackProblems += $_.Exception.Message }
        }
        if ($rollbackProblems.Count -gt 0) {
            $preserveStaging = $true
            throw "Publication failed: $publicationError; rollback failed: $($rollbackProblems -join '; '); recovery files preserved at $stagingPath"
        }
        throw $publicationError
    }
    Get-ChildItem -LiteralPath $distPath -File | Select-Object Name, Length, LastWriteTime
}
finally {
    if (-not $preserveStaging -and $null -ne $stagingPath -and (Test-Path -LiteralPath $stagingPath)) {
        $stagingFull = [IO.Path]::GetFullPath($stagingPath)
        $distPrefix = [IO.Path]::GetFullPath($distPath).TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
        if (-not $stagingFull.StartsWith($distPrefix, [StringComparison]::OrdinalIgnoreCase)) {
            throw "Unsafe staging cleanup path: $stagingFull"
        }
        if ($null -ne $previousPath -and (Test-Path -LiteralPath $previousPath)) {
            Get-ChildItem -LiteralPath $previousPath -File | Remove-Item -Force
            Remove-Item -LiteralPath $previousPath -Force
        }
        Get-ChildItem -LiteralPath $stagingPath -File | Remove-Item -Force
        Remove-Item -LiteralPath $stagingPath -Force
    }
    foreach ($name in $savedEnvironment.Keys) {
        $value = $savedEnvironment[$name]
        if ($null -eq $value) {
            Remove-Item -LiteralPath "Env:$name" -ErrorAction SilentlyContinue
        }
        else {
            Set-Item -LiteralPath "Env:$name" -Value $value
        }
    }
    Pop-Location
}
