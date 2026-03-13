package helpers

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appmesh"
	"github.com/aws/aws-sdk-go-v2/service/appmesh/types"
)

// mockAppMeshClient implements AppMeshAPI for testing.
type mockAppMeshClient struct {
	listVirtualRoutersFunc     func(ctx context.Context, params *appmesh.ListVirtualRoutersInput, optFns ...func(*appmesh.Options)) (*appmesh.ListVirtualRoutersOutput, error)
	listRoutesFunc             func(ctx context.Context, params *appmesh.ListRoutesInput, optFns ...func(*appmesh.Options)) (*appmesh.ListRoutesOutput, error)
	describeRouteFunc          func(ctx context.Context, params *appmesh.DescribeRouteInput, optFns ...func(*appmesh.Options)) (*appmesh.DescribeRouteOutput, error)
	listVirtualNodesFunc       func(ctx context.Context, params *appmesh.ListVirtualNodesInput, optFns ...func(*appmesh.Options)) (*appmesh.ListVirtualNodesOutput, error)
	describeVirtualNodeFunc    func(ctx context.Context, params *appmesh.DescribeVirtualNodeInput, optFns ...func(*appmesh.Options)) (*appmesh.DescribeVirtualNodeOutput, error)
	listVirtualServicesFunc    func(ctx context.Context, params *appmesh.ListVirtualServicesInput, optFns ...func(*appmesh.Options)) (*appmesh.ListVirtualServicesOutput, error)
	describeVirtualServiceFunc func(ctx context.Context, params *appmesh.DescribeVirtualServiceInput, optFns ...func(*appmesh.Options)) (*appmesh.DescribeVirtualServiceOutput, error)
}

func (m *mockAppMeshClient) ListVirtualRouters(ctx context.Context, params *appmesh.ListVirtualRoutersInput, optFns ...func(*appmesh.Options)) (*appmesh.ListVirtualRoutersOutput, error) {
	return m.listVirtualRoutersFunc(ctx, params, optFns...)
}

func (m *mockAppMeshClient) ListRoutes(ctx context.Context, params *appmesh.ListRoutesInput, optFns ...func(*appmesh.Options)) (*appmesh.ListRoutesOutput, error) {
	return m.listRoutesFunc(ctx, params, optFns...)
}

func (m *mockAppMeshClient) DescribeRoute(ctx context.Context, params *appmesh.DescribeRouteInput, optFns ...func(*appmesh.Options)) (*appmesh.DescribeRouteOutput, error) {
	return m.describeRouteFunc(ctx, params, optFns...)
}

func (m *mockAppMeshClient) ListVirtualNodes(ctx context.Context, params *appmesh.ListVirtualNodesInput, optFns ...func(*appmesh.Options)) (*appmesh.ListVirtualNodesOutput, error) {
	return m.listVirtualNodesFunc(ctx, params, optFns...)
}

func (m *mockAppMeshClient) DescribeVirtualNode(ctx context.Context, params *appmesh.DescribeVirtualNodeInput, optFns ...func(*appmesh.Options)) (*appmesh.DescribeVirtualNodeOutput, error) {
	return m.describeVirtualNodeFunc(ctx, params, optFns...)
}

func (m *mockAppMeshClient) ListVirtualServices(ctx context.Context, params *appmesh.ListVirtualServicesInput, optFns ...func(*appmesh.Options)) (*appmesh.ListVirtualServicesOutput, error) {
	return m.listVirtualServicesFunc(ctx, params, optFns...)
}

func (m *mockAppMeshClient) DescribeVirtualService(ctx context.Context, params *appmesh.DescribeVirtualServiceInput, optFns ...func(*appmesh.Options)) (*appmesh.DescribeVirtualServiceOutput, error) {
	return m.describeVirtualServiceFunc(ctx, params, optFns...)
}

