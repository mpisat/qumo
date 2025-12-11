package relay

type Config struct {
	// Upstream server URL (optional)
	Upstream string

	// GroupCacheSize is the maximum number of group caches to keep.
	GroupCacheSize int

	// FrameCapacity is the frame buffer size in bytes.
	FrameCapacity int
}
