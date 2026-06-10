# Changelog

## [0.1.3] - 2026-06-10

### Changed
- `exec()` now consumes SSE stream — returns `str` instead of `dict`
- Strips `data: ` prefix from each SSE line and concatenates output

## [0.1.1] - 2026-06-08

### Added
- Python SDK (`pip install jhansi`) wrapping the full Petri v0.4 API
- `Sandbox.create()`, `exec()`, `upload_file()`, `upload_zip()`, `status()`, `delete()`
- Context manager support — `with Sandbox(language="python") as sb:`
- SDK docs on `docs.jhansi.io` — installation, sandbox reference, context manager, examples
- Updated quickstart to lead with SDK

## [0.1.0] - 2026-06-08

### Added
- Initial scaffold — pyproject.toml, jhansi package, passing import test
