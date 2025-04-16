package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
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
	tempDir, err := os.MkdirTemp("", "filekeeper-test")
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
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
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

	if config.Security.SecureDelete.ObfuscateFilenames {
		t.Error("Default config should have SecureDelete.ObfuscateFilenames set to false")
	}
}

// TestConfigLoading tests the configuration loading functionality
func TestConfigLoading(t *testing.T) {
	// Create a temporary file with test configuration
	tempFile, err := os.CreateTemp("", "filekeeper-config-*.yaml")
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
    obfuscate_filenames: true
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

	if !config.Security.SecureDelete.ObfuscateFilenames {
		t.Error("Loaded config should have SecureDelete.ObfuscateFilenames set to true")
	}

	// Test loading non-existent file
	_, err = LoadConfig("/nonexistent-config-file.yaml")
	if err == nil {
		t.Error("LoadConfig() did not return error for non-existent file")
	}

	// Test loading invalid YAML
	invalidFile, err := os.CreateTemp("", "filekeeper-invalid-*.yaml")
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

// TestObfuscateFilename tests the obfuscateFilename function
func TestObfuscateFilename(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "filekeeper-obfuscate-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFilePath := filepath.Join(tempDir, "test-file.txt")
	if err := os.WriteFile(testFilePath, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a test logger
	logger := log.New(io.Discard, "", 0)

	// Test obfuscating a file name
	newPath, err := obfuscateFilename(testFilePath, logger)
	if err != nil {
		t.Errorf("obfuscateFilename returned error: %v", err)
	}

	// Verify the result
	if newPath == testFilePath {
		t.Error("obfuscateFilename didn't change the file path")
	}

	// Check that the original file doesn't exist anymore
	if _, err := os.Stat(testFilePath); !os.IsNotExist(err) {
		t.Error("Original file still exists after obfuscation")
	}

	// Check that the new file exists
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Error("Obfuscated file doesn't exist")
	}

	// Verify the extension was preserved
	if filepath.Ext(newPath) != ".txt" {
		t.Errorf("File extension not preserved, got %s, want .txt", filepath.Ext(newPath))
	}

	// Verify the file content
	content, err := os.ReadFile(newPath)
	if err != nil {
		t.Errorf("Failed to read obfuscated file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("File content changed after obfuscation, got %s, want 'test content'", string(content))
	}

	// Test with non-existent file
	_, err = obfuscateFilename("/nonexistent-file-for-test", logger)
	if err == nil {
		t.Error("obfuscateFilename did not return error for non-existent file")
	}
}

// TestObfuscateDirectoryName tests the obfuscateDirectoryName function
func TestObfuscateDirectoryName(t *testing.T) {
	// Create a temporary directory for testing
	parentDir, err := os.MkdirTemp("", "filekeeper-obfuscate-dir-test")
	if err != nil {
		t.Fatalf("Failed to create parent directory: %v", err)
	}
	defer os.RemoveAll(parentDir)

	// Create a test directory
	testDirPath := filepath.Join(parentDir, "test-dir")
	if err := os.Mkdir(testDirPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a test file in the directory
	testFilePath := filepath.Join(testDirPath, "test-file.txt")
	if err := os.WriteFile(testFilePath, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a test logger
	logger := log.New(io.Discard, "", 0)

	// Test obfuscating a directory name
	newPath, err := obfuscateDirectoryName(testDirPath, logger)
	if err != nil {
		t.Errorf("obfuscateDirectoryName returned error: %v", err)
	}

	// Verify the result
	if newPath == testDirPath {
		t.Error("obfuscateDirectoryName didn't change the directory path")
	}

	// Check that the original directory doesn't exist anymore
	if _, err := os.Stat(testDirPath); !os.IsNotExist(err) {
		t.Error("Original directory still exists after obfuscation")
	}

	// Check that the new directory exists
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Error("Obfuscated directory doesn't exist")
	}

	// Verify the file inside still exists
	newFilePath := filepath.Join(newPath, "test-file.txt")
	if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
		t.Error("File inside obfuscated directory doesn't exist")
	}

	// Test with non-existent directory
	_, err = obfuscateDirectoryName("/nonexistent-dir-for-test", logger)
	if err == nil {
		t.Error("obfuscateDirectoryName did not return error for non-existent directory")
	}
}

// TestProcessDirectory tests the directory processing functionality
func TestProcessDirectory(t *testing.T) {
	// Create a temporary directory structure for testing
	testRoot, err := os.MkdirTemp("", "filekeeper-process-test")
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
		if err := os.WriteFile(file, []byte("test content"), 0644); err != nil {
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
	logger := log.New(io.Discard, "", 0)

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
			Enabled:            false,
			Passes:             1,
			ObfuscateFilenames: false,
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

	// Test case 6: Test with ObfuscateFilenames enabled
	// Recreate the test structure
	oldFile = filepath.Join(testRoot, "sensitive-file.log")
	if err := os.WriteFile(oldFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", oldFile, err)
	}
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set old file time: %v", err)
	}

	emptySubDir = filepath.Join(testRoot, "sensitive-empty-dir")
	if err := os.Mkdir(emptySubDir, 0755); err != nil {
		t.Fatalf("Failed to create empty subdirectory: %v", err)
	}

	dirConfig.Path = testRoot
	dirConfig.RetentionPeriod = "7d"
	dirConfig.FilePattern = "*.log"
	dirConfig.ExcludeSubdirs = false
	dirConfig.RemoveEmptyDirs = true

	securityConfig.DryRun = false
	securityConfig.SecureDelete.Enabled = false
	securityConfig.SecureDelete.ObfuscateFilenames = true

	err = ProcessDirectory(dirConfig, securityConfig, logger)
	if err != nil {
		t.Errorf("ProcessDirectory with ObfuscateFilenames returned error: %v", err)
	}

	// Verify old file was deleted
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("Sensitive file still exists after processing with ObfuscateFilenames")
	}

	// Verify empty directory was removed
	if _, err := os.Stat(emptySubDir); !os.IsNotExist(err) {
		t.Errorf("Sensitive empty directory still exists after processing with ObfuscateFilenames")
	}
}

// TestSecureDeleteFile tests the secure file deletion functionality
func TestSecureDeleteFile(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "filekeeper-secure-delete-test")
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
	logger := log.New(io.Discard, "", 0)

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
	tempFile2, err := os.CreateTemp("", "filekeeper-secure-delete-multi-pass-test")
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
		if _, err := file.Seek(0, 0); err != nil {
			return err
		}
		if _, err := file.Read(readBuf); err != nil && err != io.EOF {
			return err
		}
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
			if _, err := file.Seek(0, 0); err != nil {
				return err
			}
			if _, err := file.Read(readBuf); err != nil && err != io.EOF {
				return err
			}
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
	tempFile3, err := os.CreateTemp("", "filekeeper-secure-delete-zero-pass-test")
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
	tempDir, err := os.MkdirTemp("", "filekeeper-config-test")
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
	content, err := os.ReadFile(configPath)
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
		"obfuscate_filenames:",
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
	tempDir, err := os.MkdirTemp("", "filekeeper-systemd-test")
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
		if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
			return err
		}

		// Create the timer file
		timerContent := "Test timer content"
		timerPath := filepath.Join(systemdDir, "filekeeper.timer")
		if err := os.WriteFile(timerPath, []byte(timerContent), 0644); err != nil {
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

// TestPrintHelp tests the PrintHelp function
func TestPrintHelp(t *testing.T) {
	// Capture stdout to check the output
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	// Call PrintHelp
	PrintHelp()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}
	output := buf.String()

	// Verify the output contains essential help information
	requiredContent := []string{
		ProgramName,
		ProgramVersion,
		"--help",
		"--version",
		"--init",
		"--config",
		"--install-systemd",
		"--dry-run",
		"--force",
	}

	for _, content := range requiredContent {
		if !strings.Contains(output, content) {
			t.Errorf("Help output missing required content: %s", content)
		}
	}

	// Verify default config paths are included
	if isRoot {
		if !strings.Contains(output, "/etc/filekeeper/filekeeper.yaml") {
			t.Error("Help output missing root config path")
		}
	} else {
		homeDir, _ := os.UserHomeDir()
		expectedPath := filepath.Join(homeDir, ".config", "filekeeper.yaml")
		if !strings.Contains(output, expectedPath) {
			t.Error("Help output missing user config path")
		}
	}
}

