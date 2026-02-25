# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Added `CHANGELOG.md` for ongoing release notes.

## [0.1.0] - 2026-02-25

### Added
- Added brand-new `wechat-ease` core library implementation in [wechat_ease.go](/Users/jackman/Desktop/projects/wechat-ease/wechat_ease.go).
- Added full unit test coverage in [wechat_ease_test.go](/Users/jackman/Desktop/projects/wechat-ease/wechat_ease_test.go) (success flows, retries, timeout, error handling, wrappers).
- Added module definition with Go `1.23.12` in [go.mod](/Users/jackman/Desktop/projects/wechat-ease/go.mod).
- Added Chinese code comments for key behaviors (timeouts, retries, error interception, API methods).
- Added project documentation in [README.md](/Users/jackman/Desktop/projects/wechat-ease/README.md).
- Added runnable examples:
  - [examples/basic/main.go](/Users/jackman/Desktop/projects/wechat-ease/examples/basic/main.go)
  - [examples/oauth/main.go](/Users/jackman/Desktop/projects/wechat-ease/examples/oauth/main.go)
  - [examples/template/main.go](/Users/jackman/Desktop/projects/wechat-ease/examples/template/main.go)

### Changed
- Rebuilt the library architecture to `Client + Option` with zero third-party runtime dependencies.
- Improved production readiness with timeout control, retry/backoff for access token retrieval, unified API error model, and customizable error interceptor.


## 发布流程

1. 在日常开发中将变更先写入 `Unreleased`。
2. 发版时将 `Unreleased` 内容迁移到新版本（如 `0.1.1`），并写入发布日期（`YYYY-MM-DD`）。
3. 为新版本创建 Git tag（如 `v0.1.1`）。
4. 清空 `Unreleased`，保留结构，进入下一轮迭代。
