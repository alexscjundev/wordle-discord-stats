# wordle-discord-stats

A discord bot that reads from the NYT Wordle game's discord output and shows how well you're doing against your friends

## Features

- **Bot slash commands** — `/stats <user>` for a player's all-time average and rank, `/top <k>` for the leaderboard
- **DNF = 7** — failed puzzles (X/6) are scored as 7, keeping them comparable to real scores
- **Fixed nicknames** — faulty, nickname (not discord snowflake tagged) records are resolved to the corresponding user
- **Streaks, Elo, averages** — tracks current and all-time streaks, round-robin Elo ratings, and per-player score averages
- **Daily daemon** — posts a summary to the channel automatically each day

## Nick map

Some older records may be logged under a fixed nickname (plain text rather than a Discord mention). The nick map in `daemon_config.toml` maps those names to Discord user IDs so they resolve correctly on leaderboards.

### Building the mapping

Run `nickcheck` without a nick map configured. It prints every Discord snowflake with its current display name alongside all fixed nicks found in the store:

```
snowflake → display name:
  123456789012345678    Alice Example
  987654321098765432    Bob Example
  111222333444555666    Charlie Wordsworth

fixed nicks (all unmapped):
  "alice 2022"
  "alice old account"
  "bobby"
  "charlie w"
```

Use that output to populate the `[nick_map]` section of `daemon_config.toml`:

```toml
disable_messages = false

[nick_map]
"alice 2022"        = "123456789012345678"
"alice old account" = "123456789012345678"
"bobby"             = "987654321098765432"
"charlie w"         = "111222333444555666"
```

### Verifying the mapping

Run `nickcheck` again. If all fixed nicks are accounted for it prints the resolved groups and exits 0:

```
all nicks are mapped:

Alice Example:
  user_id:    123456789012345678
  fixed_nick: alice 2022
  fixed_nick: alice old account

Bob Example:
  user_id:    987654321098765432
  fixed_nick: bobby

Charlie Wordsworth:
  user_id:    111222333444555666
  fixed_nick: charlie w
```

Any unmapped fixed nicks are printed to stderr and the command exits 1.

## Running

### Locally

Build the OCR binary, then run the bot:

```sh
cd imgparse && cargo build --release && cd ..
source values.sh && go run main.go
```

`values.sh` exports the required env vars (see `values.yaml` for the full list). It is gitignored so you can put real tokens in it safely.

To run `nickcheck` locally:

```sh
source values.sh && go run ./cmd/nickcheck
```

### Docker

```sh
docker build -t wordle-stats .
```

Run the bot (default):

```sh
docker run --rm \
  -e DISCORD_TOKEN=... \
  -e DISCORD_GUILD_ID=... \
  -e DISCORD_CHANNEL_ID=... \
  -e WORDLE_BOT_USER_ID=... \
  -v $(pwd)/data:/data \
  wordle-stats
```

Run `nickcheck` against the same data volume:

```sh
docker run --rm \
  -e DISCORD_TOKEN=... \
  -e DISCORD_GUILD_ID=... \
  -v $(pwd)/data:/data \
  wordle-stats ./nickcheck
```

State (results, cursor, and `daemon_config.toml`) is persisted in the `/data` volume.
