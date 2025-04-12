package main

import (
	"log"
	"os"
	"path/filepath"

	engine "github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1"
	"gopkg.in/yaml.v2"
)

func main() {
	// Create a config instance
	config := engine.EmptyConfig()

	// Generate schema JSON
	schemaJSON, err := config.GenerateSchemaJSON()
	if err != nil {
		log.Fatalf("Failed to generate schema: %v", err)
	}

	// Set the output path
	schemaName := "backtest-engine-v1-config.json"
	schemaPath := filepath.Join("./config", schemaName)
	sampleConfigPath := filepath.Join("./config", "backtest-engine-v1-config.yaml")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(schemaPath), 0755); err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}

	// Write schema to file
	err = os.WriteFile(schemaPath, []byte(schemaJSON), 0644)
	if err != nil {
		log.Fatalf("Failed to write schema to file: %v", err)
	}

	// write sample config to file if doesn't exist
	// yaml marshal
	if _, err := os.Stat(sampleConfigPath); os.IsNotExist(err) {
		yamlBytes, err := yaml.Marshal(config)
		// add # yaml-language-server: $schema=./backtest-engine-v1-config.json to the beginning of the file
		yamlBytes = append([]byte("# yaml-language-server: $schema="+schemaName+"\n"), yamlBytes...)
		if err != nil {
			log.Fatalf("Failed to marshal sample config to yaml: %v", err)
		}
		err = os.WriteFile(sampleConfigPath, yamlBytes, 0644)
		if err != nil {
			log.Fatalf("Failed to write sample config to file: %v", err)
		}
		log.Printf("Sample config successfully generated at %s", sampleConfigPath)
	}
	log.Printf("Schema successfully generated at %s", schemaPath)
}
