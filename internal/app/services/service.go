package services

import (
	"context"
	"github.com/glebnaz/kafka-webinars-rebrain/internal/app/models"
	pb "github.com/glebnaz/kafka-webinars-rebrain/pkg/pb/api/v1"
)

type TwitterService struct {
	users models.UserStore
	feed  models.FeedStore
	//post models.PostStore
}

func NewService() *TwitterService {
	return &TwitterService{
		users: models.NewUsersStore(),
	}
}

func (t *TwitterService) CreatePost(ctx context.Context, request *pb.CreatePostRequest) (*pb.CreatePostResponse, error) {
	panic("implement me")
}

func (t *TwitterService) GetFeed(ctx context.Context, request *pb.GetFeedRequest) (*pb.GetFeedResponse, error) {
	panic("implement me")
}
