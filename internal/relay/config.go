package relay

type Config struct {
	// Upstream server URL (optional)
	Upstream string

	// GroupCacheSize is the maximum number of group caches to keep.
	GroupCacheSize int

	// FrameCapacity is the frame buffer size in bytes.
	FrameCapacity int
}

func (c *Config) groupCacheSize() int {
	if c != nil && c.GroupCacheSize > 0 {
		return c.GroupCacheSize
	}
	return DefaultGroupCacheSize
}

func (c *Config) frameCapacity() int {
	if c != nil && c.FrameCapacity > 0 {
		return c.FrameCapacity
	}
	return DefaultNewFrameCapacity
}
