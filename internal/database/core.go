package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/likimiad/ozon_fintech/internal/database/models"
	"gorm.io/gorm"
	"log/slog"
)

const (
	CacheTTL        = time.Hour // ? Cache Time-to-Live
	MemoryThreshold = 0.75      // ? Memory usage threshold for Redis
)

var (
	ErrPostDisabled = errors.New("comments are disabled for this post")
	ErrNotFound     = errors.New("record not found")
)

type PostService struct {
	DB *Database
	RC *redis.Client
}

// NewPostService creates a new PostService instance.
func NewPostService(db *Database, rc *redis.Client) *PostService {
	return &PostService{
		DB: db,
		RC: rc,
	}
}

// CreatePost adds a new post to the database and updates the cache.
func (s *PostService) CreatePost(post *models.Post) error {
	if err := s.validatePost(post); err != nil {
		return err
	}

	slog.Info("creating new post", "title", post.Title, "author", post.Author)

	if post.CreatedAt.IsZero() {
		post.CreatedAt = time.Now()
	}

	err := s.DB.Create(post).Error
	if err != nil {
		slog.Error("error creating post", "title", post.Title, "error", err)
		return err
	}

	// Clear and update post cache
	s.clearCache("posts")
	var posts []models.Post
	if err := s.DB.Preload("Comments.Replies").Find(&posts).Error; err != nil {
		slog.Error("error fetching posts from database to update cache", "error", err)
		return err
	}
	s.setToCache("posts", posts)

	return nil
}

// UpdatePost modifies an existing post and updates the cache.
func (s *PostService) UpdatePost(post *models.Post) error {
	if err := s.validatePost(post); err != nil {
		return err
	}

	err := s.DB.Save(post).Error
	if err != nil {
		slog.Error("error updating post", "title", post.Title, "error", err)
		return err
	}

	// Очистка и обновление кэша постов
	s.clearCache("posts")
	var posts []models.Post
	if err := s.DB.Preload("Comments.Replies").Find(&posts).Error; err != nil {
		slog.Error("error fetching posts from database to update cache", "error", err)
		return err
	}
	s.setToCache("posts", posts)

	// Очистка и обновление кэша конкретного поста
	s.clearCache(fmt.Sprintf("post:%d", post.ID))
	s.setToCache(fmt.Sprintf("post:%d", post.ID), post)

	return nil
}

