package wordlecount

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// messageLimit is the most characters Discord allows in one message.
const messageLimit = 2000

// catch runs run and turns a panic into an error with the stack trace, so a
// bug in a handler doesn't take the whole bot down.
func catch(run func() error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v\n\n%s", recovered, debug.Stack())
		}
	}()
	return run()
}

// reportError posts err to channelID as a dump, logging it if that fails too.
func reportError(sender messageSender, channelID string, err error) {
	fmt.Printf("wordlecount: %v\n", err)

	blocks := codeBlocks("", err.Error())
	blocks[0] = "⚠️ **Wordle tracker error**\n" + blocks[0]
	if postErr := postAll(sender, channelID, blocks); postErr != nil {
		fmt.Printf("wordlecount: posting error dump: %v\n", postErr)
	}
}

// postAll posts each message to channelID in order.
func postAll(sender messageSender, channelID string, messages []string) error {
	for _, message := range messages {
		if _, err := sender.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{Content: message}); err != nil {
			return err
		}
	}
	return nil
}

// codeBlocks splits text into code blocks small enough to post, leaving room
// for a short heading on the first. Lines are kept whole where possible.
func codeBlocks(language, text string) []string {
	// room for the ``` and the heading reportError puts on the first one
	const maxBodyLength = messageLimit - 200

	var blocks []string
	var block strings.Builder
	closeBlock := func() {
		blocks = append(blocks, "```"+language+"\n"+block.String()+"```")
		block.Reset()
	}

	for _, line := range strings.SplitAfter(text, "\n") {
		// a line too long for one block on its own just gets cut up
		for len(line) > maxBodyLength {
			if block.Len() > 0 {
				closeBlock()
			}
			block.WriteString(line[:maxBodyLength] + "\n")
			closeBlock()
			line = line[maxBodyLength:]
		}
		if block.Len()+len(line) > maxBodyLength {
			closeBlock()
		}
		block.WriteString(line)
	}
	if block.Len() > 0 || len(blocks) == 0 {
		if !strings.HasSuffix(block.String(), "\n") {
			block.WriteByte('\n')
		}
		closeBlock()
	}
	return blocks
}
