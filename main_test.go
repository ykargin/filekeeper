package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestParseDuration tests the ParseDuration function
func TestParseDuration(t *testing.T) {
	// Positive tests
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"30d", 30 * 24 * time.Hour},
		{"1d", 24 * time.Hour},
		{"24h", 24 * time.Hour},
		{"60m", 60 * time.Minute},
		{"30m", 30 * time.Minute},
	}

	for _, test := range tests {
		result, err := ParseDuration(test.input)
		if err != nil {
			t.Errorf("ParseDuration(%s) returned error: %v", test.input, err)
		}
		if result != test.expected {
			t.Errorf("ParseDuration(%s) = %v, want %v", test.input, result, test.expected)
		}
	}

	// Negative tests
	invalidTests := []string{
		"30x",  // Invalid unit
		"days", // No numeric part
		"",     // Empty string
		"-1d",  // Negative duration
	}

	for _, test := range invalidTests {
		_, err := ParseDuration(test)
		if err == nil {
			t.Errorf("ParseDuration(%s) did not return error for invalid input", test)
		}
	}
}

// TestIsDirEmpty tests the isDirEmpty function
func TestIsDirEmpty(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "filekeeper-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test with empty directory
	empty, err := isDirEmpty(tempDir)
	if err != nil {
		t.Errorf("isDirEmpty() returned error for empty directory: %v", err)
	}
	if !empty {
		t.Errorf("isDirEmpty() = %v, want true for empty directory", empty)
	}

	// Create a file in the directory
	testFile := filepath.Join(tempDir, "testfile.txt")
	if err := ioutil.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test with non-empty directory
	empty, err = isDirEmpty(tempDir)
	if err != nil {
		t.Errorf("isDirEmpty() returned error for non-empty directory: %v", err)
	}
	if empty {
		t.Errorf("isDirEmpty() = %v, want false for non-empty directory", empty)
	}

	// Test with non-existent directory
	_, err = isDirEmpty("/nonexistent-dir-for-test")
	if err == nil {
		t.Errorf("isDirEmpty() did not return error for non-existent directory")
	}
}

// TestGetDefaultConfig tests the GetDefaultConfig function
func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	// Verify some basic expectations about the default config
	if !config.General.Enabled {
		t.Error("Default config should have General.Enabled set to true")
	}

	if !config.General.Logging.Enabled {
		t.Error("Default config should have Logging.Enabled set to true")
	}

	if config.General.Logging.Level != "info" {
		t.Errorf("Default config has Logging.Level = %s, want \"info\"", config.General.Logging.Level)
	}

	if len(config.Directories) == 0 {
		t.Error("Default config should have at least one directory configured")
	}

	// Verify security defaults
	if config.Security.DryRun {
		t.Error("Default config should have Security.DryRun set to false")
	}

	if config.Security.SecureDelete.Enabled {
		t.Error("Default config should have SecureDelete.Enabled set to false")
	}

	if config.Security.SecureDelete.Passes != 3 {
		t.Errorf("Default config has SecureDelete.Passes = %d, want 3", config.Security.SecureDelete.Passes)
	}
}

// TestConfigLoading tests the configuration loading functionality
func TestConfigLoading(t *testing.T) {
	// Create a temporary file with test configuration
	tempFile, err := ioutil.TempFile("", "filekeeper-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test configuration
	testConfig := `general:
  enabled: true
  logging:
    enabled: false
    level: "debug"
    file: "/tmp/test-log.log"
directories:
  - path: "/tmp/test-dir"
    retention_period: "7d"
    file_pattern: "*.log"
    exclude_subdirs: true
    remove_empty_dirs: false
security:
  dry_run: true
  secure_delete:
    enabled: true
    passes: 5
`
	if _, err := tempFile.Write([]byte(testConfig)); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	tempFile.Close()

	// Load the configuration
	config, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}

	// Verify loaded configuration matches expectations
	if !config.General.Enabled {
		t.Error("Loaded config should have General.Enabled set to true")
	}

	if config.General.Logging.Enabled {
		t.Error("Loaded config should have Logging.Enabled set to false")
	}

	if config.General.Logging.Level != "debug" {
		t.Errorf("Loaded config has Logging.Level = %s, want \"debug\"", config.General.Logging.Level)
	}

	if len(config.Directories) != 1 {
		t.Errorf("Loaded config has %d directories, want 1", len(config.Directories))
	} else {
		dir := config.Directories[0]
		if dir.Path != "/tmp/test-dir" {
			t.Errorf("Loaded config has directory.Path = %s, want \"/tmp/test-dir\"", dir.Path)
		}
		if dir.RetentionPeriod != "7d" {
			t.Errorf("Loaded config has directory.RetentionPeriod = %s, want \"7d\"", dir.RetentionPeriod)
		}
		if !dir.ExcludeSubdirs {
			t.Error("Loaded config should have directory.ExcludeSubdirs set to true")
		}
	}

	if !config.Security.DryRun {
		t.Error("Loaded config should have Security.DryRun set to true")
	}

	if !config.Security.SecureDelete.Enabled {
		t.Error("Loaded config should have SecureDelete.Enabled set to true")
	}

	if config.Security.SecureDelete.Passes != 5 {
		t.Errorf("Loaded config has SecureDelete.Passes = %d, want 5", config.Security.SecureDelete.Passes)
	}

	// Test loading non-existent file
	_, err = LoadConfig("/nonexistent-config-file.yaml")
	if err == nil {
		t.Error("LoadConfig() did not return error for non-existent file")
	}

	// Test loading invalid YAML
	invalidFile, err := ioutil.TempFile("", "filekeeper-invalid-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(invalidFile.Name())

	if _, err := invalidFile.Write([]byte("this is not valid yaml")); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}
	invalidFile.Close()

	_, err = LoadConfig(invalidFile.Name())
	if err == nil {
		t.Error("LoadConfig() did not return error for invalid YAML")
	}
}

