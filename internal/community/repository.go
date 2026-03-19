package community

import (
	"context"
	"errors"

	"github.com/RiceSafe/rice-safe-backend/internal/platform/database"
	"github.com/google/uuid"
)

type Repository interface {
	CreatePost(ctx context.Context, post *Post) error
	GetPosts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*PostResponse, error)
	GetPostByID(ctx context.Context, postID, userID uuid.UUID) (*PostResponse, error)
	GetComments(ctx context.Context, postID uuid.UUID) ([]*CommentResponse, error)
	CreateComment(ctx context.Context, comment *Comment) error
	ToggleLike(ctx context.Context, postID, userID uuid.UUID) (bool, error)
	DeletePost(ctx context.Context, id uuid.UUID) error
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) CreatePost(ctx context.Context, post *Post) error {
	query := `
		INSERT INTO posts (user_id, content, image_url)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	return database.DB.QueryRow(ctx, query,
		post.UserID, post.Content, post.ImageURL,
	).Scan(&post.ID, &post.CreatedAt, &post.UpdatedAt)
}

func (r *repository) GetPosts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*PostResponse, error) {
	query := `
		SELECT 
			p.id, p.user_id, 
			u.username as author_name,
			u.avatar_url,
			p.content, p.image_url, p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM likes l WHERE l.post_id = p.id) as like_count,
			(SELECT COUNT(*) FROM comments c WHERE c.post_id = p.id) as comment_count,
			EXISTS(SELECT 1 FROM likes l WHERE l.post_id = p.id AND l.user_id = $1) as is_liked
		FROM posts p
		JOIN users u ON p.user_id = u.id
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := database.DB.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*PostResponse
	for rows.Next() {
		var p PostResponse
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.AuthorName, &p.AuthorAvatar,
			&p.Content, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt,
			&p.LikeCount, &p.CommentCount, &p.IsLiked,
		); err != nil {
			return nil, err
		}
		posts = append(posts, &p)
	}
	return posts, nil
}

func (r *repository) GetPostByID(ctx context.Context, postID, userID uuid.UUID) (*PostResponse, error) {
	query := `
		SELECT 
			p.id, p.user_id, 
			u.username as author_name,
			u.avatar_url,
			p.content, p.image_url, p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM likes l WHERE l.post_id = p.id) as like_count,
			(SELECT COUNT(*) FROM comments c WHERE c.post_id = p.id) as comment_count,
			EXISTS(SELECT 1 FROM likes l WHERE l.post_id = p.id AND l.user_id = $2) as is_liked
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.id = $1
	`
	var p PostResponse
	err := database.DB.QueryRow(ctx, query, postID, userID).Scan(
		&p.ID, &p.UserID, &p.AuthorName, &p.AuthorAvatar,
		&p.Content, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt,
		&p.LikeCount, &p.CommentCount, &p.IsLiked,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) GetComments(ctx context.Context, postID uuid.UUID) ([]*CommentResponse, error) {
	query := `
		SELECT 
			c.id, c.post_id, c.user_id,
			u.username as author_name,
			u.avatar_url,
			c.content, c.created_at
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.post_id = $1
		ORDER BY c.created_at ASC
	`
	rows, err := database.DB.Query(ctx, query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*CommentResponse
	for rows.Next() {
		var c CommentResponse
		if err := rows.Scan(
			&c.ID, &c.PostID, &c.UserID, &c.AuthorName, &c.AuthorAvatar,
			&c.Content, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		comments = append(comments, &c)
	}
	return comments, nil
}

func (r *repository) CreateComment(ctx context.Context, comment *Comment) error {
	query := `
		INSERT INTO comments (post_id, user_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	return database.DB.QueryRow(ctx, query,
		comment.PostID, comment.UserID, comment.Content,
	).Scan(&comment.ID, &comment.CreatedAt, &comment.UpdatedAt)
}

func (r *repository) ToggleLike(ctx context.Context, postID, userID uuid.UUID) (bool, error) {
	// Check if already liked
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM likes WHERE post_id = $1 AND user_id = $2)`
	err := database.DB.QueryRow(ctx, checkQuery, postID, userID).Scan(&exists)
	if err != nil {
		return false, err
	}

	if exists {
		// Unlike
		deleteQuery := `DELETE FROM likes WHERE post_id = $1 AND user_id = $2`
		_, err := database.DB.Exec(ctx, deleteQuery, postID, userID)
		return false, err
	} else {
		// Like
		insertQuery := `INSERT INTO likes (post_id, user_id) VALUES ($1, $2)`
		_, err := database.DB.Exec(ctx, insertQuery, postID, userID)
		return true, err
	}
}

func (r *repository) DeletePost(ctx context.Context, id uuid.UUID) error {
	tag, err := database.DB.Exec(ctx, `DELETE FROM posts WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("post not found")
	}
	return nil
}
