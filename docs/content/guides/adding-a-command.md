---
title: "Add a command"
description: "Model a new eBay record and expose it as a command, a route, and a tool at once."
weight: 10
---

A new surface is two pieces of work: a typed record with a client method that
fetches it, and an operation that declares it. Every surface then updates itself.

## 1. Model the record

In `ebay/types.go`, add a struct for the thing you are fetching, and in its own
file (for example `ebay/review.go`) a client method that returns it. The `kit`
struct tags decide how a host addresses the record:

```go
type Review struct {
    ID     string  `json:"id"     kit:"id"`             // the URI id
    Title  string  `json:"title"`
    Body   string  `json:"body"   kit:"body"`           // what cat and Markdown print
    Rating float64 `json:"rating,omitempty"`
    Seller string  `json:"seller" kit:"link,kind=ebay/seller"` // an edge to another record
}

func (c *Client) GetReview(ctx context.Context, id string) (*Review, error) {
    body, err := c.get(ctx, c.BaseURL+"/rvw/"+id)
    if err != nil {
        return nil, err
    }
    // parse body into a Review ...
    return r, nil
}
```

- `kit:"id"` marks the field that becomes the URI id.
- `kit:"body"` marks the prose that `cat` and the Markdown export render.
- `kit:"link,kind=<scheme>/<type>"` marks an outbound edge. It can point at
  another eBay type or at another site entirely, which is what lets a host walk
  the graph across tools.

## 2. Declare the operation

The handler goes in `ebay/ops.go` with its input struct, and the registration
goes in `ebay/domain.go` inside `Register`:

```go
// in ops.go
type reviewRef struct {
    ID     string  `kit:"arg" help:"review id or URL"`
    Client *Client `kit:"inject"`
}

func getReview(ctx context.Context, in reviewRef, emit func(*Review) error) error {
    r, err := in.Client.GetReview(ctx, in.ID)
    if err != nil {
        return mapErr(err)
    }
    return emit(r)
}

// in domain.go, inside Register(app):
kit.Handle(app, kit.OpMeta{
    Name: "review", Group: "read", Single: true,
    Summary: "Show one review by id",
    URIType: "review", Resolver: true,
    Args: []kit.Arg{{Name: "id", Help: "review id or URL"}},
}, getReview)
```

That is the whole change. `kit.Handle` reflects the input for flags and the
output for the record shape, so the operation immediately becomes:

```bash
ebay review <id>                          # the command
curl 'localhost:7777/v1/review/<id>'      # the route, under serve
ant get ebay://review/<id>                # the URI dereference, via a host
```

If the new type is a resolver, add it to `URLFor` and `Classify` in
`ebay/ids.go` so `ref url` and `ant resolve` can round-trip it offline.

## Resolver ops and list ops

Two flags shape how a host treats an operation:

- **`Single: true`** with **`Resolver: true`** marks the canonical one-record
  fetch for a `URIType`. It answers `ant get`. The `item`, `seller show`, and
  `category show` ops are resolvers.
- **`List: true`** marks a member-lister for a parent resource. It answers
  `ant ls`. A list op emits records that are themselves addressable, so every
  member is a URI a host can follow. `category browse`, `category tree`,
  `seller listings`, `deals`, and `search` are list ops.

## Map errors to exit codes

Return the library sentinels and let `mapErr` translate them, so every surface
reports the same outcome with the same exit code:

```go
func mapErr(err error) error {
    switch {
    case errors.Is(err, ErrNotFound):
        return errs.NotFound("%s", err.Error())
    case errors.Is(err, ErrRateLimited):
        return errs.RateLimited("%s", err.Error())
    case errors.Is(err, ErrBlocked):
        return errs.NeedAuth("%s", err.Error())
    default:
        return err
    }
}
```

`ErrBlocked` is the bot wall, which becomes need-auth (exit 4). See
[output formats](/reference/output/) for how records render, and
[resource URIs](/guides/resource-uris/) for how a host addresses them.