func TestAppMeshRouteRef_Struct(t *testing.T) {
	route := types.RouteRef{
		Arn:               aws.String("arn:aws:appmesh:us-west-2:123456789012:mesh/my-mesh/virtualRouter/my-router/route/my-route"),
		MeshName:          aws.String("my-mesh"),
		RouteName:         aws.String("my-route"),
		VirtualRouterName: aws.String("my-router"),
	}

	if aws.ToString(route.MeshName) != "my-mesh" {
		t.Errorf("Expected MeshName to be 'my-mesh', got %s", aws.ToString(route.MeshName))
	}

	if aws.ToString(route.RouteName) != "my-route" {
		t.Errorf("Expected RouteName to be 'my-route', got %s", aws.ToString(route.RouteName))
	}

	if aws.ToString(route.VirtualRouterName) != "my-router" {
		t.Errorf("Expected VirtualRouterName to be 'my-router', got %s", aws.ToString(route.VirtualRouterName))
	}

	expectedArn := "arn:aws:appmesh:us-west-2:123456789012:mesh/my-mesh/virtualRouter/my-router/route/my-route"
	if aws.ToString(route.Arn) != expectedArn {
		t.Errorf("Expected Arn to be '%s', got %s", expectedArn, aws.ToString(route.Arn))
	}
}

func TestAppMeshVirtualRouterRef_Struct(t *testing.T) {
	router := types.VirtualRouterRef{
		Arn:               aws.String("arn:aws:appmesh:us-west-2:123456789012:mesh/my-mesh/virtualRouter/my-router"),
		MeshName:          aws.String("my-mesh"),
		VirtualRouterName: aws.String("my-router"),
	}

	if aws.ToString(router.MeshName) != "my-mesh" {
		t.Errorf("Expected MeshName to be 'my-mesh', got %s", aws.ToString(router.MeshName))
	}

	if aws.ToString(router.VirtualRouterName) != "my-router" {
		t.Errorf("Expected VirtualRouterName to be 'my-router', got %s", aws.ToString(router.VirtualRouterName))
	}

	expectedArn := "arn:aws:appmesh:us-west-2:123456789012:mesh/my-mesh/virtualRouter/my-router"
	if aws.ToString(router.Arn) != expectedArn {
		t.Errorf("Expected Arn to be '%s', got %s", expectedArn, aws.ToString(router.Arn))
	}
}

func TestAppMeshRouteData_Structure(t *testing.T) {
	// Test that RouteData can be instantiated
	routeData := &types.RouteData{
		MeshName:  aws.String("test-mesh"),
		RouteName: aws.String("test-route"),
	}

	if aws.ToString(routeData.MeshName) != "test-mesh" {
		t.Errorf("Expected MeshName to be 'test-mesh', got %s", aws.ToString(routeData.MeshName))
	}

	if aws.ToString(routeData.RouteName) != "test-route" {
		t.Errorf("Expected RouteName to be 'test-route', got %s", aws.ToString(routeData.RouteName))
	}
}

// --- Regression tests for nil dereference on API errors (T-346) ---

func TestGetAllAppMeshRoutes_ListVirtualRoutersError_NoPanic(t *testing.T) {
	mock := &mockAppMeshClient{
		listVirtualRoutersFunc: func(_ context.Context, _ *appmesh.ListVirtualRoutersInput, _ ...func(*appmesh.Options)) (*appmesh.ListVirtualRoutersOutput, error) {
			return nil, fmt.Errorf("AccessDenied: not authorized")
		},
	}

	meshName := aws.String("test-mesh")
	result := getAllAppMeshRoutes(meshName, mock)
	if result != nil {
		t.Errorf("Expected nil result on ListVirtualRouters error, got %v", result)
	}
}

