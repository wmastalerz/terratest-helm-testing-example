package charts

import (
	"crypto/tls"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
_mongoCluster = "db"
_namespace = "csdb"
_mongodb = `
apiVersion: mongodb.samsung.com/v1
kind: MongoDB
metadata:
  name: %s
spec:
  image: registry:8282/mdbm/mongodb:latest
  allowUnsafeConfiguration:
    topology: true
  pmmClient:
    image: registry:8282/mdbm/pmm-client:latest
    pmmServer:
      name: db-monitor
      namespace: %s
    credential: db-monitor
  storage:
    engine: inmemory
    inMemory:
      engineConfig:
        inMemorySizeRatio: 80
  configServer:
    name: config
    membersCount: 1
  shards:
    - name: data0
      membersCount: 1
  mongos:
    count: 1
`
)
var(
	kubectlOptions = k8s.NewKubectlOptions("", "", _namespace)
	releaseName = fmt.Sprintf("mdbm-%s", strings.ToLower(random.UniqueId()))
	t = NewTestingT()
)

func TestCharts(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testing Suite")
}

type TestingT struct {
	GinkgoTInterface
	desc GinkgoTestDescription
}

func NewTestingT() TestingT {
	return TestingT{GinkgoT(), CurrentGinkgoTestDescription()}
}

func (i TestingT) Helper() {

}
func (i TestingT) Name() string {
	return i.desc.FullTestText
}

func SetupK8sConfig() {
	_, filename, _, _ := runtime.Caller(0)
	k8sConfigFilesPath := path.Join(path.Join(path.Dir(filename), "../test"), "kubeconfigs")

	fmt.Print("k8sConfigFilesPath", k8sConfigFilesPath)
	var KUBECONFIG string

	filepath.Walk(k8sConfigFilesPath, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".k8s" {
			if KUBECONFIG != "" {
				KUBECONFIG += ":" + info.Name()
			} else {
				KUBECONFIG += info.Name()
			}
		}
		return nil
	})

	os.Setenv("KUBECONFIG", k8sConfigFilesPath+"/"+KUBECONFIG)
}

// createTunnel
func createTunnel(t *testing.T, kubectlOptions *k8s.KubectlOptions, svcName string, RemotePort int) *k8s.Tunnel {
	tunnel := k8s.NewTunnel(kubectlOptions, k8s.ResourceTypeService, svcName, 0, RemotePort)
	tunnel.ForwardPort(t)
	return tunnel
}



// verifyStatus
func verifyStatus(t *testing.T, kubectlOptions *k8s.KubectlOptions, endpoint string, expectResponse int) string {
    var currentBody string
	retries := 2
	sleep := 10 * time.Second
	tlsConfig := tls.Config{
		InsecureSkipVerify: true,
	}

	http_helper.HttpGetWithRetryWithCustomValidation(
		t,
		endpoint,
		&tlsConfig,
		retries,
		sleep,
		func(statusCode int, body string) bool {
			currentBody = body
			return statusCode == expectResponse
		},
	)
	return currentBody
}

// verifyMongodbPods
func verifyMongodbPods(t *testing.T, kubectlOptions *k8s.KubectlOptions, podCount int) int {
	filters := metav1.ListOptions{
		LabelSelector: fmt.Sprintf(
			"mongodb.samsung.com/cluster=%s,mongodb.samsung.com/namespace=%s",
			_mongoCluster, _namespace,
		),
	}
	k8s.WaitUntilNumPodsCreated(t, kubectlOptions, filters, podCount, 30, 10*time.Second)
	pods := k8s.ListPods(t, kubectlOptions, filters)
	for _, pod := range pods {
		k8s.WaitUntilPodAvailable(t, kubectlOptions, pod.Name, 30, 10*time.Second)
	}
	return len(pods)
}

