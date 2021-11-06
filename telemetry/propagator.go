package telemetry

import "gocloud.dev/pubsub"

// PubsubCarrier adapts pubsub.Message to satisfy the TextMapCarrier interface.
type PubsubCarrier struct {
	*pubsub.Message
}

// Get returns the value associated with the passed key.
func (pc PubsubCarrier) Get(key string) string {
	return pc.Metadata[key]
}

// Set stores the key-value pair.
func (pc PubsubCarrier) Set(key string, value string) {
	pc.Metadata[key] = value
}

// Keys lists the keys stored in this carrier.
func (pc PubsubCarrier) Keys() []string {
	keys := make([]string, 0, len(pc.Metadata))
	for k := range pc.Metadata {
		keys = append(keys, k)
	}
	return keys
}
