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
