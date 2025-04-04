# FileKeeper

A Go utility for automated file cleanup based on retention policies.

## Overview

FileKeeper helps you manage disk space by automatically removing files that are older than a specified retention period. It's designed for Linux systems and can be easily integrated with systemd for scheduled execution.

### Supported Architectures

FileKeeper is available for the following architectures:
- x86-64 (AMD64)
- ARM (ARMv7)
- ARM64 (AArch64)

This makes it suitable for use on a wide range of devices, from standard servers to Raspberry Pi and other ARM-based devices.

## Features

- Clean up files based on configurable retention periods (e.g., 30d, 24h, 60m)
- Process multiple directories with different retention policies
- Filter files using pattern matching
- Option to remove empty directories
- Secure deletion mode for HDD storage (multi-pass overwrite)
- Dry-run mode to preview what would be deleted
- Systemd integration for scheduled execution

## Installation

### From source

1. Clone the repository:
   ```bash
   git clone https://github.com/ykargin/filekeeper.git
   cd filekeeper
   ```

2. Initialize the Go module (if not already done):
   ```bash
   go mod init github.com/ykargin/filekeeper
   ```

3. Install dependencies:
   ```bash
   go get gopkg.in/yaml.v3
   ```

4. Build the binary:
   ```bash
   go build -o filekeeper
   ```

5. Install the binary (optional):
   ```bash
   sudo install -m 755 filekeeper /usr/local/bin/
   ```

## Usage

### Initialize configuration

Create a default configuration file:

```bash
filekeeper --init
```

This creates a template configuration file at one of these locations:
- For root users: `/etc/filekeeper/filekeeper.yaml.example`
- For regular users: `~/.config/filekeeper.yaml.example`

Review and rename the file to remove the `.example` suffix.

### Run FileKeeper

```bash
filekeeper              # Run using the default configuration
filekeeper --dry-run    # Run without actually deleting files 
filekeeper --force      # Run even if disabled in configuration
```

### Command-line options

```
Options:
  --help                  Show this help message
  --version               Show version information
  --init                  Create a default configuration file
  --config PATH           Specify an alternative configuration file path
  --install-systemd       Create systemd service and timer files
  --systemd-template-only Output systemd templates without creating files
  --dry-run               Run without actually deleting any files
  --force                 Run even if disabled in the configuration
```

## Configuration

FileKeeper uses a YAML configuration file with the following structure:

```yaml
# General settings
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
    file: "/var/log/filekeeper.log"

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

  - path: "/path/to/dir2"
    retention_period: "7d"
    file_pattern: "*.tmp"
    exclude_subdirs: true
    remove_empty_dirs: false

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
```

## Automated execution with systemd

FileKeeper makes it easy to set up automated cleaning using systemd:

```bash
filekeeper --install-systemd
```

This will create:
- A systemd service file
- A systemd timer file (set to run daily)

And provides instructions for enabling the timer.

For custom systemd configurations:

```bash
filekeeper --systemd-template-only > filekeeper-systemd-templates.txt
```

## Retention Period Format

Retention periods can be specified in:
- Days: `30d` (30 days)
- Hours: `24h` (24 hours)
- Minutes: `60m` (60 minutes)

## Development

If you want to contribute to FileKeeper, you'll need to set up the Go development environment:

```bash
# Initialize the module (if starting from scratch)
go mod init github.com/ykargin/filekeeper

# Get dependencies
go get gopkg.in/yaml.v3

# Build for development (current architecture)
go build -o filekeeper

# Build for all supported architectures
./build_all.sh
```

For more detailed instructions on contributing, please see [CONTRIBUTING.md](CONTRIBUTING.md).

## Testing

FileKeeper includes a comprehensive test suite to ensure reliability and correctness:

```bash
# Run all tests
go test -v .

# Use the convenience script
./run_tests.sh
```

To generate a code coverage report:

```bash
./coverage.sh
```

This will create an HTML report (`coverage.html`) showing which parts of the code are covered by tests.

The test suite covers:
- Configuration loading and validation
- File retention period parsing
- Directory processing and file deletion logic
- Secure deletion functionality
- Empty directory detection and removal
- Systemd service file generation

## Releasing

A release script is provided to help with versioning and releasing new versions:

```bash
./release.sh 0.1.5
```

This script:
1. Runs all tests to ensure everything works
2. Updates the version number in the source code
3. Commits the version change
4. Creates a Git tag for the release

After running this script, you can push the changes and tag to GitHub, which will trigger the CI/CD pipeline to build and publish the release.

## Security Considerations

- The secure deletion option is intended for HDD storage where data recovery might be possible
- For SSD storage, TRIM operations make secure deletion unnecessary and potentially harmful
- Running in dry-run mode first is recommended to preview what will be deleted

## License

[MIT License](LICENSE)
