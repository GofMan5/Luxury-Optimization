# GofMan3 Optimization Pack

Windows-утилита для игровой и системной оптимизации с точным планом, резервной копией, проверкой результата и откатом. Продукт больше не содержит ремонт старых BAT-твиков, security-настройки, сторонние драйверы или игровые моды.

В 3.1 добавлена временная игровая boost-сессия: оптимизатор повышает профиль через UAC, запускает игру без прав администратора и автоматически возвращает исходные настройки после выхода, ошибки запуска или Ctrl+C.

## Профили

### Рекомендуемый

- включает Windows Game Mode;
- отключает фоновую запись Game DVR;
- отключает ускорение мыши и сразу применяет live-параметры через Windows API;
- убирает прозрачность и лишние анимации интерфейса.

### Максимальная производительность

Включает рекомендуемый профиль и дополнительно:

- создаёт отдельную обратимую power-схему;
- применяет только поддерживаемые CPU EPP/Boost параметры;
- отключает AC-энергосбережение PCIe/USB;
- отключает только объявленные Ethernet-драйвером EEE/Interrupt Moderation;
- убирает задержку автозагрузки и открытия меню.

Максимальный профиль увеличивает нагрев и энергопотребление. На ноутбуке программа показывает отдельное предупреждение.

## Надёжность

Каждая изменяющая операция проходит цепочку:

`plan → backup → apply → read-back verify → rollback on error`

- системные EXE запускаются по доверенным абсолютным путям;
- PowerShell получает allowlist-окружение;
- backup защищён ACL и SHA-256 seal в HKLM;
- power, Ethernet, registry и live mouse-состояние проверяются после применения;
- `restore` возвращает последнюю операцию;
- `boost` блокирует параллельные `apply/restore` до завершения игры и выполняет откат до снятия блокировки;
- `clean` работает без elevation и не следует по reparse points.

## Что намеренно отсутствует

- отключение Defender, Firewall, mitigations, служб и Windows Update;
- HPET/BCD/timer-resolution рецепты;
- memory cleaners, fixed affinity/IRQ masks и private GPU registry keys;
- HAGS/MSI/TSX настройки без измерения на конкретном ПК;
- сторонние EXE, DLL, драйверы и моды.

Решения по каждому классу твиков записываются в [docs/NOTES.md](docs/NOTES.md), изменения версий — в [docs/CHANGELOG.md](docs/CHANGELOG.md).

## CLI

```powershell
GofMan3-Optimizer-amd64.exe audit --json
GofMan3-Optimizer-amd64.exe plan --profile recommended
GofMan3-Optimizer-amd64.exe plan --profile maximum
GofMan3-Optimizer-amd64.exe apply --profile recommended
GofMan3-Optimizer-amd64.exe apply --profile maximum
GofMan3-Optimizer-amd64.exe boost --game "C:\Games\Game\Game.exe" --profile maximum -- -windowed
GofMan3-Optimizer-amd64.exe restore
GofMan3-Optimizer-amd64.exe clean --days 2
```

Без аргументов запускается mouse-first TUI. `audit` и `plan` ничего не меняют.

## Сборка и проверки

Нужен Go 1.25+.

```powershell
.\build.ps1 -Version 3.1.0
go test ./...
go test -race ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
```

Build-скрипт транзакционно публикует `amd64`, `arm64`, `386` и `SHA256SUMS.txt` в `dist`.
