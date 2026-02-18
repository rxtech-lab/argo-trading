package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/stretchr/testify/suite"
)

type SessionManagerTestSuite struct {
	suite.Suite
	tempDir string
	logger  *logger.Logger
}

func (s *SessionManagerTestSuite) SetupSuite() {
	log, err := logger.NewLogger()
	s.Require().NoError(err)
	s.logger = log
}

func (s *SessionManagerTestSuite) SetupTest() {
	// Create a temporary directory for each test
	tempDir, err := os.MkdirTemp("", "session_manager_test_*")
	s.Require().NoError(err)
	s.tempDir = tempDir
}

func (s *SessionManagerTestSuite) TearDownTest() {
	// Clean up temporary directory after each test
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

func TestSessionManagerTestSuite(t *testing.T) {
	suite.Run(t, new(SessionManagerTestSuite))
}

func (s *SessionManagerTestSuite) TestInitialize_FirstRun() {
	sm := NewSessionManager(s.logger)

	err := sm.Initialize(s.tempDir)
	s.Require().NoError(err)

	// Verify run ID is a valid UUID
	runID := sm.GetRunID()
	_, err = uuid.Parse(runID)
	s.NoError(err, "runID should be a valid UUID")

	// Verify run name is run_1
	s.Equal("run_1", sm.GetRunName())

	// Verify folder was created using run name
	runPath := sm.GetCurrentRunPath()
	s.DirExists(runPath)

	// Verify folder path format uses run_N naming
	today := time.Now().Format("2006-01-02")
	expectedPath := filepath.Join(s.tempDir, today, "run_1")
	s.Equal(expectedPath, runPath)
}

func (s *SessionManagerTestSuite) TestInitialize_SecondRun() {
	// Create first run folder
	today := time.Now().Format("2006-01-02")
	firstRunPath := filepath.Join(s.tempDir, today, "run_1")
	err := os.MkdirAll(firstRunPath, 0755)
	s.Require().NoError(err)

	sm := NewSessionManager(s.logger)

	err = sm.Initialize(s.tempDir)
	s.Require().NoError(err)

	// Verify run name is run_2 (incremented from existing run_1)
	s.Equal("run_2", sm.GetRunName())
}

func (s *SessionManagerTestSuite) TestInitialize_IncrementsRunNumber() {
	sm1 := NewSessionManager(s.logger)
	err := sm1.Initialize(s.tempDir)
	s.Require().NoError(err)
	s.Equal("run_1", sm1.GetRunName())

	// Second session in the same directory gets run_2
	sm2 := NewSessionManager(s.logger)
	err = sm2.Initialize(s.tempDir)
	s.Require().NoError(err)
	s.Equal("run_2", sm2.GetRunName())

	// UUIDs should be different
	s.NotEqual(sm1.GetRunID(), sm2.GetRunID())
}

func (s *SessionManagerTestSuite) TestHandleDateBoundary_SameDate() {
	sm := NewSessionManager(s.logger)
	err := sm.Initialize(s.tempDir)
	s.Require().NoError(err)

	originalPath := sm.GetCurrentRunPath()

	// Same date - should not create new folder
	changed, err := sm.HandleDateBoundary(time.Now())
	s.Require().NoError(err)
	s.False(changed)
	s.Equal(originalPath, sm.GetCurrentRunPath())
}

func (s *SessionManagerTestSuite) TestHandleDateBoundary_NewDate() {
	sm := NewSessionManager(s.logger)
	err := sm.Initialize(s.tempDir)
	s.Require().NoError(err)

	originalDate := sm.GetCurrentDate()
	runName := sm.GetRunName()

	// Simulate next day
	tomorrow := time.Now().AddDate(0, 0, 1)

	changed, err := sm.HandleDateBoundary(tomorrow)
	s.Require().NoError(err)
	s.True(changed)

	// Run name should remain the same
	s.Equal(runName, sm.GetRunName())

	// Date should change
	s.NotEqual(originalDate, sm.GetCurrentDate())
	s.Equal(tomorrow.Format("2006-01-02"), sm.GetCurrentDate())

	// New folder should exist
	s.DirExists(sm.GetCurrentRunPath())

	// Path should use run name
	expectedPath := filepath.Join(s.tempDir, tomorrow.Format("2006-01-02"), runName)
	s.Equal(expectedPath, sm.GetCurrentRunPath())
}

func (s *SessionManagerTestSuite) TestGetFilePath() {
	sm := NewSessionManager(s.logger)
	err := sm.Initialize(s.tempDir)
	s.Require().NoError(err)

	filePath := sm.GetFilePath("stats.yaml")

	expectedPath := filepath.Join(sm.GetCurrentRunPath(), "stats.yaml")
	s.Equal(expectedPath, filePath)
}

func (s *SessionManagerTestSuite) TestListSessionsForDate() {
	// Create multiple run folders for today
	today := time.Now().Format("2006-01-02")
	existingFolders := []string{"folder-a", "folder-b", "folder-c"}
	for _, folder := range existingFolders {
		runPath := filepath.Join(s.tempDir, today, folder)
		err := os.MkdirAll(runPath, 0755)
		s.Require().NoError(err)
	}

	sm := NewSessionManager(s.logger)
	err := sm.Initialize(s.tempDir)
	s.Require().NoError(err)

	sessions, err := sm.ListSessionsForDate(today)
	s.Require().NoError(err)

	// Should include the 3 pre-created folders plus the new run_1 folder
	s.Len(sessions, 4)
	for _, folder := range existingFolders {
		s.Contains(sessions, folder)
	}
	s.Contains(sessions, sm.GetRunName())
}

func (s *SessionManagerTestSuite) TestListSessionsForDate_NoSessions() {
	sm := NewSessionManager(s.logger)
	err := sm.Initialize(s.tempDir)
	s.Require().NoError(err)

	// Query for a date with no sessions
	sessions, err := sm.ListSessionsForDate("2020-01-01")
	s.Require().NoError(err)
	s.Empty(sessions)
}

func (s *SessionManagerTestSuite) TestGetAllDates() {
	// Create folders for multiple dates
	dates := []string{"2025-01-01", "2025-01-02", "2025-01-03"}
	for _, date := range dates {
		runPath := filepath.Join(s.tempDir, date, "run_1")
		err := os.MkdirAll(runPath, 0755)
		s.Require().NoError(err)
	}

	sm := NewSessionManager(s.logger)
	err := sm.Initialize(s.tempDir)
	s.Require().NoError(err)

	allDates, err := sm.GetAllDates()
	s.Require().NoError(err)

	// Should include the 3 created dates plus today
	s.GreaterOrEqual(len(allDates), 3)
	s.Contains(allDates, "2025-01-01")
	s.Contains(allDates, "2025-01-02")
	s.Contains(allDates, "2025-01-03")
}

func (s *SessionManagerTestSuite) TestGetSessionStart() {
	sm := NewSessionManager(s.logger)

	before := time.Now()
	err := sm.Initialize(s.tempDir)
	s.Require().NoError(err)
	after := time.Now()

	sessionStart := sm.GetSessionStart()

	// Session start should be between before and after
	s.True(sessionStart.After(before) || sessionStart.Equal(before))
	s.True(sessionStart.Before(after) || sessionStart.Equal(after))
}

func (s *SessionManagerTestSuite) TestConcurrentAccess() {
	sm := NewSessionManager(s.logger)
	err := sm.Initialize(s.tempDir)
	s.Require().NoError(err)

	// Concurrent access to various methods
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_ = sm.GetRunID()
			_ = sm.GetRunName()
			_ = sm.GetCurrentRunPath()
			_ = sm.GetCurrentDate()
			_ = sm.GetSessionStart()
			_ = sm.GetFilePath("test.yaml")
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// ============================================================================
// Additional Edge Case Tests
// ============================================================================

func (s *SessionManagerTestSuite) TestGetDataOutputPath() {
	sm := NewSessionManager(s.logger)

	err := sm.Initialize(s.tempDir)
	s.Require().NoError(err)

	s.Equal(s.tempDir, sm.GetDataOutputPath())
}

func (s *SessionManagerTestSuite) TestGetDataOutputPath_BeforeInitialize() {
	sm := NewSessionManager(s.logger)

	// Before initialization, data output path should be empty
	s.Equal("", sm.GetDataOutputPath())
}

func (s *SessionManagerTestSuite) TestGetAllDates_NoDataPath() {
	sm := NewSessionManager(s.logger)

	// Use a non-existent path
	sm.dataOutputPath = filepath.Join(s.tempDir, "nonexistent")

	dates, err := sm.GetAllDates()
	s.NoError(err)
	s.Empty(dates)
}

func (s *SessionManagerTestSuite) TestListSessionsForDate_WithFilesInDirectory() {
	// Create date folder with both run folders and files
	today := time.Now().Format("2006-01-02")
	datePath := filepath.Join(s.tempDir, today)
	err := os.MkdirAll(datePath, 0755)
	s.Require().NoError(err)

	// Create valid run folder
	runPath := filepath.Join(datePath, "session-1")
	err = os.MkdirAll(runPath, 0755)
	s.Require().NoError(err)

	// Create a file (should be ignored)
	filePath := filepath.Join(datePath, "stats.yaml")
	err = os.WriteFile(filePath, []byte("test"), 0644)
	s.Require().NoError(err)

	sm := NewSessionManager(s.logger)
	sm.dataOutputPath = s.tempDir

	sessions, err := sm.ListSessionsForDate(today)
	s.Require().NoError(err)

	// Should only contain session-1, not the file
	s.Len(sessions, 1)
	s.Contains(sessions, "session-1")
}

func (s *SessionManagerTestSuite) TestGetAllDates_WithFilesInDirectory() {
	// Create date folder
	datePath := filepath.Join(s.tempDir, "2025-01-15")
	err := os.MkdirAll(datePath, 0755)
	s.Require().NoError(err)

	// Create a file at root level (should be ignored)
	filePath := filepath.Join(s.tempDir, "config.yaml")
	err = os.WriteFile(filePath, []byte("test"), 0644)
	s.Require().NoError(err)

	// Create an invalid date folder (should be ignored)
	invalidPath := filepath.Join(s.tempDir, "not-a-date")
	err = os.MkdirAll(invalidPath, 0755)
	s.Require().NoError(err)

	sm := NewSessionManager(s.logger)
	sm.dataOutputPath = s.tempDir

	dates, err := sm.GetAllDates()
	s.Require().NoError(err)

	// Should only contain valid date folder
	s.Len(dates, 1)
	s.Contains(dates, "2025-01-15")
}

func (s *SessionManagerTestSuite) TestGetCurrentDate() {
	sm := NewSessionManager(s.logger)

	err := sm.Initialize(s.tempDir)
	s.Require().NoError(err)

	today := time.Now().Format("2006-01-02")
	s.Equal(today, sm.GetCurrentDate())
}
