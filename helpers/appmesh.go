// Package helpers provides business logic and AWS SDK interactions for various AWS services
package helpers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/appmesh"
	"github.com/aws/aws-sdk-go-v2/service/appmesh/types"
)

// AppMeshAPI defines the subset of the App Mesh client used by helpers.
type AppMeshAPI interface {
	ListVirtualRouters(ctx context.Context, params *appmesh.ListVirtualRoutersInput, optFns ...func(*appmesh.Options)) (*appmesh.ListVirtualRoutersOutput, error)
	ListRoutes(ctx context.Context, params *appmesh.ListRoutesInput, optFns ...func(*appmesh.Options)) (*appmesh.ListRoutesOutput, error)
	DescribeRoute(ctx context.Context, params *appmesh.DescribeRouteInput, optFns ...func(*appmesh.Options)) (*appmesh.DescribeRouteOutput, error)
	ListVirtualNodes(ctx context.Context, params *appmesh.ListVirtualNodesInput, optFns ...func(*appmesh.Options)) (*appmesh.ListVirtualNodesOutput, error)
	DescribeVirtualNode(ctx context.Context, params *appmesh.DescribeVirtualNodeInput, optFns ...func(*appmesh.Options)) (*appmesh.DescribeVirtualNodeOutput, error)
	ListVirtualServices(ctx context.Context, params *appmesh.ListVirtualServicesInput, optFns ...func(*appmesh.Options)) (*appmesh.ListVirtualServicesOutput, error)
	DescribeVirtualService(ctx context.Context, params *appmesh.DescribeVirtualServiceInput, optFns ...func(*appmesh.Options)) (*appmesh.DescribeVirtualServiceOutput, error)
}

// getAllAppMeshRoutes retrieves all of the routes in the mesh. Both
// ListVirtualRouters and ListRoutes paginate via NextToken; the SDK
// paginators are used so large meshes do not get truncated.
func getAllAppMeshRoutes(meshName *string, svc AppMeshAPI) []types.RouteRef {
	routersPaginator := appmesh.NewListVirtualRoutersPaginator(svc, &appmesh.ListVirtualRoutersInput{
		MeshName: meshName,
	})

	var routeslist = []types.RouteRef{}
	for routersPaginator.HasMorePages() {
		routersPage, err := routersPaginator.NextPage(context.TODO())
		if err != nil {
			fmt.Print(err)
			return nil
		}
		for _, router := range routersPage.VirtualRouters {
			routesPaginator := appmesh.NewListRoutesPaginator(svc, &appmesh.ListRoutesInput{
				MeshName:          meshName,
				VirtualRouterName: router.VirtualRouterName,
			})
			for routesPaginator.HasMorePages() {
				routesPage, err := routesPaginator.NextPage(context.TODO())
				if err != nil {
					fmt.Print(err)
					break
				}
				routeslist = append(routeslist, routesPage.Routes...)
			}
		}
	}

	return routeslist
}

// getAppMeshRouteDescriptions retrieves the details for all of the routes in the mesh
func getAppMeshRouteDescriptions(meshName *string, svc AppMeshAPI) []*types.RouteData {
	routes := getAllAppMeshRoutes(meshName, svc)
	var routedetails = []*types.RouteData{}
	for _, route := range routes {
		input := &appmesh.DescribeRouteInput{
			MeshName:          route.MeshName,
			RouteName:         route.RouteName,
			VirtualRouterName: route.VirtualRouterName,
		}
		output, err := svc.DescribeRoute(context.TODO(), input)
		if err != nil {
			fmt.Print(err)
			continue
		}
		routedetails = append(routedetails, output.Route)
	}
	return routedetails
}

// getAllAppMeshNodes retrieves all of the VirtualNodes in the mesh. The
// SDK paginator is walked so meshes with many virtual nodes are not
// truncated at the first page.
func getAllAppMeshNodes(meshName *string, svc AppMeshAPI) []*types.VirtualNodeData {
	paginator := appmesh.NewListVirtualNodesPaginator(svc, &appmesh.ListVirtualNodesInput{
		MeshName: meshName,
	})
	var nodelist = []*types.VirtualNodeData{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			fmt.Print(err)
			return nil
		}
		for _, node := range page.VirtualNodes {
			input := &appmesh.DescribeVirtualNodeInput{
				MeshName:        node.MeshName,
				VirtualNodeName: node.VirtualNodeName,
			}
			nodetails, err := svc.DescribeVirtualNode(context.TODO(), input)
			if err != nil {
				fmt.Print(err)
				continue
			}
			nodelist = append(nodelist, nodetails.VirtualNode)
		}
	}
	return nodelist
}

