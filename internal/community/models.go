package community

import (
	"time"

	"github.com/google/uuid"
)

type Post struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Content   string    `json:"content"`
	ImageURL  *string   `json:"image_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Comment struct {
	ID        uuid.UUID `json:"id"`
	PostID    uuid.UUID `json:"post_id"`
	UserID    uuid.UUID `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Like struct {
	ID        uuid.UUID `json:"id"`
	PostID    uuid.UUID `json:"post_id"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateCommentRequest struct {
	Content string `json:"content" validate:"required"`
}

// PostResponse includes author details and counts
type PostResponse struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	AuthorName   string    `json:"author_name"`
	AuthorAvatar *string   `json:"author_avatar"`
	Content      string    `json:"content"`
	ImageURL     *string   `json:"image_url"`
	LikeCount    int       `json:"like_count"`
	CommentCount int       `json:"comment_count"`
	IsLiked      bool      `json:"is_liked"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CommentResponse includes author details
type CommentResponse struct {
	ID           uuid.UUID `json:"id"`
	PostID       uuid.UUID `json:"post_id"`
	UserID       uuid.UUID `json:"user_id"`
	AuthorName   string    `json:"author_name"`
	AuthorAvatar *string   `json:"author_avatar"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}
