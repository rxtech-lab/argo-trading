package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type GenerateCmdTestSuite struct {
	suite.Suite
	tempDir string
}

func (suite *GenerateCmdTestSuite) SetupTest() {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "generate-cmd-test")
	suite.Require().NoError(err)
	suite.tempDir = tempDir

	// Change to the temporary directory
	err = os.Chdir(tempDir)
	suite.Require().NoError(err)
}

func (suite *GenerateCmdTestSuite) TearDownTest() {
	// Clean up the temporary directory
	err := os.RemoveAll(suite.tempDir)
	suite.Require().NoError(err)
}

func (suite *GenerateCmdTestSuite) TestSchemaGeneration() {
	// Run the main function
	main()

	// Check if config directory was created
	configDir := filepath.Join(suite.tempDir, "config")
	suite.True(dirExists(configDir), "Config directory should exist")

	// Check if schema file was created
	schemaPath := filepath.Join(configDir, "backtest-engine-v1-config.json")
	suite.True(fileExists(schemaPath), "Schema file should exist")

	// Read and validate schema content
	schemaContent, err := os.ReadFile(schemaPath)
	suite.Require().NoError(err)
	suite.NotEmpty(schemaContent, "Schema file should not be empty")
}

func (suite *GenerateCmdTestSuite) TestSampleConfigGeneration() {
	// Run the main function
	main()

	// Check if sample config was created
	sampleConfigPath := filepath.Join(suite.tempDir, "config", "backtest-engine-v1-config.yaml")
	suite.True(fileExists(sampleConfigPath), "Sample config file should exist")

	// Read and validate sample config content
	sampleConfigContent, err := os.ReadFile(sampleConfigPath)
	suite.Require().NoError(err)
	suite.NotEmpty(sampleConfigContent, "Sample config file should not be empty")

	// Verify schema reference in YAML
	suite.Contains(string(sampleConfigContent), "# yaml-language-server: $schema=backtest-engine-v1-config.json")
}

// Helper functions
func dirExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func TestGenerateCmdSuite(t *testing.T) {
	suite.Run(t, new(GenerateCmdTestSuite))
}
