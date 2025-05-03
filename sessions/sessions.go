package sessions

import (
	"fmt"

	. "bot/applications"
	. "bot/datastructs"

	"github.com/bwmarrin/discordgo"
)

const guildid = "1355623019971608706"
const botCategory = "1359830076455125185"

var dsgs *discordgo.Session

var Sessions map[string]*Session

var SessionChannel chan int

func SessionExists(userid string) (chan int, chan *Session) {
	val, exists := Sessions[userid]
	if !exists {
		return nil, nil
	}
	return val.ChannelIn, val.ChannelOut
}

func Close(ds *discordgo.Session, s *Session) error {
	_, err := ds.ChannelDelete(s.SessionChannel)
	if err != nil {
		fmt.Println(err)
		return err
	}
	delete(Sessions, s.Owner)
	return nil
}

func EnterLoop(s *Session) {
	ShowApplication(dsgs, s)
	for {
		code := <-s.ChannelIn
		if code == 1 {
			ShowApplication(dsgs, s)
		}
		if code == 2 {
			s.ChannelOut <- s
		}
	}
}

func SessionOpen(s *discordgo.Session, i *discordgo.InteractionCreate) (*Session, error) {
	session := Sessions[i.Member.User.ID]
	if session != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("You already have a reviewing session in <#%s>", session.CurrentApp.EmbedID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return Sessions[i.Member.User.ID], nil
	}

	botUser, err := s.User("@me")
	if err != nil {
		fmt.Println("error fetching bot user:", err)
		return nil, err
	}

	denyAll := discordgo.PermissionOverwrite{
		ID:   guildid,
		Type: discordgo.PermissionOverwriteTypeRole,
		Deny: discordgo.PermissionViewChannel,
	}

	allowUser := discordgo.PermissionOverwrite{
		ID:    i.Member.User.ID,
		Type:  discordgo.PermissionOverwriteTypeMember,
		Allow: discordgo.PermissionViewChannel,
	}

	allowBot := discordgo.PermissionOverwrite{
		ID:    botUser.ID,
		Type:  discordgo.PermissionOverwriteTypeMember,
		Allow: discordgo.PermissionViewChannel,
	}

	channel, err := s.GuildChannelCreateComplex(guildid, discordgo.GuildChannelCreateData{
		Name:     fmt.Sprintf("session-%s", i.Member.User.Username),
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: botCategory,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			&denyAll,
			&allowUser,
			&allowBot,
		},
	})
	if err != nil {
		fmt.Println("error creating channel:", err)
		return nil, err
	}

	sess := Session{
		Owner:          i.Member.User.ID,
		SessionChannel: channel.ID,
		ChannelIn:      make(chan int),
		ChannelOut:     make(chan *Session),
		CurrentApp:     nil,
	}
	Sessions[i.Member.User.ID] = &sess

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Reviewing session created successfully in <#%s>", channel.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	return &sess, nil
	//serveApplication(s, channel)
}

func Shutdown() {
	for _, session := range Sessions {
		Close(dsgs, session)
	}
}

func Start(s *discordgo.Session) {
	dsgs = s
	Sessions = make(map[string]*Session, 0)
}
