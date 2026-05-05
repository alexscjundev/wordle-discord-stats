package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"wordle-discord-stats/daemon"
	"wordle-discord-stats/nickcache"
	"wordle-discord-stats/store"

	"github.com/bwmarrin/discordgo"
)

func main() {
	token := mustEnv("DISCORD_TOKEN")
	guildID := mustEnv("DISCORD_GUILD_ID")
	resultsPath := envOr("RESULTS_FILE", "wordle_results.json")
	configPath := envOr("DAEMON_CONFIG_FILE", "daemon_config.toml")

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		fatalf("discord session: %v", err)
	}

	nc := nickcache.New(session, guildID)
	nc.Refresh()

	fixedNicks, err := scanFixedNicks(resultsPath)
	if err != nil {
		fatalf("scan results: %v", err)
	}

	_, statErr := os.Stat(configPath)
	if os.IsNotExist(statErr) {
		fmt.Printf("daemon config does not exist (%s) — showing current state\n\n", configPath)
		printInfo(nc, fixedNicks)
		return
	}
	if statErr != nil {
		fatalf("stat %s: %v", configPath, statErr)
	}

	cfg, err := daemon.LoadConfig(configPath)
	if err != nil {
		fatalf("load config: %v", err)
	}

	if len(cfg.NickMap) == 0 {
		fmt.Println("config does not have a nick_map — showing current state\n")
		printInfo(nc, fixedNicks)
		return
	}

	verifyNickMap(cfg.NickMap, fixedNicks, nc)
}

// printInfo prints the nickcache snapshot and all fixed nicks so an admin can
// populate the [nick_map] section of daemon_config.toml.
func printInfo(nc *nickcache.NickCache, fixedNicks map[string]struct{}) {
	snap := nc.Snapshot()
	ids := make([]string, 0, len(snap))
	for id := range snap {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return snap[ids[i]] < snap[ids[j]] })

	fmt.Println("snowflake → display name:")
	for _, id := range ids {
		fmt.Printf("  %-20s  %s\n", id, snap[id])
	}

	fmt.Println("\nfixed nicks (all unmapped):")
	for _, n := range sortedKeys(fixedNicks) {
		fmt.Printf("  %q\n", n)
	}
}

// verifyNickMap checks that every fixed nick in the store is present in the
// nick_map. Exits 1 if any are missing. On success, prints each resolved
// display name with the user IDs and fixed nicks that map to it.
func verifyNickMap(nickMap map[string]string, fixedNicks map[string]struct{}, nc *nickcache.NickCache) {
	var missing []string
	for nick := range fixedNicks {
		if _, ok := nickMap[nick]; !ok {
			missing = append(missing, nick)
		}
	}
	sort.Strings(missing)

	if len(missing) > 0 {
		fmt.Fprintln(os.Stderr, "not all nicks are mapped — missing:")
		for _, n := range missing {
			fmt.Fprintf(os.Stderr, "  %q\n", n)
		}
		os.Exit(1)
	}

	fmt.Println("all nicks are mapped:\n")
	// Group fixed nicks by their target snowflake.
	type entry struct {
		snowflake  string
		fixedNicks []string
	}
	bySnowflake := map[string]*entry{}
	for nick, snowflake := range nickMap {
		e, ok := bySnowflake[snowflake]
		if !ok {
			e = &entry{snowflake: snowflake}
			bySnowflake[snowflake] = e
		}
		e.fixedNicks = append(e.fixedNicks, nick)
	}

	// Sort entries by resolved display name.
	entries := make([]*entry, 0, len(bySnowflake))
	for _, e := range bySnowflake {
		sort.Strings(e.fixedNicks)
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		return nc.Get(entries[i].snowflake) < nc.Get(entries[j].snowflake)
	})

	for _, e := range entries {
		fmt.Printf("%s:\n", nc.Get(e.snowflake))
		fmt.Printf("  user_id:    %s\n", e.snowflake)
		for _, nick := range e.fixedNicks {
			fmt.Printf("  fixed_nick: %s\n", nick)
		}
	}
}

func scanFixedNicks(path string) (map[string]struct{}, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return map[string]struct{}{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := map[string]struct{}{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var r store.WordleResult
		if err := json.Unmarshal(line, &r); err != nil {
			return nil, err
		}
		if r.FixedNick != "" {
			out[r.FixedNick] = struct{}{}
		}
	}
	return out, sc.Err()
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fatalf("%s not set", key)
	}
	return v
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
