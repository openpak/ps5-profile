# Deploy: ps5-profile

`m.np.playstation.com` is one host shared by profile, friends, trophy and messaging. DNS cannot split it, so Traefik
routes by path prefix:

| Router rule | Service |
|---|---|
| ``Host(`m.np.playstation.com`) && PathPrefix(`/api/userProfile`)`` | ps5-profile `:8080` |
| ``Host(`dms.api.playstation.com`)`` | ps5-profile `:8080` |
| ``Host(`us-prof.np.community.playstation.net`)`` | ps5-profile `:8080` |

Other `/api/...` prefixes of `m.np.playstation.com` belong to ps5-friends, ps5-trophy and ps5-messaging. Note the
friends list lives under `/api/userProfile/v1/internal/users/{id}/friends` and `.../blocks` and `.../basicPresences`:
give ps5-friends a longer, more specific PathPrefix (or a regex rule) so Traefik's rule length priority sends those to it.
This service does not serve them.

## Run

```sh
cp example.env .env
set -a; source .env; set +a
go run ./cmd/ps5-profile
GOWORK=off go test ./... && GOWORK=off go vet ./...
docker build -t ps5-profile .
```

## Configuration

| Variable | Required | Default | Meaning |
|---|---|---|---|
| `PS5_ADDR` | no | `:8080` | listen address (plain HTTP behind Traefik) |
| `PS5_TOKEN_KEY` | yes | | HMAC key shared with ps5-auth, 32 bytes or more |
| `PS5_ACCOUNT_CORE_URL`, `PS5_ACCOUNT_CORE_TOKEN` | yes | | account core; both. Without them the service exits unless `PS5_DEV_MEMORY=1` |
| `PS5_DEV_MEMORY` | no | | `1`: in-memory accounts (local development only) |
| `PS5_PROFILE_AVATAR_URL` | no | | base URL of avatar images |
| `PS5_LOG_LEVEL` | no | `info` | read by ps5-common/config |

## Release

Tag `v*.*.*` publishes `ghcr.io/openpak/ps5-profile`. After changing ps5-common run `go mod vendor` here and commit.
