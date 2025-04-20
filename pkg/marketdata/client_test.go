package marketdata

import (
	"os"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/polygon-io/client-go/rest/models"
	"github.com/rxtech-lab/argo-trading/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// ClientTestSuite is a test suite for the Client implementation
type ClientTestSuite struct {
	suite.Suite
	ctrl         *gomock.Controller
	mockProvider *mocks.MockProvider
	tempDir      string
}

// SetupSuite runs once before all tests in the suite
func (suite *ClientTestSuite) SetupSuite() {
	// Create a temporary directory for test data
	tempDir, err := os.MkdirTemp("", "marketdata-client-test")
	suite.Require().NoError(err)
	suite.tempDir = tempDir
}

// TearDownSuite runs once after all tests in the suite
func (suite *ClientTestSuite) TearDownSuite() {
	// Remove the temporary directory
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// SetupTest runs before each test
func (suite *ClientTestSuite) SetupTest() {
	// Create a new mock controller and provider for each test
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockProvider = mocks.NewMockProvider(suite.ctrl)
}

// TearDownTest runs after each test
func (suite *ClientTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// TestClientDownload tests the Download method
func (suite *ClientTestSuite) TestClientDownload() {
	// Test cases
	testCases := []struct {
		name        string
		params      DownloadParams
		setupMock   func()
		expectError bool
	}{
		{
			name: "successful download",
			params: DownloadParams{
				Ticker:     "AAPL",
				StartDate:  time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:    time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
				Multiplier: 1,
				Timespan:   models.Minute,
			},
			setupMock: func() {
				// Mock ConfigWriter call
				suite.mockProvider.EXPECT().
					ConfigWriter(gomock.Any()).
					Times(1)

				// Mock successful download
				suite.mockProvider.EXPECT().
					Download(
						"AAPL",
						time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
						time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
						1,
						models.Minute,
						gomock.Any(),
					).
					Return("path/to/data", nil).
					Times(1)
			},
			expectError: false,
		},
		{
			name: "download error",
			params: DownloadParams{
				Ticker:     "INVALID",
				StartDate:  time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:    time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
				Multiplier: 1,
				Timespan:   models.Minute,
			},
			setupMock: func() {
				// Mock ConfigWriter call
				suite.mockProvider.EXPECT().
					ConfigWriter(gomock.Any()).
					Times(1)

				// Mock download error
				suite.mockProvider.EXPECT().
					Download(
						"INVALID",
						time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
						time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
						1,
						models.Minute,
						gomock.Any(),
					).
					Return("", os.ErrNotExist).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup mock expectations
			tc.setupMock()

			// Create client with mocked provider
			client := &Client{
				provider: suite.mockProvider,
				config: ClientConfig{
					ProviderType: ProviderPolygon,
					WriterType:   WriterDuckDB,
					DataPath:     suite.tempDir,
				},
				validate: validator.New(),
			}

			// Execute Download and check result
			err := client.Download(tc.params)

			if tc.expectError {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
		})
	}
}

// TestClientConfigValidation tests the validation of the ClientConfig struct
func (suite *ClientTestSuite) TestClientConfigValidation() {
	testCases := []struct {
		name        string
		config      ClientConfig
		expectError bool
		errorField  string
	}{
		{
			name: "valid polygon config",
			config: ClientConfig{
				ProviderType:  ProviderPolygon,
				WriterType:    WriterDuckDB,
				DataPath:      suite.tempDir,
				PolygonApiKey: "test-api-key",
			},
			expectError: false,
		},
		{
			name: "valid binance config with mainnet",
			config: ClientConfig{
				ProviderType: ProviderBinance,
				WriterType:   WriterDuckDB,
				DataPath:     suite.tempDir,
			},
			expectError: false,
		},
		{
			name: "valid binance config with testnet",
			config: ClientConfig{
				ProviderType: ProviderBinance,
				WriterType:   WriterDuckDB,
				DataPath:     suite.tempDir,
			},
			expectError: false,
		},
		{
			name: "missing provider type",
			config: ClientConfig{
				WriterType:    WriterDuckDB,
				DataPath:      suite.tempDir,
				PolygonApiKey: "test-api-key",
			},
			expectError: true,
			errorField:  "ProviderType",
		},
		{
			name: "invalid provider type",
			config: ClientConfig{
				ProviderType:  "invalid",
				WriterType:    WriterDuckDB,
				DataPath:      suite.tempDir,
				PolygonApiKey: "test-api-key",
			},
			expectError: true,
			errorField:  "ProviderType",
		},
		{
			name: "missing writer type",
			config: ClientConfig{
				ProviderType:  ProviderPolygon,
				DataPath:      suite.tempDir,
				PolygonApiKey: "test-api-key",
			},
			expectError: true,
			errorField:  "WriterType",
		},
		{
			name: "invalid writer type",
			config: ClientConfig{
				ProviderType:  ProviderPolygon,
				WriterType:    "invalid",
				DataPath:      suite.tempDir,
				PolygonApiKey: "test-api-key",
			},
			expectError: true,
			errorField:  "WriterType",
		},
		{
			name: "missing data path",
			config: ClientConfig{
				ProviderType:  ProviderPolygon,
				WriterType:    WriterDuckDB,
				PolygonApiKey: "test-api-key",
			},
			expectError: true,
			errorField:  "DataPath",
		},
		{
			name: "missing polygon api key",
			config: ClientConfig{
				ProviderType: ProviderPolygon,
				WriterType:   WriterDuckDB,
				DataPath:     suite.tempDir,
			},
			expectError: true,
			errorField:  "PolygonApiKey",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create validator
			validate := validator.New()

			// Validate the config
			err := validate.Struct(tc.config)

			if tc.expectError {
				suite.Error(err, "Expected validation error but got none")
				if err != nil {
					// Check if the error is related to the expected field
					suite.Contains(err.Error(), tc.errorField, "Error should be related to the expected field")
				}
			} else {
				suite.NoError(err, "Unexpected validation error")
			}
		})
	}
}

// TestDownloadParamsValidation tests the validation of the DownloadParams struct
func (suite *ClientTestSuite) TestDownloadParamsValidation() {
	now := time.Now()

	testCases := []struct {
		name        string
		params      DownloadParams
		expectError bool
		errorField  string
	}{
		{
			name: "valid download params",
			params: DownloadParams{
				Ticker:     "AAPL",
				StartDate:  now.Add(-24 * time.Hour),
				EndDate:    now,
				Multiplier: 1,
				Timespan:   models.Minute,
			},
			expectError: false,
		},
		{
			name: "missing ticker",
			params: DownloadParams{
				StartDate:  now.Add(-24 * time.Hour),
				EndDate:    now,
				Multiplier: 1,
				Timespan:   models.Minute,
			},
			expectError: true,
			errorField:  "Ticker",
		},
		{
			name: "missing start date",
			params: DownloadParams{
				Ticker:     "AAPL",
				EndDate:    now,
				Multiplier: 1,
				Timespan:   models.Minute,
			},
			expectError: true,
			errorField:  "StartDate",
		},
		{
			name: "missing end date",
			params: DownloadParams{
				Ticker:     "AAPL",
				StartDate:  now.Add(-24 * time.Hour),
				Multiplier: 1,
				Timespan:   models.Minute,
			},
			expectError: true,
			errorField:  "EndDate",
		},
		{
			name: "end date before start date",
			params: DownloadParams{
				Ticker:     "AAPL",
				StartDate:  now,
				EndDate:    now.Add(-24 * time.Hour),
				Multiplier: 1,
				Timespan:   models.Minute,
			},
			expectError: true,
			errorField:  "EndDate",
		},
		{
			name: "missing multiplier",
			params: DownloadParams{
				Ticker:    "AAPL",
				StartDate: now.Add(-24 * time.Hour),
				EndDate:   now,
				Timespan:  models.Minute,
			},
			expectError: true,
			errorField:  "Multiplier",
		},
		{
			name: "invalid multiplier (less than 1)",
			params: DownloadParams{
				Ticker:     "AAPL",
				StartDate:  now.Add(-24 * time.Hour),
				EndDate:    now,
				Multiplier: 0,
				Timespan:   models.Minute,
			},
			expectError: true,
			errorField:  "Multiplier",
		},
		{
			name: "missing timespan",
			params: DownloadParams{
				Ticker:     "AAPL",
				StartDate:  now.Add(-24 * time.Hour),
				EndDate:    now,
				Multiplier: 1,
			},
			expectError: true,
			errorField:  "Timespan",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create validator
			validate := validator.New()

			// Validate the parameters
			err := validate.Struct(tc.params)

			if tc.expectError {
				suite.Error(err, "Expected validation error but got none")
				if err != nil {
					// Check if the error is related to the expected field
					suite.Contains(err.Error(), tc.errorField, "Error should be related to the expected field")
				}
			} else {
				suite.NoError(err, "Unexpected validation error")
			}
		})
	}
}

// TestNewClient tests the NewClient constructor with various configurations
func (suite *ClientTestSuite) TestNewClient() {
	// Since we can't easily test the actual NewClient function without mocking
	// the provider creation, we'll just test the configuration validation directly.
	// This is because NewClient makes external calls to create the real providers.

	// Test cases
	testCases := []struct {
		name          string
		config        ClientConfig
		expectError   bool
		errorContains string
	}{
		{
			name: "invalid config - missing provider type",
			config: ClientConfig{
				WriterType:    WriterDuckDB,
				DataPath:      suite.tempDir,
				PolygonApiKey: "test-api-key",
			},
			expectError:   true,
			errorContains: "invalid client configuration",
		},
		{
			name: "invalid config - unknown provider type",
			config: ClientConfig{
				ProviderType:  "unknown",
				WriterType:    WriterDuckDB,
				DataPath:      suite.tempDir,
				PolygonApiKey: "test-api-key",
			},
			expectError:   true,
			errorContains: "invalid client configuration",
		},
		{
			name: "invalid config - missing polygon API key",
			config: ClientConfig{
				ProviderType: ProviderPolygon,
				WriterType:   WriterDuckDB,
				DataPath:     suite.tempDir,
			},
			expectError:   true,
			errorContains: "invalid client configuration",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			validate := validator.New()

			// Just validate the config, don't actually create the client
			// since that would try to make external API calls
			err := validate.Struct(tc.config)

			if tc.expectError {
				suite.Error(err, "Expected error but got none")
			} else {
				suite.NoError(err, "Unexpected validation error")
			}
		})
	}
}

// TestClientSuite runs the test suite
func TestClientSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
