package america

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
)

type Client interface {
	CreateDeployment(name string, JWT string, americaUrl string, projectID string, bundleID string, resourceType string, environment string, region string, parameters interface{}) (AmericaResponse, error)
	DeleteDeployment(JWT string, americaUrl string, resourceID string) (AmericaResponse, error)
	GetDeployment(deploymentId string, JWT string, americaUrl string) (AmericaDeploymentResponse, error)
	CreateBundle(JWT string, name string, america_url string, parentGroupID int) (string, error)
	CreateDynamicOperation(JWT string, americaUrl string, OperationName string, DeploymentID string, OperationProperties interface{}) (AmericaDynamicOperationResponse, error)
	GetDynamicOperation(JWT string, operationId string, americaUrl string) (AmericaDynamicOperationResponse, error)
	UpdateDeployment(JWT string, deploymentID string, name string, americaUrl string, resourceType string, environment string, region string, parameters interface{}) (AmericaResponse, error)
}

type client struct {
	log logging.Logger
}

type CreateParametersRequest struct {
	ResourceType string      `json:"Type"`
	Properties   interface{} `json:"properties"`
}

type CreateDeploymentRequest struct {
	ResourceType string                  `json:"type"`
	Name         string                  `json:"name"`
	Environment  string                  `json:"environment"`
	Region       string                  `json:"regionName"`
	Parameters   CreateParametersRequest `json:"parameters"`
}

type CreateDynamicOperationRequest struct {
	OperationName string      `json:"operationName"`
	Properties    interface{} `json:"properties"`
}

type CreateBundleRequest struct {
	Name          string `json:"name"`
	ParentGroupID int    `json:"parentGroupId"`
}

type AmericaDeploymentResponse struct {
	ID             string      `json:"uuid"`
	Name           string      `json:"name"`
	Status         string      `json:"status"`
	AdditionalInfo interface{} `json:"additionalInfo,omitempty"`
}

type AmericaDynamicOperationResponse struct {
	ID     int    `json:"id"`
	Status string `json:"operationStatus"`
}

type LatestOperationResponse struct {
	ID             int    `json:"id"`
	DeploymentUUID string `json:"deploymentUuid"`
}

type AmericaResponse struct {
	Name            string                  `json:"name"`
	LatestOperation LatestOperationResponse `json:"latestOperation"`
}

type CreateBundleResponse struct {
	BundleID string `json:"id"`
}

func (ac *client) CreateDeployment(name string, JWT string, americaUrl string, projectID string, bundleID string, resourceType string, environment string, region string, parameters interface{}) (AmericaResponse, error) {
	parametersRequest := CreateParametersRequest{
		ResourceType: resourceType,
		Properties:   parameters,
	}

	createDeploymentRequest := CreateDeploymentRequest{
		ResourceType: resourceType,
		Name:         name,
		Parameters:   parametersRequest,
		Region:       region,
		Environment:  environment,
	}

	url := fmt.Sprintf("%s/api/v2/projects/%s/bundles/%s/deployments", americaUrl, projectID, bundleID)

	headers := http.Header{
		"Authorization": []string{JWT},
	}

	response, err := ac.SendRequest("POST", url, headers, createDeploymentRequest)
	if err != nil {
		return AmericaResponse{}, err
	}

	fmt.Println(response.Body)
	var americaResponse = AmericaResponse{}

	if err := json.NewDecoder(response.Body).Decode(&americaResponse); err != nil {
		return AmericaResponse{}, errors.Wrap(err, "Cannot marshal the response json")
	}

	return americaResponse, nil
}

func (ac *client) CreateDynamicOperation(JWT string, americaUrl string, OperationName string, DeploymentID string, OperationProperties interface{}) (AmericaDynamicOperationResponse, error) {
	createDynamicOperationRequest := CreateDynamicOperationRequest{
		OperationName: OperationName,
		Properties:    OperationProperties,
	}

	url := fmt.Sprintf("%s/api/v2/deployments/%s/operations", americaUrl, DeploymentID)

	headers := http.Header{
		"Authorization": []string{JWT},
	}

	response, err := ac.SendRequest("POST", url, headers, createDynamicOperationRequest)

	if err != nil {
		return AmericaDynamicOperationResponse{}, err
	}

	var americaResponse = AmericaDynamicOperationResponse{}

	if err := json.NewDecoder(response.Body).Decode(&americaResponse); err != nil {
		return AmericaDynamicOperationResponse{}, errors.Wrap(err, "Cannot marshal the response json")
	}

	return americaResponse, nil
}

func (ac *client) DeleteDeployment(JWT string, americaUrl string, resourceID string) (AmericaResponse, error) {
	fmt.Println("Deleting deployment with ID: ", resourceID)
	url := fmt.Sprintf("%s/api/v2/deployments/%s", americaUrl, resourceID)

	headers := http.Header{
		"Authorization": []string{JWT},
	}

	response, err := ac.SendRequest("DELETE", url, headers, nil)
	if err != nil {
		return AmericaResponse{}, err
	}

	fmt.Println(response.Body)
	var americaResponse = AmericaResponse{}

	if err := json.NewDecoder(response.Body).Decode(&americaResponse); err != nil {
		return AmericaResponse{}, errors.Wrap(err, "Cannot marshal the response json")
	}

	return americaResponse, nil
}