// Test that buildRoutesHolder correctly processes HTTP routes
func TestBuildRoutesHolder_HttpRoute(t *testing.T) {
	routes := []*types.RouteData{
		{
			VirtualRouterName: aws.String("http-router"),
			Spec: &types.RouteSpec{
				HttpRoute: &types.HttpRoute{
					Action: &types.HttpRouteAction{
						WeightedTargets: []types.WeightedTarget{
							{VirtualNode: aws.String("node-a"), Weight: 100},
						},
					},
					Match: &types.HttpRouteMatch{
						Prefix: aws.String("/api"),
					},
				},
			},
		},
	}

	result := buildRoutesHolder(routes)

	if len(result["http-router"]) != 1 {
		t.Fatalf("Expected 1 route for http-router, got %d", len(result["http-router"]))
	}
	r := result["http-router"][0]
	if r.Path != "/api" {
		t.Errorf("Expected Path '/api', got '%s'", r.Path)
	}
	if r.DestinationNode != "node-a" {
		t.Errorf("Expected DestinationNode 'node-a', got '%s'", r.DestinationNode)
	}
	if r.Weight != 100 {
		t.Errorf("Expected Weight 100, got %d", r.Weight)
	}
}

// Regression test: GRPC routes must not panic (bug T-345)
func TestBuildRoutesHolder_GrpcRoute_NoPanic(t *testing.T) {
	routes := []*types.RouteData{
		{
			VirtualRouterName: aws.String("grpc-router"),
			Spec: &types.RouteSpec{
				GrpcRoute: &types.GrpcRoute{
					Action: &types.GrpcRouteAction{
						WeightedTargets: []types.WeightedTarget{
							{VirtualNode: aws.String("grpc-node"), Weight: 50},
						},
					},
					Match: &types.GrpcRouteMatch{
						ServiceName: aws.String("my.grpc.Service"),
						MethodName:  aws.String("DoStuff"),
					},
				},
			},
		},
	}

	result := buildRoutesHolder(routes)

	if len(result["grpc-router"]) != 1 {
		t.Fatalf("Expected 1 route for grpc-router, got %d", len(result["grpc-router"]))
	}
	r := result["grpc-router"][0]
	if r.DestinationNode != "grpc-node" {
		t.Errorf("Expected DestinationNode 'grpc-node', got '%s'", r.DestinationNode)
	}
	if r.Weight != 50 {
		t.Errorf("Expected Weight 50, got %d", r.Weight)
	}
	if r.Path != "my.grpc.Service/DoStuff" {
		t.Errorf("Expected Path 'my.grpc.Service/DoStuff', got '%s'", r.Path)
	}
}

// Regression test: TCP routes must not panic (bug T-345)
func TestBuildRoutesHolder_TcpRoute_NoPanic(t *testing.T) {
	routes := []*types.RouteData{
		{
			VirtualRouterName: aws.String("tcp-router"),
			Spec: &types.RouteSpec{
				TcpRoute: &types.TcpRoute{
					Action: &types.TcpRouteAction{
						WeightedTargets: []types.WeightedTarget{
							{VirtualNode: aws.String("tcp-node"), Weight: 100},
						},
					},
				},
			},
		},
	}

	result := buildRoutesHolder(routes)

	if len(result["tcp-router"]) != 1 {
		t.Fatalf("Expected 1 route for tcp-router, got %d", len(result["tcp-router"]))
	}
	r := result["tcp-router"][0]
	if r.DestinationNode != "tcp-node" {
		t.Errorf("Expected DestinationNode 'tcp-node', got '%s'", r.DestinationNode)
	}
}

// Regression test: HTTP2 routes must not panic (bug T-345)
func TestBuildRoutesHolder_Http2Route_NoPanic(t *testing.T) {
	routes := []*types.RouteData{
		{
			VirtualRouterName: aws.String("http2-router"),
			Spec: &types.RouteSpec{
				Http2Route: &types.HttpRoute{
					Action: &types.HttpRouteAction{
						WeightedTargets: []types.WeightedTarget{
							{VirtualNode: aws.String("http2-node"), Weight: 75},
						},
					},
					Match: &types.HttpRouteMatch{
						Prefix: aws.String("/v2"),
					},
				},
			},
		},
	}

	result := buildRoutesHolder(routes)

	if len(result["http2-router"]) != 1 {
		t.Fatalf("Expected 1 route for http2-router, got %d", len(result["http2-router"]))
	}
	r := result["http2-router"][0]
	if r.Path != "/v2" {
		t.Errorf("Expected Path '/v2', got '%s'", r.Path)
	}
}

