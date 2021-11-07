package cloud

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"

	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
	"gocloud.dev/blob"
	"gocloud.dev/docstore"
	"gocloud.dev/pubsub"

	// Import providers for blob storage
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"

	// Import providers for pubsub
	_ "gocloud.dev/pubsub/mempubsub"
	_ "gocloud.dev/pubsub/natspubsub"

	// Import providers for pubsub
	_ "gocloud.dev/docstore/memdocstore"
	"gocloud.dev/docstore/mongodocstore"
	_ "gocloud.dev/docstore/mongodocstore"
)

func NewDocstore(collection, idField string) (*docstore.Collection, error) {
	ctx := context.Background()
	uri := os.Getenv("DOCSTORE_URL")
	if uri == "" {
		uri = "mem://"
	}
	if strings.HasPrefix(uri, "mongo") {
		opts := options.Client()
		opts.Monitor = otelmongo.NewMonitor()
		opts.ApplyURI(os.Getenv("MONGO_SERVER_URL"))
		client, err := mongo.NewClient(opts)
		if err != nil {
			return nil, err
		}
		err = client.Connect(context.Background())
		if err != nil {
			return nil, err
		}
		u, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		mcoll := client.Database(u.Host).Collection(collection)
		return mongodocstore.OpenCollection(mcoll, idField, nil)
	}

	fullURL := fmt.Sprintf("%s%s?id_field=%s", uri, collection, idField)
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
	if strings.Contains(url, "?") {
		//url = url + "&prefix=" + prefix
	} else {
		url = url + "?prefix=" + prefix
	}
	fmt.Println("opening bucket", url)
	bucket, err := blob.OpenBucket(ctx, url)
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
