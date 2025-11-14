# Changelog

All notable changes to the Smart Contract Event Indexer project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- GraphQL/API Gateway now proxies through gRPC Query/Admin services with gqlgen resolvers
- Integration test helper utilities for service-level testing
- Complete Phase 2 Indexer Service implementation
- Blockchain connection module with Ganache support
- Event parsing for ERC20 Transfer events
- Database persistence layer with PostgreSQL
- Reorg detection and handling
- Confirmation strategy checking
- Graceful shutdown and state recovery
- Error classification and retry logic
- Comprehensive unit tests for parser module (18/18 passing)
- Integration tests for service connectivity (4/4 passing)
- Test utilities and mock data generators
- Makefile integration with test commands
- Docker Compose development environment
- Comprehensive documentation structure

### Changed
- Admin and Query services now share improved logging/configuration defaults
- Added Go build cache directories to `.gitignore`
- Organized documentation according to project standards
- Updated Makefile with test-integration commands
- Improved error handling throughout codebase
- Enhanced logging with structured context

### Fixed
- Resolved API handler/database schema mismatches that blocked REST endpoints
- XCode Command Line Tools compatibility issues
- Logger interface type mismatches
- Database schema mismatches in tests
- Compilation errors with CGO dependencies

### Technical Details
- **Language**: Go 1.21
- **Dependencies**: go-ethereum, sqlx, lib/pq, testify
- **Database**: PostgreSQL with JSONB support
- **Testing**: Unit tests + Integration tests
- **Binary Size**: 19MB (optimized)
- **Service Startup**: <2 seconds

## [0.1.0] - 2025-10-17

### Added
- Initial project structure
- Shared modules (models, database, utils)
- Basic configuration management
- Docker Compose setup
- Database migrations
- README documentation

### Technical Details
- **Architecture**: Microservices with Go
- **Database**: PostgreSQL 15
- **Development**: Docker Compose + Ganache
- **Version Control**: Git with feature branches