// Regression test: mixed route types must all be processed (bug T-345)
func TestBuildRoutesHolder_MixedRouteTypes(t *testing.T) {
	routes := []*types.RouteData{
		{
			VirtualRouterName: aws.String("router-1"),
			Spec: &types.RouteSpec{
				HttpRoute: &types.HttpRoute{
					Action: &types.HttpRouteAction{
						WeightedTargets: []types.WeightedTarget{
							{VirtualNode: aws.String("http-node"), Weight: 100},
						},
					},
					Match: &types.HttpRouteMatch{Prefix: aws.String("/http")},
				},
			},
		},
		{
			VirtualRouterName: aws.String("router-2"),
			Spec: &types.RouteSpec{
				GrpcRoute: &types.GrpcRoute{
					Action: &types.GrpcRouteAction{
						WeightedTargets: []types.WeightedTarget{
							{VirtualNode: aws.String("grpc-node"), Weight: 100},
						},
					},
					Match: &types.GrpcRouteMatch{ServiceName: aws.String("svc")},
				},
			},
		},
		{
			VirtualRouterName: aws.String("router-3"),
			Spec: &types.RouteSpec{
				TcpRoute: &types.TcpRoute{
					Action: &types.TcpRouteAction{
						WeightedTargets: []types.WeightedTarget{
							{VirtualNode: aws.String("tcp-node"), Weight: 100},
						},
					},
				},
			},
		},
	}

	result := buildRoutesHolder(routes)

	if len(result) != 3 {
		t.Errorf("Expected 3 routers in result, got %d", len(result))
	}
	if len(result["router-1"]) != 1 {
		t.Errorf("Expected 1 route for router-1, got %d", len(result["router-1"]))
	}
	if len(result["router-2"]) != 1 {
		t.Errorf("Expected 1 route for router-2, got %d", len(result["router-2"]))
	}
	if len(result["router-3"]) != 1 {
		t.Errorf("Expected 1 route for router-3, got %d", len(result["router-3"]))
	}
}

// Regression test: route with nil spec fields must not panic (bug T-345)
func TestBuildRoutesHolder_NilRouteSpec_NoPanic(t *testing.T) {
	routes := []*types.RouteData{
		{
			VirtualRouterName: aws.String("empty-router"),
			Spec:              &types.RouteSpec{},
		},
	}

	result := buildRoutesHolder(routes)

	if len(result["empty-router"]) != 0 {
		t.Errorf("Expected 0 routes for empty-router, got %d", len(result["empty-router"]))
	}
}
}

func TestGetAllAppMeshRoutes_ListRoutesError_NoPanic(t *testing.T) {
	mock := &mockAppMeshClient{
		listVirtualRoutersFunc: func(_ context.Context, _ *appmesh.ListVirtualRoutersInput, _ ...func(*appmesh.Options)) (*appmesh.ListVirtualRoutersOutput, error) {
			return &appmesh.ListVirtualRoutersOutput{
				VirtualRouters: []types.VirtualRouterRef{
					{MeshName: aws.String("test-mesh"), VirtualRouterName: aws.String("router-1")},
				},
			}, nil
		},
		listRoutesFunc: func(_ context.Context, _ *appmesh.ListRoutesInput, _ ...func(*appmesh.Options)) (*appmesh.ListRoutesOutput, error) {
			return nil, fmt.Errorf("NotFound: router not found")
		},
	}

	meshName := aws.String("test-mesh")
	result := getAllAppMeshRoutes(meshName, mock)
	if len(result) != 0 {
		t.Errorf("Expected empty result on ListRoutes error, got %d items", len(result))
	}
}

