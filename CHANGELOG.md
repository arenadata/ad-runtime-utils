## [v0.1.3] — 2025-08-21

### Added
- Skip missing external service configs instead of failing 

## [v0.1.2] — 2025-08-04

### Added
- Support for per-service external config via `path:` key under `services.<name>`
- Strict YAML validation (unknown fields are now rejected)
  
## [v0.1.1] — 2025-07-23

### Added
- Recursive detection of nested `bin/<exe>` directories (including `jre/bin` and deeper)
- Unit tests for `internal/fs.ExistsInBin`

## [v0.1.0] — 2025-07-22
### Added
- First release