package optimizer

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func boostCommand(args []string) error {
	set := flag.NewFlagSet("boost", flag.ContinueOnError)
	game := set.String("game", "", "абсолютный путь к игре")
	profile := set.String("profile", profileMaximum, "recommended или maximum")
	priority := set.String("priority", "normal", "normal, above-normal или high")
	affinityText := set.String("affinity", "", "необязательная CPU mask, например 0xFF")
	if err := set.Parse(args); err != nil {
		return err
	}
	affinity, err := parseAffinity(*affinityText)
	if err != nil {
		return err
	}
	if _, err := processPriorityClass(*priority); err != nil {
		return err
	}
	return runBoostSession(*game, *profile, *priority, affinity, set.Args())
}

func runBoostSession(game, profile, priority string, affinity uintptr, gameArgs []string) error {
	if os.Geteuid() == 0 {
		return errors.New("не запускайте игру от root")
	}
	if _, err := profileByID(profile); err != nil {
		return err
	}
	game, err := validateGameExecutable(game)
	if err != nil {
		return err
	}
	commandPath := game
	commandArgs := append([]string(nil), gameArgs...)
	if gameMode, lookErr := exec.LookPath("gamemoderun"); lookErr == nil {
		commandPath = gameMode
		commandArgs = append([]string{game}, commandArgs...)
		fmt.Println("Feral GameMode: enabled for this launch")
	} else {
		fmt.Println("Feral GameMode: unavailable, launching directly")
	}

	command := exec.Command(commandPath, commandArgs...)
	command.Dir = filepath.Dir(game)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		return fmt.Errorf("запуск игры: %w", err)
	}
	warnings, tuneErr := tuneGameProcess(command.Process.Pid, priority, affinity)
	for _, warning := range warnings {
		fmt.Fprintln(os.Stderr, "Предупреждение:", warning)
	}
	if tuneErr != nil {
		_ = command.Process.Kill()
		_, _ = command.Process.Wait()
		return fmt.Errorf("настройка игрового процесса: %w", tuneErr)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	wait := make(chan error, 1)
	go func() { wait <- command.Wait() }()
	fmt.Println("Boost-сессия активна до завершения игры; Ctrl+C корректно завершит процесс.")
	select {
	case err := <-wait:
		if err != nil {
			return fmt.Errorf("игра завершилась с ошибкой: %w", err)
		}
		return nil
	case <-ctx.Done():
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGINT)
		select {
		case err := <-wait:
			return err
		case <-time.After(5 * time.Second):
			_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
			<-wait
			return errors.New("игра не завершилась после SIGINT и была остановлена")
		}
	}
}

func validateGameExecutable(path string) (string, error) {
	if path == "" {
		return "", errors.New("укажите --game с абсолютным путём к игре")
	}
	if !filepath.IsAbs(path) {
		return "", errors.New("путь к игре должен быть абсолютным")
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("путь к игре: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", errors.New("игра должна быть обычным исполняемым файлом")
	}
	return resolved, nil
}
