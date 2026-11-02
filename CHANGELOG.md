# Changelog: ps5-profile

## Unreleased

- 2026-11-02: first cut. `/api/userProfile/v1/internal/users/{me,accountId}/profiles`, legacy `profile2` (with `fields`
  filtering of top-level names) and the `dms.api` device list (empty). Bearer auth through ps5-common/oauth, accounts
  through ps5-common/accountcore. Tests with httptest. Not deployed, no tag, never tried by a console.
