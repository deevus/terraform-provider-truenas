# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [0.15.0] - 2026-02-23


### Added

- **client:** Add Logger interface and NopLogger
- **client:** Add WithLogger option to NewSSHClient
- **provider:** Add TFLogAdapter bridging client.Logger to tflog
- Add TrueNASServices registry struct
- Wire TrueNASServices in provider.Configure()
- Add backward-compat Client field for transitional migration

### Changed

- **client:** Replace tflog with Logger interface
- Consume truenas-go library instead of internal packages
- BaseResource uses TrueNASServices instead of client.Client
- Datasources use typed service methods
- Snapshot resource uses SnapshotService
- Cron_job resource uses CronService
- Cloudsync_credentials resource uses CloudSyncService
- Cloudsync_task resource uses CloudSyncService
- Dataset resource uses DatasetService/FilesystemService/SnapshotService
- Zvol resource uses DatasetService
- Host_path resource uses FilesystemService
- File resource uses FilesystemService
- App resource uses AppService
- App_registry resource uses AppService
- Virt_config resource uses VirtService
- Virt_instance resource uses VirtService
- Vm resource uses VMService

### Documentation

- Add note about copying local Claude settings to new worktrees
- Add design plan for extracting client/ to truenas-go

### Miscellaneous

- Clean up docs/plans after extraction complete
- Update truenas-go to v0.2.1
- Update truenas-go to v0.2.2

### Build

- Upgrade truenas-go dependency from v0.2.2 to v0.2.4

### Wire

- Pass TFLogAdapter to SSHClient via factory

## [0.14.0] - 2026-02-15


### Added

- **vm:** Scaffold truenas_vm resource with schema and model types
- **vm:** Implement full CRUD, device reconciliation, and state management
- **vm:** Add command_line_args attribute
- **zvol:** Scaffold truenas_zvol resource with schema and model types
- **zvol:** Implement Create and Read operations
- **zvol:** Implement Update and Delete operations
- **vm:** Add exists attribute to raw device block

### Changed

- **vm:** Split vm.go into focused sub-modules
- Extract shared pool/dataset helpers for zvol reuse
- Add BaseResource with Configure and ImportState
- Migrate 9 resources to embed BaseResource
- Migrate 4 custom-ImportState resources to embed BaseResource

### Documentation

- Expand VM API documentation with detailed parameters and device attributes
- Add truenas_vm resource implementation plan
- Add truenas_vm resource example and documentation
- Add truenas_zvol resource implementation plan
- Add truenas_zvol resource example and documentation
- Remove completed truenas_vm resource implementation plan
- Add implementation plan for BaseResource extraction

### Fixed

- **vm:** Make display password required
- **vm:** Resolve inconsistent result after apply for exists and serial
- **vm:** Skip serial in device params when unknown (Computed)

### Miscellaneous

- Clean up plan docs

### Testing

- **vm:** Add comprehensive tests for CRUD, device reconciliation, and helpers
- **vm:** Add command_line_args to buildCreateParams and buildUpdateParams tests
- **zvol:** Add import, edge case, and shared helper tests

## [0.13.0] - 2026-02-07


### Added

