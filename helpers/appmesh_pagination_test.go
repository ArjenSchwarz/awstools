package helpers

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appmesh"
	"github.com/aws/aws-sdk-go-v2/service/appmesh/types"
)

// The App Mesh list helpers originally called ListVirtualRouters,
// ListRoutes, ListVirtualNodes, and ListVirtualServices only once. Meshes
// with enough resources to paginate had their inventory truncated at the
// first page, producing incomplete route maps and false dangling-node
// reports. These regression tests exercise the APIClient-style mock with a
// multi-page response so every helper is forced to walk NextToken.

// mockPaginatedAppMeshClient serves mesh resources from pre-configured
// slices, paginating each List call by page size using NextToken. It
// satisfies AppMeshAPI so the existing helpers can be driven by it.
type mockPaginatedAppMeshClient struct {
	routers  []types.VirtualRouterRef
	routes   []types.RouteRef
	nodes    []types.VirtualNodeRef
	services []types.VirtualServiceRef

	pageSize int

	listRoutersCalls  int
	listRoutesCalls   int
	listNodesCalls    int
	listServicesCalls int
}

func (m *mockPaginatedAppMeshClient) pageBounds(total int, token *string) (int, int, *string) {
	start := 0
	if token != nil {
		fmt.Sscanf(*token, "%d", &start)
	}
	pageSize := m.pageSize
	if pageSize <= 0 {
		pageSize = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	var next *string
	if end < total {
		tok := fmt.Sprintf("%d", end)
		next = &tok
	}
	return start, end, next
}

func (m *mockPaginatedAppMeshClient) ListVirtualRouters(_ context.Context, input *appmesh.ListVirtualRoutersInput, _ ...func(*appmesh.Options)) (*appmesh.ListVirtualRoutersOutput, error) {
	m.listRoutersCalls++
	start, end, next := m.pageBounds(len(m.routers), input.NextToken)
	return &appmesh.ListVirtualRoutersOutput{
		VirtualRouters: m.routers[start:end],
		NextToken:      next,
	}, nil
}

func (m *mockPaginatedAppMeshClient) ListRoutes(_ context.Context, input *appmesh.ListRoutesInput, _ ...func(*appmesh.Options)) (*appmesh.ListRoutesOutput, error) {
	m.listRoutesCalls++
	start, end, next := m.pageBounds(len(m.routes), input.NextToken)
	return &appmesh.ListRoutesOutput{
		Routes:    m.routes[start:end],
		NextToken: next,
	}, nil
}

func (m *mockPaginatedAppMeshClient) DescribeRoute(_ context.Context, params *appmesh.DescribeRouteInput, _ ...func(*appmesh.Options)) (*appmesh.DescribeRouteOutput, error) {
	return &appmesh.DescribeRouteOutput{
		Route: &types.RouteData{
			MeshName:          params.MeshName,
			RouteName:         params.RouteName,
			VirtualRouterName: params.VirtualRouterName,
			Spec:              &types.RouteSpec{},
		},
	}, nil
}

func (m *mockPaginatedAppMeshClient) ListVirtualNodes(_ context.Context, input *appmesh.ListVirtualNodesInput, _ ...func(*appmesh.Options)) (*appmesh.ListVirtualNodesOutput, error) {
	m.listNodesCalls++
	start, end, next := m.pageBounds(len(m.nodes), input.NextToken)
	return &appmesh.ListVirtualNodesOutput{
		VirtualNodes: m.nodes[start:end],
		NextToken:    next,
	}, nil
}

func (m *mockPaginatedAppMeshClient) DescribeVirtualNode(_ context.Context, params *appmesh.DescribeVirtualNodeInput, _ ...func(*appmesh.Options)) (*appmesh.DescribeVirtualNodeOutput, error) {
	return &appmesh.DescribeVirtualNodeOutput{
		VirtualNode: &types.VirtualNodeData{
			MeshName:        params.MeshName,
			VirtualNodeName: params.VirtualNodeName,
			Spec:            &types.VirtualNodeSpec{},
		},
	}, nil
}

func (m *mockPaginatedAppMeshClient) ListVirtualServices(_ context.Context, input *appmesh.ListVirtualServicesInput, _ ...func(*appmesh.Options)) (*appmesh.ListVirtualServicesOutput, error) {
	m.listServicesCalls++
	start, end, next := m.pageBounds(len(m.services), input.NextToken)
	return &appmesh.ListVirtualServicesOutput{
		VirtualServices: m.services[start:end],
		NextToken:       next,
	}, nil
}

func (m *mockPaginatedAppMeshClient) DescribeVirtualService(_ context.Context, params *appmesh.DescribeVirtualServiceInput, _ ...func(*appmesh.Options)) (*appmesh.DescribeVirtualServiceOutput, error) {
	return &appmesh.DescribeVirtualServiceOutput{
		VirtualService: &types.VirtualServiceData{
			MeshName:           params.MeshName,
			VirtualServiceName: params.VirtualServiceName,
			Spec:               &types.VirtualServiceSpec{},
		},
	}, nil
}

func makeRouterRefs(meshName string, n int) []types.VirtualRouterRef {
	refs := make([]types.VirtualRouterRef, n)
	for i := 0; i < n; i++ {
		refs[i] = types.VirtualRouterRef{
			MeshName:          aws.String(meshName),
			VirtualRouterName: aws.String(fmt.Sprintf("router-%03d", i)),
		}
	}
	return refs
}

func makeRouteRefs(meshName, routerName string, n int) []types.RouteRef {
	refs := make([]types.RouteRef, n)
	for i := 0; i < n; i++ {
		refs[i] = types.RouteRef{
			MeshName:          aws.String(meshName),
			VirtualRouterName: aws.String(routerName),
			RouteName:         aws.String(fmt.Sprintf("route-%03d", i)),
		}
	}
	return refs
}

func makeNodeRefs(meshName string, n int) []types.VirtualNodeRef {
	refs := make([]types.VirtualNodeRef, n)
	for i := 0; i < n; i++ {
		refs[i] = types.VirtualNodeRef{
			MeshName:        aws.String(meshName),
			VirtualNodeName: aws.String(fmt.Sprintf("node-%03d", i)),
		}
	}
	return refs
}

func makeServiceRefs(meshName string, n int) []types.VirtualServiceRef {
	refs := make([]types.VirtualServiceRef, n)
	for i := 0; i < n; i++ {
		refs[i] = types.VirtualServiceRef{
			MeshName:           aws.String(meshName),
			VirtualServiceName: aws.String(fmt.Sprintf("service-%03d", i)),
		}
	}
	return refs
}

// TestGetAllAppMeshRoutes_Pagination verifies that both ListVirtualRouters
// and ListRoutes walk every page. Before the fix the helper only issued
// one call per operation, so large meshes saw truncated route lists.
func TestGetAllAppMeshRoutes_Pagination(t *testing.T) {
	meshName := "test-mesh"
	mock := &mockPaginatedAppMeshClient{
		routers:  makeRouterRefs(meshName, 5),
		routes:   makeRouteRefs(meshName, "router-000", 5),
		pageSize: 2, // forces 3 pages per list
	}

	result := getAllAppMeshRoutes(aws.String(meshName), mock)

	// Each of the 5 routers sees the same 5 routes in this fixture, so we
	// expect 25 total routes once pagination is walked fully.
	wantRoutes := 25
	if len(result) != wantRoutes {
		t.Fatalf("getAllAppMeshRoutes returned %d routes, want %d (pagination bug: routes truncated)", len(result), wantRoutes)
	}

	if mock.listRoutersCalls < 3 {
		t.Errorf("ListVirtualRouters called %d times, want at least 3 (pagination bug: NextToken not followed)", mock.listRoutersCalls)
	}
	// ListRoutes is called once per router and each call paginates 3 times.
	if mock.listRoutesCalls < 3*len(mock.routers) {
		t.Errorf("ListRoutes called %d times, want at least %d (pagination bug: NextToken not followed)", mock.listRoutesCalls, 3*len(mock.routers))
	}
}

// TestGetAllAppMeshNodes_Pagination verifies that ListVirtualNodes walks
// every page. Before the fix the helper only returned the first page of
// nodes.
func TestGetAllAppMeshNodes_Pagination(t *testing.T) {
	meshName := "test-mesh"
	total := 7
	mock := &mockPaginatedAppMeshClient{
		nodes:    makeNodeRefs(meshName, total),
		pageSize: 3, // forces 3 pages: [0..2], [3..5], [6]
	}

	result := getAllAppMeshNodes(aws.String(meshName), mock)

	if len(result) != total {
		t.Fatalf("getAllAppMeshNodes returned %d nodes, want %d (pagination bug: nodes truncated)", len(result), total)
	}
	if mock.listNodesCalls < 3 {
		t.Errorf("ListVirtualNodes called %d times, want at least 3 (pagination bug: NextToken not followed)", mock.listNodesCalls)
	}
	for i, node := range result {
		want := fmt.Sprintf("node-%03d", i)
		if aws.ToString(node.VirtualNodeName) != want {
			t.Errorf("result[%d].VirtualNodeName = %q, want %q", i, aws.ToString(node.VirtualNodeName), want)
		}
	}
}

// TestGetAllAppMeshVirtualServices_Pagination verifies that
// ListVirtualServices walks every page. Before the fix only the first page
// of services was returned.
func TestGetAllAppMeshVirtualServices_Pagination(t *testing.T) {
	meshName := "test-mesh"
	total := 5
	mock := &mockPaginatedAppMeshClient{
		services: makeServiceRefs(meshName, total),
		pageSize: 2, // forces 3 pages: [0,1], [2,3], [4]
	}

	result := getAllAppMeshVirtualServices(aws.String(meshName), mock)

	if len(result) != total {
		t.Fatalf("getAllAppMeshVirtualServices returned %d services, want %d (pagination bug: services truncated)", len(result), total)
	}
	if mock.listServicesCalls < 3 {
		t.Errorf("ListVirtualServices called %d times, want at least 3 (pagination bug: NextToken not followed)", mock.listServicesCalls)
	}
	for i, svc := range result {
		want := fmt.Sprintf("service-%03d", i)
		if aws.ToString(svc.VirtualServiceName) != want {
			t.Errorf("result[%d].VirtualServiceName = %q, want %q", i, aws.ToString(svc.VirtualServiceName), want)
		}
	}
}
