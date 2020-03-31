package admissionreview

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	serviceResourceType = metav1.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "services",
	}
	deploymentResourceType = metav1.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	err error //TODO: is this ok?!
)

// Mutate mutates
func Mutate(body []byte, client kubernetes.Clientset) ([]byte, error) {
	log.Printf("AdmissionReview request body: %s\n", string(body))
	// unmarshal request into AdmissionReview struct
	admReview := admissionv1beta1.AdmissionReview{}
	if err := json.Unmarshal(body, &admReview); err != nil {
		return nil, fmt.Errorf("unmarshaling request failed with %s", err)
	}

	// Get sidecar data list from namespace annotation "bnhp.co.il/transformers.apitest-sidecar-data"
	namespace, err := client.CoreV1().Namespaces().Get(admReview.Request.Namespace, metav1.GetOptions{})
	var sideCarDataList []SideCarData
	if err := json.Unmarshal(
		[]byte(namespace.Annotations["bnhp.co.il/transformers.apitest-sidecar-data"]),
		&sideCarDataList); err != nil {
		return nil, fmt.Errorf("unmarshaling request failed with %s", err)
	}
	log.Println(sideCarDataList)

	responseBody := []byte{}
	admReviewResponse := admissionv1beta1.AdmissionResponse{}

	if admReview.Request.Resource == serviceResourceType {
		mutateService(*admReview.Request, &admReviewResponse, sideCarDataList)
	} else if admReview.Request.Resource == deploymentResourceType {
		mutateDeployment(*admReview.Request, &admReviewResponse, sideCarDataList)
	}

	// set response options
	admReviewResponse.Allowed = true
	admReviewResponse.UID = admReview.Request.UID
	pT := admissionv1beta1.PatchTypeJSONPatch
	admReviewResponse.PatchType = &pT
	admReviewResponse.AuditAnnotations = map[string]string{
		"apitest-sidecar": "injected",
	}
	admReviewResponse.Result = &metav1.Status{
		Status: "Success",
	}
	admReview.Response = &admReviewResponse

	// back into JSON so we can return the finished AdmissionReview w/ Response directly
	// w/o needing to convert things in the http handler
	responseBody, err = json.Marshal(admReview)
	if err != nil {
		return nil, err
	}

	log.Printf("AdmissionReview responce body: %s\n", string(responseBody))
	return responseBody, nil
}

// SERVICE
func mutateService(
	admRequest admissionv1beta1.AdmissionRequest,
	admReviewResponse *admissionv1beta1.AdmissionResponse,
	sideCarDataList []SideCarData) error {
	// get reviewedService object
	var reviewedService *corev1.Service
	if err := json.Unmarshal(admRequest.Object.Raw, &reviewedService); err != nil {
		return fmt.Errorf("unable unmarshal pod json object %v", err)
	}

	if PortOverrideNeeded(reviewedService, sideCarDataList) {
		// build patch map
		p := []map[string]interface{}{}
		// find service in sidecar data list
		for _, sideCarData := range sideCarDataList {
			for x, port := range reviewedService.Spec.Ports {
				if int32(sideCarData.CotanierPort) == port.Port {
					log.Println(fmt.Printf(
						"patching service %v.%v: /spec/ports/%v/targetPort",
						reviewedService.Namespace, reviewedService.Name, x))

					containerPort, _ := strconv.ParseInt(getEnv("PROXY_PORT", "1330"), 10, 32)
					patch := map[string]interface{}{
						"op":    "replace",
						"path":  fmt.Sprintf("/spec/ports/%d/targetPort", x),
						"value": int32(containerPort),
					}
					p = append(p, patch)
				}
			}
		}
		// parse the []map into JSON and add to admReviewResponse
		admReviewResponse.Patch, err = json.Marshal(p)
	}
	return err
}

// DEPLOYMENT
func mutateDeployment(
	admRequest admissionv1beta1.AdmissionRequest,
	admReviewResponse *admissionv1beta1.AdmissionResponse,
	sideCarDataList []SideCarData) error {
	// get reviewedDeployment object
	var revieweDeployment *appsv1.Deployment
	if err := json.Unmarshal(admRequest.Object.Raw, &revieweDeployment); err != nil {
		return fmt.Errorf("unable unmarshal pod json object %v", err)
	}

	sideCarData := DeploymentSideCarInjectionData(revieweDeployment, sideCarDataList)
	if sideCarData != nil {
		log.Println(fmt.Printf("patching Deployment %v.%v",
			revieweDeployment.Namespace, revieweDeployment.Name))
		// build patch map
		patch := map[string]interface{}{
			"op":    "replace",
			"path":  "/spec/template/spec/containers",
			"value": append(revieweDeployment.Spec.Template.Spec.Containers, SidecarCotainer(sideCarData)),
		}
		p := []map[string]interface{}{patch}

		// parse the []map into JSON and add to admReviewResponse
		admReviewResponse.Patch, err = json.Marshal(p)
	}
	return err
}

func PortOverrideNeeded(service *corev1.Service, sideCarDataList []SideCarData) bool {
	for _, data := range sideCarDataList {
		if service.Name == data.Service {
			return true
		}
	}
	return false
}

func DeploymentSideCarInjectionNeeded(deployment *appsv1.Deployment, sideCarDataList []SideCarData) bool {
	for _, data := range sideCarDataList {
		if deployment.Name == data.Deployment {
			return true
		}
	}
	return false
}

func DeploymentSideCarInjectionData(deployment *appsv1.Deployment, sideCarDataList []SideCarData) *SideCarData {
	for _, data := range sideCarDataList {
		if deployment.Name == data.Deployment {
			return &data
		}
	}
	return nil
}

func SidecarCotainer(sideCarData *SideCarData) corev1.Container {
	containerPort, _ := strconv.ParseInt(getEnv("PROXY_PORT", "1330"), 10, 32)
	sideCarProt := corev1.ContainerPort{
		ContainerPort: int32(containerPort),
		Protocol:      "TCP",
	}

	logUrl := corev1.EnvVar{
		Name:  "LOG_URL",
		Value: getEnv("LOG_URL", "http://localhost:8080"),
	}
	proxyUrlString := fmt.Sprintf("http://%v:%v", sideCarData.Service, sideCarData.CotanierPort)
	proxyUrl := corev1.EnvVar{
		Name:  "PROXY_URL",
		Value: proxyUrlString,
	}

	return corev1.Container{
		Name:  "tranfoermers-apitest-logger",
		Image: "igal1979/sidecar-proxy:avi",
		Ports: []corev1.ContainerPort{sideCarProt},
		Env:   []corev1.EnvVar{logUrl, proxyUrl},
	}
}

type SideCarData struct {
	Service      string `json:"service"`
	Deployment   string `json:"deployment"`
	CotanierPort int    `json:"cotanierPort"`
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {

		return value
	}
	return fallback
}
