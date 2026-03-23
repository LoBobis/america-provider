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
	CreateDeployment(ctx context.Context, name string,america_url string, resource_type string, parameters string) (creationResponse, error)
	GetDeployment(ctx context.Context, deployment_id string,america_url string) (AmericaDeploymentResponse, error)
}

type client struct {
	log logging.Logger
}


type CreateDeploymentRequest struct {
	ResourceType string `json:"resource_type"`
	Name string `json:"name"`
	Parameters string `json:"parameters"`
}

type AmericaDeploymentResponse struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
	Status     string                 `json:"status"`
}


type creationResponse struct {
        DeploymentId string `json:"deployment_id"`
		Message string `json:"message"`
}


func (ac *client) CreateDeployment(ctx context.Context, name string,america_url string, resource_type string, parameters string) (creationResponse, error){
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


func (ac *client) GetDeployment(ctx context.Context, deployment_id string,america_url string) (AmericaDeploymentResponse, error) {
	//fmt.Printf("Observing: %+v", cr.Status.AtProvider)
	url := america_url+"/deployments/" +  deployment_id
	ac.log.Info("AmericaURL", "url", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return AmericaDeploymentResponse{}, errors.Wrap(err, "Couldnt Create New Request") 
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return AmericaDeploymentResponse{}, errors.Wrap(err, "Couldnt send the new request")
	}
	defer resp.Body.Close()

	var c = AmericaDeploymentResponse{}

	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
        return AmericaDeploymentResponse{}, errors.Wrap(err, "Couldnt decode the response"+ deployment_id)
	}
	return c, nil
}




// NewClient returns a new Http Client
func NewClient(log logging.Logger, authorizationToken string) (Client, error) {
	return &client{
		log:                log,
	}, nil
}
