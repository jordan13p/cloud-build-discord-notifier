package data

import (
	"context"
	"fmt"

	"github.com/google/go-github/v51/github"
	"golang.org/x/oauth2"

	"github.com/dlclark/regexp2"
	log "github.com/golang/glog"
)

func GetRespUrl(repoName string, triggerName string, triggerBuildConfigPath string) string {
	var serviceName string
	var respUrl string

	// github repo map k8s deployment name
	serviceMap := map[string]string{
		"server1": "game-ap",
		"web1":    "game-web",
		"server2": "admin-ap",
		"web2":    "admin-web",
	}

	re := regexp2.MustCompile(`^(.*)-((?!app).*)-(develop|feature|issue)$`, 0)
	if matched, err := re.MatchString(triggerName); matched {
		k8sInfo := "asia-east1/temp-cluster-01/namespace1"

		serviceName, _ = serviceMap[repoName]
		if serviceName == "" {
			serviceName = repoName
		}

		respUrl = fmt.Sprintf("https://console.cloud.google.com/kubernetes/service/%s/%s/overview?project=<PROJECT_ID>", k8sInfo, serviceName)

		if err != nil {
			log.Fatalf("fatal error: %v", err)
			respUrl = ""
		}

		return respUrl
	}

	re = regexp2.MustCompile(`^(.*)-((?!app).*)-(prod)$`, 0)
	matched, err := re.MatchString(triggerName)
	re = regexp2.MustCompile(`^(ci_cd)(?!_b2b)(.*\.yaml)$`, 0)
	matched2, err := re.MatchString(triggerBuildConfigPath)
	if matched && matched2 {
		k8sInfo := "asia-southeast1/prod-asia-01/namespace2"

		serviceName, _ = serviceMap[repoName]
		if serviceName == "" {
			serviceName = repoName
		}

		respUrl = fmt.Sprintf("https://console.cloud.google.com/kubernetes/service/%s/%s/overview?project=<PROJECT_ID>", k8sInfo, serviceName)

		if err != nil {
			log.Fatalf("fatal error: %v", err)
			respUrl = ""
		}

		return respUrl
	}

	re = regexp2.MustCompile(`^(.*)-((?!app).*)-(b2b-preprod)$`, 0)
	if matched, err := re.MatchString(triggerName); matched {
		k8sInfo := "asia-east1/temp-cluster-01/namespace3"

		serviceName, _ = serviceMap[repoName]
		if serviceName == "" {
			serviceName = repoName
		}

		respUrl = fmt.Sprintf("https://console.cloud.google.com/kubernetes/service/%s/%s/overview?project=<PROJECT_ID>", k8sInfo, serviceName)

		if err != nil {
			log.Fatalf("fatal error: %v", err)
			respUrl = ""
		}

		return respUrl
	}

	re = regexp2.MustCompile(`^(.*)-((?!app).*)-(prod)$`, 0)
	matched, err = re.MatchString(triggerName)
	re = regexp2.MustCompile(`^(ci_cd)(_b2b)(.*\.yaml)$`, 0)
	matched2, err = re.MatchString(triggerBuildConfigPath)
	if matched && matched2 {
		k8sInfo := "asia-southeast1/temp-cluster-02/namespace4"

		serviceName, _ = serviceMap[repoName]
		if serviceName == "" {
			serviceName = repoName
		}

		respUrl = fmt.Sprintf("https://console.cloud.google.com/kubernetes/service/%s/%s/overview?project=<PROJECT_ID>", k8sInfo, serviceName)

		if err != nil {
			log.Fatalf("fatal error: %v", err)
			respUrl = ""
		}

		return respUrl
	}

	respUrl = ""

	return respUrl
}

func GetCommitsAuthorName(repoName string, commitSha string) string {
	var commitUser string

	if repoName == "unknown" || commitSha == "unknown" {
		return ""
	}

	// setting github personal access tokens
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ""},
	)
	tc := oauth2.NewClient(ctx, ts)

	// use commit sha to get the commit author
	client := github.NewClient(tc)

	owner := "cloud-latitude"
	repo := repoName
	sha := commitSha
	commit, _, err := client.Repositories.GetCommit(ctx, owner, repo, sha, &github.ListOptions{})
	if err != nil {
		log.Fatalf("Error getting commit:", err)
	}

	commitUser = commit.GetCommit().GetAuthor().GetName()

	return commitUser
}
