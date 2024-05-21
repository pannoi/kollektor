package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func sendRequest(url string) (map[string]interface{}, error) {
	var release map[string]interface{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if os.Getenv("GITHUB_TOKEN") != "" && strings.Contains(url, "github.com") {
		req.Header.Set("Authorization", "Bearer "+os.Getenv("GITHUB_TOKEN"))
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	err = json.Unmarshal(body, &release)
	if err != nil {
		return nil, err
	}

	return release, nil
}

func GetHelmReleaseNotes(repo string) (string, string, error) {
	artifactHubUrl := strings.Replace(repo, "/packages/", "/api/v1/packages/", 1)
	if !strings.HasPrefix(artifactHubUrl, "https://") {
		artifactHubUrl = "https://" + artifactHubUrl
	}

	release, err := sendRequest(artifactHubUrl)
	if err != nil {
		return "", "", err
	}

	var releaseUrl string
	for _, el := range release["links"].([]map[string]string) {
		if strings.Contains(el["url"], "helm") {
			releaseUrl = el["url"]
			break
		}
	}
	releaseNotes, err := GetProjectReleaseNotes(releaseUrl)
	if err != nil {
		return "", "", err
	}

	return releaseNotes, releaseUrl, nil

}

func GetHelmChartVersion(repo string) (string, error) {
	artifactHubUrl := strings.Replace(repo, "/packages/", "/api/v1/packages/", 1)
	if !strings.HasPrefix(artifactHubUrl, "https://") {
		artifactHubUrl = "https://" + artifactHubUrl
	}

	release, err := sendRequest(artifactHubUrl)
	if err != nil {
		return "", nil
	}

	return release["version"].(string), nil
}

func GetProjectReleaseNotes(repo string) (string, error) {
	githubUrl := strings.Replace(repo, "github.com", "api.github.com/repos", 1)
	githubUrl += "/releases/latest"
	if !strings.HasPrefix(githubUrl, "https://") {
		githubUrl = "https://" + githubUrl
	}

	release, err := sendRequest(githubUrl)
	if err != nil {
		return "", nil
	}

	return release["body"].(string), nil
}

func GetProjectVersion(repo string) (string, error) {
	githubUrl := strings.Replace(repo, "github.com", "api.github.com/repos", 1)
	githubUrl += "/releases/latest"
	if !strings.HasPrefix(githubUrl, "https://") {
		githubUrl = "https://" + githubUrl
	}

	release, err := sendRequest(githubUrl)
	if err != nil {
		return "", nil
	}

	var version string
	if name, ok := release["name"].(string); ok && name != "" {
		version = name
	} else if tagName, ok := release["tag_name"].(string); ok && tagName != "" {
		version = tagName
	} else {
		fmt.Println(release)
		return "", fmt.Errorf("no valid release name or tag found in response")
	}

	index := strings.Index(version, " ")
	if index != -1 {
		version = version[:index]
	}

	return version, nil
}
