package services

import (
	"context"
	"encoding/json"
	"github.com/glebnaz/kafka-webinars-rebrain/internal/app/kafka"
	"github.com/glebnaz/kafka-webinars-rebrain/internal/app/models"
	pb "github.com/glebnaz/kafka-webinars-rebrain/pkg/pb/api/v1"
	log "github.com/sirupsen/logrus"
)

type TwitterService struct {
	users models.UserStore
	feed  models.FeedStore
	//post models.PostStore

	producer *kafka.SyncProducer
}

func NewService() *TwitterService {
	p, err := kafka.NewSyncProducer([]string{"localhost:9092"}, nil)
	if err != nil {
		panic(err)
	}

	return &TwitterService{
		users:    models.NewUsersStore(),
		producer: p,
	}
}

func (t *TwitterService) CreatePost(ctx context.Context, request *pb.CreatePostRequest) (*pb.CreatePostResponse, error) {
	post := request.GetPost()

	postInternal := models.Post{
		ID:        post.Id,
		Payload:   post.Payload,
		CreatedAt: post.CreatedAt,
		CreatedBy: post.CreatedBy,
	}

	byteJSON, err := json.Marshal(postInternal)
	if err != nil {
		return nil, err
	}

	_, offset, err := t.producer.Put(ctx, "test", byteJSON, nil)
	if err != nil {
		return nil, err
	}

	log.Debugf("offset: %d", offset)

	return &pb.CreatePostResponse{Post: post}, nil
}

func (t *TwitterService) GetFeed(ctx context.Context, request *pb.GetFeedRequest) (*pb.GetFeedResponse, error) {
	var pbPosts []*pb.Post

	posts := t.feed.GetFeed(request.GetUserId())

	for _, v := range posts {
		postInternal := pb.Post{
			Id:        v.ID,
			Payload:   v.Payload,
			CreatedAt: v.CreatedAt,
			CreatedBy: v.CreatedBy,
		}
		pbPosts = append(pbPosts, &postInternal)
	}

	return &pb.GetFeedResponse{Post: pbPosts}, nil
}
