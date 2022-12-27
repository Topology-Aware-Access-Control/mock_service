package srv

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"mock_service/pkg/istio.io/fortio/log"

	"mock_service/pkg/srv/prometheus"

	"github.com/Tahler/isotope/convert/pkg/graph/script"
	"github.com/Tahler/isotope/convert/pkg/graph/svctype"
	multierror "github.com/hashicorp/go-multierror"
)

func execute(
	step interface{},
	forwardableHeader http.Header,
	serviceTypes map[string]svctype.ServiceType) (err error) {
	switch cmd := step.(type) {
	case script.SleepCommand:
		executeSleepCommand(cmd)
	case script.RequestCommand:
		err = executeRequestCommand(
			cmd, forwardableHeader, serviceTypes)
	case script.ConcurrentCommand:
		err = executeConcurrentCommand(
			cmd, forwardableHeader, serviceTypes)
	default:
		log.Fatalf("unknown command type in script: %T", cmd)
	}
	return
}

func executeSleepCommand(cmd script.SleepCommand) {
	time.Sleep(time.Duration(cmd))
}

// Execute sends an HTTP request to another service. Assumes DNS is available
// which maps exe.ServiceName to the relevant URL to reach the service.
func executeRequestCommand(
	cmd script.RequestCommand,
	forwardableHeader http.Header,
	serviceTypes map[string]svctype.ServiceType) (err error) {
	destName := cmd.ServiceName
	destType, ok := serviceTypes[destName]
	if !ok {
		err = fmt.Errorf("service %s does not exist", destName)
		return
	}
	response, err := sendRequest(destName, destType, cmd.Size, forwardableHeader)
	if err != nil {
		return
	}
	prometheus.RecordRequestSent(destName, uint64(cmd.Size))
	if response.StatusCode == 200 {
		log.Debugf("%s responded with %s", destName, response.Status)
	} else {
		log.Errf("%s responded with %s", destName, response.Status)
	}
	if response.StatusCode == http.StatusInternalServerError {
		err = fmt.Errorf("service %s responded with %s", destName, response.Status)
	}

	// Necessary for reusing HTTP/1.x "keep-alive" TCP connections.
	// https://golang.org/pkg/net/http/#Response
	readAllAndClose(response.Body)

	return
}

func readAllAndClose(r io.ReadCloser) {
	io.Copy(ioutil.Discard, r)
	r.Close()
}

// executeConcurrentCommand calls each command in exe.Commands asynchronously
// and waits for each to complete.
func executeConcurrentCommand(
	cmd script.ConcurrentCommand,
	forwardableHeader http.Header,
	serviceTypes map[string]svctype.ServiceType) (errs error) {
	numSubCmds := len(cmd)
	wg := sync.WaitGroup{}
	wg.Add(numSubCmds)
	for _, subCmd := range cmd {
		go func(step interface{}) {
			defer wg.Done()

			err := execute(step, forwardableHeader, serviceTypes)
			if err != nil {
				errs = multierror.Append(errs, err)
			}
		}(subCmd)
	}
	wg.Wait()
	return
}
