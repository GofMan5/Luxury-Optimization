# Аудит переноса старого BAT

Исходник: `GofMan3 - optimization.bat`, 1 224 строки, 37 заявленных этапов плюс ненумерованный блок визуальных эффектов. Он выполнял всё подряд, скрывал почти все ошибки и всегда показывал success banner.

| Старый этап | Решение в Go |
|---|---|
| 1. Temp/Prefetch/Event Logs | Заменён отдельной безопасной очисткой файлов старше 48 часов. Prefetch, Event Logs, Windows Update и корзина исключены. |
| 2. HPET/BCD timers | Не перенесён: debug-параметры, аппаратно-зависимый результат. Аудит только сообщает нестандартный BCD. |
| 3. Отключение SysMain | Не перенесён: ухудшает часть систем и зависит от накопителя/нагрузки. |
| 4. USB registry/ACL | Не перенесён: массовая запись в `Enum`, сломанная WMIC-логика и отсутствующий SetACL. Максимальная схема меняет только штатные AC USB power settings. |
| 5. Массовые registry tweaks | Оставлены только Game Mode и документированные user preferences. MMCSS/private GPU/Lanman/kill-timeout значения удалены. |
| 6. Memory Compression off | Не перенесён: Windows управляет этим динамически. |
| 7–8. HIPM/DIPM/IoLatencyCap scan | Не перенесён: массовая driver-registry запись без модели устройства. |
| 9. Process Mitigations off | Удалён как критически опасный. |
| 10. StorPort idle off | Не перенесён: storage-driver trade-off. |
| 11. Game Bar/DVR/FSE | Перенесено только отключение фонового capture; FSE mappings не удаляются. |
| 12. Старые memory-manager keys | Не перенесены; CFG/SEHOP/Spectre overrides входят только в ремонт старого BAT. |
| 13. OpenGL keys/opengl32.dll | Удалён. Попытка `setvaliddata` системной DLL опасна; аномальный размер обнаруживается аудитом. |
| 14. `GPU_MAX_*` | Удалены: это не универсальные игровые GPU-настройки. |
| 15. RuntimeBroker rename | Удалён. Аудит обнаруживает пропажу и рекомендует DISM/SFC. |
| 16 и 30. TCP/netsh | Удалены: конфликтовали между собой, часть команд устарела. |
| 17. CSRSS priority | Удалён из оптимизации; добавлен в диагностику и обратимый ремонт. |
| 18. Scheduled Tasks off | Не перенесён: ломает обслуживание Windows и диагностику. |
| 19. Flush DNS/stop services | Не переносится как FPS-оптимизация. |
| 20. Recycle/Windows Update deletion | Не перенесён: необратимо и не ускоряет игры. |
| 21. Privacy/UI/Game Mode/HAGS | Game Mode и лёгкие UI-настройки перенесены. Privacy/accessibility/HAGS оставлены пользователю. |
| 22. Fixed IRQ affinity | Удалён: маски не учитывали topology, hybrid CPU и processor groups. |
| 23. Memory/NTFS/fsutil | Не перенесён: устаревшие или workload-specific значения. |
| 24. Mouse Fix | Перенесено только отключение acceleration. Sensitivity и 1080p curves не навязываются. |
| 25, 28, 29. Private NVIDIA keys | Удалены из оптимизации; известные опасные значения доступны только для удаления в repair-профиле. |
| 26. Download NPI `latest` + raw profile | Удалён: elevated mutable supply chain без hash/signature; профильный URL уже отдаёт 404. |
| 27. NVIDIA telemetry/tasks | Не переносится как производительность. |
| 31. Security notifications/Defender off | Удалён как критически опасный. |
| 32. Driver updates off | Не переносится: это политика обслуживания, а не FPS. |
| 33. Firewall off | Заменён диагностикой и repair-профилем, который включает Firewall. |
| 34. Spectre/Meltdown off | Удалён как критически опасный; repair удаляет overrides. |
| 35. TSX/RTM by brand name | Удалён: наличие Intel в строке не доказывает поддержку или безопасность. |
| 36. Ultimate power | Переписан: создаётся отдельная схема, проверяется GUID, текущая схема сохраняется и возвращается при rollback. CPU idle не отключается, min CPU 100% не задаётся. |
| 37. NIC registry block | Переписан через `Get/Set-NetAdapterAdvancedProperty`: только физический Ethernet и только объявленные драйвером EEE/Interrupt Moderation. Jumbo/MSI/offloads/buffers/affinity не навязываются. |
| 38. Visual effects | Перенесены прозрачность и две лёгкие анимации; остальные пользовательские эффекты не трогаются. |

## Исправленные системные ошибки

- Нет сломанного VBS UAC и некавыченного пути с пробелами.
- Нет `wmic` и `deltree`, удалённых из современных Windows.
- Системные EXE вызываются по абсолютному пути с массивом аргументов; current-directory path hijack исключён.
- Нет mutable runtime downloads и elevated запуска непроверенных файлов.
- Нет локализованного парсинга названий power schemes: используется GUID.
- Нет fixed core/IRQ masks и проверки производителя по маркетинговой строке.
- Нет конфликтующих повторных registry writes.
- Success выводится только после повторного чтения изменённого состояния.

## Внешние артефакты

Исходный BAT не вызывает ни один вложенный CRU/ParkControl/HIDUSBF/GTA/ReShade файл. Они исключены из нового core:

- CRU 1.5.2 и helpers — unsigned и устарели;
- ParkControl/UnparkCPU дублируют штатное управление питанием;
- HIDUSBF — отдельный kernel driver без полного современного пакета/`.cat`, несовместимый с ARM64 и потенциально с Memory Integrity/Secure Boot;
- GTA XML жёстко заданы под RTX 3060, конкретные разрешения/герцовки;
- launch-параметры содержат отсутствующий `gofman3.cfg`, фиксированные `threads/refresh`, опечатку и отключение anti-cheat;
- ReShade/GTA assets не образуют полный воспроизводимый мод-пакет.

Если дополнительные инструменты понадобятся позже, их следует выпускать отдельными opt-in модулями с фиксированной версией, официальным источником, лицензией, SHA-256, ожидаемым signer и собственной процедурой rollback.
