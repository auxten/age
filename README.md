# AG Enhanced (age)

AG Enhanced (`age`) is an extension of `ag` (The Silver Searcher), adding support for searching within various compressed file formats and automatically compressing older log files. This project maintains the same usage patterns as `ag`, ensuring seamless integration for existing users with additional functionality.

## Features

- **Search within Compressed Files**: Enables searching within zip, gz, tgz, and zstd compressed files directly.
- **Automatic Log File Compression**: Automates the compression of `.log` files older than 7 days into `.log.zstd` to efficiently manage disk space.
- **Familiar `ag` Interface**: Utilizes the same command-line interface as `ag`, allowing users to integrate with no additional learning curve.

## Installation

### Prerequisites

Ensure that `ag` (The Silver Searcher) is installed on your system. For installation instructions, please refer to its [GitHub page](https://github.com/ggreer/the_silver_searcher).

### Installing AG Enhanced

Install `age` directly from the source using Go:

```bash
go install github.com/auxten/age@latest
```

This command fetches the latest version of `age` from the GitHub repository and installs it.

## Usage

Use `age` exactly like `ag`. Here are some example commands to get started:

```bash
# Perform a basic search
age "search pattern" /path/to/search

# Use additional `ag` options
age -i "search pattern" /path/to/search

# Search within compressed files
age "pattern" /path/with/compressed/files
```


### Automatic Log File Compression

This functionality runs automatically. Any `.log` file older than 7 days is compressed to `.log.zstd`, and the original file is deleted.

## Supported File Formats

- **.zip**: Searches are conducted on each file within the zip archive.
- **.gz**: Gzipped files are decompressed on-the-fly and searched.
- **.zstd**: Files compressed with Zstandard are decompressed for searching.

## Contributing

Contributions are highly appreciated.

## License

This project is licensed under the Apache-2.0 license. See the [LICENSE](LICENSE) file for more details.
