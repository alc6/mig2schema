package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
)

type Migration struct {
	Name     string
	UpFile   string
	DownFile string
}

func ParseMigrations(migrationDir string) ([]Migration, error) {
	slog.Debug("scanning migration directory", "directory", migrationDir)
	upFiles := make(map[string]string)
	downFiles := make(map[string]string)

	err := filepath.WalkDir(migrationDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		fileName := d.Name()
		slog.Debug("found file", "file", fileName, "path", path)
		
		if strings.HasSuffix(fileName, ".up.sql") {
			baseName := strings.TrimSuffix(fileName, ".up.sql")
			upFiles[baseName] = path
			slog.Debug("found up migration", "name", baseName, "file", path)
		} else if strings.HasSuffix(fileName, ".down.sql") {
			baseName := strings.TrimSuffix(fileName, ".down.sql")
			downFiles[baseName] = path
			slog.Debug("found down migration", "name", baseName, "file", path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk migration directory: %w", err)
	}

	var migrations []Migration
	for baseName, upFile := range upFiles {
		migration := Migration{
			Name:   baseName,
			UpFile: upFile,
		}
		
		if downFile, exists := downFiles[baseName]; exists {
			migration.DownFile = downFile
			slog.Debug("migration has down file", "name", baseName, "downFile", downFile)
		}
		
		migrations = append(migrations, migration)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	slog.Info("parsed migrations", "count", len(migrations), "upFiles", len(upFiles), "downFiles", len(downFiles))
	return migrations, nil
}