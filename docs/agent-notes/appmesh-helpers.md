# App Mesh Helpers

Notes on `helpers/appmesh.go` and its test files.

## Architecture

- All list helpers take `AppMeshAPI`, a narrow interface declared at the
  top of `helpers/appmesh.go`. `*appmesh.Client` satisfies it. Tests
  supply a mock.
- `AppMeshAPI` intentionally declares every List/Describe method with the
  same signature the AWS SDK v2 uses, which means the interface is also
  compatible with the SDK's per-operation `ListXxxAPIClient` interfaces
  used by paginators.

## Pagination

Four list operations are used and all are paginated (as of T-758):

- `ListVirtualRouters` via `NewListVirtualRoutersPaginator`
- `ListRoutes` via `NewListRoutesPaginator` (one paginator per router)
- `ListVirtualNodes` via `NewListVirtualNodesPaginator`
- `ListVirtualServices` via `NewListVirtualServicesPaginator`

`ListMeshes` is not currently used anywhere in the codebase — every
command takes the mesh name as a CLI argument. If that changes, follow
the same paginator pattern with `NewListMeshesPaginator`.

## Error handling

- List-page errors return `nil` from the helper so callers treat the
  mesh as unavailable. Describe-per-item errors are logged and the item
  is skipped (`continue`), matching the pre-pagination behaviour.
- There is no `context` plumbing in these helpers — they use
  `context.TODO()` consistent with the rest of `helpers/`.

## Tests

- `helpers/appmesh_test.go` — covers nil/missing-field handling for
  route spec variants (HTTP, HTTP/2, gRPC, TCP) and API error paths.
- `helpers/appmesh_pagination_test.go` — covers multi-page responses
  using `mockPaginatedAppMeshClient`, an `AppMeshAPI` mock that slices
  pre-seeded fixtures by `NextToken`. Use this mock for any future
  regression test that needs multi-page behaviour.

## Gotchas

- `GetAllUnservicedAppMeshNodes` is highly sensitive to pagination
  correctness: it classifies a node as "dangling" whenever it is not
  referenced by any route. Incomplete pagination on either routes or
  nodes produces false positives. Before T-758's fix, large meshes would
  mis-report nodes that were only referenced by routes on later pages.
- `getAllAppMeshRoutes` iterates routers and then paginates routes for
  each router. A failure on one router's routes is logged and the loop
  continues with the next router — partial results are better than
  aborting the whole traversal.
