package srv

import (
	"fmt"
	"io/ioutil"

	"mock_service/pkg/istio.io/fortio/log"

	"github.com/Tahler/isotope/convert/pkg/graph"
	"github.com/Tahler/isotope/convert/pkg/graph/svc"
	"github.com/Tahler/isotope/convert/pkg/graph/svctype"
	"github.com/ghodss/yaml"
)

// HandlerFromServiceGraphYAML makes a handler to emulate the service with name
// serviceName in the service graph represented by the YAML file at path.
func HandlerFromServiceGraphYAML(
	path string, serviceName string) (handler Handler, err error) {

	serviceGraph, err := serviceGraphFromYAMLFile(path)
	if err != nil {
		return
	}

	service, err := extractService(serviceGraph, serviceName)
	if err != nil {
		return
	}
	logService(service)

	serviceTypes := extractServiceTypes(serviceGraph)

	handler = Handler{
		Service:      service,
		ServiceTypes: serviceTypes,
	}
	return
}

func logService(service svc.Service) error {
	if log.Log(log.Info) {
		serviceYAML, err := yaml.Marshal(service)
		if err != nil {
			return err
		}
		log.Infof("acting as service %s:\n%s", service.Name, serviceYAML)
	}
	return nil
}

// serviceGraphFromYAMLFile unmarshals the ServiceGraph from the YAML at path.
func serviceGraphFromYAMLFile(
	path string) (serviceGraph graph.ServiceGraph, err error) {
	graphYAML, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	log.Debugf("unmarshalling\n%s", graphYAML)
	err = yaml.Unmarshal(graphYAML, &serviceGraph)
	if err != nil {
		return
	}
	return
}

// extractService finds the service in serviceGraph with the specified name.
func extractService(
	serviceGraph graph.ServiceGraph, name string) (
	service svc.Service, err error) {
	for _, svc := range serviceGraph.Services {
		if svc.Name == name {
			service = svc
			return
		}
	}
	err = fmt.Errorf(
		"service with name %s does not exist in %v", name, serviceGraph)
	return
}

// extractServiceTypes builds a map from service name to its type
// (i.e. HTTP or gRPC).
func extractServiceTypes(
	serviceGraph graph.ServiceGraph) map[string]svctype.ServiceType {
	types := make(map[string]svctype.ServiceType, len(serviceGraph.Services))
	for _, service := range serviceGraph.Services {
		types[service.Name] = service.Type
	}
	return types
}
