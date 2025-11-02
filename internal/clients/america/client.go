package america

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
)

type Client interface {
	CreateDeployment(ctx context.Context, name string,america_url string, resource_type string, parameters []byte) (creationResponse, error)
}

type client struct {
	log logging.Logger
}


type CreateDeploymentRequest struct {
	ResourceType string `json:"resource_type"`
	Name string `json:"name"`
	Parameters []byte `json:"parameters"`
}

type creationResponse struct {
        DeploymentId string `json:"deployment_id"`
		Message string `json:"message"`
}


func (ac *client) CreateDeployment(ctx context.Context, name string,america_url string, resource_type string, parameters []byte) (creationResponse, error){
	createDeploymentRequest:= CreateDeploymentRequest{
		ResourceType: resource_type,
		Name: name,
		Parameters: parameters,
	}

	jsonData, err := json.Marshal(createDeploymentRequest) 
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return creationResponse{}, errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.TopicList") 
	}
	req, err := http.NewRequest("POST", america_url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return creationResponse{}, errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.TopicList")
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return creationResponse{}, errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.TopicList")
	}
	defer resp.Body.Close()

	var c = creationResponse{}

	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
        return creationResponse{}, errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.TopicList") 
    }

	return c, nil
}



// NewClient returns a new Http Client
func NewClient(log logging.Logger, authorizationToken string) (Client, error) {
	return &client{
		log:                log,
	}, nil
}