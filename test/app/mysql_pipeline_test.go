package app_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	commonMySQL "github.com/xuenqlve/common/data_source/mysql"
	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/internal/config"
	"github.com/xuenqlve/kyogre/internal/data_source"
	"github.com/xuenqlve/kyogre/internal/message"
	pluginGenerator "github.com/xuenqlve/kyogre/internal/plugin/generator"
	pluginMetadata "github.com/xuenqlve/kyogre/internal/plugin/metadata"
	pluginPressure "github.com/xuenqlve/kyogre/internal/plugin/pressure"
	pluginScenario "github.com/xuenqlve/kyogre/internal/plugin/scenario"
	pkgMySQL "github.com/xuenqlve/kyogre/pkg/data_source/mysql"
	_ "github.com/xuenqlve/kyogre/pkg/generator"
	_ "github.com/xuenqlve/kyogre/pkg/metadata"
	_ "github.com/xuenqlve/kyogre/pkg/pressure"
	_ "github.com/xuenqlve/kyogre/pkg/scenario"
)

func TestMain(m *testing.M) {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	log.Init(log.DebugLevel, filepath.Dir(dir))
	os.Exit(m.Run())
}

func TestMySQLMinimalPipelineInserts1000Rows(t *testing.T) {
	const (
		pipelineName  = "test_app_mysql_minimal_pipeline"
		dataSourceKey = "app-minimal-source"
		metadataKey   = "app-minimal-meta"
		generatorKey  = "app-minimal-generator"
		scenarioKey   = "app-minimal-scenario"
		pressureKey   = "app-minimal-pressure"
		tableName     = "minimal_pipeline_events"
		wantRows      = 1000
	)

	databaseName := fmt.Sprintf("kyapp_%d", time.Now().UnixNano()%1_000_000)
	targetSchema := fmt.Sprintf("%s.%s", databaseName, tableName)

	if err := prepareMySQLDataSource(pipelineName, dataSourceKey); err != nil {
		t.Fatalf("prepare mysql data source: %v", err)
	}

	db, err := pkgMySQL.Connection(dataSourceKey)
	if err != nil {
		t.Fatalf("open mysql connection: %v", err)
	}
	defer db.Close()
	defer dropDatabase(t, db, databaseName)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metadataManager := pluginMetadata.NewManager()
	if err := metadataManager.Configure(pipelineName, map[string]config.ConfigureMold{
		metadataKey: {
			Type: "mysql",
			Config: map[string]any{
				"data-source": dataSourceKey,
				"template":    "customize",
				"databases": map[string]any{
					databaseName: []map[string]any{
						{
							"table": tableName,
							"columns": []map[string]any{
								{"column": "event_id", "type": "varchar"},
								{"column": "name", "type": "string"},
								{"column": "payload", "type": "text"},
								{"column": "created_at", "type": "timestamp"},
							},
							"indexes": []map[string]any{
								{
									"name":       "uk_event_name_time",
									"columns":    []string{"event_id", "name", "created_at"},
									"is_primary": false,
									"is_unique":  true,
								},
							},
						},
					},
				},
			},
		},
	}); err != nil {
		t.Fatalf("configure metadata manager: %v", err)
	}

	meta, err := metadataManager.GetMetadata(metadataKey)
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	defer meta.Close()
	if err := meta.Initialize(ctx); err != nil {
		t.Fatalf("initialize metadata: %v", err)
	}

	generatorManager := pluginGenerator.NewManager()
	if err := generatorManager.Configure(pipelineName, map[string]config.ConfigureMold{
		generatorKey: {
			Type:   "mysql",
			Config: map[string]any{},
		},
	}); err != nil {
		t.Fatalf("configure generator manager: %v", err)
	}

	scenarioManager := pluginScenario.NewManager()
	if err := scenarioManager.Configure(pipelineName, map[string]config.ScenarioConfigureMold{
		scenarioKey: {
			Type:       "base",
			Metadata:   metadataKey,
			Generators: []string{generatorKey},
			Config: map[string]any{
				"builder":       "mysql",
				"mode":          "row",
				"message-count": wantRows,
				"interval-ms":   0,
				"target-selector": map[string]any{
					"strategy": "round-robin",
					"schemas":  []string{targetSchema},
				},
				"operation-selector": map[string]any{
					"strategy": "round-robin",
					"values":   []string{"insert"},
				},
				"row-count-selector": map[string]any{
					"fixed": 1,
				},
			},
		},
	}, generatorManager, nil, metadataManager); err != nil {
		t.Fatalf("configure scenario manager: %v", err)
	}

	pressureManager := pluginPressure.NewManager()
	if err := pressureManager.Configure(pipelineName, map[string]config.ConfigureMold{
		pressureKey: {
			Type: "mysql-dml",
			Config: map[string]any{
				"data-source":         dataSourceKey,
				"worker-count":        1,
				"worker-queue-length": 64,
			},
		},
	}); err != nil {
		t.Fatalf("configure pressure manager: %v", err)
	}

	director, err := scenarioManager.GetDirector(scenarioKey)
	if err != nil {
		t.Fatalf("get director: %v", err)
	}
	controller, err := pressureManager.GetPressureController([]string{pressureKey})
	if err != nil {
		t.Fatalf("get pressure controller: %v", err)
	}

	point := make(message.Point, 256)
	defer func() {
		_ = director.Close(true)
	}()

	if err := controller.Start(ctx, point.OutPoint()); err != nil {
		t.Fatalf("start pressure controller: %v", err)
	}
	if err := director.Preparation(ctx); err != nil {
		t.Fatalf("prepare director: %v", err)
	}
	if err := director.Start(ctx, point.InPoint()); err != nil {
		t.Fatalf("start director: %v", err)
	}

	select {
	case <-director.Done():
	case <-ctx.Done():
		t.Fatalf("director did not finish in time: %v", ctx.Err())
	}

	if err := director.Err(); err != nil {
		t.Fatalf("director runtime err: %v", err)
	}

	gotRows := waitRowCount(t, db, databaseName, tableName, wantRows, 3*time.Second)
	point.Close()
	if err := controller.Close(); err != nil {
		t.Fatalf("close pressure controller: %v", err)
	}
	if gotRows != wantRows {
		t.Fatalf("unexpected inserted row count: got %d want %d", gotRows, wantRows)
	}
}

