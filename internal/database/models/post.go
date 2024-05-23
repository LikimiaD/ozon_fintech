package models

import (
	"time"
)

// Post represents a blog post.
type Post struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Title           string    `gorm:"not null" json:"title"`
	Content         string    `gorm:"not null" json:"content"`
	Author          string    `gorm:"not null" json:"author"`
	CommentsEnabled bool      `gorm:"not null" json:"commentsEnabled"`
	Comments        []Comment `gorm:"foreignKey:PostID" json:"comments"`
	CreatedAt       time.Time `gorm:"index" json:"createdAt"`
	UpdatedAt       time.Time `gorm:"index" json:"updatedAt"`
}
