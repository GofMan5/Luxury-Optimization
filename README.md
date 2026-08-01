# GofMan3 Optimization Pack

Windows-утилита для игровой и системной оптимизации с точным планом, резервной копией, проверкой результата и откатом. Продукт больше не содержит ремонт старых BAT-твиков, security-настройки, сторонние драйверы или игровые моды.

Временная игровая boost-сессия повышает профиль через UAC, запускает игру без прав администратора и автоматически возвращает исходные настройки после выхода, ошибки запуска или Ctrl+C. Версия 3.2 добавляет обнаружение Steam/Epic/Xbox, сохраняемые per-game профили, process-scoped priority/affinity и обратимое управление HKCU-автозагрузкой.

## Профили

### Рекомендуемый

- включает Windows Game Mode;
- отключает фоновую запись Game DVR;
- отключает ускорение мыши и сразу применяет live-параметры через Windows API;
- убирает прозрачность и лишние анимации интерфейса.

### Максимальная производительность

Включает рекомендуемый профиль и дополнительно:

- создаёт отдельную обратимую power-схему;
- оставляет CPU диапазон 5–100% вместо постоянной минимальной частоты 100%;
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
- `games.json` публикуется атомарно с ACL текущего пользователя и повторно валидируется перед запуском;
- startup-команда сначала переносится в backup, затем удаляется и проверяется; enable восстанавливает исходный `REG_SZ`/`REG_EXPAND_SZ`;
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
GofMan3-Optimizer-amd64.exe boost --game "C:\Games\Game\Game.exe" --profile maximum --priority above-normal -- -windowed
GofMan3-Optimizer-amd64.exe games scan --json
GofMan3-Optimizer-amd64.exe games add --path "C:\Games\Game\Game.exe" --name "Game" --profile maximum --priority above-normal -- -windowed
GofMan3-Optimizer-amd64.exe games list
GofMan3-Optimizer-amd64.exe games run --id 0123456789ab
GofMan3-Optimizer-amd64.exe startup list --json
GofMan3-Optimizer-amd64.exe startup disable --name "Unused Launcher"
GofMan3-Optimizer-amd64.exe startup enable --name "Unused Launcher"
GofMan3-Optimizer-amd64.exe services list --state running --json
GofMan3-Optimizer-amd64.exe network interfaces --json
GofMan3-Optimizer-amd64.exe network test --address 1.1.1.1:443 --count 10
GofMan3-Optimizer-amd64.exe benchmark template
GofMan3-Optimizer-amd64.exe benchmark compare --before before.json --after after.json
GofMan3-Optimizer-amd64.exe backups list --json     # терминал с правами администратора
GofMan3-Optimizer-amd64.exe restore --id 20260801T010203.123456789Z
GofMan3-Optimizer-amd64.exe restore
GofMan3-Optimizer-amd64.exe clean --days 2
```

Без аргументов запускается mouse-first TUI. `audit` и `plan` ничего не меняют.

## Сборка и проверки

Нужен Go 1.25+.

```powershell
.\build.ps1 -Version 3.2.0
go test ./...
go test -race ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
```

Build-скрипт транзакционно публикует `amd64`, `arm64`, `386` и `SHA256SUMS.txt` в `dist`.

Полное сравнение с BoosterX и решения по каждому классу функций: [docs/BOOSTERX-COVERAGE.md](docs/BOOSTERX-COVERAGE.md). Закрытие 15 независимых проходов и review-of-review: [docs/REVIEW-15.md](docs/REVIEW-15.md).
