package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/samber/lo"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/config/env"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/modules"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

func main() {
	cfg := config.AppPresetDefault
	ctx := context.Background()

	otel.SetAppName(cfg.App.Name)

	if env.Version == "" {
		log.Fatalln("VERSION environment variable is required, aborting")
	}

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	ctx = lo.Must(postgres.NewContext(ctx, config.PostgresPresetDefault))

	repositoryModuleInsert := dao.NewModuleInsert()
	repositoryModuleDelete := dao.NewModuleDelete()
	repositoryModuleSelect := dao.NewModuleSelect()
	repositoryModuleListVersions := dao.NewModuleListVersions()

	serviceModuleLoadSystem := services.NewModuleLoadSystem(
		repositoryModuleInsert,
		repositoryModuleDelete,
		repositoryModuleSelect,
		repositoryModuleListVersions,
	)

	var err error

	for namespace, embedFS := range modules.KnownModules {
		log.Printf("Processing namespace: %s", namespace)
		err = errors.Join(err, processNamespace(ctx, namespace, embedFS, serviceModuleLoadSystem))
	}

	if err != nil {
		log.Printf("Completed with errors: %v", err)

		return
	}

	log.Println("All namespaces processed successfully")
}

func processNamespace(
	ctx context.Context,
	namespace string,
	embedFS fs.FS,
	service *services.ModuleLoadSystem,
) error {
	var systemModules []modules.SystemModule

	err := fs.WalkDir(embedFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		data, err := fs.ReadFile(embedFS, path)
		if err != nil {
			return fmt.Errorf("read file %s: %w", path, err)
		}

		var module modules.SystemModule

		err = yaml.Unmarshal(data, &module)
		if err != nil {
			return fmt.Errorf("unmarshal file %s: %w", path, err)
		}

		systemModules = append(systemModules, module)

		return nil
	})
	if err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}

	if len(systemModules) == 0 {
		log.Printf("No modules found in namespace %s", namespace)

		return nil
	}

	err = postgres.RunInTx(ctx, nil, func(ctx context.Context, _ bun.IDB) error {
		for _, module := range systemModules {
			_, err = service.Exec(ctx, &services.ModuleLoadSystemRequest{
				Module:  module,
				Version: env.Version,
				DevMode: env.DevMode,
			})
			if err != nil {
				return fmt.Errorf("load module %s: %w", module.ID, err)
			}

			log.Printf("Loaded module: %s/%s@%s", module.Namespace, module.ID, env.Version)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("run transaction: %w", err)
	}

	return nil
}
