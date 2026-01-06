package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	engine "github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1"
	"gopkg.in/yaml.v2"
)

// validatePaths checks if the provided paths are valid
func validatePaths(schemaPath, sampleConfigPath string) error {
	if schemaPath == "" {
		return fmt.Errorf("schema path cannot be empty")
	}
	if sampleConfigPath == "" {
		return fmt.Errorf("sample config path cannot be empty")
	}
	return nil
}

// getSchemaReference returns the YAML language server schema reference string
func getSchemaReference(schemaName string) string {
	return fmt.Sprintf("# yaml-language-server: $schema=%s\n", schemaName)
}

// validateSchemaName checks if the schema name has a valid format
func validateSchemaName(schemaName string) error {
	if schemaName == "" {
		return fmt.Errorf("schema name cannot be empty")
	}
	if !strings.HasSuffix(schemaName, ".json") {
		return fmt.Errorf("schema name must have .json extension")
	}
	return nil
}

// fileExists checks if a file exists at the given path
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// generateSchemaFile creates the schema JSON file at the specified path.
func generateSchemaFile(config engine.BacktestEngineV1Config, schemaPath string) error {
	schemaJSON, err := config.GenerateSchemaJSON()
	if err != nil {
		return fmt.Errorf("failed to generate schema: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(schemaPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(schemaPath, []byte(schemaJSON), 0644); err != nil {
		return fmt.Errorf("failed to write schema to file: %w", err)
	}

	return nil
}

// generateSampleConfig creates the sample YAML config file if it doesn't exist.
func generateSampleConfig(config engine.BacktestEngineV1Config, sampleConfigPath string, schemaName string) error {
	if !fileExists(sampleConfigPath) {
		yamlBytes, err := yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal sample config to yaml: %w", err)
		}

		schemaRef := getSchemaReference(schemaName)
		yamlBytes = append([]byte(schemaRef), yamlBytes...)

		if err := os.WriteFile(sampleConfigPath, yamlBytes, 0644); err != nil {
			return fmt.Errorf("failed to write sample config to file: %w", err)
		}
	}
	return nil
}

func main() {
	// Create a config instance
	config := engine.EmptyConfig()

	// Set the output path
	schemaName := "backtest-engine-v1-config.json"
	schemaPath := filepath.Join("./config", schemaName)
	sampleConfigPath := filepath.Join("./config", "backtest-engine-v1-config.yaml")

	// Validate schema name
	if err := validateSchemaName(schemaName); err != nil {
		log.Fatalf("Invalid schema name: %v", err)
	}

	// Validate paths
	if err := validatePaths(schemaPath, sampleConfigPath); err != nil {
		log.Fatalf("Invalid paths: %v", err)
	}

	// Generate schema file
	if err := generateSchemaFile(config, schemaPath); err != nil {
		log.Fatalf("%v", err)
	}

	// Generate sample config
	if err := generateSampleConfig(config, sampleConfigPath, schemaName); err != nil {
		log.Fatalf("%v", err)
	}

	log.Printf("Schema successfully generated at %s", schemaPath)
	if fileExists(sampleConfigPath) {
		log.Printf("Sample config successfully generated at %s", sampleConfigPath)
	}
}
