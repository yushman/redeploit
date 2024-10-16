package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"regexp"
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
	DebugDownload bool   `yaml:"debug_download"`
	DebugUpload   bool   `yaml:"debug_upload"`
	UploadMethod  string `yaml:"upload_method"`
	ArtifactsPath string `yaml:"artifacts_path"`
	SkipSources   bool   `yaml:"skip_sources"`
	SkipJavaDocs  bool   `yaml:"skip_javadocs"`
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
			"Basic " + encoded,
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

func getDir(settings Settings) string {
	var dir = "temp"
	if settings.ArtifactsPath != "" {
		dir = strings.TrimRight(settings.ArtifactsPath, "/")
	}
	return dir
}

func downloadArtifact(client *http.Client, config Config, artifact Artifact) ([]string, error) {
	// Create the file
	download := config.Download
	var links []string

	artifactPath := strings.Join(strings.Split(artifact.GroupId, "."), "/") + "/" + artifact.ArtifactId + "/" + artifact.Version + "/"

	baseUrl := strings.Trim(download.Url, "/") + "/" + artifactPath

	fmt.Printf("Downloading artifact from %s\n", baseUrl)

	// Try to extract all links from html (some repositories support this)
	// Regex pattern to find all href links
	pattern := "href=[\"']([^\"']+)[\"']"

	// Compile the regex
	re := regexp.MustCompile(pattern)

	req, err := http.NewRequest("GET", baseUrl, nil)
	if err != nil {
		return links, err
	}

	header := getAuthHeader(download)
	if header != nil {
		req.Header[header.key] = []string{header.value}
	}

	resp, err := client.Do(req)
	if err != nil {
		return links, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	html := string(body)

	// Find all matches
	matches := re.FindAllStringSubmatch(html, -1)

	// Extract and print the links
	for _, match := range matches {
		if len(match) > 1 {
			link := match[1]
			// Check if the link is a valid file
			if strings.Contains(link, ".jar") ||
				strings.Contains(link, ".aar") ||
				strings.Contains(link, ".pom") ||
				(!config.Settings.SkipSources && strings.Contains(link, "-sources.jar")) ||
				(!config.Settings.SkipJavaDocs && strings.Contains(link, "-javadoc.jar")) {
				links = append(links, link) // match[1] contains the captured URL
			}
		}
	}

	if config.Settings.DebugDownload {
		fmt.Println(html)
		fmt.Println("Extracted links:")
		for _, link := range links {
			fmt.Println(link)
		}
	}

	dir := getDir(config.Settings) + "/"

	for _, link := range links {
		url := baseUrl + link
		file := dir + artifactPath + link
		err = downloadFile(client, header, url, file, config.Settings.DebugDownload)
	}

	// If no links were found, try to download known file formats - jar, aar, pom, etc...
	if len(links) == 0 {
		jarUrl := baseUrl + artifact.ArtifactId + "-" + artifact.Version + ".jar"
		aarUrl := baseUrl + artifact.ArtifactId + "-" + artifact.Version + ".aar"
		pomUrl := baseUrl + artifact.ArtifactId + "-" + artifact.Version + ".pom"

		jarFile := dir + artifactPath + artifact.ArtifactId + "-" + artifact.Version + ".jar"
		aarFile := dir + artifactPath + artifact.ArtifactId + "-" + artifact.Version + ".aar"
		pomFile := dir + artifactPath + artifact.ArtifactId + "-" + artifact.Version + ".pom"

		links = append(links, jarFile)
		links = append(links, aarFile)
		links = append(links, pomFile)

		err = downloadFile(client, header, jarUrl, jarFile, config.Settings.DebugDownload)
		if err != nil {
			fmt.Println("No jar downloaded")
		}
		err = downloadFile(client, header, aarUrl, aarFile, config.Settings.DebugDownload)
		if err != nil {
			fmt.Println("No aar downloaded")
		}
		err = downloadFile(client, header, pomUrl, pomFile, config.Settings.DebugDownload)
		if err != nil {
			fmt.Println("No pom downloaded")
		}

		if !config.Settings.SkipSources {
			sourcesUrl := baseUrl + artifact.ArtifactId + "-" + artifact.Version + "-sources.jar"
			sourcesFile := dir + "/" + artifactPath + artifact.ArtifactId + "-" + artifact.Version + "-sources.jar"
			links = append(links, sourcesFile)
			err = downloadFile(client, header, sourcesUrl, sourcesFile, config.Settings.DebugDownload)
			if err != nil {
				fmt.Println("No sources downloaded")
			}
		}

		if !config.Settings.SkipSources {
			javadocUrl := baseUrl + artifact.ArtifactId + "-" + artifact.Version + "-javadoc.jar"
			javadocFile := dir + "/" + artifactPath + artifact.ArtifactId + "-" + artifact.Version + "-javadoc.jar"
			links = append(links, javadocFile)
			err = downloadFile(client, header, javadocUrl, javadocFile, config.Settings.DebugDownload)
			if err != nil {
				fmt.Println("No javadocs downloaded")
			}
		}

		return links, err

	}

	return links, err
}

func downloadFile(client *http.Client, header *AuthHeader, url string, filePath string, debugMode bool) error {
	fmt.Printf("Downloading file from %s to %s\n", url, filePath)

	if debugMode {
		return nil
	}

	os.Remove(filePath)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	if header != nil {
		req.Header[header.key] = []string{header.value}
	}

	d, err := httputil.DumpRequest(req, true)
	if err != nil {
		return nil
	}

	fmt.Printf("Download request: %q\n", d)

	// Get the response from the URL
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm)

	if err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	out, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer out.Close()

	// Check if the response status is OK (200)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err == nil {
		fmt.Println("Success")
	}
	return err
}

