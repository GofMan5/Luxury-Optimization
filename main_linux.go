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
	if len(os.Args) == 1 || os.Args[1] != "update" {
		maybeAutoUpdate()
	}
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка:", displayText(err.Error()))
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Println(productName, version)
		printAudit(collectAudit())
		fmt.Println()
		printHelp()
		return nil
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
	_ = set.Bool("yes", false, "совместимость с Windows CLI")
	_ = set.Bool("quiet", false, "совместимость с Windows CLI")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы apply")
	}
	if _, err := profileByID(*profileID); err != nil {
		return err
	}
	fmt.Println("Linux не требует постоянного профиля: Windows-настройки безопасно пропущены. Используйте boost для сессионного GameMode/nice/affinity.")
	return nil
}

func restoreCommand(args []string) error {
	set := flag.NewFlagSet("restore", flag.ContinueOnError)
	_ = set.Bool("yes", false, "совместимость с Windows CLI")
	_ = set.Bool("quiet", false, "совместимость с Windows CLI")
	_ = set.String("id", "", "совместимость с Windows CLI")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы restore")
	}
	fmt.Println("На Linux постоянные системные настройки не изменялись; восстанавливать нечего.")
	return nil
}

func backupsCommand(args []string) error {
	if len(args) > 0 && args[0] == "list" {
		args = args[1:]
	}
	set := flag.NewFlagSet("backups list", flag.ContinueOnError)
	jsonOnly := set.Bool("json", false, "вывести JSON")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("backups поддерживает только list")
	}
	if *jsonOnly {
		fmt.Println("[]")
	} else {
		fmt.Println("Linux использует только сессионные изменения; постоянных backup нет.")
	}
	return nil
}

func cleanCommand(args []string) error {
	set := flag.NewFlagSet("clean", flag.ContinueOnError)
	days := set.Int("days", 2, "удалять временные файлы приложения старше N дней")
	yes := set.Bool("yes", false, "подтвердить")
	quiet := set.Bool("quiet", false, "не печатать результат")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы clean")
	}
	if !*yes && !confirm("Удалить только временные файлы Luxury Optimization старше "+strconv.Itoa(*days)+" дней?") {
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
	fmt.Printf("Система: %s %s (%s)\n", displayText(audit.Hardware.OS.Caption), displayText(audit.Hardware.OS.Version), displayText(audit.Hardware.OS.Architecture))
	for _, cpu := range audit.Hardware.CPUs {
		fmt.Printf("CPU: %s — %d ядер / %d потоков\n", displayText(cpu.Name), cpu.Cores, cpu.Logical)
	}
	for _, gpu := range audit.Hardware.GPUs {
		fmt.Printf("GPU: %s (%s), драйвер %s\n", displayText(gpu.Name), displayText(gpu.Vendor), displayText(gpu.DriverVersion))
	}
	for _, capability := range audit.Capabilities {
		state := "skip"
		if capability.Available {
			state = capability.Mode
		}
		fmt.Printf("[%s] %s: %s\n", state, displayText(capability.ID), displayText(capability.Detail))
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

  audit [--json] [--out report.json]   read-only аудит системы и возможностей
  plan --profile recommended|maximum  точный план; недоступное помечается как skipped
  apply --profile ...                  безопасный Linux no-op для совместимости
  boost --game /path/game [--profile maximum] [--priority above-normal] [--affinity 0xFF] -- [args]
                                        GameMode, nice и affinity только на время игры
  games scan|add|list|run|remove       Steam и сохранённые игровые профили
  startup list|disable|enable          пользовательская XDG autostart
  services list [--json]               read-only systemd inventory
  network interfaces|test              интерфейсы и TCP latency/jitter
  benchmark template|compare           FPS/1% low/p95 frametime
  backups list | restore               совместимые безопасные no-op без persistent tweaks
  clean --days 2                       только временные файлы самого приложения
  update check|install|enable|disable  GitHub Releases + SHA-256
  version                              версия
`)
}
