package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"cloud-build-discord-notifier/data"

	cbpb "cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"github.com/GoogleCloudPlatform/cloud-build-notifiers/lib/notifiers"
	log "github.com/golang/glog"
)

const (
	webhookURLSecretName = "webhookUrl"
)

func main() {
	if err := notifiers.Main(new(discordNotifier)); err != nil {
		log.Fatalf("fatal error: %v", err)
	}
}

type discordNotifier struct {
	filter     notifiers.EventFilter
	webhookURL string
}

type field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type embed struct {
	Title       string  `json:"title"`
	Color       int     `json:"color"`
	Url         string  `json:"url"`
	Description string  `json:"description"`
	Fields      []field `json:"fields"`
}

type discordMessage struct {
	Content string  `json:"content"`
	Embeds  []embed `json:"embeds"`
}

func (s *discordNotifier) SetUp(ctx context.Context, cfg *notifiers.Config, sg notifiers.SecretGetter, _ notifiers.BindingResolver) error {
	if cfg.Spec.Notification.Filter != "" {
		prd, err := notifiers.MakeCELPredicate(cfg.Spec.Notification.Filter)
		if err != nil {
			return fmt.Errorf("failed to make a CEL predicate: %w", err)
		}
		s.filter = prd
	}

	wuRef, err := notifiers.GetSecretRef(cfg.Spec.Notification.Delivery, webhookURLSecretName)
	if err != nil {
		return fmt.Errorf("failed to get Secret ref from delivery config (%v) field %q: %w", cfg.Spec.Notification.Delivery, webhookURLSecretName, err)
	}
	wuResource, err := notifiers.FindSecretResourceName(cfg.Spec.Secrets, wuRef)
	if err != nil {
		return fmt.Errorf("failed to find Secret for ref %q: %w", wuRef, err)
	}
	wu, err := sg.GetSecret(ctx, wuResource)
	if err != nil {
		return fmt.Errorf("failed to get token secret: %w", err)
	}
	s.webhookURL = wu

	return nil
}

func (s *discordNotifier) SendNotification(ctx context.Context, build *cbpb.Build) error {
	if s.filter != nil && s.filter.Apply(ctx, build) {
		return nil
	}

	log.Infof("sending discord webhook for Build %q (status: %q)", build.Id, build.Status)
	msg, err := s.buildMessage(build)
	if err != nil {
		return fmt.Errorf("failed to write discord message: %w", err)
	}
	if msg == nil {
		return nil
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("Unable to marshal payload %w", err)
	}

	log.Infof("sending payload %s", string(payload))
	resp, err := http.Post(s.webhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	log.Infof("got resp %+v", resp)

	return nil
}

func (s *discordNotifier) buildMessage(build *cbpb.Build) (*discordMessage, error) {
	var url string
	var fields []field
	var embeds []embed

	sourceText := ""
	sourceRepo := build.Source.GetRepoSource()
	// log.Infof("repo info %+v", sourceRepo)
	if sourceRepo != nil {
		sourceText = sourceRepo.GetRepoName()
	}

	// get details from build.Substitutions[]
	repoName, ok := build.Substitutions["REPO_NAME"]
	if !ok {
		repoName = "unknown"
		log.Infof("Repository name %q not present in Substitutions", repoName)
	}
	refName, ok := build.Substitutions["REF_NAME"]
	if !ok {
		refName = "unknown"
		log.Infof("Refer name %q not present in Substitutions", refName)
	}
	triggerBuildConfigPath, ok := build.Substitutions["TRIGGER_BUILD_CONFIG_PATH"]
	if !ok {
		triggerBuildConfigPath = "unknown"
		log.Infof("Trigger name %q not present in Substitutions", triggerBuildConfigPath)
	}
	triggerName, ok := build.Substitutions["TRIGGER_NAME"]
	if !ok {
		triggerName = "unknown"
		log.Infof("Trigger name %q not present in Substitutions", triggerName)
	}
	shortSha, ok := build.Substitutions["SHORT_SHA"]
	if !ok {
		shortSha = "unknown"
		log.Infof("Short SHA %q not present in Substitutions", shortSha)
	}
	commitSha, ok := build.Substitutions["COMMIT_SHA"]
	if !ok {
		commitSha = "unknown"
		log.Infof("Commit SHA %q not present in Substitutions", commitSha)
	}

	commitShaUrl := fmt.Sprintf("[%s](https://github.com/your_repo/%s/commit/%s)", shortSha, repoName, commitSha)
	if shortSha == "" || repoName == "unknown" || commitSha == "" {
		commitShaUrl = "unknown"
		log.Infof("Commit SHA Url can not get from SHORT_SHA, COMMIT_SHA or REPO_NAME")
	}

	// get commit author name from github api, and change to discord id
	commitUser := data.GetCommitsAuthorName(repoName, commitSha)
	if commitUser == "" {
		commitUser = "unknown"
	}

	// append detail data to fields
	fields = append(fields,
		field{
			Name:   "Repository",
			Value:  repoName,
			Inline: true,
		},
		field{
			Name:   "Refer",
			Value:  refName,
			Inline: true,
		},
		field{
			Name:   "Trigger Name",
			Value:  triggerName,
			Inline: false,
		},
		field{
			Name:   "Trigger Config",
			Value:  triggerBuildConfigPath,
			Inline: false,
		},
		field{
			Name:   "Commit SHA",
			Value:  commitShaUrl,
			Inline: true,
		},
		field{
			Name:   "Commit Author",
			Value:  commitUser,
			Inline: true,
		},
	)

	// get GKE Services & Ingress URL from local packages
	url = data.GetRespUrl(repoName, triggerName, triggerBuildConfigPath)
	if url == "" {
		url = build.LogUrl
	}

	switch build.Status {
	case cbpb.Build_WORKING:
		embeds = append(embeds, embed{
			Title:       "🔨 CI/CD BUILDING",
			Color:       254200,
			Url:         build.LogUrl,
			Description: "👆 Click link to view status",
			Fields:      fields,
		})

		return &discordMessage{
			Embeds: embeds,
		}, nil
	case cbpb.Build_SUCCESS:
		embeds = append(embeds, embed{
			Title:       "✅ CI/CD SUCCESS",
			Color:       6748163,
			Url:         url,
			Description: "👆 Click link to view status",
			Fields:      fields,
		},
		)

		return &discordMessage{
			Embeds: embeds,
		}, nil
	case cbpb.Build_FAILURE, cbpb.Build_INTERNAL_ERROR, cbpb.Build_TIMEOUT:
		embeds = append(embeds, embed{
			Title:       fmt.Sprintf("❌ CI/CD ERROR - %s", build.Status),
			Color:       16253797,
			Url:         build.LogUrl,
			Description: "👆 Click link to view status",
			Fields:      fields,
		},
		)

		return &discordMessage{
			// Content: content,
			Embeds: embeds,
		}, nil
	default:
		log.Infof("Unknown status %s", build.Status)

		if len(embeds) > 0 && len(sourceText) > 0 {
			embeds[0].Description = sourceText
		}

		if len(embeds) == 0 {
			log.Infof("unhandled status - skipping notification %s", build.Status)
			return nil, nil
		}

		return &discordMessage{
			Embeds: embeds,
		}, nil
	}
}
