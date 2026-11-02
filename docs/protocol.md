# Protocol: ps5-profile

Shapes come from the public psn-api and PSNAWP libraries (see the PS5 endpoint notes); nothing here was captured from a
PS5. Whether the console itself calls these paths is unverified.

All requests need `Authorization: Bearer <token>` (ps5-common/oauth). Missing or bad token: `401` with the PSN error
body `{"error":{"referenceId","code","message"}}`.

## `GET /api/userProfile/v1/internal/users/{accountId}/profiles` (`m.np.playstation.com`)

`{accountId}` is a decimal 64-bit account id or `me`. Answer:

```json
{"onlineId":"name","aboutMe":"","avatars":[{"size":"xl","url":"..."}],"languages":["en-US"],"isPlus":false,"isOfficiallyVerified":false,"isMe":true}
```

`avatars` is empty unless `PS5_PROFILE_AVATAR_URL` is set (then sizes xl, l, m, s). `400` for a malformed id, `404`
for an unknown or inactive account.

## `GET /userProfile/v1/users/{onlineId}/profile2` (`us-prof.np.community.playstation.net`)

Answer `{"profile":{...}}` with `onlineId`, `accountId`, `npId` (`onlineId@np`, plain text), `avatarUrls[{size,avatarUrl}]`,
`plus`, `aboutMe`, `languagesUsed`, `trophySummary` (level 1, nothing earned: ps5-trophy owns the real values),
`isOfficiallyVerified`. If `fields=` is given only the listed top-level names are returned; sub-selections in
parentheses are ignored. Presence, friend relation and blocking fields are not served here.

Online id lookup: the account core has no online-id index, so an online id resolves when it is the caller's own or when
this process has already resolved that account by id (cache). Otherwise `404`. A real npId is not plain text on Sony's side; the encoding is unverified.

## `GET /api/v1/devices/accounts/me` (`dms.api.playstation.com`)

`{"accountDevices":[]}`. The real body is not documented in the sources read; there is no device registry yet.

## Account model

Link namespace `ps5`: `SubjectID` = account id, `PublicCode` = online id, state active. Token claims are trusted for the
caller's own account and online id.
