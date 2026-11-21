# What the Windows installer DOES, asked of Windows rather than of the source.
#
#   pwsh -NoProfile -File .github/scripts/check_msi.ps1 -Version 0.2.0 `
#        -First <first.msi> -Second <second.msi> -Archive <qadg-gui_..._windows_amd64.zip> -Logs <dir>
#
# Two builds of one version, because every build carries a new ProductCode, and
# the second installed over the first is what a release rebuilt under the same
# number does on somebody's machine. In order:
#
#   1. the programs' versions in the installer's own File table, because they
#      decide upgrades: a program without one is a file Windows Installer leaves
#      in place, and an upgrade then reports the new version over the old
#      program (measured in Cert Doctor on Windows Server 2025, 2026-11-13).
#   2. nothing of ours on the machine before.
#   3. the first build installs: one entry in Programs and Features, the folder
#      holds the window's archive byte for byte, the folder is on the machine's
#      PATH once and a new process finds qadg there, the Start menu has the
#      window for every account.
#   4. the second build installs over it while qadg is running, with every file
#      replaced, as an upgrade to a newer version does (DEC-331 decision 3):
#      Windows Installer answers 3010, the program keeps running, and the
#      folder holds the new files and nothing set aside.
#   5. the uninstall leaves nothing of its own: no entry, nothing on PATH, no
#      shortcut, no registry key, a file somebody else put in the folder still
#      there, and the window's settings in the account untouched.
#
# The same script runs on a runner in release.yml and by hand on a test machine,
# so a change to it is tried without a tag. It needs an administrator, because
# the installer is for every account. Nothing here signs or publishes anything.
#
# Adapted on 2026-11-21 from the installer job of Cert Doctor's release
# workflow, a project of the same owner under the same licence.
param(
  [Parameter(Mandatory = $true)][string]$Version,
  [Parameter(Mandatory = $true)][string]$First,
  [Parameter(Mandatory = $true)][string]$Second,
  [Parameter(Mandatory = $true)][string]$Archive,
  [Parameter(Mandatory = $true)][string]$Logs
)
$ErrorActionPreference = 'Stop'

# One name for the product, the folder it goes into and its registry key. The
# guard in internal/guard holds this to APP_NAME and PUBLISHER in build_msi.py.
$name = 'QA Data Generator'
$publisher = 'DonislawDev'
$installDir = Join-Path $env:ProgramFiles $name
$shortcut = Join-Path ([Environment]::GetFolderPath('CommonPrograms')) ($name + '.lnk')
$regKey = 'HKLM:\Software\' + $publisher + '\' + $name

# Where the window keeps its size and its folders (DEC-325): the account's
# settings, which an uninstall must not touch.
$settings = Join-Path $env:APPDATA 'fyne\dev.donislaw.qadg'
$marker = Join-Path $settings 'check_msi.txt'

$principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
  throw 'check_msi.ps1 installs for every account, which needs an administrator. Run it elevated.'
}
foreach ($file in $First, $Second, $Archive) {
  if (-not (Test-Path -LiteralPath $file -PathType Leaf)) {
    throw "check_msi.ps1 was given $file, and there is no such file"
  }
}
New-Item -ItemType Directory -Force -Path $Logs | Out-Null

$script:failed = 0
function Check($ok, [string]$what) {
  if ($ok) {
    Write-Host "ok      $what"
  } else {
    Write-Host "FAILED  $what"
    $script:failed++
  }
}

