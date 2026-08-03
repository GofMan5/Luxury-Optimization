package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	args, resultPath := stripResultFile(os.Args[1:])
	if (len(args) == 0 || args[0] != "update") && internalParentPID(args) == 0 {
		maybeAutoUpdate()
	}
	err := run(args)
	if resultPath != "" {
		if parentPID := internalParentPID(args); parentPID != 0 {
			_ = writeElevatedResult(resultPath, parentPID, err)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка:", displayText(err.Error()))
		os.Exit(1)
	}
}

func stripResultFile(args []string) ([]string, string) {
	clean := make([]string, 0, len(args))
	result := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--result-file" && i+1 < len(args) {
			result = args[i+1]
			i++
			continue
		}
		clean = append(clean, args[i])
	}
	return clean, result
}

func internalParentPID(args []string) uint32 {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--parent-pid" {
			value, err := strconv.ParseUint(args[i+1], 10, 32)
			if err == nil {
				return uint32(value)
			}
		}
	}
	return 0
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "tui" {
		return runTUI()
	}
	switch args[0] {
	case "audit":
		return auditCommand(args[1:])
	case "plan":
		return planCommand(args[1:])
	case "apply":
		return applyCommand(args[1:])
	case "restore":
		return restoreCommand(args[1:])
	case "boost":
		return boostCommand(args[1:])
	case "clean":
		return cleanCommand(args[1:])
	case "startup":
		return startupCommand(args[1:])
	case "games":
		return gamesCommand(args[1:])
	case "services":
		return servicesCommand(args[1:])
	case "network":
		return networkCommand(args[1:])
	case "benchmark":
		return benchmarkCommand(args[1:])
	case "backups":
		return backupsCommand(args[1:])
	case "update":
		return updateCommand(args[1:])
	case "version", "--version", "-v":
		fmt.Println(productName, version)
		return nil
	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		return fmt.Errorf("неизвестная команда %q; используйте help", args[0])
	}
}

func auditCommand(args []string) error {
	set := flag.NewFlagSet("audit", flag.ContinueOnError)
	out := set.String("out", "", "сохранить JSON в файл")
	jsonOnly := set.Bool("json", false, "вывести JSON")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы audit")
	}
	audit := collectAudit()
	data, err := json.MarshalIndent(audit, "", "  ")
	if err != nil {
		return err
	}
	if *out != "" {
		if err := os.WriteFile(*out, data, 0o600); err != nil {
			return err
		}
		fmt.Println("Отчёт сохранён:", displayText(*out))
	}
	if *jsonOnly {
		fmt.Println(string(data))
		return nil
	}
	printAudit(audit)
	return nil
}

func planCommand(args []string) error {
	set := flag.NewFlagSet("plan", flag.ContinueOnError)
	profileID := set.String("profile", profileRecommended, "recommended или maximum")
	jsonOnly := set.Bool("json", false, "вывести JSON")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы plan")
	}
	plan, err := buildPlan(*profileID)
	if err != nil {
		return err
	}
	if *jsonOnly {
		data, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	printPlan(plan)
	return nil
}

func applyCommand(args []string) error {
	set := flag.NewFlagSet("apply", flag.ContinueOnError)
	profileID := set.String("profile", profileRecommended, "recommended или maximum")
	yes := set.Bool("yes", false, "подтвердить")
	quiet := set.Bool("quiet", false, "не печатать результат")
	parentPID := set.Uint("parent-pid", 0, "internal: PID исходного процесса")
	boostSession := set.Bool("boost-session", false, "internal: активная игровая сессия")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы apply")
	}
	if *boostSession && (*parentPID == 0 || !isAdministrator()) {
		return errors.New("boost-session разрешён только elevated-процессу с parent-pid")
	}
	if *profileID != profileRecommended && *profileID != profileMaximum {
		return errors.New("apply поддерживает только recommended и maximum")
	}
	if _, err := profileByID(*profileID); err != nil {
		return err
	}
	if *parentPID != 0 {
		if !isAdministrator() {
			return errors.New("parent-pid разрешён только elevated-процессу")
		}
		sid, err := userSIDFromOptimizerProcess(uint32(*parentPID))
		if err != nil {
			return err
		}
		if err := setRegistryUserSID(sid); err != nil {
			return err
		}
	}
	if !*yes && !confirm("Применить профиль "+*profileID+"? Будет создана резервная копия") {
		return errors.New("операция отменена")
	}
	if !isAdministrator() {
		return runElevatedAndWait([]string{"apply", "--profile", *profileID, "--yes", "--quiet", "--parent-pid", strconv.Itoa(os.Getpid())})
	}
	path, err := applyProfile(*profileID, *boostSession)
	if err != nil {
		return err
	}
	if !*quiet {
		fmt.Println("Готово. Резервная копия:", displayText(path))
	}
	return nil
}