// DeletePost removes a post and its comments from the database and cache.
func (s *PostService) DeletePost(id uint) error {
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("post_id = ?", id).Delete(&models.Comment{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.Post{}, id).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	// Очистка кэша
	s.clearCache("posts")
	s.clearCache(fmt.Sprintf("comments:%d", id))
	s.clearCache(fmt.Sprintf("post:%d", id))

	return nil
}

// GetPosts retrieves all posts, using cache if available.
func (s *PostService) GetPosts() ([]models.Post, error) {
	var posts []models.Post
	cacheKey := "posts"

	if err := s.getFromCache(cacheKey, &posts); err == nil {
		slog.Info("cache hit for posts")
		for i := range posts {
			if err := s.getFromCache(fmt.Sprintf("comments:%d", posts[i].ID), &posts[i].Comments); err == nil {
				for j := range posts[i].Comments {
					err = s.preloadReplies(&posts[i].Comments[j])
					if err != nil {
						slog.Error("error preloading replies for comment", "comment_id", posts[i].Comments[j].ID, "error", err)
						return nil, err
					}
				}
			}
		}
		return posts, nil
	}

	slog.Info("cache miss for posts, querying database")
	result := s.DB.Preload("Comments.Replies").Find(&posts)
	if result.Error != nil {
		slog.Error("error fetching posts from database", "error", result.Error)
		return nil, result.Error
	}

	s.setToCache(cacheKey, posts)
	for i := range posts {
		s.setToCache(fmt.Sprintf("comments:%d", posts[i].ID), posts[i].Comments)
		for j := range posts[i].Comments {
			err := s.preloadReplies(&posts[i].Comments[j])
			if err != nil {
				slog.Error("error preloading replies for comment", "comment_id", posts[i].Comments[j].ID, "error", err)
				return nil, err
			}
		}
	}
	slog.Info("posts and comments cached", "posts", posts)
	return posts, nil
}

// GetPostByID retrieves a single post by ID, using cache if available.
func (s *PostService) GetPostByID(id uint) (*models.Post, error) {
	var post models.Post
	cacheKey := fmt.Sprintf("post:%d", id)

	if err := s.getFromCache(cacheKey, &post); err == nil {
		slog.Info("cache hit for post", "post_id", id)
		if err := s.getFromCache(fmt.Sprintf("comments:%d", id), &post.Comments); err == nil {
			slog.Info("cache hit for comments", "post_id", id)
			for i := range post.Comments {
				err = s.preloadReplies(&post.Comments[i])
				if err != nil {
					slog.Error("error preloading replies for comment", "comment_id", post.Comments[i].ID, "error", err)
					return nil, err
				}
			}
			return &post, nil
		}
		return &post, nil
	}

	slog.Info("cache miss for post", "post_id", id, "operation", "querying database")
	result := s.DB.Preload("Comments.Replies").First(&post, id)
	if result.Error != nil {
		slog.Error("error fetching post from database", "post_id", id, "error", result.Error)
		return nil, result.Error
	}

	s.setToCache(cacheKey, post)
	s.setToCache(fmt.Sprintf("comments:%d", id), post.Comments)
	for i := range post.Comments {
		err := s.preloadReplies(&post.Comments[i])
		if err != nil {
			slog.Error("error preloading replies for comment", "comment_id", post.Comments[i].ID, "error", err)
			return nil, err
		}
	}
	slog.Info("post and comments cached", "post_id", id, "post", post, "comments", post.Comments)
	return &post, nil
}

// CreateComment adds a new comment to a post and updates the cache.
func (s *PostService) CreateComment(postID uint, commentID *uint, author, content string) (*models.Comment, error) {
	comment := &models.Comment{
		PostID:    postID,
		CommentID: commentID,
		Author:    author,
		Content:   content,
	}

	if err := s.validateComment(comment); err != nil {
		return nil, err
	}

	var post models.Post
	if err := s.DB.First(&post, comment.PostID).Error; err != nil {
		slog.Error("error fetching post for comment", "post_id", comment.PostID, "error", err)
		return nil, err
	}

	if !post.CommentsEnabled {
		slog.Warn("attempt to add comment to disabled post", "post_id", comment.PostID, "error", ErrPostDisabled)
		return nil, ErrPostDisabled
	}

	if comment.CommentID != nil {
		var parentComment models.Comment
		if err := s.DB.First(&parentComment, *comment.CommentID).Error; err != nil {
			slog.Error("error fetching parent comment", "comment_id", *comment.CommentID, "error", err)
			return nil, err
		}
	}

	slog.Info("creating new comment", "author", comment.Author, "post_id", comment.PostID)

	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = time.Now()
	}

	err := s.DB.Create(comment).Error
	if err != nil {
		slog.Error("error creating comment", "author", comment.Author, "error", err)
		return nil, err
	}

	s.clearCache(fmt.Sprintf("comments:%d", comment.PostID))
	var comments []models.Comment
	if err := s.DB.Where("post_id = ?", comment.PostID).Preload("Replies").Find(&comments).Error; err != nil {
		slog.Error("error fetching comments from database to update cache", "post_id", comment.PostID, "error", err)
		return nil, err
	}
	s.setToCache(fmt.Sprintf("comments:%d", comment.PostID), comments)

	return comment, nil
}

// UpdateComment modifies an existing comment and updates the cache.
func (s *PostService) UpdateComment(id uint, content string) (*models.Comment, error) {
	var comment models.Comment
	if err := s.DB.First(&comment, id).Error; err != nil {
		return nil, err
	}

	comment.Content = content
	comment.UpdatedAt = time.Now()

	if err := s.validateComment(&comment); err != nil {
		return nil, err
	}

	if err := s.DB.Save(&comment).Error; err != nil {
		return nil, err
	}

	s.clearCache(fmt.Sprintf("comments:%d", comment.PostID))
	var comments []models.Comment
	if err := s.DB.Where("post_id = ?", comment.PostID).Preload("Replies").Find(&comments).Error; err != nil {
		slog.Error("error fetching comments from database to update cache", "post_id", comment.PostID, "error", err)
		return nil, err
	}
	s.setToCache(fmt.Sprintf("comments:%d", comment.PostID), comments)

	return &comment, nil
}