# Every file under a folder with its digest, one per line, sorted.
function Listing([string]$dir) {
  if (-not (Test-Path -LiteralPath $dir)) {
    return '(no folder)'
  }
  $lines = Get-ChildItem -LiteralPath $dir -Recurse -File | ForEach-Object {
    $_.FullName.Substring($dir.Length + 1).Replace('\', '/') + ' ' + (Get-FileHash -Algorithm SHA256 -LiteralPath $_.FullName).Hash
  }
  return (@($lines | Sort-Object) -join "`n")
}

function Entries {
  return @(Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*' -ErrorAction SilentlyContinue |
    Where-Object { $_.DisplayName -eq $name })
}

function OursOnPath {
  return @(([Environment]::GetEnvironmentVariable('Path', 'Machine')).Split(';') | Where-Object { $_.TrimEnd('\') -eq $installDir })
}

# msiexec with its arguments in a list and a limit on the wait: a quote that
# swallows the file name leaves it waiting for a help window nobody can close.
function Msi([string]$verb, [string]$file, [string]$log, [string[]]$extra) {
  $argv = @($verb, ('"' + $file + '"'), '/qn', '/norestart', '/l*v', ('"' + (Join-Path $Logs $log) + '"')) + $extra
  $p = Start-Process -FilePath (Join-Path $env:SystemRoot 'System32\msiexec.exe') -ArgumentList $argv -PassThru
  $null = $p.Handle
  if (-not $p.WaitForExit(600000)) {
    throw "msiexec $verb $file did not finish in ten minutes"
  }
  Write-Host "msiexec $verb $(Split-Path -Leaf $file) $($extra -join ' ') -> $($p.ExitCode)"
  return $p.ExitCode
}

# One column of the File table of a package, by file name.
function FileVersions([string]$msi) {
  $wi = New-Object -ComObject WindowsInstaller.Installer
  $db = $wi.GetType().InvokeMember('OpenDatabase', 'InvokeMethod', $null, $wi, @($msi, [int]0))
  $view = $db.GetType().InvokeMember('OpenView', 'InvokeMethod', $null, $db, @('SELECT `FileName`, `Version` FROM `File`'))
  $view.GetType().InvokeMember('Execute', 'InvokeMethod', $null, $view, $null) | Out-Null
  $versions = @{}
  while ($rec = $view.GetType().InvokeMember('Fetch', 'InvokeMethod', $null, $view, $null)) {
    $file = ($rec.GetType().InvokeMember('StringData', 'GetProperty', $null, $rec, @([int]1)) -split '\|')[-1]
    $versions[$file] = $rec.GetType().InvokeMember('StringData', 'GetProperty', $null, $rec, @([int]2))
  }
  $view.GetType().InvokeMember('Close', 'InvokeMethod', $null, $view, $null) | Out-Null
  return $versions
}

$versions = FileVersions $First
foreach ($program in 'qadg.exe', 'qadg-gui.exe') {
  Check ($versions[$program] -eq "$Version.0") "the installer records $program as version $Version.0 ($($versions[$program]))"
}

$unpacked = Join-Path $Logs 'window-archive'
if (Test-Path -LiteralPath $unpacked) {
  Remove-Item -LiteralPath $unpacked -Recurse -Force
}
Expand-Archive -LiteralPath $Archive -DestinationPath $unpacked
$want = Listing $unpacked

function CheckInstalled {
  $entries = @(Entries)
  Check ($entries.Count -eq 1) "one entry in Programs and Features ($($entries.Count))"
  if ($entries.Count -ge 1) {
    Check ($entries[0].DisplayVersion -eq $Version) "it says $Version ($($entries[0].DisplayVersion))"
    Check ($entries[0].Publisher -eq $publisher) "publisher $publisher ($($entries[0].Publisher))"
  }
  $have = Listing $installDir
  Check ($have -eq $want) 'the folder holds the window archive, byte for byte'
  if ($have -ne $want) {
    Write-Host "the folder holds:`n$have"
  }
  Check (@(OursOnPath).Count -eq 1) "the install folder is on the machine PATH once ($(@(OursOnPath).Count))"
  Check (-not (Test-Path -LiteralPath (Join-Path $installDir 'opengl32.dll'))) 'opengl32.dll is not in the folder on PATH'
  $env:Path = [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('Path', 'User')
  $said = (@(& cmd.exe /c 'qadg --version' 2>&1)[0] | Out-String).Trim()
  Check ($said -eq "qadg $Version") "qadg found on PATH by a new process says $Version ($said)"
  Check (Test-Path -LiteralPath $shortcut) 'the Start menu has the window for every account'
  if (Test-Path -LiteralPath $shortcut) {
    $link = (New-Object -ComObject WScript.Shell).CreateShortcut($shortcut)
    Check ($link.TargetPath -eq (Join-Path $installDir 'qadg-gui.exe')) "it starts the window ($($link.TargetPath))"
  }
}

Check (@(Entries).Count -eq 0 -and -not (Test-Path -LiteralPath $installDir)) 'nothing of ours on this machine before'
Check ((Msi '/i' $First 'install.log' @()) -eq 0) 'installed'
CheckInstalled
$firstCode = @(Entries)[0].PSChildName

# qadg writing more values than anybody reads: its output is a pipe this
# script never reads, so it fills and qadg waits on it, holding qadg.exe open
# for as long as this script wants, which is what a long run of generate does
# to an installer. REINSTALLMODE=amus makes the second build replace every
# file whatever its version, which is what an upgrade to a newer version does
# to the programs.
$running = New-Object System.Diagnostics.ProcessStartInfo
$running.FileName = Join-Path $installDir 'qadg.exe'
$running.Arguments = 'quick pesel --count 2000000000 --seed installer-check'
$running.UseShellExecute = $false
$running.RedirectStandardOutput = $true
$running.CreateNoWindow = $true
$cli = [System.Diagnostics.Process]::Start($running)
Start-Sleep -Seconds 3
Check (-not $cli.HasExited) "qadg is running from the install folder (process $($cli.Id))"

$code = Msi '/i' $Second 'in-use.log' @('REINSTALLMODE=amus')
Check ($code -eq 3010) "the second build replaced every file while qadg ran, and asks for a restart ($code)"
Check (-not $cli.HasExited) 'qadg was not closed by the installer'
CheckInstalled
Check (@(Entries)[0].PSChildName -ne $firstCode) "the second build replaced the first ($firstCode -> $(@(Entries)[0].PSChildName))"
$cli.Kill()
$null = $cli.WaitForExit(30000)

$fyne = Split-Path -Parent $settings
$createdFyne = -not (Test-Path -LiteralPath $fyne)
$createdSettings = -not (Test-Path -LiteralPath $settings)
New-Item -ItemType Directory -Force -Path $settings | Out-Null
Set-Content -LiteralPath $marker -Value 'the window keeps its settings here'
Set-Content -LiteralPath (Join-Path $installDir 'somebody-elses.txt') -Value 'not ours'
$code = Msi '/x' $Second 'uninstall.log' @()
Check ($code -eq 0) "uninstalled, with a file of somebody else in the folder ($code)"
Check (@(Entries).Count -eq 0) 'no entry in Programs and Features'
Check (@(OursOnPath).Count -eq 0) 'nothing of ours on the machine PATH'
Check (-not (Test-Path -LiteralPath $shortcut)) 'no shortcut'
Check (-not (Test-Path -LiteralPath $regKey)) "no registry key $regKey"
$left = (@(Get-ChildItem -LiteralPath $installDir -Recurse -File -ErrorAction SilentlyContinue | ForEach-Object { $_.Name }) -join ', ')
Check ($left -eq 'somebody-elses.txt') "the folder holds only the file that is not ours ($left)"
Check (Test-Path -LiteralPath $marker) "the window's settings in $settings are untouched"

# What this script put on the machine goes, and nothing else.
Remove-Item -LiteralPath (Join-Path $installDir 'somebody-elses.txt') -Force -ErrorAction SilentlyContinue
if ((Test-Path -LiteralPath $installDir) -and -not (Get-ChildItem -LiteralPath $installDir -Force)) {
  Remove-Item -LiteralPath $installDir -Force
}
Remove-Item -LiteralPath $marker -Force -ErrorAction SilentlyContinue
if ($createdFyne) {
  Remove-Item -LiteralPath $fyne -Recurse -Force -ErrorAction SilentlyContinue
} elseif ($createdSettings) {
  Remove-Item -LiteralPath $settings -Recurse -Force -ErrorAction SilentlyContinue
}

if ($script:failed -ne 0) {
  throw "$($script:failed) check(s) of the installer failed. The msiexec logs are in $Logs"
}
'the installer installs, replaces itself while qadg runs, and leaves nothing of its own behind'
