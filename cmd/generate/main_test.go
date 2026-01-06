package main

import (
	"os"
	"path/filepath"
	"testing"

	engine "github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1"
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

func (suite *GenerateCmdTestSuite) TestSampleConfigNotOverwritten() {
	// Run the main function first time
	main()

	// Get the original content
	sampleConfigPath := filepath.Join(suite.tempDir, "config", "backtest-engine-v1-config.yaml")
	originalContent, err := os.ReadFile(sampleConfigPath)
	suite.Require().NoError(err)

	// Run main again - it should not overwrite the existing sample config
	main()

	// Verify the content hasn't changed
	newContent, err := os.ReadFile(sampleConfigPath)
	suite.Require().NoError(err)
	suite.Equal(string(originalContent), string(newContent), "Sample config should not be overwritten")
}

func (suite *GenerateCmdTestSuite) TestGenerateSchemaFile() {
	config := engine.EmptyConfig()
	schemaPath := filepath.Join(suite.tempDir, "test-schema", "schema.json")

	err := generateSchemaFile(config, schemaPath)
	suite.Require().NoError(err)

	// Verify file was created
	suite.True(fileExists(schemaPath), "Schema file should exist")

	// Verify content is not empty
	content, err := os.ReadFile(schemaPath)
	suite.Require().NoError(err)
	suite.NotEmpty(content, "Schema content should not be empty")
}

func (suite *GenerateCmdTestSuite) TestGenerateSchemaFileInvalidPath() {
	config := engine.EmptyConfig()
	// Use invalid path (on Unix systems, writing to /root/... without permissions should fail)
	schemaPath := filepath.Join("/root/invalid/schema.json")

	err := generateSchemaFile(config, schemaPath)
	suite.Error(err, "Should return error for invalid path")
	suite.Contains(err.Error(), "failed to", "Error should contain descriptive message")
}

func (suite *GenerateCmdTestSuite) TestGenerateSampleConfig() {
	config := engine.EmptyConfig()
	samplePath := filepath.Join(suite.tempDir, "sample-config.yaml")
	schemaName := "test-schema.json"

	err := generateSampleConfig(config, samplePath, schemaName)
	suite.Require().NoError(err)

	// Verify file was created
	suite.True(fileExists(samplePath), "Sample config file should exist")

	// Verify content includes schema reference
	content, err := os.ReadFile(samplePath)
	suite.Require().NoError(err)
	suite.Contains(string(content), "# yaml-language-server: $schema="+schemaName)
}

func (suite *GenerateCmdTestSuite) TestGenerateSampleConfigAlreadyExists() {
	config := engine.EmptyConfig()
	samplePath := filepath.Join(suite.tempDir, "existing-config.yaml")
	schemaName := "test-schema.json"

	// Create a file first
	originalContent := []byte("existing content")
	err := os.WriteFile(samplePath, originalContent, 0644)
	suite.Require().NoError(err)

	// Try to generate - should not overwrite
	err = generateSampleConfig(config, samplePath, schemaName)
	suite.Require().NoError(err)

	// Verify content is unchanged
	content, err := os.ReadFile(samplePath)
	suite.Require().NoError(err)
	suite.Equal(string(originalContent), string(content), "Existing file should not be overwritten")
}

func (suite *GenerateCmdTestSuite) TestValidatePaths() {
	// Test valid paths
	err := validatePaths("/some/path/schema.json", "/some/path/config.yaml")
	suite.NoError(err, "Valid paths should not return error")

	// Test empty schema path
	err = validatePaths("", "/some/path/config.yaml")
	suite.Error(err, "Empty schema path should return error")
	suite.Contains(err.Error(), "schema path cannot be empty")

	// Test empty sample config path
	err = validatePaths("/some/path/schema.json", "")
	suite.Error(err, "Empty sample config path should return error")
	suite.Contains(err.Error(), "sample config path cannot be empty")

	// Test both empty
	err = validatePaths("", "")
	suite.Error(err, "Both empty paths should return error")
}

func (suite *GenerateCmdTestSuite) TestValidateSchemaName() {
	// Test valid schema name
	err := validateSchemaName("schema.json")
	suite.NoError(err, "Valid schema name should not return error")

	err = validateSchemaName("my-schema-file.json")
	suite.NoError(err, "Valid schema name with dashes should not return error")

	// Test empty schema name
	err = validateSchemaName("")
	suite.Error(err, "Empty schema name should return error")
	suite.Contains(err.Error(), "schema name cannot be empty")

	// Test schema name without .json extension
	err = validateSchemaName("schema.txt")
	suite.Error(err, "Schema name without .json should return error")
	suite.Contains(err.Error(), "must have .json extension")

	err = validateSchemaName("schema")
	suite.Error(err, "Schema name without extension should return error")
}

func (suite *GenerateCmdTestSuite) TestGetSchemaReference() {
	ref := getSchemaReference("test-schema.json")
	suite.Equal("# yaml-language-server: $schema=test-schema.json\n", ref)

	ref = getSchemaReference("another.json")
	suite.Equal("# yaml-language-server: $schema=another.json\n", ref)

	ref = getSchemaReference("")
	suite.Equal("# yaml-language-server: $schema=\n", ref)
}

// Helper functions
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return !os.IsNotExist(err) && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return !os.IsNotExist(err) && !info.IsDir()
}

func TestGenerateCmdSuite(t *testing.T) {
	suite.Run(t, new(GenerateCmdTestSuite))
}
