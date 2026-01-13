package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

	// Verify run ID is run_1
	s.Equal("run_1", sm.GetRunID())
	s.Equal(1, sm.GetRunNumber())

	// Verify folder was created
	runPath := sm.GetCurrentRunPath()
	s.DirExists(runPath)

	// Verify folder path format
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

	// Verify run ID is run_2
	s.Equal("run_2", sm.GetRunID())
	s.Equal(2, sm.GetRunNumber())
}

func (s *SessionManagerTestSuite) TestInitialize_MultipleExistingRuns() {
	// Create multiple existing run folders
	today := time.Now().Format("2006-01-02")
	for i := 1; i <= 5; i++ {
		runPath := filepath.Join(s.tempDir, today, "run_"+string(rune('0'+i)))
		err := os.MkdirAll(runPath, 0755)
		s.Require().NoError(err)
	}

	sm := NewSessionManager(s.logger)

	err := sm.Initialize(s.tempDir)
	s.Require().NoError(err)

	// Verify run ID is run_6
	s.Equal("run_6", sm.GetRunID())
	s.Equal(6, sm.GetRunNumber())
}

func (s *SessionManagerTestSuite) TestInitialize_NonSequentialRuns() {
	// Create non-sequential run folders (run_1, run_3, run_7)
	today := time.Now().Format("2006-01-02")
	for _, num := range []int{1, 3, 7} {
		runPath := filepath.Join(s.tempDir, today, "run_"+string(rune('0'+num)))
		err := os.MkdirAll(runPath, 0755)
		s.Require().NoError(err)
	}

	sm := NewSessionManager(s.logger)

	err := sm.Initialize(s.tempDir)
	s.Require().NoError(err)

	// Verify run ID is run_8 (max + 1)
	s.Equal("run_8", sm.GetRunID())
	s.Equal(8, sm.GetRunNumber())
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
	runID := sm.GetRunID()

	// Simulate next day
	tomorrow := time.Now().AddDate(0, 0, 1)

	changed, err := sm.HandleDateBoundary(tomorrow)
	s.Require().NoError(err)
	s.True(changed)

	// Run ID should remain the same
	s.Equal(runID, sm.GetRunID())

	// Date should change
	s.NotEqual(originalDate, sm.GetCurrentDate())
	s.Equal(tomorrow.Format("2006-01-02"), sm.GetCurrentDate())

	// New folder should exist
	s.DirExists(sm.GetCurrentRunPath())

	// Path should be different
	expectedPath := filepath.Join(s.tempDir, tomorrow.Format("2006-01-02"), runID)
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
	for i := 1; i <= 3; i++ {
		runPath := filepath.Join(s.tempDir, today, "run_"+string(rune('0'+i)))
		err := os.MkdirAll(runPath, 0755)
		s.Require().NoError(err)
	}

	sm := NewSessionManager(s.logger)
	err := sm.Initialize(s.tempDir)
	s.Require().NoError(err)

	sessions, err := sm.ListSessionsForDate(today)
	s.Require().NoError(err)

	// Should include run_1, run_2, run_3 plus the new run_4
	s.Len(sessions, 4)
	s.Contains(sessions, "run_1")
	s.Contains(sessions, "run_2")
	s.Contains(sessions, "run_3")
	s.Contains(sessions, "run_4")
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

func (s *SessionManagerTestSuite) TestIgnoresNonRunFolders() {
	// Create various folders that shouldn't be counted as runs
	today := time.Now().Format("2006-01-02")
	todayPath := filepath.Join(s.tempDir, today)
	err := os.MkdirAll(todayPath, 0755)
	s.Require().NoError(err)

	// Create non-run folders
	nonRunFolders := []string{"backup", "run", "run_", "run_abc", "test_1"}
	for _, folder := range nonRunFolders {
		folderPath := filepath.Join(todayPath, folder)
		err := os.MkdirAll(folderPath, 0755)
		s.Require().NoError(err)
	}

	// Create a valid run folder
	validRunPath := filepath.Join(todayPath, "run_5")
	err = os.MkdirAll(validRunPath, 0755)
	s.Require().NoError(err)

	sm := NewSessionManager(s.logger)
	err = sm.Initialize(s.tempDir)
	s.Require().NoError(err)

	// Should be run_6 (5 + 1)
	s.Equal("run_6", sm.GetRunID())
	s.Equal(6, sm.GetRunNumber())
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
	runPath := filepath.Join(datePath, "run_1")
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

	// Should only contain run_1, not the file
	s.Len(sessions, 1)
	s.Contains(sessions, "run_1")
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
