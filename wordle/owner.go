package wordlecount

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// My personal ownerID so I'm the only user allowed to run the owner-only commands,
// /wordlebackfill and /post.
const ownerID = "183372080165683200"

// ownerInServer reports whether the interaction came from the bot's owner in
// a server. If not, it answers the interaction to say why.
func ownerInServer(session *discordgo.Session, interaction *discordgo.InteractionCreate) bool {
	switch {
	case invokerID(interaction.Interaction) != ownerID:
		respond(session, interaction, "You can't use this command, sorry >_<")
		return false
	case interaction.GuildID == "":
		respond(session, interaction, "Use this command in a server, silly!!")
		return false
	default:
		return true
	}
}

// invokerID returns the ID of the user who triggered the interaction, whether
// it came from a server or a DM.
func invokerID(interaction *discordgo.Interaction) string {
	switch {
	case interaction.Member != nil && interaction.Member.User != nil:
		return interaction.Member.User.ID
	case interaction.User != nil:
		return interaction.User.ID
	default:
		return ""
	}
}

// respond answers a command straight away with a message only the invoker
// can see.
func respond(session *discordgo.Session, interaction *discordgo.InteractionCreate, content string) {
	err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
	})
	if err != nil {
		fmt.Printf("wordlecount: responding to command: %v\n", err)
	}
}
