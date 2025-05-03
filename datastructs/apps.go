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

type ApplicationVerdict struct {
	App     *Application
	Verdict bool
}

type Session struct {
	Owner          string
	SessionChannel string
	ChannelIn      chan int
	ChannelOut     chan *Session
	CurrentApp     *ReviewingApplication
}

type UserApplicationGroup struct {
	PendingApplications  []Application
	ReviewedApplications []Application
}

type AllApps map[string]UserApplicationGroup