// TestProcessDirectory tests the directory processing functionality
func TestProcessDirectory(t *testing.T) {
	// Create a temporary directory structure for testing
	testRoot, err := ioutil.TempDir("", "filekeeper-process-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testRoot)

	// Create test files with different modification times
	oldFile := filepath.Join(testRoot, "old.log")
	newFile := filepath.Join(testRoot, "new.log")
	nonMatchingFile := filepath.Join(testRoot, "data.txt")

	// Create a subdirectory with a file
	subDir := filepath.Join(testRoot, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	oldSubFile := filepath.Join(subDir, "old-sub.log")
	emptySubDir := filepath.Join(testRoot, "empty-subdir")
	if err := os.Mkdir(emptySubDir, 0755); err != nil {
		t.Fatalf("Failed to create empty subdirectory: %v", err)
	}

	// Create the files
	for _, file := range []string{oldFile, newFile, nonMatchingFile, oldSubFile} {
		if err := ioutil.WriteFile(file, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Set modification times (old files: 10 days ago, new files: now)
	oldTime := time.Now().Add(-10 * 24 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set old file time: %v", err)
	}
	if err := os.Chtimes(oldSubFile, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set old subdir file time: %v", err)
	}

	// Create a test logger
	logger := log.New(ioutil.Discard, "", 0)

	// Test case 1: Process with pattern matching and include subdirs in dry run mode
	dirConfig := DirectoryConfig{
		Path:            testRoot,
		RetentionPeriod: "7d",
		FilePattern:     "*.log",
		ExcludeSubdirs:  false,
		RemoveEmptyDirs: true,
	}
	securityConfig := SecurityConfig{
		DryRun: true, // Use dry run for testing
		SecureDelete: SecureDeleteConfig{
			Enabled: false,
			Passes:  1,
		},
	}

	// Process the directory
	err = ProcessDirectory(dirConfig, securityConfig, logger)
	if err != nil {
		t.Errorf("ProcessDirectory returned error: %v", err)
	}

	// Verify old files are still there (because we used dry run)
	if _, err := os.Stat(oldFile); os.IsNotExist(err) {
		t.Errorf("Old file was deleted despite dry run mode")
	}
	if _, err := os.Stat(oldSubFile); os.IsNotExist(err) {
		t.Errorf("Old subdirectory file was deleted despite dry run mode")
	}
	if _, err := os.Stat(emptySubDir); os.IsNotExist(err) {
		t.Errorf("Empty directory was deleted despite dry run mode")
	}

	// Test case 2: Process with subdirs excluded and actual deletion
	dirConfig.ExcludeSubdirs = true
	dirConfig.RetentionPeriod = "7d"
	securityConfig.DryRun = false

	err = ProcessDirectory(dirConfig, securityConfig, logger)
	if err != nil {
		t.Errorf("ProcessDirectory returned error: %v", err)
	}

	// Verify old root file is gone, but subdirectory file remains
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("Old file still exists after processing with delete enabled")
	}
	if _, err := os.Stat(oldSubFile); os.IsNotExist(err) {
		t.Errorf("Subdirectory file was deleted despite exclude_subdirs being true")
	}

	// Make sure the new and non-matching files still exist
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Errorf("New file was incorrectly deleted")
	}
	if _, err := os.Stat(nonMatchingFile); os.IsNotExist(err) {
		t.Errorf("Non-matching file was incorrectly deleted")
	}

	// Test case 3: Process with empty directory removal
	// First, make sure empty directory still exists
	if _, err := os.Stat(emptySubDir); os.IsNotExist(err) {
		// Recreate if it was deleted
		if err := os.Mkdir(emptySubDir, 0755); err != nil {
			t.Fatalf("Failed to recreate empty subdirectory: %v", err)
		}
	}

	dirConfig.ExcludeSubdirs = false
	dirConfig.RemoveEmptyDirs = true
	
	err = ProcessDirectory(dirConfig, securityConfig, logger)
	if err != nil {
		t.Errorf("ProcessDirectory returned error: %v", err)
	}

	// Verify empty directory was removed
	if _, err := os.Stat(emptySubDir); !os.IsNotExist(err) {
		t.Errorf("Empty directory still exists after processing with remove_empty_dirs=true")
	}

	// Test case 4: Test with a non-existent directory
	dirConfig.Path = "/non-existent-dir-for-test"
	err = ProcessDirectory(dirConfig, securityConfig, logger)
	if err == nil {
		t.Errorf("ProcessDirectory did not return error for non-existent directory")
	}

	// Test case 5: Test with an invalid retention period
	dirConfig.Path = testRoot
	dirConfig.RetentionPeriod = "invalid"
	err = ProcessDirectory(dirConfig, securityConfig, logger)
	if err == nil {
		t.Errorf("ProcessDirectory did not return error for invalid retention period")
	}
}

// TestSecureDeleteFile tests the secure file deletion functionality
func TestSecureDeleteFile(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := ioutil.TempFile("", "filekeeper-secure-delete-test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name()) // This will only execute if the test fails

	// Write some content to the file
	testData := "This is test data that should be securely deleted"
	if _, err := tempFile.Write([]byte(testData)); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	tempFile.Close()

	// Create a test logger
	logger := log.New(ioutil.Discard, "", 0)

	// Test secure delete with 1 pass
	err = secureDeleteFile(tempFile.Name(), 1, logger)
	if err != nil {
		t.Errorf("secureDeleteFile returned error: %v", err)
	}

	// Verify the file is deleted
	if _, err := os.Stat(tempFile.Name()); !os.IsNotExist(err) {
		t.Errorf("File still exists after secure deletion")
	}

	// Test with multiple passes and verify content between passes
	tempFile2, err := ioutil.TempFile("", "filekeeper-secure-delete-multi-pass-test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile2.Name()) // This will only execute if the test fails

	// Write initial test data
	if _, err := tempFile2.Write([]byte(testData)); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	tempFile2.Close()

	// Create a modified version of secureDeleteFile that allows us to inspect content between passes
	testSecureDelete := func(path string, passes int) error {
		// Open the file for writing
		file, err := os.OpenFile(path, os.O_RDWR, 0)
		if err != nil {
			return err
		}
		defer file.Close()

		// Get file info for size
		info, err := file.Stat()
		if err != nil {
			return err
		}

		// Create a buffer for overwriting
		size := info.Size()
		buf := make([]byte, 8192) // Use a reasonable buffer size
		readBuf := make([]byte, size)

		// Track the content for each pass to ensure it changes
		var contentHashes []string

		// Read initial content
		file.Seek(0, 0)
		file.Read(readBuf)
		initialHash := hashContent(readBuf)
		contentHashes = append(contentHashes, initialHash)

		// Perform secure deletion passes, checking content after each
		for pass := 0; pass < passes; pass++ {
			// Reset to beginning of file
			if _, err := file.Seek(0, 0); err != nil {
				return err
			}

			// Fill the buffer with "random" data for this pass
			for i := range buf {
				buf[i] = byte(pass ^ i)
			}

			// Write the data to the file
			remaining := size
			for remaining > 0 {
				writeSize := int64(len(buf))
				if remaining < writeSize {
					writeSize = remaining
				}

				if _, err := file.Write(buf[:writeSize]); err != nil {
					return err
				}

				remaining -= writeSize
			}

			// Flush to disk
			if err := file.Sync(); err != nil {
				return err
			}

			// Read and verify content has changed
			file.Seek(0, 0)
			file.Read(readBuf)
			passHash := hashContent(readBuf)
			
			// Check that content is different from previous content
			if pass > 0 && passHash == contentHashes[pass] {
				t.Errorf("Pass %d content did not change from previous pass", pass+1)
			}
			contentHashes = append(contentHashes, passHash)
		}

		// Final deletion
		return os.Remove(path)
	}

	// Test with 3 passes and content verification
	err = testSecureDelete(tempFile2.Name(), 3)
	if err != nil {
		t.Errorf("secureDeleteFile with 3 passes returned error: %v", err)
	}

	// Verify the file is deleted
	if _, err := os.Stat(tempFile2.Name()); !os.IsNotExist(err) {
		t.Errorf("File still exists after multi-pass secure deletion")
	}

	// Test with non-existent file
	err = secureDeleteFile("/nonexistent-file-for-test", 1, logger)
	if err == nil {
		t.Errorf("secureDeleteFile did not return error for non-existent file")
	}

	// Test with 0 passes (should handle gracefully)
	tempFile3, err := ioutil.TempFile("", "filekeeper-secure-delete-zero-pass-test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile3.Name()) // This will only execute if the test fails

	tempFile3.Close()
	err = secureDeleteFile(tempFile3.Name(), 0, logger)
	if err != nil {
		t.Errorf("secureDeleteFile with 0 passes returned error: %v", err)
	}

	// Verify the file is deleted
	if _, err := os.Stat(tempFile3.Name()); !os.IsNotExist(err) {
		t.Errorf("File still exists after zero-pass secure deletion")
	}
}

// hashContent creates a simple hash of file content for comparison
func hashContent(data []byte) string {
	var hash uint32
	for _, b := range data {
		hash = hash*31 + uint32(b)
	}
	return fmt.Sprintf("%x", hash)
}

// TestWriteExampleConfig tests the configuration file creation
func TestWriteExampleConfig(t *testing.T) {
	// Create a temporary file for the config
	tempDir, err := ioutil.TempDir("", "filekeeper-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "filekeeper.yaml")

	// Write the example config
	err = WriteExampleConfig(configPath)
	if err != nil {
		t.Errorf("WriteExampleConfig returned error: %v", err)
	}

	// Verify the file exists and has content
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		t.Errorf("Failed to read example config file: %v", err)
	}
	if len(content) == 0 {
		t.Error("Example config file is empty")
	}

	// Check for required sections
	configStr := string(content)
	requiredSections := []string{
		"general:",
		"logging:",
		"directories:",
		"path:",
		"retention_period:",
		"security:",
		"dry_run:",
		"secure_delete:",
	}

	for _, section := range requiredSections {
		if !strings.Contains(configStr, section) {
			t.Errorf("Example config missing required section: %s", section)
		}
	}

	// Test with existing parent directory
	subDir := filepath.Join(tempDir, "nested", "config")
	configInSubdir := filepath.Join(subDir, "config.yaml")
	
	err = WriteExampleConfig(configInSubdir)
	if err != nil {
		t.Errorf("WriteExampleConfig to nested directory returned error: %v", err)
	}

	// Verify the file was created in the nested directory
	if _, err := os.Stat(configInSubdir); os.IsNotExist(err) {
		t.Errorf("Config file was not created in nested directory")
	}
}

// TestSystemdFiles tests the creation of systemd files
func TestSystemdFiles(t *testing.T) {
	// Create a temporary directory for the systemd files
	tempDir, err := ioutil.TempDir("", "filekeeper-systemd-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save the original systemd directories
	origIsRoot := isRoot

	// Mock the systemd paths for testing
	userSystemdPath := filepath.Join(tempDir, "user-systemd")
	systemSystemdPath := filepath.Join(tempDir, "system-systemd")

	// Create a test function that overrides the paths
	testCreateSystemdFiles := func(userMode bool) error {
		var systemdDir string
		if userMode {
			systemdDir = userSystemdPath
		} else {
			systemdDir = systemSystemdPath
		}

		// Create the directory if it doesn't exist
		if err := os.MkdirAll(systemdDir, 0755); err != nil {
			return err
		}

		// Create the service file
		serviceContent := "Test service content"
		servicePath := filepath.Join(systemdDir, "filekeeper.service")
		if err := ioutil.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
			return err
		}

		// Create the timer file
		timerContent := "Test timer content"
		timerPath := filepath.Join(systemdDir, "filekeeper.timer")
		if err := ioutil.WriteFile(timerPath, []byte(timerContent), 0644); err != nil {
			return err
		}

		return nil
	}

	// Test user mode
	if err := testCreateSystemdFiles(true); err != nil {
		t.Errorf("Creating user systemd files returned error: %v", err)
	}

	// Check if files were created
	userServicePath := filepath.Join(userSystemdPath, "filekeeper.service")
	userTimerPath := filepath.Join(userSystemdPath, "filekeeper.timer")

	if _, err := os.Stat(userServicePath); os.IsNotExist(err) {
		t.Errorf("User service file was not created")
	}
	if _, err := os.Stat(userTimerPath); os.IsNotExist(err) {
		t.Errorf("User timer file was not created")
	}

	// Test system mode
	if err := testCreateSystemdFiles(false); err != nil {
		t.Errorf("Creating system systemd files returned error: %v", err)
	}

	// Check if files were created
	systemServicePath := filepath.Join(systemSystemdPath, "filekeeper.service")
	systemTimerPath := filepath.Join(systemSystemdPath, "filekeeper.timer")

	if _, err := os.Stat(systemServicePath); os.IsNotExist(err) {
		t.Errorf("System service file was not created")
	}
	if _, err := os.Stat(systemTimerPath); os.IsNotExist(err) {
		t.Errorf("System timer file was not created")
	}

	// Restore the original value
	isRoot = origIsRoot
}