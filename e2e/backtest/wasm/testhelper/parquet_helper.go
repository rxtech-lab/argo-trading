package testhelper

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
)

// UpdateParquetSymbol reads a Parquet file, updates all symbols to the provided symbol,
// and writes the result to a new Parquet file.
//
// Parameters:
//   - inputPath: Path to the input Parquet file
//   - outputPath: Path to write the updated Parquet file
//   - newSymbol: The symbol to replace all existing symbols with
//
// Returns:
//   - error: Any error encountered during the process
func UpdateParquetSymbol(inputPath, outputPath, newSymbol string) error {
	// Create directory for output file if it doesn't exist
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Open in-memory DuckDB database
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	// Set DuckDB optimizations
	_, err = db.Exec(`
		SET memory_limit='2GB';
		SET threads=4;
		SET temp_directory='./temp';
	`)
	if err != nil {
		return fmt.Errorf("failed to set DuckDB optimizations: %w", err)
	}

	// Create a temporary table from the Parquet file
	_, err = db.Exec(fmt.Sprintf(`
		CREATE TABLE market_data AS 
		SELECT time, '%s' AS symbol, open, high, low, close, volume
		FROM read_parquet('%s');
	`, newSymbol, inputPath))
	if err != nil {
		return fmt.Errorf("failed to create table from Parquet: %w", err)
	}

	// Export the updated data to a new Parquet file
	_, err = db.Exec(fmt.Sprintf(`
		COPY market_data TO '%s' (FORMAT PARQUET);
	`, outputPath))
	if err != nil {
		return fmt.Errorf("failed to export to Parquet: %w", err)
	}

	return nil
}

// UpdateMultipleParquetSymbols reads all Parquet files matching the pattern,
// updates the symbol in each to a unique identifier based on the filename,
// and saves the updated files to the output directory.
//
// Parameters:
//   - inputPattern: Glob pattern to match input Parquet files (e.g., "/path/to/data/*.parquet")
//   - outputDir: Directory to save the updated Parquet files
//   - symbolPrefix: Optional prefix for the new symbols (will be combined with file index)
//
// Returns:
//   - []string: Paths to the generated output files
//   - error: Any error encountered during the process
func UpdateMultipleParquetSymbols(inputPattern, outputDir, symbolPrefix string) ([]string, error) {
	// Find all matching input files
	inputFiles, err := filepath.Glob(inputPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find files matching pattern %s: %w", inputPattern, err)
	}

	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("no files found matching pattern %s", inputPattern)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	outputFiles := make([]string, 0, len(inputFiles))

	// Process each input file
	for i, inputFile := range inputFiles {
		// Generate new symbol for this file
		newSymbol := fmt.Sprintf("%s%d", symbolPrefix, i+1)

		// Generate output filename
		baseFileName := filepath.Base(inputFile)
		outputFile := filepath.Join(outputDir, baseFileName)

		// Update symbol in the file
		if err := UpdateParquetSymbol(inputFile, outputFile, newSymbol); err != nil {
			return outputFiles, fmt.Errorf("failed to update symbol in file %s: %w", inputFile, err)
		}

		outputFiles = append(outputFiles, outputFile)
	}

	return outputFiles, nil
}
