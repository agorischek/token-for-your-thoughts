package destinations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"text/template"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
	"github.com/google/go-github/v74/github"
)

type GitHubDiscussionsDestination struct {
	client        *github.Client
	owner         string
	repo          string
	category      string
	categoryID    string
	titleTemplate *template.Template

	resolveOnce sync.Once
	repoID      string
	resolveErr  error
}

func NewGitHubDiscussionsDestination(cfg config.DestinationConfig) (*GitHubDiscussionsDestination, error) {
	owner, repo, err := splitRepository(cfg.Repository)
	if err != nil {
		return nil, err
	}

	titleTemplate, err := template.New("discussion-title").Parse(cfg.TitleTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse title template: %w", err)
	}

	client := github.NewClient((&http.Client{})).WithAuthToken(cfg.Token)

	return newGitHubDiscussionsDestination(client, cfg, owner, repo, titleTemplate), nil
}

func newGitHubDiscussionsDestination(client *github.Client, cfg config.DestinationConfig, owner, repo string, titleTemplate *template.Template) *GitHubDiscussionsDestination {
	return &GitHubDiscussionsDestination{
		client:        client,
		owner:         owner,
		repo:          repo,
		category:      strings.TrimSpace(cfg.Category),
		categoryID:    strings.TrimSpace(cfg.CategoryID),
		titleTemplate: titleTemplate,
	}
}

func (d *GitHubDiscussionsDestination) Name() string {
	return "github_discussions"
}

func (d *GitHubDiscussionsDestination) Submit(ctx context.Context, item feedback.Item) error {
	if err := d.resolveRepositoryAndCategory(ctx); err != nil {
		return err
	}

	title, err := d.renderTitle(item)
	if err != nil {
		return err
	}

	body := discussionBody(item)
	var response struct {
		CreateDiscussion struct {
			Discussion struct {
				ID string `json:"id"`
			} `json:"discussion"`
		} `json:"createDiscussion"`
	}

	variables := map[string]any{
		"input": map[string]any{
			"repositoryId": d.repoID,
			"categoryId":   d.categoryID,
			"title":        title,
			"body":         body,
		},
	}
	if err := d.graphQL(ctx, createDiscussionMutation, variables, &response); err != nil {
		return fmt.Errorf("create discussion: %w", err)
	}

	if strings.TrimSpace(response.CreateDiscussion.Discussion.ID) == "" {
		return fmt.Errorf("create discussion: GitHub returned an empty discussion id")
	}
	return nil
}

func (d *GitHubDiscussionsDestination) renderTitle(item feedback.Item) (string, error) {
	var rendered bytes.Buffer
	if err := d.titleTemplate.Execute(&rendered, item); err != nil {
		return "", fmt.Errorf("render title template: %w", err)
	}

	title := strings.TrimSpace(rendered.String())
	if title == "" {
		return "", fmt.Errorf("render title template: title cannot be empty")
	}
	return title, nil
}

func (d *GitHubDiscussionsDestination) resolveRepositoryAndCategory(ctx context.Context) error {
	d.resolveOnce.Do(func() {
		var response struct {
			Repository *struct {
				ID                   string `json:"id"`
				DiscussionCategories struct {
					Nodes []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
						Slug string `json:"slug"`
					} `json:"nodes"`
				} `json:"discussionCategories"`
			} `json:"repository"`
		}

		if err := d.graphQL(ctx, repositoryDiscussionCategoriesQuery, map[string]any{
			"owner": d.owner,
			"name":  d.repo,
		}, &response); err != nil {
			d.resolveErr = fmt.Errorf("load repository details: %w", err)
			return
		}
		if response.Repository == nil {
			d.resolveErr = fmt.Errorf("load repository details: repository %s/%s was not found", d.owner, d.repo)
			return
		}

		d.repoID = response.Repository.ID
		if strings.TrimSpace(d.categoryID) != "" {
			return
		}

		for _, category := range response.Repository.DiscussionCategories.Nodes {
			if strings.EqualFold(category.ID, d.category) || strings.EqualFold(category.Name, d.category) || strings.EqualFold(category.Slug, d.category) {
				d.categoryID = category.ID
				return
			}
		}

		d.resolveErr = fmt.Errorf("load repository details: could not find discussion category %q in %s/%s", d.category, d.owner, d.repo)
	})

	return d.resolveErr
}

func (d *GitHubDiscussionsDestination) graphQL(ctx context.Context, query string, variables map[string]any, out any) error {
	payload := map[string]any{
		"query":     query,
		"variables": variables,
	}

	req, err := d.client.NewRequest("POST", "graphql", payload)
	if err != nil {
		return fmt.Errorf("build GraphQL request: %w", err)
	}
	req = req.WithContext(ctx)

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if _, err := d.client.Do(ctx, req, &envelope); err != nil {
		return fmt.Errorf("send GraphQL request: %w", err)
	}
	if len(envelope.Errors) > 0 {
		messages := make([]string, 0, len(envelope.Errors))
		for _, graphQLError := range envelope.Errors {
			if strings.TrimSpace(graphQLError.Message) != "" {
				messages = append(messages, graphQLError.Message)
			}
		}
		if len(messages) == 0 {
			return fmt.Errorf("GraphQL request failed")
		}
		return fmt.Errorf("%s", strings.Join(messages, "; "))
	}
	if out == nil || len(envelope.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("decode GraphQL response: %w", err)
	}
	return nil
}

func discussionBody(item feedback.Item) string {
	var builder strings.Builder

	builder.WriteString(item.Feedback)
	builder.WriteString("\n\n")
	builder.WriteString("_From ")
	builder.WriteString(item.Provider)
	builder.WriteString(" via ")
	builder.WriteString(strings.ToUpper(item.Source))
	builder.WriteString(" at ")
	builder.WriteString(item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	builder.WriteString("_")

	if len(item.Metadata) > 0 {
		builder.WriteString("\n\n```json\n")
		builder.WriteString(item.MarkdownMetadataJSON())
		builder.WriteString("\n```")
	}

	return builder.String()
}

func splitRepository(repository string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(repository), "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("github_discussions destination repository must be in owner/repo form")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

const repositoryDiscussionCategoriesQuery = `
query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    id
    discussionCategories(first: 100) {
      nodes {
        id
        name
        slug
      }
    }
  }
}`

const createDiscussionMutation = `
mutation($input: CreateDiscussionInput!) {
  createDiscussion(input: $input) {
    discussion {
      id
    }
  }
}`
