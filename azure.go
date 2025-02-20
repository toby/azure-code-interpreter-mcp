package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

const apiVersion = "2024-02-02-preview"

type azureClient struct {
	baseURL string
	token   string
}

func newAzureClient(c config) (*azureClient, error) {
	u, err := baseURL(c)
	if err != nil {
		return nil, err
	}
	t, err := authToken()
	if err != nil {
		return nil, err
	}
	return &azureClient{
		baseURL: u,
		token:   t,
	}, nil
}

func (c *azureClient) execute(sessionID string, code string) (string, error) {
	er := executeRequest{}
	er.Properties.CodeInputType = "inline"
	er.Properties.ExecutionType = "synchronous"
	er.Properties.Code = code
	jbuf := new(bytes.Buffer)
	err := json.NewEncoder(jbuf).Encode(er)
	if err != nil {
		return "", fmt.Errorf("failed to encode JSON: %w", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/code/execute?api-version=%s&identifier=%s", c.baseURL, apiVersion, sessionID), jbuf)
	if err != nil {
		return "", fmt.Errorf("failed create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.authRequest(req)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	exr := &executeResponse{}
	err = json.NewDecoder(resp.Body).Decode(exr)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	log.Println("HTTP request status: %s", resp.Status)
	log.Printf("/code/exec request status: %s\n", exr.Properties.Status)
	if exr.Properties.Stdout != "" {
		w := new(bytes.Buffer)
		fmt.Printf("/code/execute request stdout:\n\n%s\n", exr.Properties.Stdout)
		io.Copy(w, strings.NewReader(exr.Properties.Stdout))
		return w.String(), nil
	}
	return "", nil
	// if exr.Properties.Stderr != "" {
	// 	fmt.Printf("/code/execute equest stderr:\n\n%s\n", exr.Properties.Stderr)
	// 	io.Copy(w, strings.NewReader(exr.Properties.Stderr))
	// }
}

func (c *azureClient) listFiles(sessionID string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/files?api-version=%s&identifier=%s", c.baseURL, apiVersion, sessionID), nil)
	if err != nil {
		return "", fmt.Errorf("failed create request: %w", err)
	}
	c.authRequest(req)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	fr := &filesResponse{}
	err = json.NewDecoder(resp.Body).Decode(fr)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	fs := make([]string, 0, len(fr.Value))
	for _, v := range fr.Value {
		fs = append(fs, v.Properties.FileName)
	}
	fsr := strings.Join(fs, "\n")
	log.Printf("Files in session %s:\n%s\n", sessionID, fsr)
	return fsr, nil
}

func (c *azureClient) getFile(sessionID string, fileName string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/files/content/%s?api-version=%s&identifier=%s", c.baseURL, fileName, apiVersion, sessionID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed create request: %w", err)
	}
	c.authRequest(req)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	log.Printf("Get file %s in session %s:", fileName, sessionID)
	fc := bytes.NewBuffer([]byte{})
	fc.ReadFrom(resp.Body)
	return fc.Bytes(), nil
}

type requestProperties struct {
	CodeInputType string `json:"codeInputType"`
	ExecutionType string `json:"executionType"`
	Code          string `json:"code"`
}

type executeRequest struct {
	Properties requestProperties `json:"properties"`
}

type executeResponseProperties struct {
	Status string `json:"status"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

type executeResponse struct {
	Properties executeResponseProperties `json:"properties"`
}

type filesResponse struct {
	Value []filesResponseValue `json:"value"`
}

type filesResponseValue struct {
	Properties filesResponseProperties `json:"properties"`
}

type filesResponseProperties struct {
	FileName         string `json:"filename"`
	Size             int    `json:"size"`
	LastModifiedTime string `json:"lastModifiedTime"`
}

func (c *azureClient) authRequest(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
}

func authToken() (string, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	opts := policy.TokenRequestOptions{
		Scopes: []string{"https://dynamicsessions.io/.default"},
	}
	tok, err := cred.GetToken(ctx, opts)
	if err != nil {
		return "", err
	}
	return tok.Token, nil
}

func baseURL(c config) (string, error) {
	if c.SubscriptionID == "" {
		return "", fmt.Errorf("AZURE_SUBSCRIPTION_ID is required")
	}
	if c.ResourceGroup == "" {
		return "", fmt.Errorf("AZURE_RESOURCE_GROUP is required")
	}
	if c.SessionPool == "" {
		return "", fmt.Errorf("AZURE_SESSION_POOL is required")
	}
	if c.Region == "" {
		return "", fmt.Errorf("AZURE_REGION is required")
	}
	return fmt.Sprintf("https://%s.dynamicsessions.io/subscriptions/%s/resourceGroups/%s/sessionPools/%s", c.Region, c.SubscriptionID, c.ResourceGroup, c.SessionPool), nil
}
