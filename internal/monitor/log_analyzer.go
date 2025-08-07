package monitor

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/0xReLogic/Aegis-Orchestrator/pkg/logger"
)

// LogPattern represents a pattern to look for in logs
type LogPattern struct {
	Name        string
	Pattern     *regexp.Regexp
	Severity    string
	Description string
	Count       int
	LastSeen    time.Time
}

// LogAnalyzer analyzes logs for patterns
type LogAnalyzer struct {
	patterns     []*LogPattern
	logFiles     map[string]string
	patternMutex sync.RWMutex
	fileMutex    sync.RWMutex
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// NewLogAnalyzer creates a new log analyzer
func NewLogAnalyzer() *LogAnalyzer {
	return &LogAnalyzer{
		patterns: make([]*LogPattern, 0),
		logFiles: make(map[string]string),
		stopChan: make(chan struct{}),
	}
}

// AddPattern adds a pattern to look for
func (l *LogAnalyzer) AddPattern(name, pattern, severity, description string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	l.patternMutex.Lock()
	defer l.patternMutex.Unlock()

	l.patterns = append(l.patterns, &LogPattern{
		Name:        name,
		Pattern:     re,
		Severity:    severity,
		Description: description,
		Count:       0,
		LastSeen:    time.Time{},
	})

	logger.Info("Added log pattern: %s (%s)", name, pattern)
	return nil
}

// AddLogFile adds a log file to monitor
func (l *LogAnalyzer) AddLogFile(serviceName, filePath string) error {
	// Check if file exists and is readable
	_, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("error accessing log file: %w", err)
	}

	l.fileMutex.Lock()
	defer l.fileMutex.Unlock()

	l.logFiles[serviceName] = filePath
	logger.Info("Added log file for service %s: %s", serviceName, filePath)
	return nil
}

// Start starts the log analyzer
func (l *LogAnalyzer) Start(interval time.Duration) {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-l.stopChan:
				logger.Info("Stopping log analyzer")
				return
			case <-ticker.C:
				l.analyzeAllLogs()
			}
		}
	}()

	logger.Info("Started log analyzer with interval %s", interval)
}

// Stop stops the log analyzer
func (l *LogAnalyzer) Stop() {
	close(l.stopChan)
	l.wg.Wait()
	logger.Info("Log analyzer stopped")
}

// analyzeAllLogs analyzes all registered log files
func (l *LogAnalyzer) analyzeAllLogs() {
	l.fileMutex.RLock()
	defer l.fileMutex.RUnlock()

	for service, filePath := range l.logFiles {
		if err := l.analyzeLogFile(service, filePath); err != nil {
			logger.Error("Error analyzing log file for service %s: %v", service, err)
		}
	}
}

// analyzeLogFile analyzes a single log file
func (l *LogAnalyzer) analyzeLogFile(serviceName, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening log file: %w", err)
	}
	defer file.Close()

	// Get file size to determine where to start reading
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info: %w", err)
	}

	// Only read the last 10MB of the file to avoid memory issues with large logs
	var startOffset int64 = 0
	if fileInfo.Size() > 10*1024*1024 {
		startOffset = fileInfo.Size() - 10*1024*1024
	}

	_, err = file.Seek(startOffset, 0)
	if err != nil {
		return fmt.Errorf("error seeking in file: %w", err)
	}

	scanner := bufio.NewScanner(file)
	lineCount := 0
	matchCount := 0

	l.patternMutex.RLock()
	patterns := make([]*LogPattern, len(l.patterns))
	copy(patterns, l.patterns)
	l.patternMutex.RUnlock()

	// Scan through the file
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// Check each pattern
		for _, pattern := range patterns {
			if pattern.Pattern.MatchString(line) {
				matchCount++
				l.patternMutex.Lock()
				pattern.Count++
				pattern.LastSeen = time.Now()
				l.patternMutex.Unlock()

				logger.Warn("Log pattern match for service %s: %s - %s",
					serviceName, pattern.Name, strings.TrimSpace(line))
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning log file: %w", err)
	}

	logger.Debug("Analyzed %d lines in log file for service %s, found %d matches",
		lineCount, serviceName, matchCount)
	return nil
}

// GetPatternMatches returns the current pattern matches
func (l *LogAnalyzer) GetPatternMatches() []*LogPattern {
	l.patternMutex.RLock()
	defer l.patternMutex.RUnlock()

	// Create a copy to avoid race conditions
	result := make([]*LogPattern, len(l.patterns))
	for i, pattern := range l.patterns {
		patternCopy := *pattern
		result[i] = &patternCopy
	}

	return result
}

// ResetPatternCounts resets the pattern match counts
func (l *LogAnalyzer) ResetPatternCounts() {
	l.patternMutex.Lock()
	defer l.patternMutex.Unlock()

	for _, pattern := range l.patterns {
		pattern.Count = 0
	}

	logger.Info("Reset all log pattern counts")
}
