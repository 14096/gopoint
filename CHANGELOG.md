# Changelog

All notable changes to this project are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to
[Semantic Versioning](https://semver.org/). Pre-release versions (`-beta1`, ...) may contain breaking changes.

## [Unreleased]

## [0.0.1-beta3] - 2026-11-26

### Added

- SharePoint: `Client.Batch` (`$batch`), `Files().UploadLarge` (chunked upload) and `Webhooks` (with a notification `Handler`).
- Graph: `Client.Batch` (`$batch`), `Delta` on drives and list items, `Subscriptions` (with a notification `Handler`) and `TermStore`.
- `httpx`: per-call `Option`s (`WithHeader`, `WithQuery`, `WithOData`, `WithIfMatch`, `WithIfNoneMatch`, `WithNoAuth`), accepted by the `Client` helpers.
- `httpx`: generic `Get[T]`, `Post[T]` and `Call[T]`; `Client.Send` and `Response.ETag`; `Client.DoHTTP` and `NewRequestFromHTTP` for `*http.Request` interop.
- `httpx`: `Config.Middleware` and `Config.ShouldRetry`; `Delta` for Graph delta queries; `ISVUserAgent` and `Response.RateLimit`; `NewError`.

### Changed

- 504 responses are now retried for idempotent methods.
- `Client.Call`, `Get`, `Post`, `Patch`, `Put` and `Delete` take trailing variadic `...Option` (source compatible).
- `drives.Item` has a `Deleted` field, set on items removed since the previous `Delta` call.

## [0.0.1-beta2] - 2026-11-26

### Added

- SharePoint REST services: `Pages`, `Folders`, `Navigation`, `Permissions`, `RecycleBin`, `Users`, `Search` and `Sharing`, `Views` (list views) and `HubSites`.
- Graph services: `Pages`, `Folders`, `Permissions`, `RecycleBin` (beta), `Users`, `Search` and `Sharing`.

## [0.0.1-beta1] - 2026-11-26

First release.