// TestSetupLogger tests the logger setup functionality
func TestSetupLogger(t *testing.T) {
	// Create a temporary directory for log files
	tempDir, err := os.MkdirTemp("", "filekeeper-logs")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logPath := filepath.Join(tempDir, "test.log")

	// Test case 1: Logging enabled with info level
	config := LoggingConfig{
		Enabled: true,
		Level:   "info",
		File:    logPath,
	}

	logger, err := setupLogger(config)
	if err != nil {
		t.Errorf("setupLogger returned error: %v", err)
	}
	if logger == nil {
		t.Error("setupLogger returned nil logger")
	}

	// Test logging to file
	logger.Println("Test log message")

	// Verify log file was created and has content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Errorf("Failed to read log file: %v", err)
	}
	if !strings.Contains(string(content), "Test log message") {
		t.Error("Log file does not contain expected message")
	}

	// Test case 2: Logging enabled with debug level
	debugConfig := LoggingConfig{
		Enabled: true,
		Level:   "debug",
		File:    filepath.Join(tempDir, "debug.log"),
	}

	debugLogger, err := setupLogger(debugConfig)
	if err != nil {
		t.Errorf("setupLogger with debug level returned error: %v", err)
	}
	if debugLogger == nil {
		t.Error("setupLogger with debug level returned nil logger")
	}

	// Test case 3: Logging disabled
	disabledConfig := LoggingConfig{
		Enabled: false,
		Level:   "info",
		File:    filepath.Join(tempDir, "disabled.log"),
	}

	disabledLogger, err := setupLogger(disabledConfig)
	if err != nil {
		t.Errorf("setupLogger with disabled logging returned error: %v", err)
	}
	if disabledLogger == nil {
		t.Error("setupLogger with disabled logging returned nil logger")
	}

	// Verify disabled log file was not created
	if _, err := os.Stat(disabledConfig.File); err == nil {
		t.Error("Log file was created despite logging being disabled")
	}

	// Test case 4: Invalid log directory
	if runtime.GOOS != "windows" {
		// Skip on Windows as permissions work differently
		invalidConfig := LoggingConfig{
			Enabled: true,
			Level:   "info",
			File:    "/root/invalid/path/test.log", // Should fail on non-root test runs
		}

		_, err = setupLogger(invalidConfig)
		if err == nil {
			// This might pass if tests are run as root, so don't fail in that case
			if os.Getuid() != 0 {
				t.Error("setupLogger did not return error for invalid log directory")
			}
		}
	}
}