func TestGetAppMeshRouteDescriptions_DescribeRouteError_NoPanic(t *testing.T) {
	mock := &mockAppMeshClient{
		listVirtualRoutersFunc: func(_ context.Context, _ *appmesh.ListVirtualRoutersInput, _ ...func(*appmesh.Options)) (*appmesh.ListVirtualRoutersOutput, error) {
			return &appmesh.ListVirtualRoutersOutput{
				VirtualRouters: []types.VirtualRouterRef{
					{MeshName: aws.String("test-mesh"), VirtualRouterName: aws.String("router-1")},
				},
			}, nil
		},
		listRoutesFunc: func(_ context.Context, _ *appmesh.ListRoutesInput, _ ...func(*appmesh.Options)) (*appmesh.ListRoutesOutput, error) {
			return &appmesh.ListRoutesOutput{
				Routes: []types.RouteRef{
					{MeshName: aws.String("test-mesh"), RouteName: aws.String("route-1"), VirtualRouterName: aws.String("router-1")},
				},
			}, nil
		},
		describeRouteFunc: func(_ context.Context, _ *appmesh.DescribeRouteInput, _ ...func(*appmesh.Options)) (*appmesh.DescribeRouteOutput, error) {
			return nil, fmt.Errorf("AccessDenied: not authorized")
		},
	}

	meshName := aws.String("test-mesh")
	result := getAppMeshRouteDescriptions(meshName, mock)
	if len(result) != 0 {
		t.Errorf("Expected empty result on DescribeRoute error, got %d items", len(result))
	}
}

func TestGetAllAppMeshNodes_ListVirtualNodesError_NoPanic(t *testing.T) {
	mock := &mockAppMeshClient{
		listVirtualNodesFunc: func(_ context.Context, _ *appmesh.ListVirtualNodesInput, _ ...func(*appmesh.Options)) (*appmesh.ListVirtualNodesOutput, error) {
			return nil, fmt.Errorf("AccessDenied: not authorized")
		},
	}

	meshName := aws.String("test-mesh")
	result := getAllAppMeshNodes(meshName, mock)
	if result != nil {
		t.Errorf("Expected nil result on ListVirtualNodes error, got %v", result)
	}
}

func TestGetAllAppMeshNodes_DescribeVirtualNodeError_NoPanic(t *testing.T) {
	mock := &mockAppMeshClient{
		listVirtualNodesFunc: func(_ context.Context, _ *appmesh.ListVirtualNodesInput, _ ...func(*appmesh.Options)) (*appmesh.ListVirtualNodesOutput, error) {
			return &appmesh.ListVirtualNodesOutput{
				VirtualNodes: []types.VirtualNodeRef{
					{MeshName: aws.String("test-mesh"), VirtualNodeName: aws.String("node-1")},
				},
			}, nil
		},
		describeVirtualNodeFunc: func(_ context.Context, _ *appmesh.DescribeVirtualNodeInput, _ ...func(*appmesh.Options)) (*appmesh.DescribeVirtualNodeOutput, error) {
			return nil, fmt.Errorf("NotFound: node not found")
		},
	}

	meshName := aws.String("test-mesh")
	result := getAllAppMeshNodes(meshName, mock)
	if len(result) != 0 {
		t.Errorf("Expected empty result on DescribeVirtualNode error, got %d items", len(result))
	}
}

func TestGetAllAppMeshVirtualServices_ListError_NoPanic(t *testing.T) {
	mock := &mockAppMeshClient{
		listVirtualServicesFunc: func(_ context.Context, _ *appmesh.ListVirtualServicesInput, _ ...func(*appmesh.Options)) (*appmesh.ListVirtualServicesOutput, error) {
			return nil, fmt.Errorf("AccessDenied: not authorized")
		},
	}

	meshName := aws.String("test-mesh")
	result := getAllAppMeshVirtualServices(meshName, mock)
	if result != nil {
		t.Errorf("Expected nil result on ListVirtualServices error, got %v", result)
	}
}