func restoreCommand(args []string) error {
	set := flag.NewFlagSet("restore", flag.ContinueOnError)
	yes := set.Bool("yes", false, "подтвердить")
	quiet := set.Bool("quiet", false, "не печатать результат")
	parentPID := set.Uint("parent-pid", 0, "internal: PID исходного процесса")
	boostSession := set.Bool("boost-session", false, "internal: активная игровая сессия")
	backupID := set.String("id", "", "ID из backups list")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы restore")
	}
	if *boostSession && (*parentPID == 0 || !isAdministrator()) {
		return errors.New("boost-session разрешён только elevated-процессу с parent-pid")
	}
	if *backupID != "" && !backupIDPattern.MatchString(*backupID) {
		return errors.New("неверный backup ID")
	}
	question := "Восстановить последнюю применённую резервную копию?"
	if *backupID != "" {
		question = "Восстановить резервную копию " + *backupID + "?"
	}
	if !*yes && !confirm(question) {
		return errors.New("операция отменена")
	}
	requestSID := ""
	if *parentPID != 0 {
		if !isAdministrator() {
			return errors.New("parent-pid разрешён только elevated-процессу")
		}
		var err error
		requestSID, err = userSIDFromOptimizerProcess(uint32(*parentPID))
		if err != nil {
			return err
		}
	} else {
		var err error
		requestSID, err = currentUserSID()
		if err != nil {
			return err
		}
	}
	if !isAdministrator() {
		elevatedArgs := []string{"restore", "--yes", "--quiet", "--parent-pid", strconv.Itoa(os.Getpid())}
		if *backupID != "" {
			elevatedArgs = append(elevatedArgs, "--id", *backupID)
		}
		return runElevatedAndWait(elevatedArgs)
	}
	var path string
	var err error
	if *backupID == "" {
		path, err = restoreLatest(requestSID, *boostSession)
	} else {
		path, err = restoreSelected(requestSID, *backupID, *boostSession)
	}
	if err != nil {
		return err
	}
	if !*quiet {
		fmt.Println("Восстановлено:", displayText(path))
	}
	return nil
}

func cleanCommand(args []string) error {
	set := flag.NewFlagSet("clean", flag.ContinueOnError)
	days := set.Int("days", 2, "удалять временные файлы старше N дней")
	yes := set.Bool("yes", false, "подтвердить")
	quiet := set.Bool("quiet", false, "не печатать результат")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы clean")
	}
	if !*yes && !confirm("Удалить временные файлы старше "+strconv.Itoa(*days)+" дней?") {
		return errors.New("операция отменена")
	}
	result, err := cleanTemporaryFiles(*days)
	if err != nil {
		return err
	}
	if !*quiet {
		fmt.Printf("Удалено: %d файлов, %d папок, %.1f МБ; пропущено: %d\n", result.Files, result.Dirs, float64(result.Bytes)/(1024*1024), result.Skipped)
	}
	return nil
}

func confirm(question string) bool {
	fmt.Print(question, " [да/нет]: ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "да" || line == "yes" || line == "y"
}

func printAudit(audit Audit) {
	fmt.Println(productName, audit.Version)
	for _, cpu := range audit.Hardware.CPUs {
		fmt.Printf("CPU: %s — %d ядер / %d потоков\n", displayText(cpu.Name), cpu.Cores, cpu.Logical)
	}
	for _, gpu := range audit.Hardware.GPUs {
		fmt.Printf("GPU: %s (%s), драйвер %s\n", displayText(gpu.Name), displayText(gpu.Vendor), displayText(gpu.DriverVersion))
	}
	fmt.Println("Активная схема:", audit.ActivePowerGUID)
	if len(audit.Findings) == 0 {
		fmt.Println("Рекомендуемый игровой профиль уже применён полностью.")
	}
	for _, finding := range audit.Findings {
		fmt.Printf("%s — %s\n", displayText(finding.Title), displayText(finding.Evidence))
	}
	for _, warning := range audit.Warnings {
		fmt.Println("Предупреждение:", displayText(warning))
	}
}

func printPlan(plan Plan) {
	fmt.Println(displayText(plan.Profile.Name))
	for _, item := range plan.Items {
		marker := "="
		if item.Changed {
			marker = "→"
		}
		fmt.Printf("[%s] %s: %s %s %s\n", displayText(item.Category), displayText(item.Name), displayText(item.Current), marker, displayText(item.Desired))
	}
	for _, warning := range plan.Warnings {
		fmt.Println("Предупреждение:", displayText(warning))
	}
}

func printHelp() {
	fmt.Print(`Luxury Optimization

Без аргументов запускается mouse-first TUI.

  audit [--json] [--out report.json]   read-only аудит
  plan --profile recommended|maximum
                                        точный план без изменений
  apply --profile recommended|maximum  применить с backup и проверкой
  boost --game C:\Games\Game.exe --profile maximum [--priority above-normal] [--affinity 0xFF] -- [аргументы игры]
                                        применить профиль только на время игры
  startup list [--json]                показать registry-автозагрузку
  startup disable|enable --name NAME   обратимо управлять HKCU Run
  games scan [--json]                  найти Steam, Epic и Xbox игры
  games add|list|run|remove            сохранить и запускать per-game профили
  services list [--json]               read-only список служб Windows
  network interfaces|test              интерфейсы или TCP latency/jitter
  benchmark template|compare           сравнить FPS/1% low/p95 frametime
  backups list [--json]                sealed restore center (нужен admin)
  update check|install|enable|disable  обновления из GitHub Releases с SHA-256
  restore [--id BACKUP_ID]             откатить последнюю или выбранную операцию
  clean --days 2                       безопасно очистить старые temp-файлы
  version                              версия
`)
}
