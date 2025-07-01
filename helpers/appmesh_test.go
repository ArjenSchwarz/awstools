package helpers

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appmesh/types"
)

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

// Integration tests would require App Mesh access
func TestGetAllAppMeshRoutes_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires App Mesh client interface implementation")
}

func TestGetAppMeshRouteDescriptions_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires App Mesh client interface implementation")
}
