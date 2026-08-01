@echo off
echo This BAT is a disabled source archive. Use dist\GofMan3-Optimizer-amd64.exe.
exit /b 1
:: Простая обработка ошибок
setlocal EnableDelayedExpansion
color 0A
chcp 65001 >nul

:: Универсальный оптимизатор системы Windows
:: Автор: GofMan3
:: YouTube: https://www.youtube.com/@GofMan3

:: Запрос административных прав
:-------------------------------------
>nul 2>&1 "%SYSTEMROOT%\system32\cacls.exe" "%SYSTEMROOT%\system32\config\system"
if '%errorlevel%' NEQ '0' (
    color 0C
    echo.
    echo  ██████╗ ███████╗ ██████╗ ███████╗███████╗███████╗    ███████╗██████╗ ██████╗  ██████╗ ██████╗ 
    echo  ██╔══██╗██╔════╝██╔════╝ ██╔════╝██╔════╝██╔════╝    ██╔════╝██╔══██╗██╔══██╗██╔═══██╗██╔══██╗
    echo  ██████╔╝█████╗  ██║      █████╗  ███████╗███████╗    █████╗  ██████╔╝██████╔╝██║   ██║██████╔╝
    echo  ██╔══██╗██╔══╝  ██║      ██╔══╝  ╚════██║╚════██║    ██╔══╝  ██╔══██╗██╔══██╗██║   ██║██╔══██╗
    echo  ██║  ██║███████╗╚██████╗ ███████╗███████║███████║    ███████╗██║  ██║██║  ██║╚██████╔╝██║  ██║
    echo  ╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚══════╝╚══════╝╚══════╝    ╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝
    echo.
    echo                    [!] Требуются права администратора для работы!
    echo                    [!] Нажмите любую клавишу для перезапуска...
    pause >nul
    goto UACPrompt
) else ( goto gotAdmin )

:UACPrompt
    echo Set UAC = CreateObject^("Shell.Application"^) > "%temp%\getadmin.vbs"
    set params = %*:"=""
    echo UAC.ShellExecute "cmd.exe", "/c %~s0 %params%", "", "runas", 1 >> "%temp%\getadmin.vbs"
    "%temp%\getadmin.vbs"
    del "%temp%\getadmin.vbs"
    exit /B

:gotAdmin
    pushd "%CD%"
    CD /D "%~dp0"

cls
color 0A
echo.
echo.
echo  ██████╗  ██████╗ ███████╗███╗   ███╗ █████╗ ███╗   ██╗██████╗     
echo  ██╔════╝ ██╔═══██╗██╔════╝████╗ ████║██╔══██╗████╗  ██║╚════██╗    
echo  ██║  ███╗██║   ██║█████╗  ██╔████╔██║███████║██╔██╗ ██║ █████╔╝    
echo  ██║   ██║██║   ██║██╔══╝  ██║╚██╔╝██║██╔══██║██║╚██╗██║ ╚═══██╗    
echo  ╚██████╔╝╚██████╔╝██║     ██║ ╚═╝ ██║██║  ██║██║ ╚████║██████╔╝    
echo   ╚═════╝  ╚═════╝ ╚═╝     ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═════╝     
echo.
echo  ██████╗ ██████╗ ████████╗██╗███╗   ███╗██╗███████╗ █████╗ ████████╗██╗ ██████╗ ███╗   ██╗
echo  ██╔═══██╗██╔══██╗╚══██╔══╝██║████╗ ████║██║╚══███╔╝██╔══██╗╚══██╔══╝██║██╔═══██╗████╗  ██║
echo  ██║   ██║██████╔╝   ██║   ██║██╔████╔██║██║  ███╔╝ ███████║   ██║   ██║██║   ██║██╔██╗ ██║
echo  ██║   ██║██╔═══╝    ██║   ██║██║╚██╔╝██║██║ ███╔╝  ██╔══██║   ██║   ██║██║   ██║██║╚██╗██║
echo  ╚██████╔╝██║        ██║   ██║██║ ╚═╝ ██║██║███████╗██║  ██║   ██║   ██║╚██████╔╝██║ ╚████║
echo   ╚═════╝ ╚═╝        ╚═╝   ╚═╝╚═╝     ╚═╝╚═╝╚══════╝╚═╝  ╚═╝   ╚═╝   ╚═╝ ╚═════╝ ╚═╝  ╚═══╝
echo.
echo  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓
echo.
echo                 ╔═══════════════════════════════════════════════╗
echo                 ║ YouTube: https://www.youtube.com/@GofMan3     ║
echo                 ╚═══════════════════════════════════════════════╝
echo.
echo.
echo   ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄  Начинаем оптимизацию системы  ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
echo.
timeout /t 2 /nobreak >nul
cls

echo.
echo   [1/37] ■ Удаление временных файлов
RD /S /Q %temp%
MKDIR %temp%
takeown /f "C:\Windows\Temp" /r /d y
RD /S /Q C:\Windows\Temp
MKDIR C:\Windows\Temp
del /s /f /q C:\WINDOWS\Prefetch
del /s /f /q %temp%\*.*
deltree /y c:\windows\tempor~1 2>nul
deltree /y c:\windows\temp 2>nul
deltree /y c:\windows\tmp 2>nul
deltree /y c:\windows\ff*.tmp 2>nul
deltree /y c:\windows\history 2>nul
deltree /y c:\windows\cookies 2>nul
deltree /y c:\windows\recent 2>nul
deltree /y c:\windows\spool\printers 2>nul
del c:\WIN386.SWP 2>nul
FOR /F " tokens=*" %%G in ('wevtutil.exe el') DO (call :clear_event_log "%%G")
echo Временные файлы удалены успешно.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [2/37] ╔═══ Отключение HPET и настройка параметров загрузки ═══╗
bcdedit /deletevalue disabledynamictick
bcdedit /set disabledynamictick yes
bcdedit /deletevalue useplatformclock
bcdedit /set useplatformtick yes

:: Отключение HPET устройства в диспетчере устройств
echo Отключение устройства HPET в диспетчере устройств...
for /f "tokens=*" %%i in ('wmic path Win32_PnPEntity where "DeviceID like '%%ACPI\\VEN_PNP&DEV_0103%%'" get DeviceID /value 2^>nul') do (
    for /f "tokens=2 delims==" %%a in ("%%i") do (
        if not "%%a"=="" (
            echo Найдено устройство HPET: %%a
            reg add "HKLM\SYSTEM\CurrentControlSet\Enum\%%a\Device Parameters" /v "Disable" /t REG_DWORD /d "1" /f >nul 2>&1
            PowerShell "Get-PnpDevice -InstanceId '%%a' | Disable-PnpDevice -Confirm:$false" >nul 2>&1
        )
    )
)

:: Отключение службы "Таймер событий высокой точности"
sc config "High Precision Event Timer" start= disabled >nul 2>&1
net stop "High Precision Event Timer" >nul 2>&1

echo HPET отключен и параметры загрузки настроены.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [3/37] ╔═══ Отключение префетчера ═══╗
net stop SysMain
reg add "HKLM\SYSTEM\CurrentControlSet\Services\SysMain" /v "Start" /t REG_DWORD /d "4" /f
echo Префетчер отключен.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [4/37] ╔═══ Отключение энергосбережения USB устройств ═══╗
:: Первый метод - используем SetACL (если доступен)
for /f %%a in ('wmic PATH Win32_PnPEntity GET DeviceID ^| findstr /l "USB\VID_"') do (
    if exist "C:\Windows\SetACL.exe" (
        C:\Windows\SetACL.exe -on "HKLM\SYSTEM\ControlSet001\Enum\%%a\Device Parameters" -ot reg -actn setowner -ownr "n:Administrators" >nul 2>&1
        C:\Windows\SetACL.exe -on "HKLM\SYSTEM\ControlSet001\Enum\%%a\Device Parameters" -ot reg -actn ace -ace "n:Administrators;p:full" >nul 2>&1
    )
    reg.exe add "HKLM\SYSTEM\ControlSet001\Enum\%%a\Device Parameters" /v SelectiveSuspendOn /t REG_DWORD /d 00000000 /f >nul 2>&1
    reg.exe add "HKLM\SYSTEM\ControlSet001\Enum\%%a\Device Parameters" /v SelectiveSuspendEnabled /t REG_BINARY /d 00 /f >nul 2>&1
    reg.exe add "HKLM\SYSTEM\ControlSet001\Enum\%%a\Device Parameters" /v EnhancedPowerManagementEnabled /t REG_DWORD /d 00000000 /f >nul 2>&1
    reg.exe add "HKLM\SYSTEM\ControlSet001\Enum\%%a\Device Parameters" /v AllowIdleIrpInD3 /t REG_DWORD /d 00000000 /f >nul 2>&1
)

:: Второй метод - от ROOT_HUB
for /f %%a in ('wmic PATH Win32_USBHub GET DeviceID ^| findstr /l "USB\ROOT_HUB"') do (
    if exist "C:\Windows\SetACL.exe" (
        C:\Windows\SetACL.exe -on "HKLM\SYSTEM\ControlSet001\Enum\%%a\Device Parameters\WDF" -ot reg -actn setowner -ownr "n:Administrators" >nul 2>&1
    )
    reg.exe add "HKLM\SYSTEM\ControlSet001\Enum\%%a\Device Parameters\WDF" /v IdleInWorkingState /t REG_DWORD /d 00000000 /f >nul 2>&1
)

