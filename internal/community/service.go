package community

import (
	"context"
	"mime/multipart"

	"github.com/RiceSafe/rice-safe-backend/internal/platform/storage"
	"github.com/google/uuid"
)

type Service interface {
	CreatePost(ctx context.Context, userID uuid.UUID, content string, imageHeader *multipart.FileHeader) (*Post, error)
	GetPosts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*PostResponse, error)
	GetPostByID(ctx context.Context, postID, userID uuid.UUID) (*PostResponse, error)
	GetComments(ctx context.Context, postID uuid.UUID) ([]*CommentResponse, error)
	CreateComment(ctx context.Context, userID, postID uuid.UUID, content string) (*Comment, error)
	ToggleLike(ctx context.Context, userID, postID uuid.UUID) (bool, error)
}

type service struct {
	repo    Repository
	storage storage.Service
}

func NewService(repo Repository, storage storage.Service) Service {
	return &service{repo: repo, storage: storage}
}

func (s *service) CreatePost(ctx context.Context, userID uuid.UUID, content string, imageHeader *multipart.FileHeader) (*Post, error) {
	var imageURL string
	if imageHeader != nil {
		// Upload to GCS
		url, err := s.storage.UploadFile(imageHeader, "community")
		if err != nil {
			return nil, err
		}
		imageURL = url
	}

	post := &Post{
		UserID:   userID,
		Content:  content,
		ImageURL: imageURL,
	}

	if err := s.repo.CreatePost(ctx, post); err != nil {
		return nil, err
	}

	// Calculate current signed URL if exists
	if post.ImageURL != "" {
		signed, err := s.storage.GetFileUrl(post.ImageURL)
		if err == nil {
			post.ImageURL = signed
		}
	}

	return post, nil
}

func (s *service) GetPosts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*PostResponse, error) {
	posts, err := s.repo.GetPosts(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Sign URLs for posts and authors
	for _, p := range posts {
		if p.ImageURL != "" {
			if signed, err := s.storage.GetFileUrl(p.ImageURL); err == nil {
				p.ImageURL = signed
			}
		}
		if p.AuthorAvatar != "" {
			if signed, err := s.storage.GetFileUrl(p.AuthorAvatar); err == nil {
				p.AuthorAvatar = signed
			}
		}
	}
	return posts, nil
}

func (s *service) GetPostByID(ctx context.Context, postID, userID uuid.UUID) (*PostResponse, error) {
	p, err := s.repo.GetPostByID(ctx, postID, userID)
	if err != nil {
		return nil, err
	}

	if p.ImageURL != "" {
		if signed, err := s.storage.GetFileUrl(p.ImageURL); err == nil {
			p.ImageURL = signed
		}
	}
	if p.AuthorAvatar != "" {
		if signed, err := s.storage.GetFileUrl(p.AuthorAvatar); err == nil {
			p.AuthorAvatar = signed
		}
	}
	return p, nil
}

func (s *service) GetComments(ctx context.Context, postID uuid.UUID) ([]*CommentResponse, error) {
	comments, err := s.repo.GetComments(ctx, postID)
	if err != nil {
		return nil, err
	}

	// Sign avatar URLs
	for _, c := range comments {
		if c.AuthorAvatar != "" {
			if signed, err := s.storage.GetFileUrl(c.AuthorAvatar); err == nil {
				c.AuthorAvatar = signed
			}
		}
	}
	return comments, nil
}

func (s *service) CreateComment(ctx context.Context, userID, postID uuid.UUID, content string) (*Comment, error) {
	comment := &Comment{
		PostID:  postID,
		UserID:  userID,
		Content: content,
	}
	if err := s.repo.CreateComment(ctx, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *service) ToggleLike(ctx context.Context, userID, postID uuid.UUID) (bool, error) {
	return s.repo.ToggleLike(ctx, postID, userID)
}