// getAllAppMeshVirtualServices retrieves all of the VirtualServices in
// the mesh. The SDK paginator is walked so meshes with many virtual
// services are not truncated at the first page.
func getAllAppMeshVirtualServices(meshName *string, svc AppMeshAPI) []*types.VirtualServiceData {
	paginator := appmesh.NewListVirtualServicesPaginator(svc, &appmesh.ListVirtualServicesInput{
		MeshName: meshName,
	})
	var servicelist = []*types.VirtualServiceData{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			fmt.Print(err)
			return nil
		}
		for _, service := range page.VirtualServices {
			input := &appmesh.DescribeVirtualServiceInput{
				MeshName:           service.MeshName,
				VirtualServiceName: service.VirtualServiceName,
			}
			servicedetails, err := svc.DescribeVirtualService(context.TODO(), input)
			if err != nil {
				fmt.Print(err)
				continue
			}
			servicelist = append(servicelist, servicedetails.VirtualService)
		}
	}
	return servicelist
}

// buildRoutesHolder processes route data and builds a map of router name to routes.
// Handles HTTP, HTTP/2, gRPC, and TCP route types.
func buildRoutesHolder(routes []*types.RouteData) map[string][]AppMeshVirtualServiceRoute {
	routesholder := make(map[string][]AppMeshVirtualServiceRoute)
	for _, route := range routes {
		var targets []types.WeightedTarget
		var path string

		switch {
		case route.Spec.HttpRoute != nil:
			targets = route.Spec.HttpRoute.Action.WeightedTargets
			path = aws.ToString(route.Spec.HttpRoute.Match.Prefix)
		case route.Spec.Http2Route != nil:
			targets = route.Spec.Http2Route.Action.WeightedTargets
			path = aws.ToString(route.Spec.Http2Route.Match.Prefix)
		case route.Spec.GrpcRoute != nil:
			targets = route.Spec.GrpcRoute.Action.WeightedTargets
			path = grpcMatchPath(route.Spec.GrpcRoute.Match)
		case route.Spec.TcpRoute != nil:
			targets = route.Spec.TcpRoute.Action.WeightedTargets
		default:
			continue
		}

		routerName := aws.ToString(route.VirtualRouterName)
		for _, action := range targets {
			target := AppMeshVirtualServiceRoute{
				Router:          routerName,
				Path:            path,
				DestinationNode: aws.ToString(action.VirtualNode),
				Weight:          action.Weight,
			}
			routesholder[routerName] = append(routesholder[routerName], target)
		}
	}
	return routesholder
}

// grpcMatchPath builds a path string from a gRPC route match.
func grpcMatchPath(match *types.GrpcRouteMatch) string {
	if match == nil {
		return ""
	}
	serviceName := aws.ToString(match.ServiceName)
	methodName := aws.ToString(match.MethodName)
	if serviceName != "" && methodName != "" {
		return serviceName + "/" + methodName
	}
	if serviceName != "" {
		return serviceName
	}
	return methodName
}

// GetAllAppMeshPaths retrieves all the connections in the mesh
func GetAllAppMeshPaths(meshName *string, svc AppMeshAPI) []AppMeshVirtualService {
	var result []AppMeshVirtualService
	services := getAllAppMeshVirtualServices(meshName, svc)
	routes := getAppMeshRouteDescriptions(meshName, svc)
	routesholder := buildRoutesHolder(routes)
	for _, service := range services {
		switch v := service.Spec.Provider.(type) {
		case *types.VirtualServiceProviderMemberVirtualRouter:
			serviceroutes := AppMeshVirtualService{
				VirtualServiceName:   aws.ToString(service.VirtualServiceName),
				VirtualServiceRoutes: routesholder[aws.ToString(v.Value.VirtualRouterName)],
			}
			result = append(result, serviceroutes)
		default:
			fmt.Println("union is nil or unknown type")
		}
	}
	return result
}

