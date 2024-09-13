package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Settings  Settings   `yaml:"settings"`
	Download  Endpoint   `yaml:"download"`
	Upload    Endpoint   `yaml:"upload"`
	Artifacts []Artifact `yaml:"artifacts"`
}

type Settings struct {
	Temp          bool `yaml:"temp"`
	DebugDownload bool `yaml:"debug_download"`
	DebugUpload   bool `yaml:"debug_upload"`
}

type Endpoint struct {
	Url        string `yaml:"url"`
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	Token      string `yaml:"token"`
	AuthHeader string `yaml:"auth_header"`
}

type Artifact struct {
	GroupId    string `yaml:"group_id"`
	ArtifactId string `yaml:"artifact_id"`
	Version    string `yaml:"version"`
}

type AuthHeader struct {
	key   string
	value string
}

func getAuthHeader(endpoint Endpoint) *AuthHeader {
	var authHeader *AuthHeader
	if endpoint.User != "" && endpoint.Password != "" {
		data := endpoint.User + ":" + endpoint.Password
		encoded := base64.StdEncoding.EncodeToString([]byte(data))
		authHeader = &AuthHeader{
			"Authorization",
			encoded,
		}
	} else if endpoint.Token != "" {
		authHeader = &AuthHeader{
			"Authorization",
			"Bearer " + endpoint.Token,
		}
	} else if endpoint.AuthHeader != "" {
		split := strings.Split(endpoint.AuthHeader, ":")
		authHeader = &AuthHeader{
			split[0],
			split[1],
		}
	}
	return authHeader
}

func downloadArtifact(client *http.Client, download Endpoint, artifact Artifact) error {
	// Create the file

	baseUrl := download.Url + "/" + strings.Join(strings.Split(artifact.GroupId, "."), "/") + "/" + artifact.ArtifactId
	jarUrl := baseUrl + "/" + artifact.ArtifactId + "-" + artifact.Version + ".jar"
	aarUrl := baseUrl + "/" + artifact.ArtifactId + "-" + artifact.Version + ".aar"
	pomUrl := baseUrl + "/" + artifact.ArtifactId + "-" + artifact.Version + ".pom"

	jarFile := "temp/" + artifact.ArtifactId + "-" + artifact.Version + ".jar"
	aarFile := "temp/" + artifact.ArtifactId + "-" + artifact.Version + ".aar"
	pomFile := "temp/" + artifact.ArtifactId + "-" + artifact.Version + ".pom"

	authHeader := getAuthHeader(download)

	err := downloadFile(client, authHeader, jarUrl, jarFile)
	err = downloadFile(client, authHeader, aarUrl, aarFile)
	if err != nil {
		return err
	}
	err = downloadFile(client, authHeader, pomUrl, pomFile)
	return err
}

func downloadFile(client *http.Client, header *AuthHeader, url string, filePath string) error {
	fmt.Printf("Downloading file %s\n", filePath)
	os.Remove(filePath)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	if header != nil {
		req.Header.Add(header.key, header.value)
	}

	// Get the response from the URL
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Check if the response status is OK (200)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func uploadArtifacts(client *http.Client, upload Endpoint, artifact Artifact) error {
	baseUrl := upload.Url + "/" + strings.Join(strings.Split(artifact.GroupId, "."), "/") + "/" + artifact.ArtifactId
	jarUrl := baseUrl + "/" + artifact.ArtifactId + "-" + artifact.Version + ".jar"
	aarUrl := baseUrl + "/" + artifact.ArtifactId + "-" + artifact.Version + ".aar"
	pomUrl := baseUrl + "/" + artifact.ArtifactId + "-" + artifact.Version + ".pom"

	jarFile := "temp/" + artifact.ArtifactId + "-" + artifact.Version + ".jar"
	aarFile := "temp/" + artifact.ArtifactId + "-" + artifact.Version + ".aar"
	pomFile := "temp/" + artifact.ArtifactId + "-" + artifact.Version + ".pom"

	authHeader := getAuthHeader(upload)

	err := uploadFile(client, authHeader, jarUrl, jarFile)
	err = uploadFile(client, authHeader, aarUrl, aarFile)
	if err != nil {
		return err
	}
	err = uploadFile(client, authHeader, pomUrl, pomFile)
	return err
}

func uploadFile(client *http.Client, header *AuthHeader, url string, filePath string) error {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Prepare a buffer to hold the multipart form data
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Create a form file field
	part, err := writer.CreateFormFile("file", file.Name())
	if err != nil {
		return err
	}

	// Copy the file content into the form field
	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	// Close the writer to finalize the multipart form data
	err = writer.Close()
	if err != nil {
		return err
	}

	// Send the POST request to upload the file
	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return err
	}

	if header != nil {
		req.Header.Add(header.key, header.value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read and print the response
	responseBody, _ := io.ReadAll(resp.Body)
	fmt.Println(string(responseBody))

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("no file specified")
		return
	}
	configFile := os.Args[1]

	// Read the YAML file
	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Unmarshal the YAML data into the Config struct
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Print the parsed data
	fmt.Printf("Settings: %+v\n", config.Settings)
	fmt.Printf("Download: %+v\n", config.Download)
	fmt.Printf("Artifacts: %+v\n", config.Artifacts)

	// Create an HTTP client
	client := &http.Client{}

	os.MkdirAll("temp", os.ModePerm)
	defer os.RemoveAll("temp")

	for _, a := range config.Artifacts {
		err := downloadArtifact(client, config.Download, a)
		if err != nil {
			log.Printf("error downloading artifact %s: %v\n", a.ArtifactId, err)
			continue
		}
		err = uploadArtifacts(client, config.Upload, a)
		if err != nil {
			log.Printf("error uploading artifact %s: %v\n", a.ArtifactId, err)
			continue
		}
	}
}