// verifyMongodbCr
func verifyMongodbCr(t *testing.T, kubectlOptions *k8s.KubectlOptions, endStage string) string {
    var out string
	timeout := 60 * time.Second
	filter := "-o=jsonpath='{range .items[*]}{.status.stage}{\"\\n\"}{end}'"

	deadline := time.Now().Add(timeout)
	for {
		time.Sleep(10 * time.Second)
		out, _ = k8s.RunKubectlAndGetOutputE(t, kubectlOptions, "get", "mdb", filter)
		if strings.Contains(out, endStage) || time.Now().After(deadline) {
			break
		}
	}
    return out
}


var _ = Describe("When I deploy helm chart mdbm operator and CR mongodb", func() {
	Context("The Operator and Mongodb have been deployed", Ordered, func() {
		options := &helm.Options{
			SetValues: map[string]string{"global.images.registry": "registry:8282"},
		}
		cr := fmt.Sprintf(_mongodb, _mongoCluster,_namespace)

		BeforeAll(func() {
			SetupK8sConfig()
			helmChartPath := "../charts/mdbm"
			retries := 15
			sleep := 30 * time.Second
			svcNameFinal := "db-monitor"

			helm.Install(t, options, helmChartPath, releaseName)
			k8s.WaitUntilPodAvailable(t, kubectlOptions, svcNameFinal+"-0", retries, sleep)
			time.Sleep(60 * time.Second)
			err := k8s.KubectlApplyFromStringE(t, kubectlOptions, cr)
			assert.Nil(t, err, "create mongodb cluster")

		})

		AfterAll(func() {
			helm.Delete(t, options, releaseName, true)
			k8s.KubectlDeleteFromStringE(t, kubectlOptions, cr)
		})

		Describe("I send HTTP GET request to service", func() {
			It("Should PMM return dashboard graph/login contain MONITORING", func() {
				svcPort := 443
				svcName := "db-monitor"
				expectResponse := 200
				expectBody := "Monitoring"

				tunnel := createTunnel(&testing.T{}, kubectlOptions, svcName, svcPort)
				endpoint := fmt.Sprintf("https://%s/graph/login", tunnel.Endpoint())
				finalBody := verifyStatus(&testing.T{}, kubectlOptions, endpoint, expectResponse)
				Expect(finalBody).Should(ContainSubstring(expectBody))
			})
			It("Should API return api/v1/hosts response with RESULTS", func() {
				svcPort := 5000
				svcName := "db-api"
				expectResponse := 200
				expectBody := "results"

				tunnel := createTunnel(&testing.T{}, kubectlOptions, svcName, svcPort)
				endpoint:= fmt.Sprintf("http://%s/api/v1/hosts", tunnel.Endpoint())
				finalBody := verifyStatus(&testing.T{}, kubectlOptions, endpoint, expectResponse)
				Expect(finalBody).Should(ContainSubstring(expectBody))
			})
			It("Should OPERATOR not access into svc because UNAUTHORIZED", func() {
				svcPort := 8443
				svcName := "db-operator"
				expectResponse := 401
				expectBody := "Unauthorized"

				tunnel := createTunnel(&testing.T{}, kubectlOptions, svcName, svcPort)
				endpoint := fmt.Sprintf("https://%s/", tunnel.Endpoint())
				finalBody := verifyStatus(&testing.T{}, kubectlOptions, endpoint, expectResponse)
				Expect(finalBody).Should(ContainSubstring(expectBody))
			})
		})

		Describe("I read the number of pods and CR final stage" , func() {
			It("Should have expected number of pod", func() {
				podCount := 3
				currCount := verifyMongodbPods(&testing.T{}, kubectlOptions, podCount)
				Expect(currCount).To(Equal(podCount), "min. 3 pods (data0-0, config0-0, mongo0-0)" )
			})
			It("Should have stage DONE", func() {
				endStatus := "Done"
				finalStage :=	verifyMongodbCr(&testing.T{}, kubectlOptions, endStatus)
				Expect(finalStage).Should(ContainSubstring(endStatus))
			})
		})
	})
})