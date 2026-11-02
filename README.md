# PS5 Profile Server

OpenPak's PS5 Profile Server.

| | |
| --- | --- |
| Network | PlayStation 5 |
| Role | User profile slice of the PSN web API: profiles by account id, legacy profile2, device list |
| Used by | PlayStation 5 system software, PlayStation app clients |
| Image | `ghcr.io/openpak/ps5-profile` |
| Licence | AGPL-3.0-only |

## What it serves

| Endpoint | What |
| --- | --- |
| `GET /api/userProfile/v1/internal/users/me/profiles` (`m.np.playstation.com`) | The caller's profile |
| `GET /api/userProfile/v1/internal/users/{accountId}/profiles` (`m.np.playstation.com`) | A profile by account id |
| `GET /api/v1/devices/accounts/me` (`dms.api.playstation.com`) | Device list (empty) |
| `GET /userProfile/v1/users/{onlineId}/profile2` (`us-prof.np.community.playstation.net`) | Legacy profile |

All but `/healthz` need a bearer token from ps5-auth.

## Run

```sh
PS5_TOKEN_KEY=$(head -c 32 /dev/zero | tr '\0' k) go run ./cmd/ps5-profile
GOWORK=off go test ./...
```

## Docs

- [Protocol](docs/protocol.md)
- [Deploy](docs/deploy.md)

## Licence

AGPL-3.0-only. See [LICENSE](LICENSE).
