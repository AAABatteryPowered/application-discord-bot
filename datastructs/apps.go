package datastructs

import (
	"github.com/bwmarrin/discordgo"
)

type Application struct {
	Link    string
	Author  *discordgo.Member
	Verdict *bool
}

type ReviewingApplication struct {
	App     *Application
	EmbedID string
}

type Session struct {
	Owner          string
	SessionChannel string
	ChannelIn      chan int
	ChannelOut     chan *Session
	CurrentApp     *ReviewingApplication
}

type UserApplicationGroup []Application

type allApps map[string]UserApplicationGroup

var Applications allApps
var PastApplications allApps