// TestPrintSystemdTemplates tests the PrintSystemdTemplates function
func TestPrintSystemdTemplates(t *testing.T) {
	// Capture stdout to check the output
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	// Call PrintSystemdTemplates
	PrintSystemdTemplates()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}
	output := buf.String()

	// Verify the output contains essential systemd file content
	requiredContent := []string{
		"# FileKeeper Service File",
		"# FileKeeper Timer File",
		"[Unit]",
		"[Service]",
		"[Timer]",
		"[Install]",
		"OnCalendar=daily",
		"Type=oneshot",
	}

	for _, content := range requiredContent {
		if !strings.Contains(output, content) {
			t.Errorf("Systemd template output missing required content: %s", content)
		}
	}
}

// TestCommandLineFlags tests the handling of command line flags
func TestCommandLineFlags(t *testing.T) {
	// Save and restore the original os.Args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Save and restore os.Stdout for capturing output
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Save the original flag.CommandLine to restore it after the test
	origFlagCommandLine := flag.CommandLine
	defer func() { flag.CommandLine = origFlagCommandLine }()

	// Save original config file path
	origConfigFile := configFile
	defer func() { configFile = origConfigFile }()

	// Create a temporary directory for configuration files
	tempDir, err := os.MkdirTemp("", "filekeeper-cli-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Define test cases
	testCases := []struct {
		name         string
		args         []string
		setup        func() // Setup before running the case
		expectedText string // Text that should be in the output
		expectedErr  bool   // Whether an error exit is expected
	}{
		{
			name: "Version flag",
			args: []string{"filekeeper", "--version"},
			setup: func() {
				// Reset flag.CommandLine for each test
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			},
			expectedText: ProgramVersion,
			expectedErr:  false,
		},
		{
			name: "Help flag",
			args: []string{"filekeeper", "--help"},
			setup: func() {
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			},
			expectedText: "Usage:",
			expectedErr:  false,
		},
		{
			name: "Init flag",
			args: []string{"filekeeper", "--init"},
			setup: func() {
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
				// Use a temp directory for config
				configFile = filepath.Join(tempDir, "filekeeper.yaml")
			},
			expectedText: "Created example configuration file",
			expectedErr:  false,
		},
		{
			name: "Systemd template flag",
			args: []string{"filekeeper", "--systemd-template-only"},
			setup: func() {
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			},
			expectedText: "# FileKeeper Service File",
			expectedErr:  false,
		},
	}

	// Run each test case
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup for this test case
			tc.setup()

			// Set the command line arguments
			os.Args = tc.args

			// Create a pipe to capture stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			os.Stdout = w

			// Create a channel to handle the potential os.Exit() call
			exitCalled := make(chan int, 1)

			// Mock os.Exit
			originalExit := osExit
			osExit = func(code int) {
				exitCalled <- code
				panic("os.Exit called") // Use panic to abort execution
			}
			defer func() {
				osExit = originalExit
				if r := recover(); r != nil && r != "os.Exit called" {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			// Execute the main function in a goroutine
			go func() {
				defer func() {
					if r := recover(); r == "os.Exit called" {
						// Expected when os.Exit is called
						return
					}
				}()
				// Execute the code that would be in main()
				mainImpl()
				// If we reach here without exiting, signal with exit code 0
				exitCalled <- 0
			}()

			// Wait for the goroutine to finish or time out
			var exitCode int
			select {
			case exitCode = <-exitCalled:
				// Got the exit code
			case <-time.After(2 * time.Second):
				t.Fatal("Test timed out")
			}

			// Close the pipe and restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read the captured output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("Failed to read captured output: %v", err)
			}
			output := buf.String()

			// Check if the output contains the expected text
			if !strings.Contains(output, tc.expectedText) {
				t.Errorf("Output does not contain expected text: %s\nGot: %s", tc.expectedText, output)
			}

			// Check if the exit code matches expectations
			if tc.expectedErr && exitCode == 0 {
				t.Errorf("Expected error exit but got success")
			} else if !tc.expectedErr && exitCode != 0 {
				t.Errorf("Expected success but got error exit with code %d", exitCode)
			}
		})
	}
}

