package project

import (
	"fmt"
	"net/url"
	"strings"
)

type TemplateProvider string

const (
	ProviderGitHub    TemplateProvider = "github"
	ProviderGitLab    TemplateProvider = "gitlab"
	ProviderBitbucket TemplateProvider = "bitbucket"
)

func ResolveTemplateURL(from string, provider TemplateProvider) (string, error) {
	from = strings.TrimSpace(from)
	if from == "" {
		return "", fmt.Errorf("template reference cannot be empty")
	}

	// If it's already a full URL, leave it as is.
	if strings.Contains(from, "://") {
		// basic validation
		if _, err := url.Parse(from); err != nil {
			return "", fmt.Errorf("invalid URL: %w", err)
		}
		return from, nil
	}

	// Short form expected: "owner/repo"
	parts := strings.Split(from, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid template reference %q, expected owner/repo", from)
	}
	owner, repo := parts[0], parts[1]
	if owner == "" || repo == "" {
		return "", fmt.Errorf("invalid template reference %q, expected owner/repo", from)
	}

	switch provider {
	case ProviderGitHub, "":
		return fmt.Sprintf("https://github.com/%s/%s.git", owner, repo), nil
	case ProviderGitLab:
		return fmt.Sprintf("https://gitlab.com/%s/%s.git", owner, repo), nil
	case ProviderBitbucket:
		return fmt.Sprintf("https://bitbucket.org/%s/%s.git", owner, repo), nil
	default:
		return "", fmt.Errorf("unsupported provider %q (supported: github, gitlab, bitbucket)", provider)
	}
}
