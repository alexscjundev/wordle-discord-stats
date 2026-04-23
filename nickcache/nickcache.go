package nickcache

import (
	"log/slog"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type NickCache struct {
	session *discordgo.Session
	guildID string
	mu      sync.RWMutex
	nicks   map[string]string // snowflake → display name
}

func New(session *discordgo.Session, guildID string) *NickCache {
	return &NickCache{
		session: session,
		guildID: guildID,
		nicks:   map[string]string{},
	}
}

// Get returns the display name for a player key. For Discord snowflakes it
// resolves via the cache; for fixed nicks (not in the cache) it returns the
// key unchanged, so callers need not distinguish between the two.
func (c *NickCache) Get(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if name, ok := c.nicks[key]; ok {
		return name
	}
	return key
}

func (c *NickCache) Refresh() {
	members, err := c.session.GuildMembers(c.guildID, "", 1000)
	if err != nil {
		slog.Error("nickcache: fetch guild members", "err", err)
		return
	}

	fresh := make(map[string]string, len(members))
	for _, m := range members {
		name := m.Nick
		if name == "" {
			name = m.User.Username
		}
		fresh[m.User.ID] = name
	}

	c.mu.Lock()
	c.nicks = fresh
	c.mu.Unlock()
	slog.Debug("nickcache: refreshed", "count", len(fresh))
}

func (c *NickCache) Start(interval time.Duration) {
	c.Refresh()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			c.Refresh()
		}
	}()
}
