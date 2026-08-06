//go:build linux

package optimizer

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func (s *Service) ApplyProfile(profile string) (MutationResult, error) {
	if _, err := profileByID(profile); err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Message: "Linux не применяет Windows-only persistent tweaks; используйте сессионный GameMode/nice/affinity."}, nil
}

func (s *Service) ApplyTweak(id string) (MutationResult, error) {
	s.tweakMu.Lock()
	defer s.tweakMu.Unlock()
	plan, err := buildPlan(profileMaximum)
	if err != nil {
		return MutationResult{}, err
	}
	for _, item := range plan.Items {
		if item.ID == id {
			return MutationResult{Message: "Этот твик применяется только к игровой сессии через GameMode/priority и не меняет Linux постоянно."}, nil
		}
	}
	return MutationResult{}, errors.New("твик недоступен на этой системе")
}

func (s *Service) RestoreTweak(string) (MutationResult, error) {
	s.tweakMu.Lock()
	defer s.tweakMu.Unlock()
	return MutationResult{Message: "Постоянные настройки Linux не изменялись; откат не требуется."}, nil
}

func decorateTweakRestoreState(plan *Plan) {
	for index := range plan.Items {
		plan.Items[index].Reversible = false
		plan.Items[index].ManualAvailable = false
	}
}

func (s *Service) Restore(string) (MutationResult, error) {
	return MutationResult{Message: "На Linux постоянные системные настройки не изменялись."}, nil
}

func (s *Service) Backups() ([]BackupSummary, error) { return []BackupSummary{}, nil }

func (s *Service) SetService(string, bool) (MutationResult, error) {
	return MutationResult{}, errors.New("управление системными службами Linux недоступно; список остаётся read-only")
}

func (s *Service) SystemRestorePoints() ([]SystemRestorePoint, error) {
	return []SystemRestorePoint{}, nil
}

func (s *Service) OpenSystemRestore() (MutationResult, error) {
	return MutationResult{}, errors.New("системное восстановление Windows недоступно на Linux")
}

func (s *Service) CheckpointStatus(profile string) (CheckpointStatus, error) {
	if _, err := profileByID(profile); err != nil {
		return CheckpointStatus{}, err
	}
	return CheckpointStatus{Ready: true, Profile: profile}, nil
}

func (s *Service) CreateCheckpoint(profile string) (CheckpointStatus, error) {
	return s.CheckpointStatus(profile)
}

func launchRecoveryCLI(args []string) (int, error) {
	executable, err := os.Executable()
	if err != nil {
		return 0, err
	}
	command := exec.Command(executable, args...)
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := command.Start(); err != nil {
		return 0, err
	}
	go func() { _ = command.Wait() }()
	return command.Process.Pid, nil
}
