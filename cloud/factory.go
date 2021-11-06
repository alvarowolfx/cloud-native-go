package cloud

import (
	"context"
	"fmt"
	"os"

	"gocloud.dev/pubsub"
)

func NewTopic() (*pubsub.Topic, error) {
	ctx := context.Background()
	url := os.Getenv("PUBSUB_TOPIC_URL")
	if url == "" {
		url = "mem://events"
	}
	t, err := pubsub.OpenTopic(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("could not open topic: %v", err)
	}
	return t, nil
}

func NewTopicSub() (*pubsub.Subscription, error) {
	ctx := context.Background()
	url := os.Getenv("PUBSUB_SUB_URL")
	if url == "" {
		url = "mem://events"
	}
	sub, err := pubsub.OpenSubscription(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("could not open topic: %v", err)
	}
	return sub, nil
}
