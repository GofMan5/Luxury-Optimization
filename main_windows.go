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

var version = "3.0.0-dev"

func main() {
	args, resultPath := stripResultFile(os.Args[1:])
	err := run(args)
	if resultPath != "" {
		if parentPID := internalParentPID(args); parentPID != 0 {
			_ = writeElevatedResult(resultPath, parentPID, err)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка:", err)
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
	case "clean":
		return cleanCommand(args[1:])
	case "version", "--version", "-v":
		fmt.Println("GofMan3 Optimizer", version)
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
	audit := collectAudit()
	data, err := json.MarshalIndent(audit, "", "  ")
	if err != nil {
		return err
	}
	if *out != "" {
		if err := os.WriteFile(*out, data, 0o600); err != nil {
			return err
		}
		fmt.Println("Отчёт сохранён:", *out)
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
	if err := set.Parse(args); err != nil {
		return err
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
		sid, err := currentUserSID()
		if err != nil {
			return err
		}
		_ = sid // SID is derived again from the live parent token in the elevated child.
		return runElevatedAndWait([]string{"apply", "--profile", *profileID, "--yes", "--quiet", "--parent-pid", strconv.Itoa(os.Getpid())})
	}
	path, err := applyProfile(*profileID)
	if err != nil {
		return err
	}
	if !*quiet {
		fmt.Println("Готово. Резервная копия:", path)
	}
	return nil
}

func restoreCommand(args []string) error {
	set := flag.NewFlagSet("restore", flag.ContinueOnError)
	yes := set.Bool("yes", false, "подтвердить")
	quiet := set.Bool("quiet", false, "не печатать результат")
	parentPID := set.Uint("parent-pid", 0, "internal: PID исходного процесса")
	if err := set.Parse(args); err != nil {
		return err
	}
	if !*yes && !confirm("Восстановить последнюю применённую резервную копию?") {
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
		return runElevatedAndWait([]string{"restore", "--yes", "--quiet", "--parent-pid", strconv.Itoa(os.Getpid())})
	}
	path, err := restoreLatest(requestSID)
	if err != nil {
		return err
	}
	if !*quiet {
		fmt.Println("Восстановлено:", path)
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
	fmt.Println("GofMan3 Optimizer", audit.Version)
	for _, cpu := range audit.Hardware.CPUs {
		fmt.Printf("CPU: %s — %d ядер / %d потоков\n", cpu.Name, cpu.Cores, cpu.Logical)
	}
	for _, gpu := range audit.Hardware.GPUs {
		fmt.Printf("GPU: %s (%s), драйвер %s\n", gpu.Name, gpu.Vendor, gpu.DriverVersion)
	}
	fmt.Println("Активная схема:", audit.ActivePowerGUID)
	if len(audit.Findings) == 0 {
		fmt.Println("Рекомендуемый игровой профиль уже применён полностью.")
	}
	for _, finding := range audit.Findings {
		fmt.Printf("%s — %s\n", finding.Title, finding.Evidence)
	}
	for _, warning := range audit.Warnings {
		fmt.Println("Предупреждение:", warning)
	}
}

func printPlan(plan Plan) {
	fmt.Println(plan.Profile.Name)
	for _, item := range plan.Items {
		marker := "="
		if item.Changed {
			marker = "→"
		}
		fmt.Printf("[%s] %s: %s %s %s\n", item.Category, item.Name, item.Current, marker, item.Desired)
	}
	for _, warning := range plan.Warnings {
		fmt.Println("Предупреждение:", warning)
	}
}

func printHelp() {
	fmt.Print(`GofMan3 Optimizer

Без аргументов запускается mouse-first TUI.

  audit [--json] [--out report.json]   read-only аудит
  plan --profile recommended|maximum
                                        точный план без изменений
  apply --profile recommended|maximum  применить с backup и проверкой
  restore                              откатить последнюю операцию
  clean --days 2                       безопасно очистить старые temp-файлы
  version                              версия
`)
}
