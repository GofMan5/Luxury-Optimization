package optimizer

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
)

func boostCommand(args []string) error {
	set := flag.NewFlagSet("boost", flag.ContinueOnError)
	game := set.String("game", "", "абсолютный путь к EXE игры")
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

func runBoostSession(game, profile, priority string, affinity uintptr, gameArgs []string) (err error) {
	if isAdministrator() {
		return errors.New("boost запускается без прав администратора, чтобы игра не получила elevation")
	}
	if _, err := profileByID(profile); err != nil {
		return err
	}
	game, err = validateGameExecutable(game)
	if err != nil {
		return err
	}
	releaseLock, err := acquireBoostSessionLock()
	if err != nil {
		return err
	}
	defer releaseLock()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	parentPID := strconv.Itoa(os.Getpid())
	if err := runElevatedAndWait([]string{"apply", "--profile", profile, "--yes", "--quiet", "--parent-pid", parentPID, "--boost-session"}); err != nil {
		return fmt.Errorf("применение boost-профиля: %w", err)
	}
	defer func() {
		restoreErr := runElevatedAndWait([]string{"restore", "--yes", "--quiet", "--parent-pid", parentPID, "--boost-session"})
		if restoreErr != nil {
			restoreErr = fmt.Errorf("восстановление после boost-сессии: %w", restoreErr)
		} else {
			fmt.Println("Boost-сессия завершена, исходные настройки восстановлены.")
		}
		err = errors.Join(err, restoreErr)
	}()

	command := exec.Command(game, gameArgs...)
	command.Dir = filepath.Dir(game)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Start(); err != nil {
		return fmt.Errorf("запуск игры: %w", err)
	}
	tuningErr := tuneGameProcess(uint32(command.Process.Pid), priority, affinity)
	if tuningErr != nil {
		fmt.Fprintln(os.Stderr, "Предупреждение: настройка процесса:", displayText(tuningErr.Error()))
	}
	fmt.Println("Boost-профиль применён. Ожидаю завершения игры; Ctrl+C завершит boost-сессию.")
	wait := make(chan error, 1)
	go func() { wait <- command.Wait() }()
	select {
	case err = <-wait:
		if err != nil {
			err = fmt.Errorf("игра завершилась с ошибкой: %w", err)
		}
	case <-ctx.Done():
		fmt.Println("Получен Ctrl+C, восстанавливаю исходные настройки.")
	}
	return errors.Join(err, tuningErr)
}

func validateGameExecutable(path string) (string, error) {
	if path == "" {
		return "", errors.New("укажите --game с абсолютным путём к EXE игры")
	}
	if !filepath.IsAbs(path) {
		return "", errors.New("путь к игре должен быть абсолютным")
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("путь к игре: %w", err)
	}
	if !strings.EqualFold(filepath.Ext(resolved), ".exe") {
		return "", errors.New("игра должна быть EXE-файлом")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("путь к игре не является обычным файлом")
	}
	file, err := os.Open(resolved)
	if err != nil {
		return "", err
	}
	defer file.Close()
	var magic [2]byte
	if _, err := io.ReadFull(file, magic[:]); err != nil || string(magic[:]) != "MZ" {
		return "", errors.New("файл игры не имеет PE/MZ-заголовка")
	}
	return resolved, nil
}