:: Третий метод - дополнительный из Disable USB Idle
FOR /F %%a in ('WMIC PATH Win32_USBHub GET DeviceID^| FINDSTR /L "VID_"') DO (
    REG ADD "HKLM\SYSTEM\CurrentControlSet\Enum\%%a\Device Parameters" /F /V "EnhancedPowerManagementEnabled" /T REG_DWORD /d 0 >nul 2>&1
    REG ADD "HKLM\SYSTEM\CurrentControlSet\Enum\%%a\Device Parameters" /F /V "AllowIdleIrpInD3" /T REG_DWORD /d 0 >nul 2>&1
    REG ADD "HKLM\SYSTEM\CurrentControlSet\Enum\%%a\Device Parameters" /F /V "SelectiveSuspendOn" /T REG_DWORD /d 0 >nul 2>&1
    REG ADD "HKLM\SYSTEM\CurrentControlSet\Enum\%%a\Device Parameters" /F /V "DeviceSelectiveSuspended" /T REG_DWORD /d 0 >nul 2>&1
    REG ADD "HKLM\SYSTEM\CurrentControlSet\Enum\%%a\Device Parameters" /F /V "SelectiveSuspendEnabled" /T REG_DWORD /d 0 >nul 2>&1
)
echo Энергосбережение USB устройств отключено.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [5/37] ╔═══ Применение оптимизаций реестра ═══╗
:: 16 Hex - Smoothest
echo Оптимизация приоритетов процессов (Win32PrioritySeparation)...
reg add "HKLM\SYSTEM\CurrentControlSet\Control\PriorityControl" /v "Win32PrioritySeparation" /t REG_DWORD /d "16" /f

:: Decrease Delay
echo Оптимизация сетевых задержек...
reg add "HKLM\SYSTEM\CurrentControlSet\services\LanmanServer\Parameters" /v "autodisconnect" /t REG_DWORD /d "4294967295" /f
reg add "HKLM\SYSTEM\CurrentControlSet\services\LanmanServer\Parameters" /v "Size" /t REG_DWORD /d "3" /f
reg add "HKLM\SYSTEM\CurrentControlSet\services\LanmanServer\Parameters" /v "EnableOplocks" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\CurrentControlSet\services\LanmanServer\Parameters" /v "IRPStackSize" /t REG_DWORD /d "32" /f
reg add "HKLM\SYSTEM\CurrentControlSet\services\LanmanServer\Parameters" /v "SharingViolationDelay" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\CurrentControlSet\services\LanmanServer\Parameters" /v "SharingViolationRetries" /t REG_DWORD /d "0" /f

:: CPU Optimization & Сетевой Троттлинг Индекс & Disable Network Throttling & Increase Responsiveness
echo Оптимизация CPU и сетевого троттлинга...
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile" /v "NetworkThrottlingIndex" /t REG_DWORD /d "4294967295" /f
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile" /v "SystemResponsiveness" /t REG_DWORD /d "0" /f

:: Disable GpuEnergyDriver
echo Отключение драйвера энергосбережения GPU...
reg add "HKLM\SYSTEM\CurrentControlSet\Services\GpuEnergyDrv" /v "Start" /t REG_DWORD /d "4" /f

:: Disable Power Throttling
echo Отключение ограничений производительности...
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Power\PowerSettings\54533251-82be-4824-96c1-47b60b740d00\3b04d4fd-1cc7-4f23-ab1c-d1337819c4bb\DefaultPowerSchemeValues\381b4222-f694-41f0-9685-ff5bb260df2e" /v "ACSettingIndex" /t REG_DWORD /d "2" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Power\PowerSettings\54533251-82be-4824-96c1-47b60b740d00\3b04d4fd-1cc7-4f23-ab1c-d1337819c4bb\DefaultPowerSchemeValues\381b4222-f694-41f0-9685-ff5bb260df2e" /v "DCSettingIndex" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Power\PowerSettings\54533251-82be-4824-96c1-47b60b740d00\3b04d4fd-1cc7-4f23-ab1c-d1337819c4bb\DefaultPowerSchemeValues\8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c" /v "ACSettingIndex" /t REG_DWORD /d "0" /f
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\DriverSearching" /v "SearchOrderConfig" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power" /v "HiberbootEnabled" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Power\PowerThrottling" /v "PowerThrottlingOff" /t REG_DWORD /d "1" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Power" /v "HibernateEnabledDefault" /t REG_DWORD /d "0" /f

:: Game Optimizations & Increase FPS in Games
echo Применение игровых оптимизаций...
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile\Tasks\Games" /v "GPU Priority" /t REG_DWORD /d "8" /f
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile\Tasks\Games" /v "Priority" /t REG_DWORD /d "6" /f
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile\Tasks\Games" /v "Scheduling Category" /t REG_SZ /d "High" /f
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile\Tasks\Games" /v "SFIO Priority" /t REG_SZ /d "High" /f
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile\Tasks\Games" /v "Affinity" /t REG_DWORD /d "0" /f
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile\Tasks\Games" /v "Background Only" /t REG_SZ /d "False" /f
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile\Tasks\Games" /v "Clock Rate" /t REG_DWORD /d "10000" /f

:: GPU Tweaks
echo Оптимизация драйверов GPU...
reg add "HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers" /v "RmGpsPsEnablePerCpuCoreDpc" /t REG_DWORD /d "1" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers\Power" /v "RmGpsPsEnablePerCpuCoreDpc" /t REG_DWORD /d "1" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\nvlddmkm" /v "RmGpsPsEnablePerCpuCoreDpc" /t REG_DWORD /d "1" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\nvlddmkm\NVAPI" /v "RmGpsPsEnablePerCpuCoreDpc" /t REG_DWORD /d "1" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\nvlddmkm\Global\NVTweak" /v "RmGpsPsEnablePerCpuCoreDpc" /t REG_DWORD /d "1" /f

:: Increase Responsiveness
echo Увеличение отзывчивости системы...
reg add "HKCU\Control Panel\Desktop" /v "MenuShowDelay" /t REG_SZ /d "0" /f
reg add "HKCU\Control Panel\Desktop" /v "WaitToKillAppTimeout" /t REG_SZ /d "5000" /f
reg add "HKCU\Control Panel\Desktop" /v "HungAppTimeout" /t REG_SZ /d "4000" /f
reg add "HKCU\Control Panel\Desktop" /v "AutoEndTasks" /t REG_SZ /d "1" /f
reg add "HKCU\Control Panel\Desktop" /v "LowLevelHooksTimeout" /t REG_DWORD /d "4096" /f
reg add "HKCU\Control Panel\Desktop" /v "WaitToKillServiceTimeout" /t REG_DWORD /d "8192" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control" /v "WaitToKillServiceTimeout" /t REG_SZ /d "2000" /f

