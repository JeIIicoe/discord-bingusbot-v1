# BINGUS BOT 9000

A little Discord bot that keeps score of your server's Wordle games.

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

### Optional settings

These also go in `.env`:

- `WORDLE_TIMEZONE`: the timezone used to decide which day a post belongs to. Defaults to `Europe/London`.
- `DATA_DIR`: where the scores are saved. Defaults to `data`.

Back up the `data` folder now and then, as it's the only copy of everyone's scores.

## Deploying to a VPS

Every push to `main` is built and deployed to the server by GitHub Actions. To set it up once:

1. Make an SSH key just for deploys: `ssh-keygen -t ed25519 -f deploy_key -N ""`
2. Copy the `deploy` folder to the server and run `sh deploy/setup-server.sh "$(cat deploy_key.pub)"` as root.
3. Put your `.env` in `/opt/bingusbot`, owned by `bingus` with `chmod 600`.
4. In the repo's **Settings → Environments**, create an environment called `production` with these secrets:
   - `VPS_HOST`: the server's IP address
   - `VPS_SSH_KEY`: the contents of `deploy_key` (the private half)
   - `VPS_KNOWN_HOSTS`: the output of `ssh-keyscan <server IP>`
5. On an Arm server (Hetzner's CAX range), add a repository variable `VPS_ARCH` set to `arm64`.

Logs are available on the server with `journalctl -u bingusbot -f`.
