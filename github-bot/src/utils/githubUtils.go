package utils

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/v41/github"
)

// Gets all labels of a repo
func GetLabels(ctx context.Context, client *github.Client, owner, repo string)([]*github.Label) {
	opts := &github.ListOptions{PerPage: 100}
	repoLabels, _, err := client.Issues.ListLabels(ctx, owner, repo, opts)

	if err != nil {
		log.Printf("Error listing labels: %v", err)
	}
	return repoLabels
}

// AddLabels adds labels to a specified GitHub issue.
func AddLabels(ctx context.Context, client *github.Client, owner, repo string, issueNumber int, labels []string) {
	_, _, err := client.Issues.AddLabelsToIssue(ctx, owner, repo, issueNumber, labels)
	if err != nil {
		log.Printf("Error adding labels to issue #%d: %v", issueNumber, err)
	}
}

// CommentOnIssue comments on a specified GitHub issue.
func CommentOnIssue(ctx context.Context, client *github.Client, owner, repo string, issueNumber int, comment string) {
	commentRequest := &github.IssueComment{Body: &comment}
	_, _, err := client.Issues.CreateComment(ctx, owner, repo, issueNumber, commentRequest)
	if err != nil {
		log.Printf("Error commenting on issue #%d: %v", issueNumber, err)
	}
}

// CreatePriorityLabels creates a set of priority labels in the specified GitHub repository if they do not already exist.
func CreatePriorityLabels(ctx context.Context, client *github.Client, owner, repo string) error {
	// Define the priority labels to be created
	priorityLabels := []struct {
		Name        string
		Color       string
		Description string
	}{
		{"priority: critical", "800080", "Critical priority request. Must fix"},
		{"priority: high", "d81b60", "High priority request"},
		{"priority: medium", "e65100", "Medium priority request"},
		{"priority: low", "fdd835", "Low priority request"},
	}

	// Iterate over the priority labels and create them if they do not exist
	for _, label := range priorityLabels {
		exists, err := labelExists(ctx, client, owner, repo, label.Name)
		if err != nil {
			return fmt.Errorf("failed to check if label exists: %v", err)
		}

		if !exists {
			newLabel := &github.Label{
				Name:        github.String(label.Name),
				Color:       github.String(label.Color),
				Description: github.String(label.Description),
			}

			_, _, err := client.Issues.CreateLabel(ctx, owner, repo, newLabel)
			if err != nil {
				return fmt.Errorf("failed to create label '%s': %v", label.Name, err)
			}
			fmt.Printf("Label '%s' created successfully\n", label.Name)
		} else {
			fmt.Printf("Label '%s' already exists\n", label.Name)
		}
	}

	return nil
}

// labelExists checks if a label exists in the specified GitHub repository.
func labelExists(ctx context.Context, client *github.Client, owner, repo, labelName string) (bool, error) {
	labels, _, err := client.Issues.ListLabels(ctx, owner, repo, nil)
	if err != nil {
		return false, err
	}

	for _, label := range labels {
		if *label.Name == labelName {
			return true, nil
		}
	}
	return false, nil
}

func DeleteComment(ctx context.Context, client *github.Client, owner string, repo string, commentId int64) (bool, error) {
	_, err := client.Issues.DeleteComment(ctx, owner, repo, commentId)
	if err != nil {
		return false, err
	}
	return true, nil
}
