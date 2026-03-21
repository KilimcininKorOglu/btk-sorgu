# Changelog

## [3.0.2] - 2026-03-21

### Changed
- Add Makefile and GoReleaser config, adapt build.bat for project
- Remove build.sh script in favor of Makefile
- Remove dead gzip handling in getCaptcha
- Drain response body in getSessionCookies for connection reuse

### Fixed
- Only allow q-quit in non-input TUI states (domains with "q" were untypeable)
- Display correct result after refreshing non-last history entry
- Detect unexpected BTK response format instead of false positive "accessible"
- Check scanner error after loading .env file
- Clear error on Esc from result state
- Reset refreshingIdx on query error
- Use errors.New in retry error wrapping to prevent format string issues
- Set timestamp on query results for history tracking
- Handle saveHistory errors and display to user