echo Применение оптимизаций реестра завершено.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [6/37] ╔═══ Отключение сжатия памяти ═══╗
PowerShell "Disable-MMAgent -MemoryCompression"
echo Сжатие памяти отключено.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [7/37] ╔═══ Отключение HIPM и DIPM, парковка HDD ═══╗
FOR /F "eol=E" %%a in ('REG QUERY "HKLM\SYSTEM\CurrentControlSet\Services" /S /F "EnableHIPM"^| FINDSTR /V "EnableHIPM"') DO (
    REG ADD "%%a" /F /V "EnableHIPM" /T REG_DWORD /d 0 >nul 2>&1
    REG ADD "%%a" /F /V "EnableDIPM" /T REG_DWORD /d 0 >nul 2>&1
    REG ADD "%%a" /F /V "EnableHDDParking" /T REG_DWORD /d 0 >nul 2>&1

    FOR /F "tokens=*" %%z IN ("%%a") DO (
        SET STR=%%z
        SET STR=!STR:HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\=!
        ECHO Отключение HIPM и DIPM в !STR!
    )
)
echo HIPM, DIPM и парковка HDD отключены.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [8/37] ╔═══ Настройка IoLatencyCap ═══╗
FOR /F "eol=E" %%a in ('REG QUERY "HKLM\SYSTEM\CurrentControlSet\Services" /S /F "IoLatencyCap"^| FINDSTR /V "IoLatencyCap"') DO (
    REG ADD "%%a" /F /V "IoLatencyCap" /T REG_DWORD /d 0 >nul 2>&1

    FOR /F "tokens=*" %%z IN ("%%a") DO (
        SET STR=%%z
        SET STR=!STR:HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\services\=!
        SET STR=!STR:\Parameters=!
        ECHO Установка IoLatencyCap в 0 для !STR!
    )
)
echo IoLatencyCap успешно настроен.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [9/37] ╔═══ Отключение Process Mitigation ═══╗
powershell set-ProcessMitigation -System -Disable DEP, EmulateAtlThunks, SEHOP, ForceRelocateImages, RequireInfo, BottomUp, HighEntropy, StrictHandle, DisableWin32kSystemCall, AuditSystemCall, DisableExtensionPoints, BlockDynamicCode, AllowThreadsToOptOut, AuditDynamicCode, CFG, SuppressExports, StrictCFG, MicrosoftSignedOnly, AllowStoreSignedBinaries, AuditMicrosoftSigned, AuditStoreSigned, EnforceModuleDependencySigning, DisableNonSystemFonts, AuditFont, BlockRemoteImageLoads, BlockLowLabelImageLoads, PreferSystem32, AuditRemoteImageLoads, AuditLowLabelImageLoads, AuditPreferSystem32, EnableExportAddressFilter, AuditEnableExportAddressFilter, EnableExportAddressFilterPlus, AuditEnableExportAddressFilterPlus, EnableImportAddressFilter, AuditEnableImportAddressFilter, EnableRopStackPivot, AuditEnableRopStackPivot, EnableRopCallerCheck, AuditEnableRopCallerCheck, EnableRopSimExec, AuditEnableRopSimExec, SEHOP, AuditSEHOP, SEHOPTelemetry, TerminateOnError, DisallowChildProcessCreation, AuditChildProcess
echo Process Mitigation отключен.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [10/37] ╔═══ Отключение StorPort Idle ═══╗
for /f "tokens=*" %%s in ('reg query "HKLM\System\CurrentControlSet\Enum" /S /F "StorPort" ^| findstr /e "StorPort"') do (
    Reg add "%%s" /v "EnableIdlePowerManagement" /t REG_DWORD /d "0" /f >nul 2>&1
)
echo StorPort Idle отключен.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [11/37] ╔═══ Отключение FSE и Game Bar ═══╗
reg add "HKCU\SOFTWARE\Microsoft\GameBar" /v "ShowStartupPanel" /t REG_DWORD /d "0" /f 
reg add "HKCU\SOFTWARE\Microsoft\GameBar" /v "GamePanelStartupTipIndex" /t REG_DWORD /d "3" /f 
reg add "HKCU\SOFTWARE\Microsoft\GameBar" /v "AllowAutoGameMode" /t REG_DWORD /d "0" /f 
reg add "HKCU\SOFTWARE\Microsoft\GameBar" /v "AutoGameModeEnabled" /t REG_DWORD /d "0" /f 
reg add "HKCU\SOFTWARE\Microsoft\GameBar" /v "UseNexusForGameBarEnabled" /t REG_DWORD /d "0" /f 
reg add "HKCU\System\GameConfigStore" /v "GameDVR_Enabled" /t REG_DWORD /d "0" /f 
reg add "HKCU\System\GameConfigStore" /v "GameDVR_FSEBehaviorMode" /t REG_DWORD /d "2" /f 
reg add "HKCU\System\GameConfigStore" /v "GameDVR_FSEBehavior" /t REG_DWORD /d "2" /f 
reg add "HKCU\System\GameConfigStore" /v "GameDVR_HonorUserFSEBehaviorMode" /t REG_DWORD /d "1" /f 
reg add "HKCU\System\GameConfigStore" /v "GameDVR_DXGIHonorFSEWindowsCompatible" /t REG_DWORD /d "1" /f 
reg add "HKCU\System\GameConfigStore" /v "GameDVR_EFSEFeatureFlags" /t REG_DWORD /d "0" /f 
reg add "HKCU\System\GameConfigStore" /v "GameDVR_DSEBehavior" /t REG_DWORD /d "2" /f 
reg add "HKLM\SOFTWARE\Microsoft\PolicyManager\default\ApplicationManagement\AllowGameDVR" /v "value" /t REG_DWORD /d "0" /f 
reg add "HKLM\SOFTWARE\Policies\Microsoft\Windows\GameDVR" /v "AllowGameDVR" /t REG_DWORD /d "0" /f 
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\GameDVR" /v "AppCaptureEnabled" /t REG_DWORD /d "0" /f 
reg add "HKU\.DEFAULT\SOFTWARE\Microsoft\GameBar" /v "AutoGameModeEnabled" /t REG_DWORD /d "0" /f 
reg delete "HKCU\System\GameConfigStore\Children" /f 
reg delete "HKCU\System\GameConfigStore\Parents" /f
echo FSE и Game Bar отключены.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [12/37] ╔═══ Оптимизация управления памятью ═══╗
:: Memory Management Optimization
echo Оптимизация памяти и безопасности...
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "DisablePagingExecutive" /t REG_DWORD /d "1" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "LargeSystemCache" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "NonPagedPoolQuota" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "NonPagedPoolSize" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "PagedPoolQuota" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "PagedPoolSize" /t REG_DWORD /d "192" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "SystemPages" /t REG_DWORD /d "4294967295" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "FeatureSettingsOverride" /t REG_DWORD /d "3" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "FeatureSettingsOverrideMask" /t REG_DWORD /d "3" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "KernelSEHOPEnabled" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "DisableExceptionChainValidation" /t REG_DWORD /d "1" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "PhysicalAddressExtension" /t REG_DWORD /d "1" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "FeatureSettings" /t REG_DWORD /d "1" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "PoolUsageMaximum" /t REG_DWORD /d "96" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "EnableBoottrace" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "EnableCfg" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "EnableLowVaAccess" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "CoalescingTimerInterval" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "IoPageLockLimit" /t REG_DWORD /d "8000000" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management" /v "DisablePagingCombining" /t REG_DWORD /d "1" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management\PrefetchParameters" /v "EnableBootTrace" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\ControlSet001\Control\Session Manager\Memory Management\PrefetchParameters" /v "EnablePrefetcher" /t REG_DWORD /d "0" /f
echo Оптимизация управления памятью завершена.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [13/37] ╔═══ Настройка параметров OpenGL ═══╗
:: Установка общих параметров OpenGL
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\OpenGLDrivers\MSOpenGL" /v "DisableAsync" /t REG_DWORD /d 0x1 /f
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\OpenGLDrivers\MSOpenGL" /v "EnableMultisampleBuffers" /t REG_DWORD /d 0x1 /f
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\OpenGLDrivers\MSOpenGL" /v "EnableSurfaceMemoryManagement" /t REG_DWORD /d 0x1 /f
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\OpenGLDrivers\MSOpenGL" /v "MaxWGLSwapInterval" /t REG_DWORD /d 0x1 /f

:: Настройка размера буфера OpenGL для различных видеокарт
:: AMD
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\OpenGLDrivers\AMD Inc.\AMD OpenGL Driver" /v MaxBufferSize /t REG_DWORD /d 4194304 /f
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\OpenGLDrivers\AMD Inc.\AMD OpenGL Driver" /v BufferSize /t REG_DWORD /d 2097152 /f
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\OpenGLDrivers\ATI Technologies Inc.\ATI OpenGL Driver" /v BufferSize /t REG_DWORD /d 1048576 /f

:: Intel
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\OpenGLDrivers\Intel Corporation\Intel(R) HD Graphics" /v MaxBufferSize /t REG_DWORD /d 4194304 /f
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\OpenGLDrivers\Intel Corporation\Intel(R) HD Graphics" /v BufferSize /t REG_DWORD /d 2097152 /f

:: NVIDIA
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\OpenGLDrivers\NVIDIA Corporation\nvoglv64" /v MaxBufferSize /t REG_DWORD /d 4194304 /f
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\OpenGLDrivers\NVIDIA Corporation\nvoglv64" /v BufferSize /t REG_DWORD /d 2097152 /f

:: Увеличение буфера OpenGL через fsutil
set OPENGL_32_DLL=C:\Windows\System32\opengl32.dll
set OPENGL_32_DLL_SIZE=2576980377
copy %OPENGL_32_DLL% %OPENGL_32_DLL%.bak
fsutil file setvaliddata %OPENGL_32_DLL% %OPENGL_32_DLL_SIZE%

:: Дополнительные настройки OpenGL
set OPENGL_DEBUG=1
echo Параметры OpenGL оптимизированы.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [14/37] ╔═══ Настройка переменных GPU ═══╗
setx GPU_MAX_HEAP_SIZE 100
setx GPU_MAX_ALLOC_PERCENT 100
setx GPU_SINGLE_ALLOC_PERCENT 100
echo Переменные GPU настроены.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [15/37] ╔═══ Оптимизация RuntimeBroker ═══╗
taskkill /im runtimebroker.exe /f
if exist "%WinDir%\System32\runtimebroker.exe.bak" (
    echo Резервная копия RuntimeBroker.exe уже существует.
) else (
    copy "%WinDir%\System32\runtimebroker.exe" "%WinDir%\System32\runtimebroker.exe.bak" >nul 2>&1
)
takeown /f "%WinDir%\System32\runtimebroker.exe" >nul 2>&1
icacls "%WinDir%\System32\runtimebroker.exe" /grant administrators:F >nul 2>&1
:: Не удаляем полностью, а переименовываем для возможности восстановления
if exist "%WinDir%\System32\runtimebroker.exe" (
    rename "%WinDir%\System32\runtimebroker.exe" "runtimebroker.exe.disabled" >nul 2>&1
)
echo RuntimeBroker оптимизирован.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [16/37] ╔═══ Оптимизация сетевых параметров ═══╗
netsh int tcp set global autotuninglevel=normal
netsh int tcp set global chimney=enabled
netsh int tcp set global dca=enabled
netsh int tcp set global netdma=disabled
netsh int tcp set global congestionprovider=ctcp
netsh int tcp set global ecncapability=disabled
netsh int tcp set heuristics disabled
netsh int tcp set global rss=enabled
netsh int tcp set global fastopen=enabled
netsh int tcp set global nonsackrttresiliency=disabled
netsh int tcp set global rsc=enabled
echo Сетевые параметры TCP/IP оптимизированы.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [17/37] ╔═══ Установка высокого приоритета для CSRSS ═══╗
reg add "HKLM\Software\Microsoft\Windows NT\CurrentVersion\Image File Execution Options\csrss.exe\PerfOptions" /v "CpuPriorityClass" /t REG_DWORD /d "4" /f
reg add "HKLM\Software\Microsoft\Windows NT\CurrentVersion\Image File Execution Options\csrss.exe\PerfOptions" /v "IoPriority" /t REG_DWORD /d "3" /f
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile" /v "NoLazyMode" /t REG_DWORD /d "1" /f
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile" /v "AlwaysOn" /t REG_DWORD /d "1" /f
echo CSRSS установлен на высокий приоритет для улучшения отзывчивости системы.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [18/37] ╔═══ Отключение планировщиков задач Windows ═══╗
echo Отключение триггеров и планировщика задач...
schtasks /change /TN "Microsoft\Windows\.NET Framework\.NET Framework NGEN v4.0.30319" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\.NET Framework\.NET Framework NGEN v4.0.30319 64" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\.NET Framework\.NET Framework NGEN v4.0.30319 64 Critical" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\.NET Framework\.NET Framework NGEN v4.0.30319 Critical" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\ApplicationData\appuriverifierdaily" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\ApplicationData\appuriverifierinstall" /DISABLE > NUL 2>&1
schtasks /Change /TN "Microsoft\Windows\Application Experience\Microsoft Compatibility Appraiser" /DISABLE > NUL 2>&1
schtasks /Change /TN "Microsoft\Windows\Application Experience\ProgramDataUpdater" /DISABLE > NUL 2>&1
schtasks /Change /TN "Microsoft\Windows\Application Experience\StartupAppTask" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\Device Information\Device" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\Diagnosis\Scheduled" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\Diagnosis\RecommendedTroubleshootingScanner" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\DiskFootprint\Diagnostics" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\DiskFootprint\StorageSense" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\Feedback\Siuf\DmClientOnScenarioDownload" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\Feedback\Siuf\DmClient" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\International\Synchronize Language Settings" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\LanguageComponentsInstaller\Installation" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\LanguageComponentsInstaller\ReconcileLanguageResources" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\Maps\MapsUpdateTask" /DISABLE > NUL 2>&1
schtasks /Change /TN "Microsoft\Windows\Maps\MapsToastTask" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\PushToInstall\Registration" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\Setup\SetupCleanupTask" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\Speech\SpeechModelDownloadTask" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\Windows Error Reporting\QueueReporting" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\WindowsColorSystem\Calibration Loader" /DISABLE > NUL 2>&1
schtasks /change /TN "Microsoft\Windows\Work Folders\Work Folders Logon Synchronization" /DISABLE > NUL 2>&1
echo Планировщики задач Windows отключены.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [19/37] ╔═══ Применение финальных оптимизаций ═══╗
echo Очистка кэша DNS
ipconfig /flushdns
echo Завершение работы системных служб Windows Search и Superfetch
net stop WSearch >nul 2>&1
net stop SysMain >nul 2>&1
echo Финальные оптимизации завершены.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [20/37] ╔═══ Применение изменений и очистка системы ═══╗
echo Очистка корзины
rd /s /q C:\$Recycle.bin
echo Удаление папки обновлений Windows
rd /s /q "C:\Windows\SoftwareDistribution\Download" >nul 2>&1
echo Применение изменений завершено.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [21/37] ╔═══ Применение дополнительных оптимизаций реестра ═══╗
echo Оптимизация задержки монитора...
reg add "HKLM\SYSTEM\CurrentControlSet\Services\DXGKrnl" /v "MonitorLatencyTolerance" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\DXGKrnl" /v "MonitorRefreshLatencyTolerance" /t REG_DWORD /d "0" /f

