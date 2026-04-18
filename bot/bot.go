package bot

import (
	"fmt"
	"log/slog"

	"wordle-discord-stats/nickcache"
	"wordle-discord-stats/store"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	session *discordgo.Session
	guildID string
	store   store.Store
	nicks   *nickcache.NickCache
}

var slashCommands = []*discordgo.ApplicationCommand{
	{
		Name:        "stats",
		Description: "Get average Wordle score for a user",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "User to look up",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "since_day",
				Description: "Include results on or after this Wordle day (0 = all time)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "scoring_type",
				Description: "Scoring method",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "average", Value: "average"},
					{Name: "bayesian weighted", Value: "bayesian_weighted"},
				},
			},
		},
	},
	{
		Name:        "top",
		Description: "Get top Wordle scores",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "k",
				Description: "Number of users to show",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "since_day",
				Description: "Include results on or after this Wordle day (0 = all time)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "scoring_type",
				Description: "Scoring method",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "average", Value: "average"},
					{Name: "bayesian weighted", Value: "bayesian_weighted"},
				},
			},
		},
	},
}

func New(token, guildID string, st store.Store, nc *nickcache.NickCache) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	b := &Bot{session: s, guildID: guildID, store: st, nicks: nc}
	s.AddHandler(b.handleInteraction)

	if err := s.Open(); err != nil {
		return nil, err
	}

	for _, cmd := range slashCommands {
		if _, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd); err != nil {
			return nil, fmt.Errorf("register command %s: %w", cmd.Name, err)
		}
	}
	slog.Info("registered slash commands", "count", len(slashCommands))

	return b, nil
}

func (b *Bot) SetNickCache(nc *nickcache.NickCache) { b.nicks = nc }
func (b *Bot) Close()                               { b.session.Close() }
func (b *Bot) Session() *discordgo.Session          { return b.session }

func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	switch i.ApplicationCommandData().Name {
	case "stats":
		b.handleStats(s, i)
	case "top":
		b.handleTop(s, i)
	}
}
