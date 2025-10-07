# Changelog

All notable changes to OCX Protocol will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Professional repository structure with CONTRIBUTING.md, CHANGELOG.md, CODE_OF_CONDUCT.md
- Comprehensive documentation organization in docs/ directory
- Clean, minimal README without emojis

### Changed
- Reorganized all documentation into docs/ directory
- Updated README to professional, enterprise-grade format
- Improved commit message standards (Conventional Commits)

## [0.1.1] - 2025-10-07

### Added
- Token bucket rate limiting (10 req/s, burst 20)
- Request size limits (10MB maximum)
- Security headers middleware (HSTS, CSP, X-Frame-Options)
- Professional website deployment to ocx.world
- Comprehensive audit reports and deployment guides

### Fixed
- Receipt determinism issues (millisecond precision)
- Duplicate method declarations in reputation handlers
- Compilation errors in server build
- Website test page artifacts and inconsistencies

### Security
- Implemented DoS protection via rate limiting
- Added request size validation to prevent memory exhaustion
- Applied security headers to all HTTP responses
- Fixed nanosecond timing variance in receipt generation

## [0.1.0] - 2025-09-20

### Added
- Deterministic Virtual Machine (D-MVM) with seccomp sandboxing
- Ed25519 cryptographic receipt system
- REST API server with authentication and idempotency
- PostgreSQL integration for receipt storage
- Standalone Rust verification library
- Health check endpoints (/livez, /readyz, /health)
- Prometheus metrics endpoint
- Docker and Kubernetes deployment configurations
- Comprehensive smoke tests and demo scripts

### Changed
- Migrated from prototype to production-ready architecture
- Improved error handling and logging throughout codebase
- Optimized receipt generation performance

### Security
- Implemented cgroup resource limits
- Added seccomp syscall filtering
- Enabled canonical CBOR encoding for deterministic receipts
- Integrated Ed25519 signature verification

## [0.0.1] - 2025-08-15

### Added
- Initial prototype release
- Basic execution engine
- Simple receipt generation
- Command-line interface
- Basic API server

---

## Version History Notes

### Versioning Scheme

We use [Semantic Versioning](https://semver.org/):
- **MAJOR**: Incompatible API changes
- **MINOR**: Backwards-compatible functionality additions
- **PATCH**: Backwards-compatible bug fixes

### Release Process

1. Update CHANGELOG.md with changes
2. Update version in relevant files
3. Create git tag: `git tag -a v0.1.0 -m "Release v0.1.0"`
4. Push tag: `git push origin v0.1.0`
5. Create GitHub release with changelog notes

### Change Categories

- **Added**: New features
- **Changed**: Changes in existing functionality
- **Deprecated**: Soon-to-be removed features
- **Removed**: Removed features
- **Fixed**: Bug fixes
- **Security**: Security improvements or fixes

---

[Unreleased]: https://github.com/KuroKernel/ocx-protocol/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/KuroKernel/ocx-protocol/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/KuroKernel/ocx-protocol/compare/v0.0.1...v0.1.0
[0.0.1]: https://github.com/KuroKernel/ocx-protocol/releases/tag/v0.0.1
