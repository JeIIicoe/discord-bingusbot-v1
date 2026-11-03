# BINGUS BOT 9000

A fun little Discord bot that keeps score of your server's Wordle games.

Whenever the Wordle app posts the day's results, Bingus Bot adds them to a weekly tally and replies with the running scores. On Sunday it posts the final leaderboard, with medals for the top three, then starts afresh on Monday.

## How scoring works

- Each game scores the number of guesses it took, so lower is better.
- A failed game (X/6) or a missed day scores 7.
- The lowest total for the week wins.
- Each server keeps its own scores.

## Commands

| Command | What it does |
| --- | --- |
| `/hello` | Says hello. |
| `/stats` | Shows your lifetime Wordle stats for the server, including this week so far. |
| `/wordlebackfill` | Sets the bot up in a channel by loading last week's and this week's Wordle results. Owner only. |
| `/post` | Posts this week's leaderboard straight away, without waiting for Sunday. Owner only. |

## Running it

You'll need [Go](https://go.dev/dl/) installed.

1. Create a bot in the [Discord Developer Portal](https://discord.com/developers/applications) and turn on **Message Content Intent** under **Bot**.
2. Invite it to your server with permission to read messages, send messages and read message history.
3. Make a `.env` file next to `main.go`:

   ```
   DISCORD_TOKEN=your-bot-token
   ```

4. Run `go run .`