func uploadArtifacts(client *http.Client, config Config, artifact Artifact, files []string) error {
	upload := config.Upload

	baseUrl := strings.Trim(upload.Url, "/") + "/" + strings.Join(strings.Split(artifact.GroupId, "."), "/") + "/" + artifact.ArtifactId + "/" + artifact.Version
	fmt.Printf("Uploading artifacts to %s\n", baseUrl)

	dir := getDir(config.Settings)

	var err error

	authHeader := getAuthHeader(upload)

	var method = "POST"
	if config.Settings.UploadMethod == "PUT" {
		method = "PUT"
	}

	for _, file := range files {
		url := baseUrl + "/" + file
		filePath := dir + file
		err = uploadFile(client, authHeader, url, filePath, method, config.Settings.DebugUpload)
		if err != nil {
			fmt.Printf("Error uploading file %s\n    to %s\n    caused by %v\n", filePath, url, err)
		}
	}
	return err
}

func uploadFile(client *http.Client, header *AuthHeader, url string, filePath string, method string, debugMode bool) error {
	fmt.Printf("Uploading file from %s to %s\n", filePath, url)

	if debugMode {
		return nil
	}

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
	req, err := http.NewRequest(method, url, &b)
	if err != nil {
		return err
	}

	if header != nil {
		req.Header[header.key] = []string{header.value}
	}

	d, err := httputil.DumpRequest(req, true)
	if err != nil {
		return nil
	}

	fmt.Printf("Upload request: %q\n", d)

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

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
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

	// Make directories
	path := getDir(config.Settings)
	pathExists, err := exists(path)

	if err != nil {
		log.Fatalf("error: %v", err)
	}
	if !pathExists {
		os.MkdirAll(path, os.ModePerm)
	}

	if config.Settings.ArtifactsPath == "" {
		// if no need to save artifacts - remove temp directory
		defer os.RemoveAll(path)
	}

	// Download and upload artifacts
	for _, a := range config.Artifacts {
		files, err := downloadArtifact(client, config, a)
		if err != nil {
			log.Printf("error downloading artifact %s: %v\n", a.ArtifactId, err)
			continue
		}
		err = uploadArtifacts(client, config, a, files)
		if err != nil {
			log.Printf("error uploading artifact %s: %v\n", a.ArtifactId, err)
			continue
		}
	}
}
