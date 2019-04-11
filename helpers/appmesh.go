package helpers

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/appmesh"
)

var appmeshSession = appmesh.New(session.New())

// AppmeshSession returns a shared AppmeshSession
func AppmeshSession() *appmesh.AppMesh {
	return appmeshSession
}

// getAllAppMeshRoutes retrieves all of the routes in the mesh
func getAllAppMeshRoutes(meshName *string, svc *appmesh.AppMesh) []*appmesh.RouteRef {
	routersInput := &appmesh.ListVirtualRoutersInput{
		MeshName: meshName,
	}

	routers, err := svc.ListVirtualRouters(routersInput)
	if err != nil {
		fmt.Print(err)
	}
	var routeslist = []*appmesh.RouteRef{}
	for _, routers := range routers.VirtualRouters {
		routesInput := &appmesh.ListRoutesInput{
			MeshName:          meshName,
			VirtualRouterName: routers.VirtualRouterName,
		}
		routes, err := svc.ListRoutes(routesInput)
		if err != nil {
			fmt.Print(err)
		}
		for _, route := range routes.Routes {
			routeslist = append(routeslist, route)
		}
	}

	return routeslist
}

// getAppMeshRouteDescriptions retrieves the details for all of the routes in the mesh
func getAppMeshRouteDescriptions(meshName *string, svc *appmesh.AppMesh) []*appmesh.RouteData {
	routes := getAllAppMeshRoutes(meshName, svc)
	var routedetails = []*appmesh.RouteData{}
	for _, route := range routes {
		input := &appmesh.DescribeRouteInput{
			MeshName:          route.MeshName,
			RouteName:         route.RouteName,
			VirtualRouterName: route.VirtualRouterName,
		}
		output, err := svc.DescribeRoute(input)
		if err != nil {
			fmt.Print(err)
		}
		routedetails = append(routedetails, output.Route)
	}
	return routedetails
}

// getAllAppMeshNodes retrieves all of the VirtualNodes in the mesh
func getAllAppMeshNodes(meshName *string, svc *appmesh.AppMesh) []*appmesh.VirtualNodeData {
	nodesInput := &appmesh.ListVirtualNodesInput{
		MeshName: meshName,
	}
	nodes, err := svc.ListVirtualNodes(nodesInput)
	if err != nil {
		fmt.Print(err)
	}
	var nodelist = []*appmesh.VirtualNodeData{}
	for _, node := range nodes.VirtualNodes {
		input := &appmesh.DescribeVirtualNodeInput{
			MeshName:        node.MeshName,
			VirtualNodeName: node.VirtualNodeName,
		}
		nodetails, err := svc.DescribeVirtualNode(input)
		if err != nil {
			fmt.Print(err)
		}
		nodelist = append(nodelist, nodetails.VirtualNode)
	}
	return nodelist
}

// getAllAppMeshVirtualServices retrieves all of the VirtualServices in the mesh
func getAllAppMeshVirtualServices(meshName *string) []*appmesh.VirtualServiceData {
	svc := AppmeshSession()
	servicesInput := &appmesh.ListVirtualServicesInput{
		MeshName: meshName,
	}
	services, err := svc.ListVirtualServices(servicesInput)
	if err != nil {
		fmt.Print(err)
	}
	var servicelist = []*appmesh.VirtualServiceData{}
	for _, service := range services.VirtualServices {
		input := &appmesh.DescribeVirtualServiceInput{
			MeshName:           service.MeshName,
			VirtualServiceName: service.VirtualServiceName,
		}
		servicedetails, err := svc.DescribeVirtualService(input)
		if err != nil {
			fmt.Print(err)
		}
		servicelist = append(servicelist, servicedetails.VirtualService)
	}
	return servicelist
}

// GetAllAppMeshPaths retrieves all the connections in the mesh
func GetAllAppMeshPaths(meshName *string, svc *appmesh.AppMesh) []AppMeshVirtualService {
	var result []AppMeshVirtualService
	routesholder := make(map[string][]AppMeshVirtualServiceRoute)
	services := getAllAppMeshVirtualServices(meshName)
	routes := getAppMeshRouteDescriptions(meshName, svc)
	for _, route := range routes {
		for _, action := range route.Spec.HttpRoute.Action.WeightedTargets {
			target := AppMeshVirtualServiceRoute{
				Router:          aws.StringValue(route.VirtualRouterName),
				Path:            aws.StringValue(route.Spec.HttpRoute.Match.Prefix),
				DestinationNode: aws.StringValue(action.VirtualNode),
				Weight:          aws.Int64Value(action.Weight),
			}
			routerName := aws.StringValue(route.VirtualRouterName)
			if targets, ok := routesholder[routerName]; ok {
				routesholder[routerName] = append(targets, target)
			} else {
				values := []AppMeshVirtualServiceRoute{target}
				routesholder[routerName] = values
			}

		}
	}
	for _, service := range services {
		serviceroutes := AppMeshVirtualService{
			VirtualServiceName:   aws.StringValue(service.VirtualServiceName),
			VirtualServiceRoutes: routesholder[aws.StringValue(service.Spec.Provider.VirtualRouter.VirtualRouterName)],
		}
		result = append(result, serviceroutes)
	}
	return result
}

// GetAllUnservicedAppMeshNodes returns a slice of nodes that don't serve as the backend for any service
func GetAllUnservicedAppMeshNodes(meshname *string, svc *appmesh.AppMesh) []string {
	routes := GetAllAppMeshPaths(meshname, svc)
	nodes := getAllAppMeshNodes(meshname, svc)
	var nodelist []string
	var pathlist []string
	var dangling []string
	for _, node := range nodes {
		nodelist = append(nodelist, aws.StringValue(node.VirtualNodeName))
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
func GetAllAppMeshNodeConnections(meshname *string, svc *appmesh.AppMesh) []AppMeshVirtualNode {
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
			VirtualNodeName: aws.StringValue(node.VirtualNodeName),
			BackendServices: connectedServices,
			BackendNodes:    backendNodes,
		}
		nodelist = append(nodelist, nodeinfo)
	}
	return nodelist
}

// getAppMeshVirtualNodeBackendServices2 retrieves a list of all the backend services for a node
func getAppMeshVirtualNodeBackendServices2(meshname *string, nodename *string, svc *appmesh.AppMesh) []string {
	var backendlists []string
	input := &appmesh.DescribeVirtualNodeInput{
		MeshName:        meshname,
		VirtualNodeName: nodename,
	}
	nodetails, err := svc.DescribeVirtualNode(input)
	if err != nil {
		fmt.Print(err)
	}
	for _, backend := range nodetails.VirtualNode.Spec.Backends {
		backendlists = append(backendlists, aws.StringValue(backend.VirtualService.VirtualServiceName))
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
	Weight          int64
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
	service.VirtualServicePaths = append(paths, path)
}