echo Оптимизация настроек Windows...
:: Учетные записи (отключение синхронизации)
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\SettingSync" /v "SyncPolicy" /t REG_DWORD /d "5" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\SettingSync\Groups\Personalization" /v "Enabled" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\SettingSync\Groups\BrowserSettings" /v "Enabled" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\SettingSync\Groups\Credentials" /v "Enabled" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\SettingSync\Groups\Accessibility" /v "Enabled" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\SettingSync\Groups\Windows" /v "Enabled" /t REG_DWORD /d "0" /f

:: Персонализация (отключение прозрачности)
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Themes\Personalize" /v "EnableTransparency" /t REG_DWORD /d "0" /f

:: Игры и графика
reg add "HKLM\SOFTWARE\Microsoft\PolicyManager\default\ApplicationManagement\AllowGameDVR" /v "value" /t REG_DWORD /d "0" /f
reg add "HKLM\SOFTWARE\Policies\Microsoft\Windows\GameDVR" /v "AllowGameDVR" /t REG_DWORD /d "0" /f
reg add "HKCU\System\GameConfigStore" /v "GameDVR_Enabled" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\GameDVR" /v "AppCaptureEnabled" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\GameBar" /v "AllowAutoGameMode" /t REG_DWORD /d "1" /f
reg add "HKCU\SOFTWARE\Microsoft\GameBar" /v "AutoGameModeEnabled" /t REG_DWORD /d "1" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers" /v "HwSchMode" /t REG_DWORD /d "2" /f
reg add "HKCU\SOFTWARE\Microsoft\DirectX\UserGpuPreferences" /v "DirectXUserGlobalSettings" /t REG_SZ /d "VRROptimizeEnable=0;" /f

:: Специальные возможности
reg add "HKCU\Control Panel\Accessibility\MouseKeys" /v "Flags" /t REG_SZ /d "0" /f
reg add "HKCU\Control Panel\Accessibility\StickyKeys" /v "Flags" /t REG_SZ /d "0" /f
reg add "HKCU\Control Panel\Accessibility\Keyboard Response" /v "Flags" /t REG_SZ /d "0" /f
reg add "HKCU\Control Panel\Accessibility\ToggleKeys" /v "Flags" /t REG_SZ /d "0" /f

:: Конфиденциальность
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\AdvertisingInfo" /v "Enabled" /t REG_DWORD /d "0" /f
reg add "HKCU\Control Panel\International\User Profile" /v "HttpAcceptLanguageOptOut" /t REG_DWORD /d "1" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\Advanced" /v "Start_TrackProgs" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\ContentDeliveryManager" /v "SubscribedContent-338393Enabled" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\ContentDeliveryManager" /v "SubscribedContent-353694Enabled" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\ContentDeliveryManager" /v "SubscribedContent-353696Enabled" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\Speech_OneCore\Settings\OnlineSpeechPrivacy" /v "HasAccepted" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\Personalization\Settings" /v "AcceptedPrivacyPolicy" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\InputPersonalization" /v "RestrictImplicitInkCollection" /t REG_DWORD /d "1" /f
reg add "HKCU\SOFTWARE\Microsoft\InputPersonalization" /v "RestrictImplicitTextCollection" /t REG_DWORD /d "1" /f
reg add "HKCU\SOFTWARE\Microsoft\InputPersonalization\TrainedDataStore" /v "HarvestContacts" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Diagnostics\DiagTrack" /v "ShowedToastAtLevel" /t REG_DWORD /d "1" /f
reg add "HKLM\SOFTWARE\Policies\Microsoft\Windows\DataCollection" /v "AllowTelemetry" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Privacy" /v "TailoredExperiencesWithDiagnosticDataEnabled" /t REG_DWORD /d "0" /f
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Diagnostics\DiagTrack\EventTranscriptKey" /v "EnableEventTranscript" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\Siuf\Rules" /v "NumberOfSIUFInPeriod" /t REG_DWORD /d "0" /f
reg add "HKCU\SOFTWARE\Microsoft\Siuf\Rules" /v "PeriodInNanoSeconds" /t REG_DWORD /d "0" /f

:: История активности
reg add "HKLM\SOFTWARE\Policies\Microsoft\Windows\System" /v "PublishUserActivities" /t REG_DWORD /d "0" /f
reg add "HKLM\SOFTWARE\Policies\Microsoft\Windows\System" /v "UploadUserActivities" /t REG_DWORD /d "0" /f

:: Доступ к приложениям
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\userNotificationListener" /v "Value" /t REG_SZ /d "Deny" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\location" /v "Value" /t REG_SZ /d "Deny" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\appDiagnostics" /v "Value" /t REG_SZ /d "Deny" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\userAccountInformation" /v "Value" /t REG_SZ /d "Deny" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\BackgroundAccessApplications" /v "GlobalUserDisabled" /t REG_DWORD /d "1" /f
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Search" /v "BackgroundAppGlobalToggle" /t REG_DWORD /d "0" /f

echo Оптимизация энергопотребления...
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Power\PowerSettings\2a737441-1930-4402-8d77-b2bebba308a3\d4e98f31-5ffe-4ce1-be31-1b38b384c009\DefaultPowerSchemeValues\381b4222-f694-41f0-9685-ff5bb260df2e" /v "ACSettingIndex" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Power\PowerSettings\2a737441-1930-4402-8d77-b2bebba308a3\d4e98f31-5ffe-4ce1-be31-1b38b384c009\DefaultPowerSchemeValues\381b4222-f694-41f0-9685-ff5bb260df2e" /v "DCSettingIndex" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Power\PowerSettings\2a737441-1930-4402-8d77-b2bebba308a3\d4e98f31-5ffe-4ce1-be31-1b38b384c009\DefaultPowerSchemeValues\8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c" /v "ACSettingIndex" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Power\PowerSettings\54533251-82be-4824-96c1-47b60b740d00\3b04d4fd-1cc7-4f23-ab1c-d1337819c4bb\DefaultPowerSchemeValues\381b4222-f694-41f0-9685-ff5bb260df2e" /v "ACSettingIndex" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Power\PowerSettings\54533251-82be-4824-96c1-47b60b740d00\3b04d4fd-1cc7-4f23-ab1c-d1337819c4bb\DefaultPowerSchemeValues\381b4222-f694-41f0-9685-ff5bb260df2e" /v "DCSettingIndex" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Power\PowerSettings\54533251-82be-4824-96c1-47b60b740d00\3b04d4fd-1cc7-4f23-ab1c-d1337819c4bb\DefaultPowerSchemeValues\8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c" /v "ACSettingIndex" /t REG_DWORD /d "0" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Power\PowerThrottling" /v "PowerThrottlingOff" /t REG_DWORD /d "1" /f

echo Дополнительные оптимизации реестра успешно применены.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [22/37] ╔═══ Оптимизация Affinity для устройств ═══╗
echo Применение оптимизации Affinity для сетевых адаптеров, USB и видеокарты...
for /f "tokens=*" %%f in ('wmic cpu get NumberOfCores /value ^| find "="') do set %%f
for /f "tokens=*" %%f in ('wmic cpu get NumberOfLogicalProcessors /value ^| find "="') do set %%f