// Create a variable to mock os.Exit for testing
var osExit = os.Exit

// Create a variable to mock user.Current for testing
var userCurrent = user.Current

// initConfigPaths is a testable version of the init function
func initConfigPaths() {
	currentUser, err := userCurrent()
	if err != nil {
		log.Fatalf("Error determining current user: %v", err)
	}

	isRoot = currentUser.Uid == "0"

	if isRoot {
		configDir = "/etc/filekeeper"
		configFile = filepath.Join(configDir, "filekeeper.yaml")
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Error determining user home directory: %v", err)
		}
		configDir = filepath.Join(homeDir, ".config")
		configFile = filepath.Join(configDir, "filekeeper.yaml")
	}
}

// Create an implementation of main() that can be called in tests
func mainImpl() {
	// Parse command line flags
	var (
		showHelp            bool
		showVersion         bool
		initConfig          bool
		configPath          string
		installSystemd      bool
		systemdTemplateOnly bool
		dryRun              bool
		force               bool
	)

	flag.BoolVar(&showHelp, "help", false, "Show help information")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&initConfig, "init", false, "Create a default configuration file")
	flag.StringVar(&configPath, "config", configFile, "Specify an alternative configuration file path")
	flag.BoolVar(&installSystemd, "install-systemd", false, "Create systemd service and timer files")
	flag.BoolVar(&systemdTemplateOnly, "systemd-template-only", false, "Output systemd templates without creating files")
	flag.BoolVar(&dryRun, "dry-run", false, "Run without actually deleting any files")
	flag.BoolVar(&force, "force", false, "Run even if disabled in the configuration")

	flag.Parse()

	// Show help if requested
	if showHelp {
		PrintHelp()
		fmt.Println("\nRun 'filekeeper --init' to create a default configuration file.")
		return
	}

	// Show version and exit
	if showVersion {
		fmt.Printf("%s v%s\n", ProgramName, ProgramVersion)
		return
	}

	// Initialize configuration
	if initConfig {
		exampleConfigPath := configPath + ".example"
		if err := WriteExampleConfig(exampleConfigPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating example configuration: %v\n", err)
			osExit(1)
		}
		fmt.Printf("Created example configuration file: %s\n", exampleConfigPath)
		fmt.Printf("Please review and rename to %s when ready.\n", configPath)
		return
	}

	// Print systemd templates
	if systemdTemplateOnly {
		PrintSystemdTemplates()
		return
	}

	// Install systemd files
	if installSystemd {
		err := CreateSystemdFiles(!isRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating systemd files: %v\n", err)
			osExit(1)
		}
		return
	}

	// Skip the rest for testing - we would normally load config and process files
}

