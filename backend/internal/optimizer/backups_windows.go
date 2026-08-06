package optimizer

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
)

func backupsCommand(args []string) error {
	if len(args) > 0 && args[0] == "list" {
		args = args[1:]
	} else if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return errors.New("backups поддерживает только list")
	}
	set := flag.NewFlagSet("backups list", flag.ContinueOnError)
	jsonOnly := set.Bool("json", false, "вывести JSON")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы backups list")
	}
	if !isAdministrator() {
		return errors.New("backups list запускается из терминала с правами администратора")
	}
	sid, err := currentUserSID()
	if err != nil {
		return err
	}
	summaries, err := listBackupSummaries(sid)
	if err != nil {
		return err
	}
	if *jsonOnly {
		data, err := json.MarshalIndent(summaries, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	for _, backup := range summaries {
		fmt.Printf("%s  %s  %-15s %s  restorable=%t\n", backup.ID, backup.CreatedAt.Local().Format("2006-01-02 15:04:05"), backup.Status, backup.Profile, backup.Restorable)
	}
	return nil
}

func listBackupSummaries(targetSID string) ([]BackupSummary, error) {
	if !sidPattern.MatchString(targetSID) {
		return nil, errors.New("неверный SID")
	}
	dir, names, err := backupFiles()
	if err != nil {
		return nil, err
	}
	result := make([]BackupSummary, 0, len(names))
	for _, name := range names {
		id := strings.TrimSuffix(name, ".json")
		backup, err := loadBackupFile(filepath.Join(dir, name), id)
		if err != nil {
			return nil, err
		}
		if backup.TargetUserSID != targetSID {
			continue
		}
		if err := validateBackup(backup); err != nil {
			return nil, err
		}
		result = append(result, BackupSummary{ID: backup.ID, CreatedAt: backup.CreatedAt, Profile: backup.Profile, TweakID: backup.TweakID, Status: backup.Status, Restorable: backupRestorable(backup.Status)})
	}
	return result, nil
}
