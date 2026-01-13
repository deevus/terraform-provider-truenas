# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [0.2.0] - 2026-01-13


### Added

- Add force_destroy option for recursive dataset deletion
- Add RemoveAll and Chown operations to SSH client
- Add force_destroy option to host_path resource for non-empty directories
- Add force_destroy option to file resource for permission-locked files
- Add ChmodRecursive method to SSH client
- Add mountpoint permission management to dataset resource
- **dataset:** Add full_path computed attribute to schema
- **dataset:** Sync full_path from API response
- **dataset:** Deprecate mount_path in favor of full_path
- **dataset:** Deprecate name in favor of path with parent
- **dataset:** Support path attribute with parent, prefer over name
- **host_path:** Add deprecation warning recommending datasets
- Support file ownership in file resource operations
- Add path traversal protection to file resource
- **provider:** Add host_key_fingerprint schema attribute to SSH block
- **ssh:** Implement host key verification using fingerprint
- **errors:** Add NewHostKeyError constructor for host key verification

### Changed

- Improve host path creation and error handling
- Remove unused hasPermissions method from HostPathResource
- Replace TrueNAS API calls with SFTP for directory operations
- Extract dataset query and mapping logic into reusable functions
- Add uid/gid parameters to WriteFile interface
- Replace SFTP with TrueNAS API and sudo for file operations

### Documentation

- Add comprehensive midclt API reference documentation
- Add structured schemas and examples for pool API endpoints
- Add detailed schemas and examples for filesystem operations
- Enhance app API documentation with schemas and job operation notes
- Add comprehensive plan for app data management improvements
- Add design plan for deprecating host_path resource
- Add implementation plan for dataset schema improvements
- **dataset:** Update schema descriptions for new path usage
- Add SSH host key verification design document
- Add plan for fixing documentation gaps after recent implementation changes
- Add SSH host key and sudo requirements to templates
- Update provider examples with host_key_fingerprint
- Document missing resource attributes and deprecations
- Update README with SSH host key and sudo requirements

### Fixed

- Improve TrueNAS error parsing to strip process exit and traceback noise
- Prevent state drift for optional computed attributes
- Resolve permission issues during force_destroy deletion
- Restore parent directory permissions after force_destroy deletion
- Query dataset after creation to populate all computed attributes
- **dataset:** Update error message to mention all valid configurations
- **dataset:** Simplify name deprecation message
- Resolve deadlock in SFTP connection initialization
- Correct SSH host key fingerprint command in help text

### Miscellaneous

- Mark documentation gaps plan as completed

### Testing

- Add comprehensive tests for MockClient SFTP methods
- Add error parsing tests for TrueNAS error messages
- Add comprehensive tests for YAMLStringType implementation
- Add edge case tests for FileResource operations
- Add error handling tests for AppResource Create and Update
- Add comprehensive tests for optional computed attribute behavior
- **dataset:** Update test helpers to include full_path attribute
- Update SSH client tests for API-based file operations
- **ssh:** Add host key verification unit tests
- Add host_key_fingerprint attribute to SSH configuration tests

### Build

- Add automated release workflow with changelog generation

## [0.1.2] - 2026-01-11


### Documentation

- Add README

### Fixed

- Correct compose_config drift detection

## [0.1.1] - 2026-01-11


### Documentation

- Update provider source to deevus/truenas
- Add TrueNAS user setup instructions with screenshot

## [0.1.0] - 2026-01-11


### Added

- **client:** Add error types and parsing
- **client:** Add Client interface and MockClient
- **client:** Add SSH client implementation
- **client:** Add job polling with exponential backoff
- **client:** Add midclt command builder and param types
- **provider:** Add schema and configuration
- **datasources:** Add truenas_pool data source
- **datasources:** Add truenas_dataset data source
- **resources:** Add truenas_dataset resource
- **resources:** Add truenas_host_path resource
- **resources:** Add truenas_app resource
- **app:** Simplify resource for custom Docker Compose apps
- **client:** Extend Client interface with SFTP methods
- **client:** Implement WriteFile SFTP method
- **client:** Implement ReadFile SFTP method
- **client:** Implement DeleteFile, FileExists, MkdirAll SFTP methods
- **resources:** Add file resource scaffold
- **resources:** Implement file resource validation
- **resources:** Implement file resource Create operation
- **resources:** Implement file resource Read operation
- **resources:** Implement file resource Update and Delete operations
- **provider:** Register truenas_file resource
- **release:** Add GoReleaser and GitHub Actions for Terraform Registry publishing

### Documentation

- Add comprehensive implementation plan with TDD tasks
- Add terraform example files
- Add documentation templates and generation task
- Add truenas_file implementation plan
- Add truenas_file resource documentation

### Fixed

- **client:** Prevent command injection in SSH params
- **client:** Use io.ReadAll for robust large file reading
- **client:** Apply mode parameter in MkdirAll
- **resources:** Explicitly set ID in Update for consistency
- **file:** Handle unknown values in validation and use ID for import path
- **file:** Set defaults for mode/uid/gid in Update when unknown
- **app:** Query state after create/update instead of parsing progress output

### Miscellaneous

- Mise.toml
- Initialize go module with dependencies
- Update module path to github.com/deevus
- Add mise configuration and task runners
- Update mise.toml to use latest Go version
- Add main entry point and provider stub
- Add staticcheck to mise.toml
- Change `jj describe` to `jj commit` in implementation plan
- Update CLAUDE.md with mise instructions
- Add github.com/pkg/sftp dependency
- Add PolyForm Noncommercial license
- **license:** Change from PolyForm Noncommercial to MIT
- Add GPG public key for release verification

### Testing

- **client:** Add missing tests for 100% coverage
- **file:** Add failing tests for unknown value validation and import path

### Debug

- **app:** Add logging for app.update response

