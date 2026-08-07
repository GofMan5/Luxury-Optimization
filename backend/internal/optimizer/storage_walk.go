package optimizer

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type storageDirTask struct {
	path     string
	identity os.FileInfo
}

type storageDirItem struct {
	entry fs.DirEntry
	info  os.FileInfo
	err   error
}

type storageDirResult struct {
	task  storageDirTask
	items []storageDirItem
	err   error
}

func walkStorageEntries(ctx context.Context, root string, visit func(string, fs.DirEntry, os.FileInfo, error) error) error {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return err
	}
	workerCount := min(8, max(2, runtime.GOMAXPROCS(0)/2))
	ctx, cancel := context.WithCancel(ctx)
	tasks := make(chan storageDirTask)
	results := make(chan storageDirResult, workerCount)
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case task, ok := <-tasks:
					if !ok {
						return
					}
					result := readStorageDirectory(task)
					select {
					case results <- result:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}
	defer func() {
		cancel()
		close(tasks)
		workers.Wait()
	}()

	queue := []storageDirTask{{path: root, identity: rootInfo}}
	active := 0
	for len(queue) > 0 || active > 0 {
		var dispatch chan<- storageDirTask
		var next storageDirTask
		if len(queue) > 0 && active < workerCount {
			dispatch = tasks
			next = queue[len(queue)-1]
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case dispatch <- next:
			queue = queue[:len(queue)-1]
			active++
		case result := <-results:
			active--
			if result.err != nil {
				if sameStoragePath(result.task.path, root) {
					return result.err
				}
				if err := visit(result.task.path, nil, nil, result.err); err != nil && !errors.Is(err, fs.SkipDir) {
					return err
				}
				continue
			}
			for _, item := range result.items {
				path := filepath.Join(result.task.path, item.entry.Name())
				visitErr := visit(path, item.entry, item.info, item.err)
				if errors.Is(visitErr, fs.SkipDir) {
					continue
				}
				if visitErr != nil {
					return visitErr
				}
				if item.err == nil && item.info.IsDir() {
					queue = append(queue, storageDirTask{path: path, identity: item.info})
				}
			}
		}
	}
	return nil
}

func readStorageDirectory(task storageDirTask) storageDirResult {
	current, err := os.Lstat(task.path)
	if err != nil {
		return storageDirResult{task: task, err: err}
	}
	if !current.IsDir() || current.Mode()&os.ModeSymlink != 0 || storageEntryIsReparse(current) || !os.SameFile(task.identity, current) {
		return storageDirResult{task: task, err: errors.New("directory changed or became a link during scan")}
	}
	directory, err := os.Open(task.path)
	if err != nil {
		return storageDirResult{task: task, err: err}
	}
	entries, readErr := directory.ReadDir(-1)
	closeErr := directory.Close()
	if readErr != nil {
		return storageDirResult{task: task, err: readErr}
	}
	if closeErr != nil {
		return storageDirResult{task: task, err: closeErr}
	}
	items := make([]storageDirItem, len(entries))
	for index, entry := range entries {
		items[index].entry = entry
		items[index].info, items[index].err = entry.Info()
	}
	return storageDirResult{task: task, items: items}
}
