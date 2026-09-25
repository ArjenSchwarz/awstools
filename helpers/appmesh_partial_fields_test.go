package helpers

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appmesh"
	"github.com/aws/aws-sdk-go-v2/service/appmesh/types"
)

// T-1634: a partial route can have a recognized route type but no Action.
// It must be skipped while other routes from the same router remain visible.
func TestBuildRoutesHolder_MissingNestedFields(t *testing.T) {
	valid := &types.RouteData{
		VirtualRouterName: aws.String("router"),
		Spec: &types.RouteSpec{TcpRoute: &types.TcpRoute{
			Action: &types.TcpRouteAction{WeightedTargets: []types.WeightedTarget{{VirtualNode: aws.String("healthy")}}},
		}},
	}
	tests := []struct {
		name        string
		spec        *types.RouteSpec
		wantPartial bool
	}{
		{"HTTP action", &types.RouteSpec{HttpRoute: &types.HttpRoute{Match: &types.HttpRouteMatch{}}}, false},
		{"HTTP2 action", &types.RouteSpec{Http2Route: &types.HttpRoute{Match: &types.HttpRouteMatch{}}}, false},
		{"gRPC action", &types.RouteSpec{GrpcRoute: &types.GrpcRoute{Match: &types.GrpcRouteMatch{}}}, false},
		{"TCP action", &types.RouteSpec{TcpRoute: &types.TcpRoute{}}, false},
		{"HTTP match", &types.RouteSpec{HttpRoute: &types.HttpRoute{Action: &types.HttpRouteAction{WeightedTargets: []types.WeightedTarget{{VirtualNode: aws.String("partial")}}}}}, true},
		{"HTTP2 match", &types.RouteSpec{Http2Route: &types.HttpRoute{Action: &types.HttpRouteAction{WeightedTargets: []types.WeightedTarget{{VirtualNode: aws.String("partial")}}}}}, true},
		{"gRPC match", &types.RouteSpec{GrpcRoute: &types.GrpcRoute{Action: &types.GrpcRouteAction{WeightedTargets: []types.WeightedTarget{{VirtualNode: aws.String("partial")}}}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("buildRoutesHolder panicked on missing nested field: %v", p)
				}
			}()
			routes := []*types.RouteData{{VirtualRouterName: aws.String("router"), Spec: tt.spec}, valid}
			got := buildRoutesHolder(routes)["router"]
			wantCount := 1
			if tt.wantPartial {
				wantCount = 2
			}
			if len(got) != wantCount {
				t.Fatalf("got %d routes, want %d", len(got), wantCount)
			}
			if got[len(got)-1].DestinationNode != "healthy" {
				t.Errorf("valid route lost: %+v", got)
			}
			if tt.wantPartial && (got[0].DestinationNode != "partial" || got[0].Path != "") {
				t.Errorf("partial match route = %+v, want destination partial and empty path", got[0])
			}
		})
	}
}

// T-1634: a typed-nil provider member must be skipped without losing a
// different, valid service in the same mesh.
func TestGetAllAppMeshPaths_MissingProviderMember(t *testing.T) {
	var nilRouter *types.VirtualServiceProviderMemberVirtualRouter
	var nilNode *types.VirtualServiceProviderMemberVirtualNode
	tests := []struct {
		name     string
		provider types.VirtualServiceProvider
	}{
		{"typed nil router", nilRouter},
		{"typed nil node", nilNode},
		{"router without name", &types.VirtualServiceProviderMemberVirtualRouter{}},
		{"node without name", &types.VirtualServiceProviderMemberVirtualNode{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("GetAllAppMeshPaths panicked on missing provider member: %v", p)
				}
			}()
			mock := &mockAppMeshClient{
				listVirtualServicesFunc: func(context.Context, *appmesh.ListVirtualServicesInput, ...func(*appmesh.Options)) (*appmesh.ListVirtualServicesOutput, error) {
					return &appmesh.ListVirtualServicesOutput{VirtualServices: []types.VirtualServiceRef{
						{VirtualServiceName: aws.String("partial")},
						{VirtualServiceName: aws.String("healthy")},
					}}, nil
				},
				describeVirtualServiceFunc: func(_ context.Context, input *appmesh.DescribeVirtualServiceInput, _ ...func(*appmesh.Options)) (*appmesh.DescribeVirtualServiceOutput, error) {
					provider := tt.provider
					if aws.ToString(input.VirtualServiceName) == "healthy" {
						provider = &types.VirtualServiceProviderMemberVirtualNode{Value: types.VirtualNodeServiceProvider{VirtualNodeName: aws.String("node")}}
					}
					return &appmesh.DescribeVirtualServiceOutput{VirtualService: &types.VirtualServiceData{
						VirtualServiceName: input.VirtualServiceName,
						Spec:               &types.VirtualServiceSpec{Provider: provider},
					}}, nil
				},
				listVirtualRoutersFunc: func(context.Context, *appmesh.ListVirtualRoutersInput, ...func(*appmesh.Options)) (*appmesh.ListVirtualRoutersOutput, error) {
					return &appmesh.ListVirtualRoutersOutput{}, nil
				},
			}
			got := GetAllAppMeshPaths(aws.String("mesh"), mock)
			if len(got) != 1 || got[0].VirtualServiceName != "healthy" || len(got[0].VirtualServiceRoutes) != 1 || got[0].VirtualServiceRoutes[0].DestinationNode != "node" {
				t.Errorf("GetAllAppMeshPaths() = %+v, want only healthy service", got)
			}
		})
	}
}

// T-1634: typed-nil and empty backend members are omitted while valid
// backend services continue to be returned.
func TestGetAppMeshVirtualNodeBackendServices2_MissingMember(t *testing.T) {
	var nilBackend *types.BackendMemberVirtualService
	tests := []struct {
		name    string
		backend types.Backend
	}{
		{"typed nil", nilBackend},
		{"missing name", &types.BackendMemberVirtualService{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("backend extraction panicked on missing member: %v", p)
				}
			}()
			mock := &mockAppMeshClient{
				describeVirtualNodeFunc: func(context.Context, *appmesh.DescribeVirtualNodeInput, ...func(*appmesh.Options)) (*appmesh.DescribeVirtualNodeOutput, error) {
					return &appmesh.DescribeVirtualNodeOutput{VirtualNode: &types.VirtualNodeData{Spec: &types.VirtualNodeSpec{
						Backends: []types.Backend{tt.backend, &types.BackendMemberVirtualService{Value: types.VirtualServiceBackend{VirtualServiceName: aws.String("healthy")}}},
					}}}, nil
				},
			}
			got := getAppMeshVirtualNodeBackendServices2(aws.String("mesh"), aws.String("node"), mock)
			if len(got) != 1 || got[0] != "healthy" {
				t.Errorf("backends = %v, want [healthy]", got)
			}
		})
	}
}