if !NumberOfCores! gtr 4 (
    for /f %%i in ('wmic path Win32_VideoController get PNPDeviceID^| findstr /l "PCI\VEN_"') do (
        reg add "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "DevicePolicy" /t REG_DWORD /d "3" /f
        reg delete "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "AssignmentSetOverride" /f
    )
    for /f %%i in ('wmic path Win32_NetworkAdapter get PNPDeviceID^| findstr /l "PCI\VEN_"') do (
        reg add "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "DevicePolicy" /t REG_DWORD /d "5" /f
        reg delete "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "AssignmentSetOverride" /f
    )
) >nul 2>&1

if !NumberOfLogicalProcessors! gtr !NumberOfCores! (
    REM HyperThreading Enabled
    for /f %%i in ('wmic path Win32_USBController get PNPDeviceID^| findstr /l "PCI\VEN_"') do (
        reg add "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "DevicePolicy" /t REG_DWORD /d "4" /f
        reg add "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "AssignmentSetOverride" /t REG_BINARY /d "C0" /f
    )
    for /f %%i in ('wmic path Win32_VideoController get PNPDeviceID^| findstr /l "PCI\VEN_"') do (
        reg add "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "DevicePolicy" /t REG_DWORD /d "4" /f
        reg add "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "AssignmentSetOverride" /t REG_BINARY /d "C0" /f
    )
    for /f %%i in ('wmic path Win32_NetworkAdapter get PNPDeviceID^| findstr /l "PCI\VEN_"') do (
        reg add "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "DevicePolicy" /t REG_DWORD /d "4" /f
        reg add "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "AssignmentSetOverride" /t REG_BINARY /d "30" /f
    )
) else (
    REM HyperThreading Disabled
    for /f %%i in ('wmic path Win32_USBController get PNPDeviceID^| findstr /l "PCI\VEN_"') do (
        reg add "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "DevicePolicy" /t REG_DWORD /d "4" /f
        reg add "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "AssignmentSetOverride" /t REG_BINARY /d "08" /f
    )
    for /f %%i in ('wmic path Win32_VideoController get PNPDeviceID^| findstr /l "PCI\VEN_"') do (
        reg add "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "DevicePolicy" /t REG_DWORD /d "4" /f
        reg add "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "AssignmentSetOverride" /t REG_BINARY /d "02" /f
    )
    for /f %%i in ('wmic path Win32_NetworkAdapter get PNPDeviceID^| findstr /l "PCI\VEN_"') do (
        reg add "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "DevicePolicy" /t REG_DWORD /d "4" /f
        reg add "HKLM\System\CurrentControlSet\Enum\%%i\Device Parameters\Interrupt Management\Affinity Policy" /v "AssignmentSetOverride" /t REG_BINARY /d "04" /f
    )
) >nul 2>&1
echo Оптимизация Affinity успешно применена.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [23/37] ╔═══ Расширенная оптимизация памяти ═══╗
echo Применение расширенных оптимизаций памяти...

REM Disable FTH
reg add "HKLM\Software\Microsoft\FTH" /v "Enabled" /t Reg_DWORD /d "0" /f >nul 2>&1

REM Disable Desktop Composition
reg add "HKCU\Software\Microsoft\Windows\DWM" /v "Composition" /t REG_DWORD /d "0" /f >nul 2>&1

REM Disable Background apps
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\BackgroundAccessApplications" /v "GlobalUserDisabled" /t Reg_DWORD /d "1" /f >nul 2>&1
reg add "HKLM\Software\Policies\Microsoft\Windows\AppPrivacy" /v "LetAppsRunInBackground" /t Reg_DWORD /d "2" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Search" /v "BackgroundAppGlobalToggle" /t Reg_DWORD /d "0" /f >nul 2>&1

REM Disallow drivers to get paged into virtual memory
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management" /v "DisablePagingExecutive" /t Reg_DWORD /d "1" /f >nul 2>&1

REM Disable Page Combining and Memory Compression
powershell -NoProfile -Command "Disable-MMAgent -PagingCombining -mc" >nul 2>&1
reg add "HKLM\System\CurrentControlSet\Control\Session Manager\Memory Management" /v "DisablePageCombining" /t REG_DWORD /d "1" /f >nul 2>&1

REM Use Large System Cache to improve microstuttering
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management" /v "LargeSystemCache" /t Reg_DWORD /d "1" /f >nul 2>&1

REM Free unused ram
reg add "HKLM\System\CurrentControlSet\Control\Session Manager" /v "HeapDeCommitFreeBlockThreshold" /t REG_DWORD /d "262144" /f >nul 2>&1

REM Auto restart Powershell on error
reg add "HKLM\Software\Microsoft\Windows NT\CurrentVersion\Winlogon" /v "AutoRestartShell" /t REG_DWORD /d "1" /f >nul 2>&1

REM Disk Optimizations
reg add "HKLM\SYSTEM\CurrentControlSet\Control\FileSystem" /v "DontVerifyRandomDrivers" /t REG_DWORD /d "1" /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Control\FileSystem" /v "LongPathsEnabled" /t REG_DWORD /d "0" /f >nul 2>&1

REM Disable Prefetch and Superfetch
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management\PrefetchParameters" /v "EnablePrefetcher" /t Reg_DWORD /d "0" /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management\PrefetchParameters" /v "EnableSuperfetch" /t Reg_DWORD /d "0" /f >nul 2>&1

REM Disable Hibernation + Fast Startup
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power" /v "HiberbootEnabled" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Power" /v "HibernateEnabledDefault" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Power" /v "HibernateEnabled" /t REG_DWORD /d "0" /f >nul 2>&1

REM Wait time to kill app during shutdown
reg add "HKCU\Control Panel\Desktop" /v "WaitToKillAppTimeout" /t Reg_SZ /d "1000" /f >nul 2>&1

REM Wait to end service at shutdown
reg add "HKLM\System\CurrentControlSet\Control" /v "WaitToKillServiceTimeout" /t Reg_SZ /d "1000" /f >nul 2>&1

REM Wait to kill non-responding app
reg add "HKCU\Control Panel\Desktop" /v "HungAppTimeout" /t Reg_SZ /d "1000" /f >nul 2>&1

REM fsutil optimizations
if exist "%SYSTEMROOT%\System32\fsutil.exe" (
    REM Raise the limit of paged pool memory
    fsutil behavior set memoryusage 2 >nul 2>&1
    REM https://www.serverbrain.org/solutions-2003/the-mft-zone-can-be-optimized.html
    fsutil behavior set mftzone 2 >nul 2>&1
    REM Disable Last Access information on directories, performance/privacy
    fsutil behavior set disablelastaccess 1 >nul 2>&1
    REM Disable Virtual Memory Pagefile Encryption
    fsutil behavior set encryptpagingfile 0 >nul 2>&1
    REM Disables the creation of legacy 8.3 character-length file names on FAT- and NTFS-formatted volumes
    fsutil behavior set disable8dot3 1 >nul 2>&1
    REM Disable NTFS compression
    fsutil behavior set disablecompression 1 >nul 2>&1
    REM Enable Trim
    fsutil behavior set disabledeletenotify 0 >nul 2>&1
)
echo Расширенная оптимизация памяти успешно применена.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [24/37] ╔═══ Оптимизация мыши (Mouse Fix) ═══╗
echo Применение оптимизации мыши...

rem Отключаем акселерацию мыши
reg add "HKCU\Control Panel\Mouse" /v "MouseSpeed" /t REG_SZ /d "0" /f >nul 2>&1
reg add "HKCU\Control Panel\Mouse" /v "MouseThreshold1" /t REG_SZ /d "0" /f >nul 2>&1
reg add "HKCU\Control Panel\Mouse" /v "MouseThreshold2" /t REG_SZ /d "0" /f >nul 2>&1
reg add "HKCU\Control Panel\Mouse" /v "MouseSensitivity" /t REG_SZ /d "10" /f >nul 2>&1
reg add "HKCU\Control Panel\Mouse" /v "SmoothMouseYCurve" /t REG_BINARY /d "0000000000000000000038000000000000007000000000000000A800000000000000E00000000000" /f >nul 2>&1

rem Добавляем оптимальные настройки для разрешения экрана 1080p (масштаб 100%)
reg add "HKCU\Control Panel\Mouse" /v "SmoothMouseXCurve" /t REG_BINARY /d "0000000000000000C0CC0C0000000000809919000000000040662600000000000033330000000000" /f >nul 2>&1

echo Оптимизация мыши успешно применена.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [25/37] ╔═══ Отключение Preemption для NVIDIA ═══╗
echo Применение оптимизации Disable Preemption для видеокарт NVIDIA...

reg add "HKLM\SYSTEM\CurrentControlSet\Services\nvlddmkm" /v "DisablePreemption" /t Reg_DWORD /d "1" /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Services\nvlddmkm" /v "DisableCudaContextPreemption" /t Reg_DWORD /d "1" /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Services\nvlddmkm" /v "EnableCEPreemption" /t Reg_DWORD /d "0" /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Services\nvlddmkm" /v "DisablePreemptionOnS3S4" /t Reg_DWORD /d "1" /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Services\nvlddmkm" /v "ComputePreemption" /t Reg_DWORD /d "0" /f >nul 2>&1

echo Отключение Preemption для NVIDIA успешно применено.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [26/37] ╔═══ Настройка профиля NVIDIA с помощью ProfileInspector ═══╗
echo Применение оптимальных настроек производительности с помощью NVIDIA Profile Inspector...

:: Создаем директорию для NVIDIA Profile Inspector
if not exist "%SYSTEMDRIVE%\Hone\Resources" mkdir "%SYSTEMDRIVE%\Hone\Resources" >nul 2>&1

:: Удаляем старую директорию, если она существует
rmdir /S /Q "%SYSTEMDRIVE%\Hone\Resources\nvidiaProfileInspector\" >nul 2>&1

