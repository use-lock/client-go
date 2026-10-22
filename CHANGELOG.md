# Changelog

## [0.2.0](https://github.com/use-lock/client-go/compare/v0.1.0...v0.2.0) (2026-10-22)


### ⚠ BREAKING CHANGES

* the Patch* methods are removed; use the Update* methods, which now send PATCH with partial bodies.

### Features

* follow Lock's PATCH-only API and client revocation ([16db41b](https://github.com/use-lock/client-go/commit/16db41b0604d9aa24f99b29dcb782871e2240df5))

## 0.1.0 (2026-09-16)


### ⚠ BREAKING CHANGES

* generate grouped clients with typed responses and API errors

### Features

* add generated Go client and release automation ([6ccabc3](https://github.com/use-lock/client-go/commit/6ccabc3c894fb1b62e7f2a16b5474cd898f06d39))
* generate grouped clients with typed responses and API errors ([bbb4ba4](https://github.com/use-lock/client-go/commit/bbb4ba4d9fd8532a6667b0c53916e84dd3306212))