func prepareMySQLDataSource(pipeline, name string) error {
	ds, err := data_source.GetDataSource(pkgMySQL.MySQL)
	if err != nil {
		return err
	}
	return ds.Configure(pipeline, map[string]any{
		name: commonMySQL.Config{
			Host:     getenvOrDefault("KYOGRE_TEST_MYSQL_HOST", "127.0.0.1"),
			Port:     getenvIntOrDefault("KYOGRE_TEST_MYSQL_PORT", 3306),
			Username: getenvOrDefault("KYOGRE_TEST_MYSQL_USERNAME", "root"),
			Password: getenvOrDefault("KYOGRE_TEST_MYSQL_PASSWORD", "root"),
		},
	})
}

func queryRowCount(t *testing.T, db *sql.DB, database, table string) int {
	t.Helper()
	query := fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s`", database, table)
	var count int
	if err := db.QueryRow(query).Scan(&count); err != nil {
		t.Fatalf("query row count: %v", err)
	}
	return count
}

func waitRowCount(t *testing.T, db *sql.DB, database, table string, want int, timeout time.Duration) int {
	t.Helper()
	deadline := time.Now().Add(timeout)
	last := 0
	for {
		last = queryRowCount(t, db, database, table)
		if last >= want {
			return last
		}
		if time.Now().After(deadline) {
			return last
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func dropDatabase(t *testing.T, db *sql.DB, database string) {
	t.Helper()
	if _, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", database)); err != nil {
		t.Fatalf("drop database %s: %v", database, err)
	}
}

func getenvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvIntOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