func TestGetAllAppMeshVirtualServices_DescribeError_NoPanic(t *testing.T) {
	mock := &mockAppMeshClient{
		listVirtualServicesFunc: func(_ context.Context, _ *appmesh.ListVirtualServicesInput, _ ...func(*appmesh.Options)) (*appmesh.ListVirtualServicesOutput, error) {
			return &appmesh.ListVirtualServicesOutput{
				VirtualServices: []types.VirtualServiceRef{
					{MeshName: aws.String("test-mesh"), VirtualServiceName: aws.String("svc-1")},
				},
			}, nil
		},
		describeVirtualServiceFunc: func(_ context.Context, _ *appmesh.DescribeVirtualServiceInput, _ ...func(*appmesh.Options)) (*appmesh.DescribeVirtualServiceOutput, error) {
			return nil, fmt.Errorf("NotFound: service not found")
		},
	}

	meshName := aws.String("test-mesh")
	result := getAllAppMeshVirtualServices(meshName, mock)
	if len(result) != 0 {
		t.Errorf("Expected empty result on DescribeVirtualService error, got %d items", len(result))
	}
}

func TestGetAppMeshVirtualNodeBackendServices2_Error_NoPanic(t *testing.T) {
	mock := &mockAppMeshClient{
		describeVirtualNodeFunc: func(_ context.Context, _ *appmesh.DescribeVirtualNodeInput, _ ...func(*appmesh.Options)) (*appmesh.DescribeVirtualNodeOutput, error) {
			return nil, fmt.Errorf("AccessDenied: not authorized")
		},
	}

	meshName := aws.String("test-mesh")
	nodeName := aws.String("node-1")
	result := getAppMeshVirtualNodeBackendServices2(meshName, nodeName, mock)
	if result != nil {
		t.Errorf("Expected nil result on DescribeVirtualNode error, got %v", result)
	}
}

func TestGetAppMeshRouteDescriptions_PartialError_SkipsFailed(t *testing.T) {
	callCount := 0
	mock := &mockAppMeshClient{
		listVirtualRoutersFunc: func(_ context.Context, _ *appmesh.ListVirtualRoutersInput, _ ...func(*appmesh.Options)) (*appmesh.ListVirtualRoutersOutput, error) {
			return &appmesh.ListVirtualRoutersOutput{
				VirtualRouters: []types.VirtualRouterRef{
					{MeshName: aws.String("test-mesh"), VirtualRouterName: aws.String("router-1")},
				},
			}, nil
		},
		listRoutesFunc: func(_ context.Context, _ *appmesh.ListRoutesInput, _ ...func(*appmesh.Options)) (*appmesh.ListRoutesOutput, error) {
			return &appmesh.ListRoutesOutput{
				Routes: []types.RouteRef{
					{MeshName: aws.String("test-mesh"), RouteName: aws.String("route-ok"), VirtualRouterName: aws.String("router-1")},
					{MeshName: aws.String("test-mesh"), RouteName: aws.String("route-fail"), VirtualRouterName: aws.String("router-1")},
				},
			}, nil
		},
		describeRouteFunc: func(_ context.Context, params *appmesh.DescribeRouteInput, _ ...func(*appmesh.Options)) (*appmesh.DescribeRouteOutput, error) {
			callCount++
			if aws.ToString(params.RouteName) == "route-fail" {
				return nil, fmt.Errorf("NotFound")
			}
			return &appmesh.DescribeRouteOutput{
				Route: &types.RouteData{
					MeshName:  params.MeshName,
					RouteName: params.RouteName,
				},
			}, nil
		},
	}

	meshName := aws.String("test-mesh")
	result := getAppMeshRouteDescriptions(meshName, mock)
	if len(result) != 1 {
		t.Errorf("Expected 1 result (skipping failed route), got %d", len(result))
	}
	if callCount != 2 {
		t.Errorf("Expected DescribeRoute to be called 2 times, got %d", callCount)
	}
}
