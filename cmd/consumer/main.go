package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/glebnaz/kafka-webinars-rebrain/internal/app/kafka"
	"github.com/glebnaz/kafka-webinars-rebrain/internal/app/models"
	"google.golang.org/grpc"
	"os"

	grpcInternal "github.com/glebnaz/go-platform/grpc"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	level, err := log.ParseLevel(os.Getenv("LOGLVL"))
	if err != nil {
		level = log.DebugLevel
	}
	log.SetLevel(level)
}

var feed = models.NewFeedStore()

type HandlerMsg struct {
	feed *models.FeedStore
}

func DecodeB64(message string) (retour string, err error) {
	base64Text := make([]byte, base64.StdEncoding.DecodedLen(len(message)))
	_, err = base64.StdEncoding.Decode(base64Text, []byte(message))
	if err != nil {
		return "", err
	}
	fmt.Printf("base64: %s\n", base64Text)
	return string(base64Text), nil
}

func (h HandlerMsg) Handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var post models.Post

	fmt.Println(string(msg.Value))

	//byteData,err := base64.StdEncoding.DecodeString(string(bytes.Trim(msg.Value, "\xef\xbb\xbf")))
	byteData, err := DecodeB64(string(msg.Value[1:(len(msg.Value) - 1)]))
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println(string(byteData))

	err = json.Unmarshal([]byte(byteData), &post)
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println(string(msg.Value))

	feed.AddPostToUsers([]int32{1, 2, 3}, post)

	return nil
}

func main() {

	feed := models.NewFeedStore()

	msgh := HandlerMsg{
		feed: &feed,
	}

	cg := kafka.NewConsumerGroup([]string{"localhost:9092"}, "test")

	con, err := cg.ConsumeTopic(context.Background(), []string{"test"}, msgh, kafka.WithInitialOffset(sarama.OffsetNewest))
	if err != nil {
		panic(err)
	}

	con.Run()

	ctx := context.Background()

	app := newApp(ctx, grpc.UnaryInterceptor(grpcInternal.NewServerUnaryLoggerInterceptor()))

	if err := app.Start(ctx); err != nil {
		log.Errorf("bad start app: %s", err)
	}
}