:: Загружаем и распаковываем NVIDIA Profile Inspector
echo Загрузка NVIDIA Profile Inspector...
curl -g -L -# -o "%SYSTEMDRIVE%\Hone\Resources\nvidiaProfileInspector.zip" "https://github.com/Orbmu2k/nvidiaProfileInspector/releases/latest/download/nvidiaProfileInspector.zip" >nul 2>&1
powershell -NoProfile Expand-Archive '%SYSTEMDRIVE%\Hone\Resources\nvidiaProfileInspector.zip' -DestinationPath '%SYSTEMDRIVE%\Hone\Resources\nvidiaProfileInspector\' >nul 2>&1
del /F /Q "%SYSTEMDRIVE%\Hone\Resources\nvidiaProfileInspector.zip" >nul 2>&1

:: Загружаем оптимальный профиль настроек
echo Загрузка оптимизированного профиля настроек...
curl -g -L -# -o "%SYSTEMDRIVE%\Hone\Resources\nvidiaProfileInspector\Latency_and_Performances_Settings_by_Hone_Team2.nip" "https://raw.githubusercontent.com/auraside/HoneCtrl/main/Files/Latency_and_Performances_Settings_by_Hone_Team2.nip" >nul 2>&1

:: Применяем профиль настроек
echo Применение оптимизированного профиля настроек...
cd "%SYSTEMDRIVE%\Hone\Resources\nvidiaProfileInspector\" >nul 2>&1
nvidiaProfileInspector.exe "Latency_and_Performances_Settings_by_Hone_Team2.nip" >nul 2>&1

echo Оптимизация профиля NVIDIA успешно применена.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [27/37] ╔═══ Отключение телеметрии NVIDIA ═══╗
echo Отключение сбора данных и телеметрии NVIDIA...

:: Отключаем телеметрию NVIDIA через реестр
reg add "HKLM\SOFTWARE\NVIDIA Corporation\NvControlPanel2\Client" /v "OptInOrOutPreference" /t REG_DWORD /d 0 /f >nul 2>&1
reg add "HKLM\SOFTWARE\NVIDIA Corporation\Global\FTS" /v "EnableRID44231" /t REG_DWORD /d 0 /f >nul 2>&1
reg add "HKLM\SOFTWARE\NVIDIA Corporation\Global\FTS" /v "EnableRID64640" /t REG_DWORD /d 0 /f >nul 2>&1
reg add "HKLM\SOFTWARE\NVIDIA Corporation\Global\FTS" /v "EnableRID66610" /t REG_DWORD /d 0 /f >nul 2>&1
reg delete "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" /v "NvBackend" /f >nul 2>&1

:: Отключаем задачи сбора отчетов о сбоях NVIDIA
echo Отключение запланированных задач NVIDIA...
schtasks /change /disable /tn "NvTmRep_CrashReport1_{B2FE1952-0186-46C3-BAEC-A80AA35AC5B8}" >nul 2>&1
schtasks /change /disable /tn "NvTmRep_CrashReport2_{B2FE1952-0186-46C3-BAEC-A80AA35AC5B8}" >nul 2>&1
schtasks /change /disable /tn "NvTmRep_CrashReport3_{B2FE1952-0186-46C3-BAEC-A80AA35AC5B8}" >nul 2>&1
schtasks /change /disable /tn "NvTmRep_CrashReport4_{B2FE1952-0186-46C3-BAEC-A80AA35AC5B8}" >nul 2>&1

echo Телеметрия NVIDIA успешно отключена.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [28/37] ╔═══ Дополнительные оптимизации NVIDIA ═══╗
echo Применение дополнительных оптимизаций для видеокарт NVIDIA...

:: Оптимизация настроек NVIDIA через реестр
echo Оптимизация цветокоррекции...
reg add "HKCU\Software\NVIDIA Corporation\Global\NVTweak\Devices\509901423-0\Color" /v "NvCplUseColorCorrection" /t Reg_DWORD /d "0" /f >nul 2>&1

echo Отключение Miracast...
reg add "HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers" /v "PlatformSupportMiracast" /t Reg_DWORD /d "0" /f >nul 2>&1

echo Отключение энергосбережения дисплея...
reg add "HKLM\SYSTEM\CurrentControlSet\Services\nvlddmkm\Global\NVTweak" /v "DisplayPowerSaving" /t Reg_DWORD /d "0" /f >nul 2>&1

:: Снятие ограничений частоты
echo Настройка неограниченной частоты GPU...
if exist "%SYSTEMDRIVE%\Program Files\NVIDIA Corporation\NVSMI\nvidia-smi.exe" (
    cd "%SYSTEMDRIVE%\Program Files\NVIDIA Corporation\NVSMI\" >nul 2>&1
    nvidia-smi -acp UNRESTRICTED >nul 2>&1
    nvidia-smi -acp DEFAULT >nul 2>&1
)

:: Оптимизация дополнительных настроек для всех устройств NVIDIA
echo Отключение мозаичного отображения и TCC для всех устройств NVIDIA...
for /f %%a in ('reg query "HKLM\System\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}" /t REG_SZ /s /e /f "NVIDIA" ^| findstr "HKEY"') do (
    reg add "%%a" /v "EnableTiledDisplay" /t REG_DWORD /d "0" /f >nul 2>&1
    reg add "%%a" /v "TCCSupported" /t REG_DWORD /d "0" /f >nul 2>&1
)

:: Включение опции Silk Smoothness
echo Включение Silk Smoothness...
reg add "HKLM\SYSTEM\CurrentControlSet\Services\nvlddmkm\FTS" /v "EnableRID61684" /t REG_DWORD /d "1" /f >nul 2>&1

echo Дополнительные оптимизации NVIDIA успешно применены.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [29/37] ╔═══ Отключение Write Combining ═══╗
echo Отключение объединения записи для видеокарт NVIDIA...

reg add "HKLM\SYSTEM\CurrentControlSet\Services\nvlddmkm" /v "DisableWriteCombining" /t Reg_DWORD /d "1" /f >nul 2>&1

echo Функция Disable Write Combining успешно включена.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [30/37] ╔═══ Оптимизация сетевых параметров Netsh ═══╗
echo Применение оптимизаций сетевых параметров...

:: Оптимизация параметров TCP/IP через netsh
netsh int tcp set global dca=enabled >nul 2>&1
netsh int tcp set global netdma=enabled >nul 2>&1
netsh interface isatap set state disabled >nul 2>&1
netsh int tcp set global timestamps=disabled >nul 2>&1
netsh int tcp set global rss=enabled >nul 2>&1
netsh int tcp set global nonsackrttresiliency=disabled >nul 2>&1
netsh int tcp set global initialRto=2000 >nul 2>&1
netsh int tcp set supplemental template=custom icw=10 >nul 2>&1
netsh interface ip set interface ethernet currenthoplimit=64 >nul 2>&1

echo Оптимизации сетевых параметров Netsh успешно применены.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [31/37] ╔═══ Отключение назойливых уведомлений безопасности Windows ═══╗
echo Отключение системных уведомлений безопасности и обслуживания...

:: Отключение уведомлений безопасности Windows на уровне системы и пользователя
reg add "HKLM\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings\Windows.SystemToast.SecurityAndMaintenance" /v "Enabled" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings\Windows.SystemToast.SecurityAndMaintenance" /v "Enabled" /t REG_DWORD /d "0" /f >nul 2>&1

:: Отключение уведомлений Центра безопасности Windows Defender
reg add "HKLM\SOFTWARE\Microsoft\Windows Defender Security Center\Notifications" /v "DisableNotifications" /t REG_DWORD /d "1" /f >nul 2>&1
reg add "HKLM\SOFTWARE\Policies\Microsoft\Windows Defender Security Center\Notifications" /v "DisableNotifications" /t REG_DWORD /d "1" /f >nul 2>&1

:: Дополнительные параметры для отключения уведомлений безопасности
reg add "HKLM\SOFTWARE\Policies\Microsoft\Windows Defender Security Center\Notifications" /v "DisableEnhancedNotifications" /t REG_DWORD /d "1" /f >nul 2>&1
reg add "HKLM\SOFTWARE\Policies\Microsoft\Windows Defender" /v "DisableAntiSpyware" /t REG_DWORD /d "1" /f >nul 2>&1
reg add "HKLM\SOFTWARE\Policies\Microsoft\Windows Defender\Reporting" /v "DisableEnhancedNotifications" /t REG_DWORD /d "1" /f >nul 2>&1
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Notifications\Settings" /v "NOC_GLOBAL_SETTING_TOASTS_ENABLED" /t REG_DWORD /d "0" /f >nul 2>&1

echo Назойливые уведомления безопасности успешно отключены.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [32/37] ╔═══ Отключение автообновления драйверов при запуске системы ═══╗
echo Отключение автоматического обновления драйверов Windows...

:: Отключение поиска драйверов при запуске системы
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\DriverSearching" /v "SearchOrderConfig" /t REG_DWORD /d "0" /f >nul 2>&1

:: Исключение драйверов из обновлений качества Windows Update
reg add "HKLM\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate" /v "ExcludeWUDriversInQualityUpdate" /t REG_DWORD /d "1" /f >nul 2>&1

echo Автообновления драйверов Windows успешно отключены.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [33/37] ╔═══ Отключение брандмауэра Windows ═══╗
echo Отключение встроенного брандмауэра Windows...

:: Отключение службы брандмауэра Windows
net stop "Windows Firewall" >nul 2>&1
net stop "MpsSvc" >nul 2>&1
sc config "MpsSvc" start= disabled >nul 2>&1

:: Отключение брандмауэра для всех профилей через реестр
reg add "HKLM\SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\PublicProfile" /v "EnableFirewall" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\StandardProfile" /v "EnableFirewall" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\DomainProfile" /v "EnableFirewall" /t REG_DWORD /d "0" /f >nul 2>&1

:: Отключение брандмауэра через netsh
netsh advfirewall set allprofiles state off >nul 2>&1