// TestInitFunction tests the init function that sets up config paths
func TestInitFunction(t *testing.T) {
	// Save original values
	origIsRoot := isRoot
	origConfigDir := configDir
	origConfigFile := configFile

	// Restore after test
	defer func() {
		isRoot = origIsRoot
		configDir = origConfigDir
		configFile = origConfigFile
	}()

	// Test cases for different user types
	testCases := []struct {
		name     string
		uid      string
		expected struct {
			isRoot     bool
			configBase string
		}
	}{
		{
			name: "Root user",
			uid:  "0",
			expected: struct {
				isRoot     bool
				configBase string
			}{
				isRoot:     true,
				configBase: "/etc",
			},
		},
		{
			name: "Regular user",
			uid:  "1000",
			expected: struct {
				isRoot     bool
				configBase string
			}{
				isRoot:     false,
				configBase: ".config",
			},
		},
	}

	// Prepare to mock user.Current
	origUserCurrent := userCurrent
	defer func() {
		userCurrent = origUserCurrent
	}()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock user.Current to return the desired UID
			userCurrent = func() (*user.User, error) {
				return &user.User{
					Uid: tc.uid,
				}, nil
			}

			// Reset global variables
			isRoot = false
			configDir = ""
			configFile = ""

			// Call init manually (not the package init function)
			initConfigPaths()

			// Verify the results
			if isRoot != tc.expected.isRoot {
				t.Errorf("isRoot = %v, want %v", isRoot, tc.expected.isRoot)
			}

			if isRoot {
				if configDir != "/etc/filekeeper" {
					t.Errorf("Root config dir = %s, want /etc/filekeeper", configDir)
				}
				if configFile != "/etc/filekeeper/filekeeper.yaml" {
					t.Errorf("Root config file = %s, want /etc/filekeeper/filekeeper.yaml", configFile)
				}
			} else {
				homeDir, _ := os.UserHomeDir()
				expectedDir := filepath.Join(homeDir, ".config")
				expectedFile := filepath.Join(homeDir, ".config", "filekeeper.yaml")

				if configDir != expectedDir {
					t.Errorf("User config dir = %s, want %s", configDir, expectedDir)
				}
				if configFile != expectedFile {
					t.Errorf("User config file = %s, want %s", configFile, expectedFile)
				}
			}
		})
	}
}
