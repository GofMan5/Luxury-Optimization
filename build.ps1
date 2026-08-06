param(
    [ValidatePattern('^1\.0\.\d+(?:-[0-9A-Za-z.-]+)?$')]
    [string]$Version = '1.0.2'
)

$ErrorActionPreference = 'Stop'
$projectRoot = $PSScriptRoot
$backendPath = Join-Path $projectRoot 'backend'
$distPath = Join-Path $projectRoot 'dist'
$stagingPath = $null
$previousPath = $null
$preserveStaging = $false
$savedEnvironment = @{}
foreach ($name in @('CGO_ENABLED', 'GOOS', 'GOARCH')) {
    $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}

$targets = @(
    [pscustomobject]@{ OS = 'windows'; Arch = 'amd64'; Name = 'Luxury-Optimization-windows-amd64.exe' },
    [pscustomobject]@{ OS = 'windows'; Arch = 'arm64'; Name = 'Luxury-Optimization-windows-arm64.exe' },
    [pscustomobject]@{ OS = 'windows'; Arch = '386';   Name = 'Luxury-Optimization-windows-386.exe' },
    [pscustomobject]@{ OS = 'linux';   Arch = 'amd64'; Name = 'Luxury-Optimization-linux-amd64' },
    [pscustomobject]@{ OS = 'linux';   Arch = 'arm64'; Name = 'Luxury-Optimization-linux-arm64' }
)

Push-Location $backendPath
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

    foreach ($target in $targets) {
        $env:GOOS = $target.OS
        $env:GOARCH = $target.Arch
        $outputPath = Join-Path $stagingPath $target.Name
        go build -mod=readonly -trimpath -ldflags "-s -w -X github.com/GofMan5/Luxury-Optimization/internal/optimizer.version=$Version" -o $outputPath ./cmd/luxury-optimization
        if ($LASTEXITCODE -ne 0) { throw "go build $($target.OS)/$($target.Arch) failed with exit code $LASTEXITCODE" }
    }

    $hashByName = @{}
    $hashLines = foreach ($target in $targets) {
        $artifactPath = Join-Path $stagingPath $target.Name
        if (-not (Test-Path -LiteralPath $artifactPath -PathType Leaf)) { throw "Missing artifact: $($target.Name)" }
        $hash = (Get-FileHash -LiteralPath $artifactPath -Algorithm SHA256).Hash.ToLowerInvariant()
        $hashByName[$target.Name] = $hash
        "$hash  $($target.Name)"
    }
    $hashPath = Join-Path $stagingPath 'SHA256SUMS.txt'
    [IO.File]::WriteAllLines($hashPath, [string[]]$hashLines, [Text.UTF8Encoding]::new($false))
    $artifactNames = @($targets | ForEach-Object { $_.Name }) + 'SHA256SUMS.txt'

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

    foreach ($legacyName in @('GofMan3-Optimizer-amd64.exe', 'GofMan3-Optimizer-arm64.exe', 'GofMan3-Optimizer-386.exe')) {
        $legacyPath = Join-Path $distPath $legacyName
        if (Test-Path -LiteralPath $legacyPath -PathType Leaf) { Remove-Item -LiteralPath $legacyPath -Force }
    }
    Get-ChildItem -LiteralPath $distPath -File | Sort-Object Name | Select-Object Name, Length, LastWriteTime
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