// DeleteComment logically deletes a comment and updates the cache.
func (s *PostService) DeleteComment(id uint) error {
	var comment models.Comment
	if err := s.DB.First(&comment, id).Error; err != nil {
		return err
	}

	comment.IsDeleted = true
	comment.Content = "Comment deleted by user"
	comment.UpdatedAt = time.Now()

	if err := s.DB.Save(&comment).Error; err != nil {
		return err
	}

	s.clearCache(fmt.Sprintf("comments:%d", comment.PostID))
	var comments []models.Comment
	if err := s.DB.Where("post_id = ?", comment.PostID).Preload("Replies").Find(&comments).Error; err != nil {
		slog.Error("error fetching comments from database to update cache", "post_id", comment.PostID, "error", err)
		return err
	}
	s.setToCache(fmt.Sprintf("comments:%d", comment.PostID), comments)

	return nil
}

// getFromCache retrieves data from Redis cache.
func (s *PostService) getFromCache(key string, dest interface{}) error {
	data, err := s.RC.Get(context.Background(), key).Result()
	if errors.Is(err, redis.Nil) {
		return ErrNotFound
	} else if err != nil {
		slog.Warn("error fetching data from cache", "key", key, "error", err)
		return err
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		slog.Warn("error unmarshaling cached data", "key", key, "error", err)
		return err
	}

	return nil
}

// setToCache stores data in Redis cache.
func (s *PostService) setToCache(key string, value interface{}) {
	data, err := json.Marshal(value)
	if err != nil {
		slog.Warn("failed to marshal data for caching", "key", key, "error", err)
		return
	}

	err = s.RC.Set(context.Background(), key, data, CacheTTL).Err()
	if err != nil {
		slog.Warn("failed to set data to cache", "key", key, "error", err)
		return
	}

	slog.Info("data set to cache", "key", key)
	s.checkMemoryUsage()
}

// clearCache removes cache entries matching a pattern.
func (s *PostService) clearCache(pattern string) {
	keys, err := s.RC.Keys(context.Background(), pattern).Result()
	if err != nil {
		slog.Warn("failed to get keys for clearing cache", "pattern", pattern, "error", err)
		return
	}

	for _, key := range keys {
		s.RC.Del(context.Background(), key)
		slog.Info("cleared cache", "key", key)
	}
}

// checkMemoryUsage checks Redis memory usage and clears expired keys if needed.
func (s *PostService) checkMemoryUsage() {
	info, err := s.RC.Info(context.Background(), "memory").Result()
	if err != nil {
		slog.Warn("failed to get Redis memory info", "error", err)
		return
	}

	var usedMemory, totalMemory int64
	fmt.Sscanf(info, "used_memory:%d\ntotal_system_memory:%d\n", &usedMemory, &totalMemory)

	slog.Info("information about redis service", "used_memory", usedMemory, "total_system_memory", totalMemory)

	if float64(usedMemory)/float64(totalMemory) > MemoryThreshold {
		s.clearExpiredKeys()
	}
}

// clearExpiredKeys clears keys from Redis that have expired TTL.
func (s *PostService) clearExpiredKeys() {
	keys, err := s.RC.Keys(context.Background(), "*").Result()
	if err != nil {
		slog.Warn("failed to get keys for clearing expired cache", "error", err)
		return
	}

	for _, key := range keys {
		ttl, err := s.RC.TTL(context.Background(), key).Result()
		if err != nil || ttl < 0 {
			s.RC.Del(context.Background(), key)
			slog.Info("cleared expired cache", "key", key)
		}
	}
}

// PreloadComments preloads comments for a given post.
func (s *PostService) PreloadComments(post *models.Post) error {
	err := s.DB.Preload("Comments").Find(&post.Comments).Error
	if err != nil {
		slog.Error("error preloading comments for post", "post_id", post.ID, "error", err)
		return err
	}
	for i := range post.Comments {
		err = s.preloadReplies(&post.Comments[i])
		if err != nil {
			slog.Error("error preloading replies for comment", "comment_id", post.Comments[i].ID, "error", err)
			return err
		}
	}
	slog.Info("successfully preloaded comments and replies for post", "post_id", post.ID)
	return nil
}

// preloadReplies recursively preloads replies for a given comment.
func (s *PostService) preloadReplies(comment *models.Comment) error {
	err := s.DB.Preload("Replies").Find(&comment.Replies, "comment_id = ?", comment.ID).Error
	if err != nil {
		slog.Error("error preloading replies for comment", "comment_id", comment.ID, "error", err)
		return err
	}
	for i := range comment.Replies {
		err = s.preloadReplies(&comment.Replies[i])
		if err != nil {
			slog.Error("error preloading nested replies for comment", "comment_id", comment.Replies[i].ID, "error", err)
			return err
		}
	}
	return nil
}
