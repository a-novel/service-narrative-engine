package main

import (
	"context"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func main() {
	ctx := lo.Must(postgres.NewContext(context.Background(), config.PostgresPresetDefault))
	lo.Must0(postgres.RunMigrationsContext(ctx, migrations.Migrations))
}
