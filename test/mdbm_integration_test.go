package test

import (
	"crypto/tls"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
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

var (
kubectlOptions = k8s.NewKubectlOptions("", "", _namespace)
releaseName = fmt.Sprintf("mdbm-%s", strings.ToLower(random.UniqueId()))
)

// TestMain MDBM charts installation and verification (may take to 7 minutes)
func TestMain(m *testing.T) {

	helmChartPath := "../charts/mdbm"
	retries := 15
	sleep := 30 * time.Second
	options := &helm.Options{
		SetValues: map[string]string{"global.images.registry": "registry:8282"},
	}

	svcNameFinal := "db-monitor"

	defer helm.Delete(m, options, releaseName, true)
	helm.Install(m, options, helmChartPath, releaseName)
	k8s.WaitUntilPodAvailable(m, kubectlOptions, svcNameFinal+"-0", retries, sleep)
    time.Sleep(60 * time.Second)

    cr := fmt.Sprintf(_mongodb, _mongoCluster,_namespace)
	defer k8s.KubectlDeleteFromStringE(m, kubectlOptions, cr)
	err := k8s.KubectlApplyFromStringE(m, kubectlOptions, cr)
	assert.Nil(m, err, "create mongodb cluster")

	m.Run("mdbm", func(t *testing.T) {
        t.Run("Pmm", svcPmm)
        t.Run("Api", svcApi)
        t.Run("Operator", svcOperator)
		t.Run("Mongodb", mongodbCluster)
	})
}

// svcPmm
func svcPmm(t *testing.T) {
	svcPort := 443
	svcName := "db-monitor"
	expectBody := "Monitoring"
	expectResponse := 200

	tunnel := createTunnel(t, kubectlOptions, svcName, svcPort)
	endpoint := fmt.Sprintf("https://%s/graph/login", tunnel.Endpoint())
	verifyStatus(t, kubectlOptions, endpoint, expectBody, expectResponse)
}

// svcApi
func svcApi(t *testing.T) {
	svcPort := 5000
	svcName := "db-api"
	expectBody := "results"
	expectResponse := 200

	tunnel := createTunnel(t, kubectlOptions, svcName, svcPort)
	endpoint:= fmt.Sprintf("http://%s/api/v1/hosts", tunnel.Endpoint())
	verifyStatus(t, kubectlOptions, endpoint, expectBody, expectResponse)
}

// svcOperator
func svcOperator(t *testing.T) {
	svcPort := 8443
	svcName := "db-operator"
	expectBody := "Unauthorized"
	expectResponse := 401

	tunnel := createTunnel(t, kubectlOptions, svcName, svcPort)
	endpoint := fmt.Sprintf("https://%s/", tunnel.Endpoint())
	verifyStatus(t, kubectlOptions, endpoint, expectBody, expectResponse)
}

// mongodbCluster
func mongodbCluster(t *testing.T) {
	podCount := 3
	endStatus := "Done"

	verifyMongodbPods(t, kubectlOptions, podCount)
	verifyMongodbCr(t, kubectlOptions, endStatus)
}


// createTunnel
func createTunnel(t *testing.T, kubectlOptions *k8s.KubectlOptions, svcName string, RemotePort int) *k8s.Tunnel {
	tunnel := k8s.NewTunnel(kubectlOptions, k8s.ResourceTypeService, svcName, 0, RemotePort)
	tunnel.ForwardPort(t)
	return tunnel
}

// verifyStatus
func verifyStatus(t *testing.T, kubectlOptions *k8s.KubectlOptions, endpoint string, expectBody string, expectResponse int) {
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
			return statusCode == expectResponse && strings.Contains(body, expectBody)
		},
	)
}

// verifyMongodbPods
func verifyMongodbPods(t *testing.T, kubectlOptions *k8s.KubectlOptions, podCount int) {
	t.Parallel()

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
}

// verifyMongodbCr
func verifyMongodbCr(t *testing.T, kubectlOptions *k8s.KubectlOptions, endStage string) {
timeout := 60 * time.Second
filter := "-o=jsonpath='{range .items[*]}{.status.stage}{\"\\n\"}{end}'"

deadline := time.Now().Add(timeout)
for {
    out, _ := k8s.RunKubectlAndGetOutputE(t, kubectlOptions, "get", "mdb", filter)
    if strings.Contains(out, endStage) || time.Now().After(deadline) {
        break
    }
}
}