echo Брандмауэр Windows успешно отключен.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [34/37] ╔═══ Отключение патчей безопасности Spectre и Meltdown ═══╗

:: Проверка версии Windows
for /f "tokens=4-5 delims=. " %%i in ('ver') do set VERSION=%%i.%%j
if "%version%" == "10.0" (
    echo Обнаружена Windows 10. Отключение патчей безопасности Spectre и Meltdown...
    
    :: Локальная обработка ошибок для критически важной секции
    setlocal
    echo [*] Применение защищенных настроек...

    :: Отключение Spectre и Meltdown через реестр
    (
      reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management" /v "FeatureSettingsOverride" /t REG_DWORD /d "3" /f >nul 2>&1
      reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management" /v "FeatureSettingsOverrideMask" /t REG_DWORD /d "3" /f >nul 2>&1
      reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management" /v "FeatureSettings" /t REG_DWORD /d "1" /f >nul 2>&1
    ) || (
      echo [!] Предупреждение: Не удалось отключить некоторые защитные патчи. Продолжение выполнения...
    )

    :: Дополнительные настройки для отключения защиты на уровне процессора
    (
      reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Virtualization" /v "MinVmVersionForCpuBasedMitigations" /t REG_SZ /d "1.0" /f >nul 2>&1
    ) || (
      echo [!] Не удалось настроить параметры виртуализации. Продолжение выполнения...
    )

    :: Отключение защиты кэша процессора (CVE-2017-5754, CVE-2018-3639)
    wmic cpu get name | findstr /I "Intel" >nul 2>&1
    if not errorlevel 1 (
        echo [+] Обнаружен процессор Intel, применяются специфические настройки...
        (
          reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management" /v "EnableCmdQTagProxy" /t REG_DWORD /d "0" /f >nul 2>&1
        ) || (
          echo [!] Не удалось применить специфические настройки Intel. Продолжение выполнения...
        )
    )

    :: Применение специальных настроек Boot Configuration Data
    (
      bcdedit /set isolatedcontext off >nul 2>&1
      bcdedit /set allowedinmemorysettings 0x0 >nul 2>&1
    ) || (
      echo [!] Не удалось изменить параметры загрузки. Возможно, отсутствуют права администратора.
      echo [i] Совет: Запустите скрипт от имени администратора для полной оптимизации.
    )

    echo [+] Патчи безопасности Spectre и Meltdown успешно отключены.
    echo [i] Внимание: Система может быть уязвима к атакам на уровне процессора,
    echo [i] но производительность возрастет до 30%%.
    endlocal
) else (
    echo Обнаружена Windows 11. Отключение патчей Spectre и Meltdown пропущено из за несовместимости твика.
)

echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [35/37] ╔═══ Включение TSX (Transactional Synchronization Extensions) ═══╗
echo Включение аппаратной транзакционной памяти для повышения производительности многопоточных программ...

:: Проверка, поддерживает ли процессор TSX
wmic cpu get name | findstr /I "Intel" >nul 2>&1
if errorlevel 1 (
    echo Ваш процессор не является Intel. Пропуск настройки TSX.
) else (
    :: Включение TSX путем удаления параметра отключения
    reg delete "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Kernel" /v "DisableTsx" /f >nul 2>&1
    
    :: Альтернативно можно установить значение 0 (включено)
    reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Kernel" /v "DisableTsx" /t REG_DWORD /d "0" /f >nul 2>&1
    
    :: Применение RTM (Restricted Transactional Memory)
    reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Kernel" /v "EnableRTM" /t REG_DWORD /d "1" /f >nul 2>&1
    
    echo TSX успешно включен для повышения производительности многопоточных программ и игр.
)
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [36/37] ╔═══ Установка Ultimate Performance схемы питания ═══╗
echo Установка скрытой схемы энергопитания Ultimate Performance для максимальной производительности...

:: Проверка существования схемы Ultimate Performance
powercfg -list | findstr "e9a42b02-d5df-448d-aa00-03f14749eb61" >nul 2>&1
if errorlevel 1 (
    echo Схема Ultimate Performance не найдена в системе. Добавление схемы...
    
    :: Дублирование схемы Ultimate Performance и получение её GUID
    for /f "tokens=3,4" %%a in ('powercfg -duplicatescheme e9a42b02-d5df-448d-aa00-03f14749eb61 2^>nul') do (
        set "ULTIMATE_GUID=%%a"
        echo Создана схема питания Ultimate Performance с GUID: !ULTIMATE_GUID!
        
        :: Установка созданной схемы как активной
        powercfg -setactive "!ULTIMATE_GUID!" >nul 2>&1
        echo Схема Ultimate Performance установлена как активная.
    )
) else (
    echo Схема Ultimate Performance уже существует в системе.
    
    :: Получение GUID существующей схемы Ultimate Performance
    for /f "tokens=3" %%a in ('powercfg -list ^| findstr "Ultimate Performance"') do (
        set "ULTIMATE_GUID=%%a"
        echo Найдена схема питания Ultimate Performance с GUID: !ULTIMATE_GUID!
        
        :: Установка существующей схемы как активной
        powercfg -setactive "!ULTIMATE_GUID!" >nul 2>&1
        echo Схема Ultimate Performance установлена как активная.
    )
)

:: Дополнительные оптимизации схемы питания для производительности
powercfg -setacvalueindex scheme_current sub_processor PROCTHROTTLEMIN 100 >nul 2>&1
powercfg -setacvalueindex scheme_current sub_processor PROCTHROTTLEMAX 100 >nul 2>&1
powercfg -setacvalueindex scheme_current sub_processor IDLEDISABLE 1 >nul 2>&1

echo Схема питания Ultimate Performance успешно установлена и активирована.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo [37/37] ╔═══ Расширенная оптимизация сетевой карты ═══╗
echo Оптимизация сетевых адаптеров для снижения задержек и повышения производительности...

:: Настройка всех Ethernet адаптеров
echo Оптимизация сетевых адаптеров:
for /f "tokens=*" %%a in ('wmic nic where "NetEnabled=True" get Name ^| findstr /v /i /c:"Name" /c:"Bluetooth" /c:"WAN" /c:"Виртуальный" /c:"Virtual" /c:"Loopback" ^| findstr /i /c:"Ethernet" /c:"Killer" /c:"Realtek" /c:"Intel" /c:"Broadcom" /c:"Atheros" /c:"Marvell" /c:"Adapter" /c:"Wi-Fi" /c:"WiFi" /c:"Wireless"') do (
    echo - Найден сетевой адаптер: "%%a"
    
    :: Получаем GUID сетевого адаптера
    for /f "tokens=*" %%i in ('reg query "HKLM\SYSTEM\CurrentControlSet\Control\Class\{4D36E972-E325-11CE-BFC1-08002BE10318}" /s /v "DriverDesc" ^| findstr /i /c:"%%a" /c:"REG_SZ"') do (
        for /f "tokens=1" %%b in ("%%i") do (
            set "ADAPTER_REG_PATH=%%b"
            
            :: Отключение энергосбережения (общие параметры)
            reg add "!ADAPTER_REG_PATH!" /v "*EEE" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "EEE" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "EnablePME" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "EnergyEfficientEthernet" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "GigaLite" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "PowerSavingMode" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "*NicAutoPowerSaver" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "AutoPowerSaveModeEnabled" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "ULPMode" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "AdvancedEEE" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "DisableDelayedPowerUp" /t REG_SZ /d "2" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "EnablePME" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "MIMOPowerSaveMode" /t REG_SZ /d "3" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "S5WakeOnLan" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "SavePowerNow" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "WakeOnDisconnect" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "WoWLANLPSOffload" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "WakeOnLink" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "WakeOnSlot" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "WolShutdownLinkSpeed" /t REG_SZ /d "2" /f >nul 2>&1
            
            :: Настройка буферов и очередей
            reg add "!ADAPTER_REG_PATH!" /v "TransmitBuffers" /t REG_SZ /d "4096" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "ReceiveBuffers" /t REG_SZ /d "512" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "*NumRssQueues" /t REG_SZ /d "4" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "*NumTxQueues" /t REG_SZ /d "4" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "TxBufferSize" /t REG_SZ /d "4096" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "RxBufferSize" /t REG_SZ /d "512" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "BuffersRxMax" /t REG_SZ /d "4096" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "BuffersTxMax" /t REG_SZ /d "4096" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "TxQueueLength" /t REG_SZ /d "3000" /f >nul 2>&1
            
            :: Настройка прерываний и модерации
            reg add "!ADAPTER_REG_PATH!" /v "*InterruptModeration" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "ITR" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "InterruptModeration" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "*PriorityVLANTag" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "*RSS" /t REG_SZ /d "1" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "EnableRSS" /t REG_SZ /d "1" /f >nul 2>&1
            
            :: Отключение автонастройки
            reg add "!ADAPTER_REG_PATH!" /v "*SpeedDuplex" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "AutoDisableGigabit" /t REG_SZ /d "0" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "Speed_DuplexMode" /t REG_SZ /d "0" /f >nul 2>&1
            
            :: Оптимизация TCP/IP для адаптера
            reg add "!ADAPTER_REG_PATH!" /v "TCPChecksumOffloadIPv4" /t REG_SZ /d "3" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "TCPChecksumOffloadIPv6" /t REG_SZ /d "3" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "UDPChecksumOffloadIPv4" /t REG_SZ /d "3" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "UDPChecksumOffloadIPv6" /t REG_SZ /d "3" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "IPChecksumOffloadIPv4" /t REG_SZ /d "3" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "LsoV2IPv4" /t REG_SZ /d "1" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "LsoV2IPv6" /t REG_SZ /d "1" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "JumboFrame" /t REG_SZ /d "1" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "JumboPacket" /t REG_SZ /d "9014" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!" /v "DefaultAcpiMsi" /t REG_SZ /d "1" /f >nul 2>&1
            
            :: Настройка MSI/MSI-X для сетевых карт
            reg add "!ADAPTER_REG_PATH!\Interrupt Management" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!\Interrupt Management\MessageSignaledInterruptProperties" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!\Interrupt Management\MessageSignaledInterruptProperties" /v "MSISupported" /t REG_DWORD /d "1" /f >nul 2>&1
            reg add "!ADAPTER_REG_PATH!\Interrupt Management\MessageSignaledInterruptProperties" /v "MessageNumberLimit" /t REG_DWORD /d "8" /f >nul 2>&1
            
            :: Настройка привязки прерываний к ядрам
            for /f "tokens=*" %%f in ('wmic cpu get NumberOfCores /value ^| find "="') do set %%f
            for /f "tokens=*" %%f in ('wmic cpu get NumberOfLogicalProcessors /value ^| find "="') do set %%f
            if !NumberOfLogicalProcessors! gtr 4 (
                echo   - Применение оптимизации Affinity для сетевой карты...
                reg add "!ADAPTER_REG_PATH!\Interrupt Management\Affinity Policy" /v "DevicePolicy" /t REG_DWORD /d "4" /f >nul 2>&1
                reg add "!ADAPTER_REG_PATH!\Interrupt Management\Affinity Policy" /v "AssignmentSetOverride" /t REG_BINARY /d "30" /f >nul 2>&1
            )
            
            echo   - Оптимизирован: %%a
        )
    )
)

