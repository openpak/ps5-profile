# Changelog: ps5-profile

## v0.1.0 (2026-11-03)

- 2026-11-03: refuses to start without `PS5_ACCOUNT_CORE_URL` / `PS5_ACCOUNT_CORE_TOKEN` unless `PS5_DEV_MEMORY=1`.

- 2026-11-02: first cut. `/api/userProfile/v1/internal/users/{me,accountId}/profiles`, legacy `profile2` (with `fields`
  filtering of top-level names) and the `dms.api` device list (empty). Bearer auth through ps5-common/oauth, accounts
  through ps5-common/accountcore. Tests with httptest. Not deployed, no tag, never tried by a console.
