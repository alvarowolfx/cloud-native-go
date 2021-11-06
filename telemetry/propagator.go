package telemetry

// PubsubCarrier adapts pubsub.Message to satisfy the TextMapCarrier interface.
type PubsubMetadataCarrier map[string]string

// Get returns the value associated with the passed key.
func (pc PubsubMetadataCarrier) Get(key string) string {
	return pc[key]
}

// Set stores the key-value pair.
func (pc PubsubMetadataCarrier) Set(key string, value string) {
	pc[key] = value
}

// Keys lists the keys stored in this carrier.
func (pc PubsubMetadataCarrier) Keys() []string {
	keys := make([]string, 0, len(pc))
	for k := range pc {
		keys = append(keys, k)
	}
	return keys
}