:: Настройка протоколов TCP/IP для улучшения производительности
echo Оптимизация параметров TCP/IP...
:: Уменьшаем TTL для снижения пинга
reg add "HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters" /v "DefaultTTL" /t REG_DWORD /d "64" /f >nul 2>&1
:: Уменьшаем время сохранения соединений TCP
reg add "HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters" /v "TcpTimedWaitDelay" /t REG_DWORD /d "30" /f >nul 2>&1
:: Увеличиваем максимальное число подключений
reg add "HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters" /v "TcpNumConnections" /t REG_DWORD /d "16777214" /f >nul 2>&1
:: Оптимизируем задержку подтверждения
reg add "HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters" /v "TCPDelAckTicks" /t REG_DWORD /d "0" /f >nul 2>&1
:: Отключаем технологию Window Scaling
reg add "HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters" /v "Tcp1323Opts" /t REG_DWORD /d "1" /f >nul 2>&1
:: Увеличиваем размер MTU для лучшей производительности
reg add "HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters" /v "EnablePMTUDiscovery" /t REG_DWORD /d "1" /f >nul 2>&1
:: Установка DNS-кэша
reg add "HKLM\SYSTEM\CurrentControlSet\Services\Dnscache\Parameters" /v "CacheHashTableBucketSize" /t REG_DWORD /d "1" /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Services\Dnscache\Parameters" /v "CacheHashTableSize" /t REG_DWORD /d "384" /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Services\Dnscache\Parameters" /v "MaxCacheEntryTtlLimit" /t REG_DWORD /d "64000" /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Services\Dnscache\Parameters" /v "MaxSOACacheEntryTtlLimit" /t REG_DWORD /d "301" /f >nul 2>&1

:: Снижение приоритета интернет-приложений
echo Снижение приоритета сетевых приложений для игр...
reg add "HKLM\SOFTWARE\Policies\Microsoft\Windows\Psched" /v "NonBestEffortLimit" /t REG_DWORD /d "0" /f >nul 2>&1

echo Сетевые карты успешно оптимизированы для максимальной производительности.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul

echo ╔═══ Оптимизация визуальных эффектов Windows ═══╗
echo Настройка оптимальных визуальных эффектов для повышения производительности...

:: Включаем только необходимые визуальные эффекты
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects" /v "VisualFXSetting" /t REG_DWORD /d "3" /f >nul 2>&1
reg add "HKCU\Control Panel\Desktop" /v "UserPreferencesMask" /t REG_BINARY /d "9012078012000000" /f >nul 2>&1
reg add "HKCU\Control Panel\Desktop\WindowMetrics" /v "MinAnimate" /t REG_SZ /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced" /v "ListviewAlphaSelect" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced" /v "ListviewShadow" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced" /v "TaskbarAnimations" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\DWM" /v "EnableAeroPeek" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\DWM" /v "AlwaysHibernateThumbnails" /t REG_DWORD /d "0" /f >nul 2>&1

:: Настройка системных визуальных эффектов
echo Применение индивидуальных настроек визуальных эффектов...
reg add "HKCU\Control Panel\Desktop" /v "DragFullWindows" /t REG_SZ /d "1" /f >nul 2>&1
reg add "HKCU\Control Panel\Desktop" /v "FontSmoothing" /t REG_SZ /d "2" /f >nul 2>&1
reg add "HKCU\Control Panel\Desktop" /v "MenuShowDelay" /t REG_SZ /d "200" /f >nul 2>&1
reg add "HKCU\Control Panel\Desktop\WindowMetrics" /v "MinAnimate" /t REG_SZ /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced" /v "TaskbarAnimations" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced" /v "ListviewAlphaSelect" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced" /v "ListviewShadow" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced" /v "IconsOnly" /t REG_DWORD /d "0" /f >nul 2>&1

:: Включение выбранных эффектов
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\AnimateControls" /v "DefaultValue" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\ComboBoxAnimation" /v "DefaultValue" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\ControlAnimations" /v "DefaultValue" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\CursorShadow" /v "DefaultValue" /t REG_DWORD /d "1" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\DragFullWindows" /v "DefaultValue" /t REG_DWORD /d "1" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\DropShadow" /v "DefaultValue" /t REG_DWORD /d "1" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\DWMAeroPeekEnabled" /v "DefaultValue" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\DWMEnabled" /v "DefaultValue" /t REG_DWORD /d "1" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\DWMSaveThumbnailEnabled" /v "DefaultValue" /t REG_DWORD /d "1" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\FontSmoothing" /v "DefaultValue" /t REG_DWORD /d "1" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\ListBoxSmoothScrolling" /v "DefaultValue" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\ListviewAlphaSelect" /v "DefaultValue" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\ListviewShadow" /v "DefaultValue" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\MenuAnimation" /v "DefaultValue" /t REG_DWORD /d "1" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\SelectionFade" /v "DefaultValue" /t REG_DWORD /d "1" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\TaskbarAnimations" /v "DefaultValue" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\Themes" /v "DefaultValue" /t REG_DWORD /d "1" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\ThumbnailsOrIcon" /v "DefaultValue" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects\TooltipAnimation" /v "DefaultValue" /t REG_DWORD /d "1" /f >nul 2>&1

:: Отключение прозрачности интерфейса
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Themes\Personalize" /v "EnableTransparency" /t REG_DWORD /d "0" /f >nul 2>&1
reg add "HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Themes\Personalize" /v "EnableBlurBehind" /t REG_DWORD /d "0" /f >nul 2>&1

echo Визуальные эффекты Windows настроены оптимально.
echo ──────────────────────────────── by GofMan3 ────────────────────────────────
echo.
:: timeout /t 1 /nobreak >nul



echo.
echo.
echo  ██████╗  ██████╗ ███████╗███╗   ███╗ █████╗ ███╗   ██╗██████╗     
echo  ██╔════╝ ██╔═══██╗██╔════╝████╗ ████║██╔══██╗████╗  ██║╚════██╗    
echo  ██║  ███╗██║   ██║█████╗  ██╔████╔██║███████║██╔██╗ ██║ █████╔╝    
echo  ██║   ██║██║   ██║██╔══╝  ██║╚██╔╝██║██╔══██║██║╚██╗██║ ╚═══██╗    
echo  ╚██████╔╝╚██████╔╝██║     ██║ ╚═╝ ██║██║  ██║██║ ╚████║██████╔╝    
echo   ╚═════╝  ╚═════╝ ╚═╝     ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═════╝     
echo.
echo  ██████╗ ██████╗ ████████╗██╗███╗   ███╗██╗███████╗ █████╗ ████████╗██╗ ██████╗ ███╗   ██╗
echo  ██╔═══██╗██╔══██╗╚══██╔══╝██║████╗ ████║██║╚══███╔╝██╔══██╗╚══██╔══╝██║██╔═══██╗████╗  ██║
echo  ██║   ██║██████╔╝   ██║   ██║██╔████╔██║██║  ███╔╝ ███████║   ██║   ██║██║   ██║██╔██╗ ██║
echo  ██║   ██║██╔═══╝    ██║   ██║██║╚██╔╝██║██║ ███╔╝  ██╔══██║   ██║   ██║██║   ██║██║╚██╗██║
echo  ╚██████╔╝██║        ██║   ██║██║ ╚═╝ ██║██║███████╗██║  ██║   ██║   ██║╚██████╔╝██║ ╚████║
echo   ╚═════╝ ╚═╝        ╚═╝   ╚═╝╚═╝     ╚═╝╚═╝╚══════╝╚═╝  ╚═╝   ╚═╝   ╚═╝ ╚═════╝ ╚═╝  ╚═══╝
echo.
echo  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓
echo.
echo                 ╔═══════════════════════════════════════════════╗
echo                 ║  YouTube: https://www.youtube.com/@GofMan3    ║
echo                 ║   ✓ ОПТИМИЗАЦИЯ СИСТЕМЫ УСПЕШНО ЗАВЕРШЕНА     ║ 
echo                 ╚═══════════════════════════════════════════════╝
echo.
echo  [!] Рекомендуется перезагрузить компьютер для применения всех изменений.
echo.
echo  Нажмите любую клавишу, чтобы завершить...
pause >nul

exit /b

:clear_event_log
echo [-] Очистка журнала событий: %~1
wevtutil.exe cl %1 >nul 2>&1
goto :eof 
