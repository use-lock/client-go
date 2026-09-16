# Lock Go Client

Typed Go clients for [Lock](https://github.com/use-lock/lock), generated from its OpenAPI specifications.

## Installation

Requires Go 1.25 or later.

```sh
go get github.com/use-lock/client-go
```

## Packages

| Package | API | Base URL |
| --- | --- | --- |
| `github.com/use-lock/client-go/admin` | Create, update, list, and delete realms | `https://lock.example/api` |
| `github.com/use-lock/client-go/management` | Manage a realm's clients, resources, and social providers | `https://lock.example/api` |
| `github.com/use-lock/client-go/auth` | OAuth, OpenID Connect, user info, and discovery | The realm issuer, such as `https://staging.example` |

The root package provides the shared `APIError` and `Response` types. Both are also available through each API package.

Admin and management requests use the `/api` base path. Their tokens have different audiences: `{issuer}/admin-api` for realm administration and `{issuer}/api` for management operations. Use a token with the audience and scopes required by the operation.

## Quick Start

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/use-lock/client-go/admin"
)

func main() {
    client, err := admin.NewClient("https://lock.example/api",
        admin.WithToken(os.Getenv("LOCK_ADMIN_ACCESS_TOKEN")),
        admin.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
    )
    if err != nil {
        log.Fatal(err)
    }

    realm, err := client.GetRealm(context.Background(), "staging")
    if err != nil {
        log.Fatal(err)
    }

    log.Println(realm.Data.Name)
}
```

Methods return decoded results such as `RealmResponse` or `RealmCollection`. There is no separate `WithResponses` client and no need to inspect `JSON200` fields.

## Management

Use the management client with a token addressed to the Management API. Pass the realm slug to each operation:

```go
client, err := management.NewClient("https://lock.example/api",
    management.WithToken(token),
)
if err != nil {
    return err
}

clients, err := client.ListClients(ctx, "staging", nil)
if err != nil {
    return err
}

for _, client := range clients.Data {
    fmt.Println(client.Name)
}
```

Import `github.com/use-lock/client-go/management` for this example. List methods accept an optional parameter struct for pagination, sorting, and filters; `nil` uses the server defaults.

## Error Handling

An unexpected HTTP status returns an `*APIError`. Use `errors.As` to access its status, headers, raw body, OAuth error code, and validation details:

```go
realm, err := client.GetRealm(ctx, "staging")
if err != nil {
    var apiError *admin.APIError
    if errors.As(err, &apiError) {
        switch apiError.StatusCode {
        case http.StatusNotFound:
            fmt.Println("Realm not found")
        case http.StatusUnprocessableEntity:
            fmt.Println(apiError.Errors)
        default:
            fmt.Println(apiError.Error())
        }
    }
    return err
}

fmt.Println(realm.Data.Name)
```

Network failures, context cancellation, and invalid success bodies also return errors. `errors.Is(err, context.Canceled)` works for canceled requests.

## OAuth and Discovery

Create an auth client with the realm issuer as its base URL:

```go
client, err := auth.NewClient("https://staging.example")
if err != nil {
    return err
}

metadata, err := client.GetProviderMetadata(ctx)
if err != nil {
    return err
}

fmt.Println(metadata.Issuer)
```

Import `github.com/use-lock/client-go/auth` for this example. The package also provides `IssueToken`, `IntrospectToken`, `RevokeToken`, and `GetUserInfo`. See the [token request example](auth/client_test.go) for constructing a client-credentials request.

Browser operations such as `Authorize` and `Logout` return a `Response` containing `StatusCode`, `Header`, and `Body`, since they may return HTML or redirects. The default HTTP client does not follow redirects. A client supplied through `WithHTTPClient` uses its own redirect policy.

## Configuration

- `WithToken(token)` adds a bearer token. It does not obtain or refresh tokens.
- `WithHTTPClient(client)` accepts an `*http.Client` or another implementation of `Do(*http.Request) (*http.Response, error)`. Use it to configure timeouts and transport behavior. The default client has no timeout.
- `WithRequestEditorFn(fn)` modifies requests before they are sent, for example to add authentication headers. Individual methods also accept request editors after their normal arguments.

For partial updates, pointer fields represent optional values. Nullable fields use [`nullable`](https://github.com/oapi-codegen/nullable) to distinguish an omitted value from an explicit JSON `null`. See the [patch example](management/client_test.go).

## Development

```sh
make check       # Go vet, race tests, and formatting checks
make generate    # Regenerate all three clients from the committed specifications
make sync-openapi # Copy the group exports from ../lock and regenerate
```

To use another checkout of Lock:

```sh
make sync-openapi OPENAPI_DIR=/path/to/lock
```

Public operation names live in `openapi/operations.json`. The generator in `internal/generate` prepares named response schemas and applies the templates in `openapi/templates`. Change those sources instead of editing `client.gen.go` files.

## License

[MIT](LICENSE). The adapted oapi-codegen templates retain their [Apache-2.0 license](openapi/templates/LICENSE).