// GetAllUnservicedAppMeshNodes returns a slice of nodes that don't serve as the backend for any service
func GetAllUnservicedAppMeshNodes(meshname *string, svc AppMeshAPI) []string {
	routes := GetAllAppMeshPaths(meshname, svc)
	nodes := getAllAppMeshNodes(meshname, svc)
	var nodelist []string
	var pathlist []string
	var dangling []string
	for _, node := range nodes {
		nodelist = append(nodelist, aws.ToString(node.VirtualNodeName))
	}
	for _, route := range routes {
		for _, path := range route.VirtualServiceRoutes {
			pathlist = append(pathlist, path.DestinationNode)
		}
	}
	for _, node := range nodelist {
		if !stringInSlice(node, pathlist) {
			dangling = append(dangling, node)
		}
	}
	return dangling
}

// GetAllAppMeshNodeConnections retrieves all nodes and which services/nodes they connect to
func GetAllAppMeshNodeConnections(meshname *string, svc AppMeshAPI) []AppMeshVirtualNode {
	services := GetAllAppMeshPaths(meshname, svc)
	var nodelist []AppMeshVirtualNode
	servicelist := make(map[string]AppMeshVirtualService)
	for _, service := range services {
		for _, path := range service.VirtualServiceRoutes {
			destinationNode := path.DestinationNode
			backends := getAppMeshVirtualNodeBackendServices2(meshname, &destinationNode, svc)
			if len(backends) == 0 {
				pathDetails := AppMeshVirtualServicePath{
					VirtualNode: destinationNode,
				}
				service.AddPath(pathDetails)
			} else {
				for _, backend := range backends {
					pathDetails := AppMeshVirtualServicePath{
						VirtualNode: destinationNode,
						ServiceName: backend,
					}
					service.AddPath(pathDetails)
				}
			}
		}
		servicelist[service.VirtualServiceName] = service
	}
	nodes := getAllAppMeshNodes(meshname, svc)
	for _, node := range nodes {
		connectedServices := getAppMeshVirtualNodeBackendServices2(meshname, node.VirtualNodeName, svc)
		var backendNodes []string
		for _, service := range connectedServices {
			connectedservice := servicelist[service]
			for _, path := range connectedservice.VirtualServicePaths {
				backendNodes = append(backendNodes, path.VirtualNode)
			}
		}
		nodeinfo := AppMeshVirtualNode{
			VirtualNodeName: aws.ToString(node.VirtualNodeName),
			BackendServices: connectedServices,
			BackendNodes:    backendNodes,
		}
		nodelist = append(nodelist, nodeinfo)
	}
	return nodelist
}

// getAppMeshVirtualNodeBackendServices2 retrieves a list of all the backend services for a node
func getAppMeshVirtualNodeBackendServices2(meshname *string, nodename *string, svc AppMeshAPI) []string {
	var backendlists []string
	input := &appmesh.DescribeVirtualNodeInput{
		MeshName:        meshname,
		VirtualNodeName: nodename,
	}
	nodetails, err := svc.DescribeVirtualNode(context.TODO(), input)
	if err != nil {
		fmt.Print(err)
		return nil
	}
	for _, backend := range nodetails.VirtualNode.Spec.Backends {
		switch v := backend.(type) {
		case *types.BackendMemberVirtualService:
			backendlists = append(backendlists, aws.ToString(v.Value.VirtualServiceName))
		default:
			fmt.Println("union is nil or unknown type")
		}
	}
	return backendlists
}

// AppMeshVirtualService contains information about an App Mesh Virtual Service
type AppMeshVirtualService struct {
	VirtualServiceName   string
	VirtualServiceRoutes []AppMeshVirtualServiceRoute
	VirtualServicePaths  []AppMeshVirtualServicePath
}

// AppMeshVirtualServiceRoute contains information about an App Mesh route
type AppMeshVirtualServiceRoute struct {
	Router          string
	Path            string
	DestinationNode string
	Weight          int32
}

// AppMeshVirtualServicePath shows virtual nodes and their backend that a service might be connected to
type AppMeshVirtualServicePath struct {
	VirtualNode string
	ServiceName string
}

// AppMeshVirtualNode contains information about an App Mesh Virtual Node
type AppMeshVirtualNode struct {
	VirtualNodeName string
	BackendServices []string
	BackendNodes    []string
}

// AddPath adds a path to an AppMeshVirtualService
func (service *AppMeshVirtualService) AddPath(path AppMeshVirtualServicePath) {
	var paths []AppMeshVirtualServicePath
	if service.VirtualServicePaths != nil {
		paths = service.VirtualServicePaths
	}
	paths = append(paths, path)
	service.VirtualServicePaths = paths
}
