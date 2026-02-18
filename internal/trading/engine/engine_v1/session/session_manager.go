package session

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"go.uber.org/zap"
)

// SessionManager handles session lifecycle and folder management for live trading.
// It creates and manages the folder structure:
//
//	{dataOutputPath}/{YYYY-MM-DD}/run_N/
type SessionManager struct {
	dataOutputPath string
	runID          string // UUID for unique identification
	runName        string // run_N for folder naming
	sessionStart   time.Time
	currentDate    string
	currentRunPath string
	mu             sync.Mutex
	logger         *logger.Logger
}

// NewSessionManager creates a new SessionManager instance.
func NewSessionManager(log *logger.Logger) *SessionManager {
	return &SessionManager{
		dataOutputPath: "",
		runID:          "",
		runName:        "",
		sessionStart:   time.Time{},
		currentDate:    "",
		currentRunPath: "",
		mu:             sync.Mutex{},
		logger:         log,
	}
}

// Initialize sets up the session manager with the data output path.
// It determines the next run number and creates the folder structure.
func (s *SessionManager) Initialize(dataOutputPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.dataOutputPath = dataOutputPath
	s.sessionStart = time.Now()
	s.currentDate = s.sessionStart.Format("2006-01-02")

	// Generate a unique run ID
	s.runID = uuid.New().String()

	// Determine the next run number for folder naming
	nextRun, err := s.nextRunNumber()
	if err != nil {
		return fmt.Errorf("failed to determine next run number: %w", err)
	}

	s.runName = fmt.Sprintf("run_%d", nextRun)

	// Create folder structure
	if err := s.createFolderStructure(); err != nil {
		return fmt.Errorf("failed to create folder structure: %w", err)
	}

	s.logger.Info("Session initialized",
		zap.String("run_id", s.runID),
		zap.String("date", s.currentDate),
		zap.String("path", s.currentRunPath),
	)

	return nil
}

// nextRunNumber scans the current date folder and returns the next run number.
//
//nolint:funcorder // helper method used by Initialize
func (s *SessionManager) nextRunNumber() (int, error) {
	datePath := filepath.Join(s.dataOutputPath, s.currentDate)

	entries, err := os.ReadDir(datePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, nil
		}

		return 0, fmt.Errorf("failed to read date directory: %w", err)
	}

	maxRun := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, "run_") {
			continue
		}

		numStr := strings.TrimPrefix(name, "run_")

		num, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}

		if num > maxRun {
			maxRun = num
		}
	}

	return maxRun + 1, nil
}

// createFolderStructure creates the folder structure for the current session.
//
//nolint:funcorder // helper method used by Initialize and HandleDateBoundary
func (s *SessionManager) createFolderStructure() error {
	s.currentRunPath = filepath.Join(s.dataOutputPath, s.currentDate, s.runName)

	if err := os.MkdirAll(s.currentRunPath, 0755); err != nil {
		return fmt.Errorf("failed to create run folder: %w", err)
	}

	return nil
}

// HandleDateBoundary checks if the date has changed and creates a new folder if needed.
// Returns true if a new folder was created (date boundary crossed).
func (s *SessionManager) HandleDateBoundary(timestamp time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newDate := timestamp.Format("2006-01-02")
	if newDate == s.currentDate {
		return false, nil
	}

	// Date has changed - create new folder with same run number
	oldDate := s.currentDate
	s.currentDate = newDate

	if err := s.createFolderStructure(); err != nil {
		return false, fmt.Errorf("failed to create folder for new date: %w", err)
	}

	s.logger.Info("Date boundary crossed, created new folder",
		zap.String("old_date", oldDate),
		zap.String("new_date", newDate),
		zap.String("run_id", s.runID),
		zap.String("new_path", s.currentRunPath),
	)

	return true, nil
}

// GetCurrentRunPath returns the current run folder path.
func (s *SessionManager) GetCurrentRunPath() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentRunPath
}

// GetRunID returns the session run ID (UUID format).
func (s *SessionManager) GetRunID() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.runID
}

// GetRunName returns the session run name (run_N format) used for folder naming.
func (s *SessionManager) GetRunName() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.runName
}

// GetSessionStart returns the session start time.
func (s *SessionManager) GetSessionStart() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.sessionStart
}

// GetCurrentDate returns the current date in YYYY-MM-DD format.
func (s *SessionManager) GetCurrentDate() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentDate
}

// GetDataOutputPath returns the base data output path.
func (s *SessionManager) GetDataOutputPath() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.dataOutputPath
}

// GetFilePath returns the full path for a file in the current run folder.
func (s *SessionManager) GetFilePath(filename string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return filepath.Join(s.currentRunPath, filename)
}

// ListSessionsForDate returns all run IDs for a given date.
func (s *SessionManager) ListSessionsForDate(date string) ([]string, error) {
	datePath := filepath.Join(s.dataOutputPath, date)

	if _, err := os.Stat(datePath); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(datePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read date directory: %w", err)
	}

	var runs []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		runs = append(runs, entry.Name())
	}

	sort.Strings(runs)

	return runs, nil
}

// GetAllDates returns all dates with session data.
func (s *SessionManager) GetAllDates() ([]string, error) {
	if _, err := os.Stat(s.dataOutputPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(s.dataOutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read data output directory: %w", err)
	}

	var dates []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Validate date format YYYY-MM-DD
		if _, err := time.Parse("2006-01-02", entry.Name()); err == nil {
			dates = append(dates, entry.Name())
		}
	}

	sort.Strings(dates)

	return dates, nil
}
