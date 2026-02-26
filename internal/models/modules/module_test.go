package modules_test

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/lib"
	"github.com/a-novel/service-narrative-engine/internal/models/modules"
	"github.com/a-novel/service-narrative-engine/internal/services"
	servicesmocks "github.com/a-novel/service-narrative-engine/internal/services/mocks"
)

const testSystemModuleVersion = "1.0.0"

// TestKnownModules verifies every YAML module file in every KnownModules namespace:
//  1. Correctly unmarshals into a SystemModule with no empty fields.
//  2. Produces a valid JSON schema via lib.BuildSchema (no flags, AI flag, Partial flag).
//  3. Can be loaded through the ModuleLoadSystem service with a mocked DAO.
func TestKnownModules(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name      string
		rawYAML   []byte
		namespace string
	}

	// Collect all YAML files from every known namespace.
	var testCases []testCase

	for namespace, embedFS := range modules.KnownModules {
		err := fs.WalkDir(embedFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			if ext := strings.ToLower(filepath.Ext(path)); ext != ".yaml" && ext != ".yml" {
				return nil
			}

			data, err := fs.ReadFile(embedFS, path)
			if err != nil {
				return err
			}

			testCases = append(testCases, testCase{
				name:      namespace + "/" + path,
				rawYAML:   data,
				namespace: namespace,
			})

			return nil
		})
		require.NoError(t, err)
	}

	require.NotEmpty(t, testCases, "KnownModules must define at least one module")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// fresh re-unmarshals the raw YAML each time to prevent mutation side effects
			// from BuildSchema calls (the Partial flag mutates the RawSchema in place).
			fresh := func() modules.SystemModule {
				var m modules.SystemModule
				require.NoError(t, yaml.Unmarshal(tc.rawYAML, &m))
				m.Namespace = tc.namespace // namespace is the map key, not in the YAML file
				return m
			}

			// ----------------------------------------------------------------
			// 1. Unmarshal validity: all fields must be present and non-empty.
			// ----------------------------------------------------------------

			mod := fresh()
			require.NotEmpty(t, mod.ID, "module ID must not be empty")
			require.NotEmpty(t, mod.Namespace, "module Namespace must not be empty")
			require.NotEmpty(t, mod.Description, "module Description must not be empty")
			require.NotNil(t, mod.Schema, "module Schema must not be nil")

			// ----------------------------------------------------------------
			// 2. Schema validity across all BuildSchema flag combinations.
			// ----------------------------------------------------------------

			t.Run("BuildSchema/NoFlags", func(t *testing.T) {
				t.Parallel()
				_, err := lib.BuildSchema(fresh().Schema)
				require.NoError(t, err)
			})

			t.Run("BuildSchema/AI", func(t *testing.T) {
				t.Parallel()
				_, err := lib.BuildSchema(fresh().Schema, lib.SchemaBuildFlagAI)
				require.NoError(t, err)
			})

			t.Run("BuildSchema/Partial", func(t *testing.T) {
				t.Parallel()
				_, err := lib.BuildSchema(fresh().Schema, lib.SchemaBuildFlagPartial)
				require.NoError(t, err)
			})

			// ----------------------------------------------------------------
			// 3. ModuleLoadSystem service (non-dev mode, all DAOs mocked).
			// ----------------------------------------------------------------

			t.Run("ModuleLoadSystem", func(t *testing.T) {
				t.Parallel()

				m := fresh()
				baseTime := time.Now()

				insertMock := servicesmocks.NewMockModuleLoadSystemRepository(t)
				deleteMock := servicesmocks.NewMockModuleLoadSystemRepositoryDelete(t)
				selectMock := servicesmocks.NewMockModuleLoadSystemRepositorySelect(t)
				listVersionsMock := servicesmocks.NewMockModuleLoadSystemRepositoryListVersions(t)

				// In non-dev mode the service calls only the insert repository.
				insertMock.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ModuleInsertRequest) bool {
						return req.ID == m.ID &&
							req.Namespace == m.Namespace &&
							req.Version == testSystemModuleVersion &&
							req.Preversion == "" &&
							req.Description == m.Description &&
							time.Since(req.Now) < time.Minute
					})).
					Return(&dao.Module{
						ID:          m.ID,
						Namespace:   m.Namespace,
						Version:     testSystemModuleVersion,
						Description: m.Description,
						Schema:      m.Schema,
						CreatedAt:   baseTime,
					}, nil)

				svc := services.NewModuleLoadSystem(insertMock, deleteMock, selectMock, listVersionsMock)

				result, err := svc.Exec(context.Background(), &services.ModuleLoadSystemRequest{
					Module:  m,
					Version: testSystemModuleVersion,
					DevMode: false,
				})

				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, m.ID, result.ID)
				require.Equal(t, m.Namespace, result.Namespace)
				require.Equal(t, testSystemModuleVersion, result.Version)
			})
		})
	}
}