func (ac *client) UpdateDeployment(JWT string, deploymentID string, name string, americaUrl string, resourceType string, environment string, region string, parameters interface{}) (AmericaResponse, error) {
	parametersRequest := CreateParametersRequest{
		ResourceType: resourceType,
		Properties:   parameters,
	}

	createDeploymentRequest := CreateDeploymentRequest{
		ResourceType: resourceType,
		Name:         name,
		Parameters:   parametersRequest,
		Region:       region,
		Environment:  environment,
	}

	url := fmt.Sprintf("%s/api/v2/deployments/%s", americaUrl, deploymentID)

	headers := http.Header{
		"Authorization": []string{JWT},
	}

	response, err := ac.SendRequest("PUT", url, headers, createDeploymentRequest)
	if err != nil {
		return AmericaResponse{}, err
	}
	fmt.Println(response.Body)
	var americaResponse = AmericaResponse{}

	if err := json.NewDecoder(response.Body).Decode(&americaResponse); err != nil {
		return AmericaResponse{}, errors.Wrap(err, "Cannot marshal the response json")
	}

	return americaResponse, nil
}

func (ac *client) CreateBundle(JWT string, name string, america_url string, parentGroupID int) (string, error) {
	createBundleRequest := CreateBundleRequest{
		Name:          name,
		ParentGroupID: parentGroupID,
	}

	jsonData, err := json.Marshal(createBundleRequest)
	if err != nil {
		fmt.Println("dorti")
		fmt.Println("Error marshalling JSON:", err)
		return "", errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.TopicList")
	}

	url := fmt.Sprintf("%s/api/v2/projects/10076/bundles", america_url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return "", errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.TopicList")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", JWT)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return "", errors.Wrap(err, "Got error from america")
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Assuming BUNDLE_UUID_KEY is a constant
	bundleUUID, ok := result["id"]
	if !ok {
		return "", errors.Wrap(err, "Got error from america")
	}

	bundleUUIDFloat, ok := bundleUUID.(float64)
	if !ok {
		// You might need to convert this properly depending on your UUID format
		return "", fmt.Errorf("unexpected UUID type: %T", bundleUUID)
	}

	bundleUUIDStr := fmt.Sprintf("%v", bundleUUIDFloat) // or "%f" or another format as needed

	return bundleUUIDStr, nil
}

func (ac *client) GetDeployment(deploymentId string, JWT string, americaUrl string) (AmericaDeploymentResponse, error) {
	url := fmt.Sprintf("%s/api/v2/deployments/%s", americaUrl, deploymentId)

	headers := http.Header{
		"Authorization": []string{JWT},
	}

	response, err := ac.SendRequest("GET", url, headers, nil)

	if err != nil {
		return AmericaDeploymentResponse{}, err
	}

	var americaDeploymentResponse = AmericaDeploymentResponse{}

	if err := json.NewDecoder(response.Body).Decode(&americaDeploymentResponse); err != nil {
		return AmericaDeploymentResponse{}, errors.Wrap(err, "Cannot marshal the response json")
	}

	return americaDeploymentResponse, nil
}

func (ac *client) GetDynamicOperation(JWT string, operationId string, americaUrl string) (AmericaDynamicOperationResponse, error) {
	url := fmt.Sprintf("%s/api/v2/deployment-operations/%s", americaUrl, operationId)

	headers := http.Header{
		"Authorization": []string{JWT},
	}

	response, err := ac.SendRequest("GET", url, headers, nil)

	if err != nil {
		return AmericaDynamicOperationResponse{}, err
	}

	var americaDeploymentResponse = AmericaDynamicOperationResponse{}

	if err := json.NewDecoder(response.Body).Decode(&americaDeploymentResponse); err != nil {
		return AmericaDynamicOperationResponse{}, errors.Wrap(err, "Cannot marshal the response json")
	}

	return americaDeploymentResponse, nil
}

// NewClient returns a new Http Client
func NewClient(log logging.Logger, authorizationToken string) (Client, error) {
	return &client{
		log: log,
	}, nil
}

// SendRequest sends an HTTP request and marshals the response into the provided object
func (ac *client) SendRequest(method string, url string, headers http.Header, body interface{}) (*http.Response, error) {
	// Create a new request
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Wrap(err, "cannot marshal request body")
		}
		// optional debug print
		fmt.Println("payload:", string(jsonData))
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return &http.Response{}, errors.Errorf("Failed to create request: %v", err)
	}

	// Set headers if provided
	if headers != nil {
		req.Header = headers
	}

	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return &http.Response{}, errors.Errorf("Failed to send request: %v", err)
	}
	//defer resp.Body.Close()

	return resp, nil
}
