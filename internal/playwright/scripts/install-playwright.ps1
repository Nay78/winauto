param(
    [Parameter(Mandatory=$true)]
    [string]$WsPath,
    [int]$Port = 9323
)

$installDir = "C:\ProgramData\win-automation\playwright"
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

# Write ws_path.txt
Set-Content -Path "$installDir\ws_path.txt" -Value $WsPath

# Create scheduled task
$taskName = "WinAutomation-Playwright"
$nodePath = "C:\Program Files\nodejs\node.exe"
$scriptPath = "$installDir\launch-server.js"
$action = New-ScheduledTaskAction -Execute $nodePath -Argument "`"$scriptPath`" --host=0.0.0.0 --port=$Port"
$trigger = New-ScheduledTaskTrigger -AtStartup
$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -RunLevel Highest
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries

Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Principal $principal -Settings $settings

Write-Host "state=installed task=$taskName port=$Port"
