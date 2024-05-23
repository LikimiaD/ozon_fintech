package models

import (
	"time"
)

// Comment represents a comment on a post.
type Comment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PostID    uint      `gorm:"not null" json:"postId"`
	CommentID *uint     `gorm:"index" json:"commentId"` // ID of the parent comment
	Author    string    `gorm:"not null" json:"author"`
	Content   string    `gorm:"not null;size:2000" json:"content"`
	IsDeleted bool      `gorm:"not null" json:"isDeleted"`
	Replies   []Comment `gorm:"foreignKey:CommentID;constraint:OnDelete:CASCADE" json:"replies"`
	CreatedAt time.Time `gorm:"index" json:"createdAt"`
	UpdatedAt time.Time `gorm:"index" json:"updatedAt"`
}
