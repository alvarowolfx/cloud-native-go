package cloud

import (
	"context"
	"fmt"
	"os"

	"gocloud.dev/blob"
	"gocloud.dev/docstore"
	"gocloud.dev/pubsub"

	// Import providers for blob storage
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"

	// Import providers for pubsub
	_ "gocloud.dev/pubsub/mempubsub"
	_ "gocloud.dev/pubsub/natspubsub"

	// Import providers for pubsub
	_ "gocloud.dev/docstore/memdocstore"
	_ "gocloud.dev/docstore/mongodocstore"
)

func NewDocstore(collection, idField string) (*docstore.Collection, error) {
	ctx := context.Background()
	url := os.Getenv("DOCSTORE_URL")
	if url == "" {
		url = "mem://"
	}

	fullURL := fmt.Sprintf("%s%s?id_field=%s", url, collection, idField)
	coll, err := docstore.OpenCollection(ctx, fullURL)
	if err != nil {
		return nil, fmt.Errorf("could not open bucket: %v", err)
	}
	return coll, nil
}

func NewBucket(prefix string) (*blob.Bucket, error) {
	ctx := context.Background()
	url := os.Getenv("BUCKET_URL")
	if url == "" {
		url = "file://./tmp/"
	}
	bucket, err := blob.OpenBucket(ctx, url+"?prefix="+prefix)
	if err != nil {
		return nil, fmt.Errorf("could not open bucket: %v", err)
	}
	return bucket, nil
}

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