- Add truenas_app_registry resource (#37)

## [0.12.0] - 2026-02-04


### Added

- Early version detection with Connect/Version pattern (#24)
- Add truenas_virt_config data source and resource (#29)
- Add truenas_virt_instance resource (#30)
- Add addresses attribute to virt_instance resource (#36)

### Documentation

- **api:** Expand filesystem API documentation with 10 new methods (#21)

### Fixed

- SSH transport job polling for TrueNAS 24.x (#23)
- WebSocket transient network resilience (#26)
- Preserve custom docs with tfplugindocs templates (#35)

### Testing

- Add comprehensive coverage for version parsing and MockClient methods
- Add comprehensive version comparison tests (#28)

## [0.11.0] - 2026-01-28


### Added

- Add case-insensitive string custom type for semantic equality
- Enrich TrueNAS job errors with app lifecycle log details
- **cloudsync:** Add include parameter support (#18)
- **cloudsync:** Add TrueNAS 24.x API compatibility for credentials (#19)

### Changed

- Make WebSocket credentials optional in schema
- Improve test helper readability with structured parameters
- **api:** Add GetVersionOrDiag helper to reduce duplication

### Documentation

- Clarify TrueNAS 25.0+ requirement for WebSocket transport

### Fixed

- Enforce TrueNAS 25.0+ requirement for WebSocket transport
- Preserve user-specified case in desired_state attribute
- **app:** Resolve Value Conversion Error for CaseInsensitiveStringType
- **dataset:** Convert quota/refquota to integers for API compatibility (#20)

### Miscellaneous

- Remove outdated planning documents

### Testing

- Add coverage for desired_state case preservation behavior

## [0.10.0] - 2026-01-26


### Added

- **dataset:** Add ValidateConfig for mode requirement
- **cron_job:** Rename stdout/stderr to capture_stdout/capture_stderr (#14)
- Add rate limiting and retry logic (#16)
- Add JSON-RPC WebSocket transport for TrueNAS API calls (#17)

### Changed

- **dataset:** Remove unreachable stripacl fallback

### Documentation

- Add design for mode validation with uid/gid
- Add implementation plan for mode validation
- **dataset:** Document mode requirement with uid/gid

### Testing

- **dataset:** Add ValidateConfig tests for mode requirement
- **dataset:** Verify ValidateConfig interface in TestNewDatasetResource
- **dataset:** Remove obsolete stripacl test

## [0.9.0] - 2026-01-21


### Added

- **api:** Add cron job API response struct
- **resources:** Add cron job resource scaffolding
- **resources:** Add cron job helper functions
- **resources:** Implement cron job Create method
- **resources:** Implement cron job Read method
- **resources:** Implement cron job Update method
- **resources:** Implement cron job Delete method
- **provider:** Register cron job resource

### Documentation

- **plans:** Add cron job resource implementation plan
- Add truenas_cron_job resource documentation

### Testing

- **resources:** Add cron job basic tests
- **resources:** Add cron job test helpers
- **cron_job:** Add Create method tests
- **cron_job:** Add Read method tests
- **cron_job:** Add Update method tests
- **cron_job:** Add Delete method tests
- **cron_job:** Add schedule variation tests

## [0.8.0] - 2026-01-20


### Added

- **errors:** Add app lifecycle fields to TrueNASError struct
- **errors:** Detect app lifecycle pattern in error messages
- **errors:** Implement app lifecycle log parsing
- **errors:** Prefer AppLifecycleError in Error() output
- **jobs:** Fetch and parse app lifecycle log on failure
- Add setup-dev task for Terraform provider local development

### Documentation

- Add app lifecycle error handling design and implementation plan

### Fixed

- **ssh:** Use app lifecycle error enrichment in CallAndWait
- **errors:** Use cat command instead of non-existent API

### Testing

- **errors:** Add tests for app lifecycle pattern detection
- **errors:** Add tests for app lifecycle log parsing
- **jobs:** Add test for app lifecycle log fetching
- **jobs:** Verify graceful handling when log fetch fails

## [0.7.0] - 2026-01-20


### Added

- **app:** Add restart_triggers attribute for file dependency restarts

### Fixed

- **app:** Predict state value during plan when drift detected

### Miscellaneous

- Add .beads/ to gitignore
- Remove jj from development workflow

### Testing

- Add midclt response fixtures for unit testing
- Add comprehensive tests for restart_triggers edge cases

### Ci

- Add test workflow with gotestfmt

## [0.6.0] - 2026-01-18


### Added

- **api:** Add cloud sync API response types
- **resources:** Add cloud sync credentials resource skeleton
- **resources:** Implement cloud sync credentials schema with provider blocks
- **resources:** Implement cloud sync credentials Configure
- **resources:** Add cloud sync credentials model types
- **resources:** Implement cloud sync credentials Create for S3
- **resources:** Implement cloud sync credentials Read
- **resources:** Implement cloud sync credentials Update
- **resources:** Implement cloud sync credentials Delete
- **provider:** Register cloud sync credentials resource
- **resources:** Add cloud sync task resource skeleton
- **resources:** Implement cloud sync task schema and model types
- **resources:** Implement cloud sync task Create
- **resources:** Implement cloud sync task Read
- **resources:** Implement cloud sync task Update
- **resources:** Implement cloud sync task Delete
- **provider:** Register cloud sync task resource
- **datasources:** Add cloud sync credentials data source

### Changed

- Replace WriteFile positional parameters with structured WriteFileParams

### Documentation

- Add cloud sync resources design
- Add cloud sync implementation plan with TDD
- Add cloud sync resource documentation

### Fixed

- **resources:** Change provider block attributes to Optional with runtime validation
- **resources:** Change encryption.password to Optional with runtime validation
- Correct cloud sync credentials provider payload structure
- Correct CloudSync API object types for credentials and bandwidth limits

### Testing

- **resources:** Add cloud sync credentials metadata test
- **resources:** Add cloud sync credentials test helpers
- **resources:** Add cloud sync credentials error path tests
- **resources:** Add B2, GCS, Azure provider create tests
- **resources:** Add cloud sync credentials import test
- **resources:** Add cloud sync task test helpers
- **resources:** Add cloud sync task error path tests
- **resources:** Add provider-specific create tests for cloud sync task
- **resources:** Add schedule validation tests for cloud sync task
- **resources:** Add schedule validation tests for cloud sync task
- **resources:** Add encryption tests for cloud sync task

## [0.5.0] - 2026-01-16


### Added

- **app:** Add state constants and normalization helper
- **app:** Add isStableState and isValidDesiredState helpers
- **app:** Add case-insensitive state plan modifier
- **app:** Add desired_state and state_timeout schema attributes
- **app:** Add waitForStableState polling helper
- **app:** Add queryAppState helper method
- **app:** Add reconcileDesiredState method
- **app:** Handle desired_state in Create lifecycle
- **app:** Handle desired_state reconciliation in Update lifecycle
- **app:** Preserve desired_state in Read lifecycle
- Add job logs excerpt to error reporting
- **snapshot:** Add resource scaffold and schema
- **snapshot:** Implement Create with hold support
- **snapshot:** Implement Read with not-found handling
- **snapshot:** Implement Update for hold/release
- **snapshot:** Implement Delete with hold release and error tests
- **snapshots:** Add data source scaffold and schema
- **snapshots:** Implement data source with filtering
- **dataset:** Add snapshot_id for clone creation

### Documentation

- Add design for desired_state attribute on truenas_app
- Add implementation plan for desired_state attribute
- Add desired_state and state_timeout to app resource docs
- Add snapshot resource design
- Add snapshot implementation plan
- Add snapshot resource and data source documentation
- Add version-aware API resolution design

### Fixed

- Version-aware API resolution and desired_state normalization
- Parse system.version as raw string, not JSON
- **snapshot:** Correct API field mapping for name and hold detection
- **app:** Add UseStateForUnknown to state attribute
- **app:** Smart plan modifier for computed state attribute

### Testing

- **app:** Add CRASHED state handling tests
- Expand test coverage for edge cases and error handling
- Add comprehensive tests for TrueNAS job logs excerpt handling
- **snapshot:** Add schema and configure tests
- **snapshot:** Add ImportState test
- **provider:** Update expected counts for snapshot resource and data source
- Add comprehensive error handling and edge case tests for snapshot operations

### Build

- Enable git-cliff for automated changelog generation

### Ci

- Fix release process with git-cliff integration

## [0.3.0] - 2026-01-14


### Added

- **ssh:** Add MaxSessions field to SSHConfig
- **ssh:** Add session semaphore to SSHClient
- **ssh:** Add acquireSession helper method
- **ssh:** Add semaphore to Call method
- **ssh:** Add semaphore to CallAndWait method
- **provider:** Add max_sessions configuration option
- **ssh:** Add runSudoOutput and switch ReadFile to sudo cat

### Documentation

- Add SSH session semaphore design
- Add SSH session semaphore implementation plan
- Regenerate provider docs with max_sessions option
- Add design for ReadFile sudo cat implementation

### Fixed

- **ssh:** Reduce default max_sessions from 10 to 5

## [0.2.1] - 2026-01-14


### Fixed

- Add required stripacl option when setting ownership without mode

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

