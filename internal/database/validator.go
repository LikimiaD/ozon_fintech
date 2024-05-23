package database

import (
	"errors"
	"github.com/likimiad/ozon_fintech/internal/database/models"
)

var (
	ErrEmptyTitle   = errors.New("title cannot be empty")
	ErrEmptyContent = errors.New("content cannot be empty")
	ErrContentLimit = errors.New("content exceeds maximum length of 2000 characters")
	ErrEmptyAuthor  = errors.New("author cannot be empty")
)

// validatePost checks if the post has valid fields.
func (s *PostService) validatePost(post *models.Post) error {
	if post.Title == "" {
		return ErrEmptyTitle
	}
	if post.Content == "" {
		return ErrEmptyContent
	}
	if post.Author == "" {
		return ErrEmptyAuthor
	}
	return nil
}

// validateComment checks if the comment has valid fields.
func (s *PostService) validateComment(comment *models.Comment) error {
	if comment.Content == "" {
		return ErrEmptyContent
	}
	if len(comment.Content) > 2000 {
		return ErrContentLimit
	}
	if comment.Author == "" {
		return ErrEmptyAuthor
	}
	return nil
}
