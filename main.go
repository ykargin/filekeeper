// FileKeeper - a program to remove files older than a specified retention period based on configuration
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Version information
const (
	ProgramName    = "FileKeeper"
	ProgramVersion = "0.1.4"
)

// Config represents the main configuration structure
type Config struct {
	General     GeneralConfig     `yaml:"general"`
	Directories []DirectoryConfig `yaml:"directories"`
	Security    SecurityConfig    `yaml:"security"`
}

// GeneralConfig contains general program settings
type GeneralConfig struct {
	Enabled bool          `yaml:"enabled"`
	Logging LoggingConfig `yaml:"logging"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Enabled bool   `yaml:"enabled"`
	Level   string `yaml:"level"`
	File    string `yaml:"file"`
}

// DirectoryConfig contains settings for a directory to process
type DirectoryConfig struct {
	Path            string `yaml:"path"`
	RetentionPeriod string `yaml:"retention_period"`
	FilePattern     string `yaml:"file_pattern"`
	ExcludeSubdirs  bool   `yaml:"exclude_subdirs"`
	RemoveEmptyDirs bool   `yaml:"remove_empty_dirs"`
}

// SecurityConfig contains security settings
type SecurityConfig struct {
	DryRun       bool               `yaml:"dry_run"`
	SecureDelete SecureDeleteConfig `yaml:"secure_delete"`
}

// SecureDeleteConfig contains secure deletion settings
type SecureDeleteConfig struct {
	Enabled bool `yaml:"enabled"`
	Passes  int  `yaml:"passes"`
}

// Global variables
var (
	isRoot     bool
	configDir  string
	configFile string
)

// Init determines if the program is running as root and sets the appropriate config paths
func init() {
	currentUser, err := user.Current()
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

// ParseDuration parses a duration string like "30d", "24h", "60m"
func ParseDuration(durationStr string) (time.Duration, error) {
	// Handle days specially since Go doesn't have a built-in "d" unit
	if strings.HasSuffix(durationStr, "d") {
		value := strings.TrimSuffix(durationStr, "d")
		var days int
		_, err := fmt.Sscanf(value, "%d", &days)
		if err == nil {
			return time.Hour * 24 * time.Duration(days), nil
		}
		return 0, fmt.Errorf("invalid day format: %s", durationStr)
	}

	// For other units, use the standard time.ParseDuration
	return time.ParseDuration(durationStr)
}

// GetDefaultConfig returns a default configuration
func GetDefaultConfig() Config {
	logFile := "/var/log/filekeeper.log"
	if !isRoot {
		homeDir, _ := os.UserHomeDir()
		logFile = filepath.Join(homeDir, ".local", "share", "filekeeper", "filekeeper.log")
	}

	return Config{
		General: GeneralConfig{
			Enabled: true,
			Logging: LoggingConfig{
				Enabled: true,
				Level:   "info",
				File:    logFile,
			},
		},
		Directories: []DirectoryConfig{
			{
				Path:            "/path/to/dir1",
				RetentionPeriod: "30d",
				FilePattern:     "*.log",
				ExcludeSubdirs:  false,
				RemoveEmptyDirs: true,
			},
		},
		Security: SecurityConfig{
			DryRun: false,
			SecureDelete: SecureDeleteConfig{
				Enabled: false,
				Passes:  3,
			},
		},
	}
}

// WriteExampleConfig writes a default configuration file
func WriteExampleConfig(configPath string) error {
	// Create the directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	config := GetDefaultConfig()
	// Удаляем неиспользуемую переменную data
	// data, err := yaml.Marshal(&config)
	// if err != nil {
	//     return err
	// }

	// Add comments to the yaml file
	configWithComments := `# General settings
general:
  # Enable/disable program operation
  enabled: true
  # Logging settings
  logging:
    # Enable/disable logging
    enabled: true
    # Logging level (debug, info, warn, error)
    level: "info"
    # Path to log file
    file: "` + config.General.Logging.File + `"

# List of directories to process
directories:
  - path: "/path/to/dir1"
    # File retention period (format: 30d, 24h, 60m)
    retention_period: "30d"
    # File matching pattern (optional)
    file_pattern: "*.log"
    # Exclude subdirectories?
    exclude_subdirs: false
    # Remove empty directories?
    remove_empty_dirs: true

# Security settings
security:
  # Dry run mode: only output files that would be deleted without actual deletion
  dry_run: false
  # Secure deletion settings (for HDD)
  secure_delete:
    # Enable/disable secure deletion
    enabled: false
    # Number of passes for data overwrite
    passes: 3
`

	return ioutil.WriteFile(configPath, []byte(configWithComments), 0644)
}

// CreateSystemdFiles creates the systemd service and timer files
func CreateSystemdFiles(userMode bool) error {
	var systemdDir string

	// Determine the systemd directory based on whether we're in user mode
	if userMode {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		systemdDir = filepath.Join(homeDir, ".config", "systemd", "user")
	} else {
		systemdDir = "/etc/systemd/system"
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		return err
	}

	// Create the service file
	serviceContent := `[Unit]
Description=FileKeeper - Scheduled file cleanup based on retention policy
Documentation=https://github.com/ykargin/filekeeper

[Service]
Type=oneshot
ExecStart=/usr/local/bin/filekeeper

# Security settings - adjust as needed
ProtectSystem=strict
ProtectHome=read-only
PrivateTmp=true
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
`

	// If in user mode, remove system-specific security settings
	if userMode {
		serviceContent = `[Unit]
Description=FileKeeper - Scheduled file cleanup based on retention policy
Documentation=https://github.com/ykargin/filekeeper

[Service]
Type=oneshot
ExecStart=filekeeper

[Install]
WantedBy=default.target
`
	}

	servicePath := filepath.Join(systemdDir, "filekeeper.service")
	if err := ioutil.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return err
	}

	// Create the timer file
	timerContent := `[Unit]
Description=Run FileKeeper daily to clean up old files
Documentation=https://github.com/ykargin/filekeeper

[Timer]
OnCalendar=daily
Persistent=true
RandomizedDelaySec=1hour

[Install]
WantedBy=timers.target
`

	timerPath := filepath.Join(systemdDir, "filekeeper.timer")
	if err := ioutil.WriteFile(timerPath, []byte(timerContent), 0644); err != nil {
		return err
	}

	// Output activation commands
	fmt.Println("Systemd files created successfully:")
	fmt.Println("  - Service: " + servicePath)
	fmt.Println("  - Timer: " + timerPath)

	if userMode {
		fmt.Println("\nTo activate, run:")
		fmt.Println("  systemctl --user daemon-reload")
		fmt.Println("  systemctl --user enable --now filekeeper.timer")
	} else {
		fmt.Println("\nTo activate, run:")
		fmt.Println("  systemctl daemon-reload")
		fmt.Println("  systemctl enable --now filekeeper.timer")
	}

	return nil
}

// PrintSystemdTemplates prints the systemd templates to stdout
func PrintSystemdTemplates() {
	fmt.Println("# FileKeeper Service File (filekeeper.service)")
	fmt.Println("# Save to /etc/systemd/system/ (for system-wide) or ~/.config/systemd/user/ (for user)")
	fmt.Println(`
[Unit]
Description=FileKeeper - Schedule file cleanup based on retention policy
Documentation=https://github.com/ykargin/filekeeper

[Service]
Type=oneshot
ExecStart=/usr/local/bin/filekeeper

# Security settings - adjust as needed
ProtectSystem=strict
ProtectHome=read-only
PrivateTmp=true
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
`)

	fmt.Println("\n# FileKeeper Timer File (filekeeper.timer)")
	fmt.Println("# Save to /etc/systemd/system/ (for system-wide) or ~/.config/systemd/user/ (for user)")
	fmt.Println(`
[Unit]
Description=Run FileKeeper daily to clean up old files
Documentation=https://github.com/ykargin/filekeeper

[Timer]
OnCalendar=daily
Persistent=true
RandomizedDelaySec=1hour

[Install]
WantedBy=timers.target
`)
}

// LoadConfig loads the configuration from a file
func LoadConfig(configPath string) (Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	config := Config{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, err
	}

	return config, nil
}

// ProcessDirectory processes a directory according to its configuration
func ProcessDirectory(dirConfig DirectoryConfig, securityConfig SecurityConfig, logger *log.Logger) error {
	logger.Printf("Processing directory: %s", dirConfig.Path)

	// Parse retention period
	retention, err := ParseDuration(dirConfig.RetentionPeriod)
	if err != nil {
		return fmt.Errorf("invalid retention period '%s': %v", dirConfig.RetentionPeriod, err)
	}

	// Calculate cutoff time
	cutoff := time.Now().Add(-retention)
	logger.Printf("Retention period: %s (removing files before %s)", dirConfig.RetentionPeriod, cutoff.Format(time.RFC3339))

	// Prepare to walk directory
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Printf("Error accessing path %s: %v", path, err)
			return nil // Continue walking
		}

		// Skip the root directory itself
		if path == dirConfig.Path {
			return nil
		}

		// Skip directories if we're not removing empty ones or if we're excluding subdirectories
		if info.IsDir() {
			if dirConfig.ExcludeSubdirs && path != dirConfig.Path {
				return filepath.SkipDir
			}
			// We'll handle directories in a second pass
			return nil
		}

		// Check if the file matches the pattern
		if dirConfig.FilePattern != "" {
			match, err := filepath.Match(dirConfig.FilePattern, filepath.Base(path))
			if err != nil {
				logger.Printf("Error matching pattern '%s' for file %s: %v", dirConfig.FilePattern, path, err)
				return nil
			}
			if !match {
				return nil // Skip files that don't match the pattern
			}
		}

		// Check if the file is older than the cutoff
		if info.ModTime().Before(cutoff) {
			if securityConfig.DryRun {
				logger.Printf("Would delete file: %s (modified: %s)", path, info.ModTime().Format(time.RFC3339))
				fmt.Printf("Would delete file: %s (modified: %s)\n", path, info.ModTime().Format(time.RFC3339))
			} else {
				if securityConfig.SecureDelete.Enabled {
					if err := secureDeleteFile(path, securityConfig.SecureDelete.Passes, logger); err != nil {
						logger.Printf("Error securely deleting file %s: %v", path, err)
					} else {
						logger.Printf("Securely deleted file: %s", path)
					}
				} else {
					if err := os.Remove(path); err != nil {
						logger.Printf("Error deleting file %s: %v", path, err)
					} else {
						logger.Printf("Deleted file: %s", path)
					}
				}
			}
		}

		return nil
	}

	// Walk the directory
	if err := filepath.Walk(dirConfig.Path, walkFn); err != nil {
		return err
	}

	// Second pass: remove empty directories if configured
	if dirConfig.RemoveEmptyDirs {
		logger.Printf("Checking for empty directories in %s", dirConfig.Path)

		// We need to walk from the deepest directories first
		var dirs []string
		walkDirsFn := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Continue walking
			}
			if info.IsDir() && path != dirConfig.Path {
				dirs = append(dirs, path)
			}
			return nil
		}

		if err := filepath.Walk(dirConfig.Path, walkDirsFn); err != nil {
			return err
		}

		// Sort directories by depth (most nested first)
		for i := len(dirs) - 1; i >= 0; i-- {
			dir := dirs[i]

			// Skip if we're excluding subdirectories
			if dirConfig.ExcludeSubdirs && dir != dirConfig.Path {
				continue
			}

			// Check if directory is empty
			empty, err := isDirEmpty(dir)
			if err != nil {
				logger.Printf("Error checking if directory %s is empty: %v", dir, err)
				continue
			}

			if empty {
				if securityConfig.DryRun {
					logger.Printf("Would remove empty directory: %s", dir)
				} else {
					if err := os.Remove(dir); err != nil {
						logger.Printf("Error removing directory %s: %v", dir, err)
					} else {
						logger.Printf("Removed empty directory: %s", dir)
					}
				}
			}
		}
	}

	return nil
}

// isDirEmpty checks if a directory is empty
func isDirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// Try to read just one file
	_, err = f.Readdirnames(1)
	if err == nil {
		// Directory is not empty
		return false, nil
	}
	if err != nil && err.Error() == "EOF" {
		// Directory is empty
		return true, nil
	}
	return false, err
}

// secureDeleteFile performs secure deletion of a file by overwriting with random data
func secureDeleteFile(path string, passes int, logger *log.Logger) error {
	// Open the file for writing
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
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

	// Perform the secure deletion passes
	for pass := 0; pass < passes; pass++ {
		logger.Printf("Secure delete pass %d/%d for %s", pass+1, passes, path)

		// Reset to beginning of file
		if _, err := file.Seek(0, 0); err != nil {
			return err
		}

		// Fill the buffer with random data for this pass
		for i := range buf {
			buf[i] = byte(pass ^ i)
		}

		// Write the random data to the file
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
	}

	// Final deletion
	return os.Remove(path)
}

// setupLogger sets up the logger based on configuration
func setupLogger(config LoggingConfig) (*log.Logger, error) {
	if !config.Enabled {
		// If logging is disabled, use a no-op logger
		return log.New(ioutil.Discard, "", 0), nil
	}

	// Ensure the log directory exists
	logDir := filepath.Dir(config.File)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// Open log file
	logFile, err := os.OpenFile(config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	// Set up logger
	var logFlags int
	switch strings.ToLower(config.Level) {
	case "debug":
		logFlags = log.Ldate | log.Ltime | log.Lshortfile
	default:
		logFlags = log.Ldate | log.Ltime
	}

	return log.New(logFile, "", logFlags), nil
}

// PrintHelp prints the help information
func PrintHelp() {
	fmt.Printf("%s v%s - A program to remove files older than a specified retention period\n\n", ProgramName, ProgramVersion)
	fmt.Println("Usage:")
	fmt.Println("  filekeeper [options]")
	fmt.Println("\nOptions:")
	fmt.Println("  --help                  Show this help message")
	fmt.Println("  --version               Show version information")
	fmt.Println("  --init                  Create a default configuration file")
	fmt.Println("  --config PATH           Specify an alternative configuration file path")
	fmt.Println("  --install-systemd       Create systemd service and timer files")
	fmt.Println("  --systemd-template-only Output systemd templates without creating files")
	fmt.Println("  --dry-run               Run without actually deleting any files")
	fmt.Println("  --force                 Run even if disabled in the configuration")

	fmt.Println("\nDefault configuration paths:")
	if isRoot {
		fmt.Println("  - System config (root): /etc/filekeeper/filekeeper.yaml")
	} else {
		homeDir, _ := os.UserHomeDir()
		fmt.Println("  - User config: " + filepath.Join(homeDir, ".config", "filekeeper.yaml"))
	}

	fmt.Println("\nExamples:")
	fmt.Println("  filekeeper --init                   # Create default configuration")
	fmt.Println("  filekeeper                          # Run with default configuration")
	fmt.Println("  filekeeper --dry-run               # Simulate deletion without removing files")
	fmt.Println("  filekeeper --install-systemd        # Install systemd service and timer")
}

func main() {
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

	// If no arguments and config doesn't exist, show help
	if len(os.Args) == 1 {
		_, err := os.Stat(configFile)
		if os.IsNotExist(err) {
			PrintHelp()
			fmt.Println("\nRun 'filekeeper --init' to create a default configuration file.")
			return
		}
		// Otherwise continue with default config
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
			os.Exit(1)
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
			os.Exit(1)
		}
		return
	}

	// Try to load configuration
	config, err := LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration from %s: %v\n", configPath, err)
		fmt.Println("Run 'filekeeper --init' to create a default configuration file.")
		os.Exit(1)
	}

	// Check if program is enabled
	if !config.General.Enabled && !force {
		fmt.Println("Program is disabled in configuration. Use --force to run anyway.")
		return
	}

	// Setup logger
	logger, err := setupLogger(config.General.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up logger: %v\n", err)
		os.Exit(1)
	}

	// Override dry run if specified in command line
	if dryRun {
		config.Security.DryRun = true
	}

	// Log startup
	logger.Printf("Starting %s v%s", ProgramName, ProgramVersion)
	logger.Printf("Configuration loaded from: %s", configPath)
	if config.Security.DryRun {
		logger.Printf("Running in dry-run mode - no files will be deleted")
		fmt.Println("Running in dry-run mode - no files will be deleted")
	}

	// Process each directory
	for _, dirConfig := range config.Directories {
		if err := ProcessDirectory(dirConfig, config.Security, logger); err != nil {
			logger.Printf("Error processing directory %s: %v", dirConfig.Path, err)
			fmt.Fprintf(os.Stderr, "Error processing directory %s: %v\n", dirConfig.Path, err)
		}
	}

	logger.Printf("Finished processing all directories")
}
