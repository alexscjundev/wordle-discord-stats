package store

import (
	"math"
	"testing"
)

func storeWith(results ...WordleResult) *FileStore {
	rs := results
	return &FileStore{scan: func() ([]WordleResult, error) { return rs, nil }}
}

func storeWithNickMap(nickMap map[string]string, results ...WordleResult) *FileStore {
	rs := results
	return &FileStore{nickMap: nickMap, scan: func() ([]WordleResult, error) { return rs, nil }}
}

// staticResolver maps keys to display names for use in tests.
type staticResolver map[string]string

func (r staticResolver) Get(key string) string {
	if name, ok := r[key]; ok {
		return name
	}
	return key
}

func done(day int, nick string, score int) WordleResult {
	return WordleResult{FixedNick: nick, Day: day, Score: score, Complete: true}
}

func doneUser(day int, userID string, score int) WordleResult {
	return WordleResult{UserID: userID, Day: day, Score: score, Complete: true}
}

func dnf(day int, nick string) WordleResult {
	return WordleResult{FixedNick: nick, Day: day, Complete: false}
}

func mustQuery(t *testing.T, st *FileStore, q Query) QueryResult {
	t.Helper()
	res, err := st.Query(q)
	if err != nil {
		t.Fatalf("query %+v: %v", q, err)
	}
	return res
}

func valueOf(entries []Entry, name string) float64 {
	for _, e := range entries {
		if e.Name == name {
			return e.Value
		}
	}
	return math.NaN()
}

func TestAvgAllTime(t *testing.T) {
	st := storeWith(
		done(1, "alex", 3), done(1, "bob", 4), done(1, "charles", 5),
		done(2, "alex", 4), done(2, "bob", 5), done(2, "charles", 6),
		done(3, "alex", 5), done(3, "bob", 4), done(3, "charles", 3),
	)

	t.Run("top-k ordered best-first", func(t *testing.T) {
		res := mustQuery(t, st, Query{Kind: KindAvgAllTime, Selector: SelectorTopK, K: 3})
		got := []string{res.Entries[0].Name, res.Entries[1].Name, res.Entries[2].Name}
		want := []string{"alex", "bob", "charles"} // avgs 4, 13/3, 14/3
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("order: got %v, want %v", got, want)
			}
		}
		if valueOf(res.Entries, "alex") != 4 {
			t.Errorf("alex avg: got %v, want 4", valueOf(res.Entries, "alex"))
		}
	})

	t.Run("min-games filter excludes", func(t *testing.T) {
		res := mustQuery(t, st, Query{Kind: KindAvgAllTime, Selector: SelectorTopK, K: 3, MinGames: 4})
		if len(res.Entries) != 0 {
			t.Errorf("want empty (all have 3 games), got %d", len(res.Entries))
		}
	})

	t.Run("dnf counts as 7", func(t *testing.T) {
		st := storeWith(done(1, "alex", 3), dnf(2, "alex"))
		res := mustQuery(t, st, Query{Kind: KindAvgAllTime, Selector: SelectorPlayer, Player: "alex"})
		if res.Entries[0].Value != 5 {
			t.Errorf("got %v, want 5 ((3+7)/2)", res.Entries[0].Value)
		}
	})
}

func TestAvgSliding(t *testing.T) {
	st := storeWith(
		done(1, "alex", 2), done(1, "bob", 3),
		done(2, "alex", 2), done(2, "bob", 3),
		done(3, "alex", 4), done(3, "bob", 4),
		done(4, "alex", 4), done(4, "bob", 4),
		done(5, "alex", 6), done(5, "bob", 6),
	)

	t.Run("last 3 days only", func(t *testing.T) {
		res := mustQuery(t, st, Query{Kind: KindAvgSliding, Selector: SelectorPlayer, Player: "alex", SlidingDays: 3})
		if math.Abs(res.Entries[0].Value-14.0/3.0) > 1e-9 {
			t.Errorf("alex avg: got %v, want %v", res.Entries[0].Value, 14.0/3.0)
		}
	})

	t.Run("zero window = all time", func(t *testing.T) {
		res := mustQuery(t, st, Query{Kind: KindAvgSliding, Selector: SelectorPlayer, Player: "alex", SlidingDays: 0})
		if res.Entries[0].Value != 3.6 {
			t.Errorf("got %v, want 3.6", res.Entries[0].Value)
		}
	})

	t.Run("window larger than data = all time", func(t *testing.T) {
		res := mustQuery(t, st, Query{Kind: KindAvgSliding, Selector: SelectorPlayer, Player: "alex", SlidingDays: 100})
		if res.Entries[0].Value != 3.6 {
			t.Errorf("got %v, want 3.6", res.Entries[0].Value)
		}
	})
}

func TestTotalElo(t *testing.T) {
	q := Query{Kind: KindTotalElo, Selector: SelectorTopK, K: 3, EloStart: 1500, EloK: 32}

	t.Run("consistent winner tops, consistent loser bottoms", func(t *testing.T) {
		st := storeWith(
			done(1, "alex", 3), done(1, "bob", 4), done(1, "charles", 5),
			done(2, "alex", 3), done(2, "bob", 4), done(2, "charles", 5),
			done(3, "alex", 3), done(3, "bob", 4), done(3, "charles", 5),
		)
		res := mustQuery(t, st, q)
		if res.Entries[0].Name != "alex" || res.Entries[2].Name != "charles" {
			t.Errorf("order: got %s/%s/%s, want alex/bob/charles",
				res.Entries[0].Name, res.Entries[1].Name, res.Entries[2].Name)
		}
		if res.Entries[0].Value <= 1500 || res.Entries[2].Value >= 1500 {
			t.Errorf("ratings: got %v/%v, expected above/below 1500", res.Entries[0].Value, res.Entries[2].Value)
		}
	})

	t.Run("solo player stays at start", func(t *testing.T) {
		st := storeWith(done(1, "alex", 3))
		res := mustQuery(t, st, Query{Kind: KindTotalElo, Selector: SelectorPlayer, Player: "alex", EloStart: 1500, EloK: 32})
		if res.Entries[0].Value != 1500 {
			t.Errorf("got %v, want 1500", res.Entries[0].Value)
		}
	})

	t.Run("ties leave ratings unchanged", func(t *testing.T) {
		st := storeWith(done(1, "alex", 3), done(1, "bob", 3), done(1, "charles", 3))
		res := mustQuery(t, st, q)
		for _, e := range res.Entries {
			if e.Value != 1500 {
				t.Errorf("%s: got %v, want 1500", e.Name, e.Value)
			}
		}
	})
}

func TestCurrentStreak(t *testing.T) {
	t.Run("counts back from latest day only", func(t *testing.T) {
		// alex: 1,2,3 → streak 3. bob: 1,3 → streak 1 (gap at 2). charles: 3 → streak 1.
		st := storeWith(
			done(1, "alex", 3), done(1, "bob", 4),
			done(2, "alex", 3),
			done(3, "alex", 3), done(3, "bob", 4), done(3, "charles", 5),
		)
		res := mustQuery(t, st, Query{Kind: KindCurrentStreak, Selector: SelectorTopK, K: 10})
		want := map[string]float64{"alex": 3, "bob": 1, "charles": 1}
		for name, v := range want {
			if got := valueOf(res.Entries, name); got != v {
				t.Errorf("%s: got %v, want %v", name, got, v)
			}
		}
	})

	t.Run("absent on latest day = zero", func(t *testing.T) {
		st := storeWith(
			done(1, "alex", 3), done(1, "david", 4),
			done(2, "alex", 3),
			done(3, "alex", 3),
		)
		res := mustQuery(t, st, Query{Kind: KindCurrentStreak, Selector: SelectorPlayer, Player: "david"})
		if res.Entries[0].Value != 0 {
			t.Errorf("got %v, want 0", res.Entries[0].Value)
		}
	})

	t.Run("single day played = streak of 1", func(t *testing.T) {
		st := storeWith(done(5, "alex", 3))
		res := mustQuery(t, st, Query{Kind: KindCurrentStreak, Selector: SelectorPlayer, Player: "alex"})
		if res.Entries[0].Value != 1 {
			t.Errorf("got %v, want 1", res.Entries[0].Value)
		}
	})
}

func TestAllTimeStreak(t *testing.T) {
	// alex: 1,2,3,5 → best run 3 ending at day 3.
	// bob:  1,3,4,5 → best run 3 ending at day 5.
	// charles: 1 only → run of 1 ending at day 1.
	st := storeWith(
		done(1, "alex", 3), done(1, "bob", 4), done(1, "charles", 5),
		done(2, "alex", 3),
		done(3, "alex", 3), done(3, "bob", 4),
		done(4, "bob", 4),
		done(5, "alex", 3), done(5, "bob", 4),
	)

	t.Run("returns best run length and its end day", func(t *testing.T) {
		res := mustQuery(t, st, Query{Kind: KindAllTimeStreak, Selector: SelectorPlayer, Player: "alex"})
		if res.Entries[0].Value != 3 || res.Entries[0].Day != 3 {
			t.Errorf("alex: got value=%v day=%d, want 3/3", res.Entries[0].Value, res.Entries[0].Day)
		}
		res = mustQuery(t, st, Query{Kind: KindAllTimeStreak, Selector: SelectorPlayer, Player: "bob"})
		if res.Entries[0].Value != 3 || res.Entries[0].Day != 5 {
			t.Errorf("bob: got value=%v day=%d, want 3/5", res.Entries[0].Value, res.Entries[0].Day)
		}
	})

	t.Run("single game = streak of 1", func(t *testing.T) {
		res := mustQuery(t, st, Query{Kind: KindAllTimeStreak, Selector: SelectorPlayer, Player: "charles"})
		if res.Entries[0].Value != 1 || res.Entries[0].Day != 1 {
			t.Errorf("got value=%v day=%d, want 1/1", res.Entries[0].Value, res.Entries[0].Day)
		}
	})

	t.Run("earliest of equal-length runs wins", func(t *testing.T) {
		// x played days 1-2 and 5-6: two runs of length 2.
		st := storeWith(
			done(1, "x", 3), done(2, "x", 3),
			done(5, "x", 3), done(6, "x", 3),
		)
		res := mustQuery(t, st, Query{Kind: KindAllTimeStreak, Selector: SelectorPlayer, Player: "x"})
		if res.Entries[0].Value != 2 || res.Entries[0].Day != 2 {
			t.Errorf("got value=%v day=%d, want 2/2", res.Entries[0].Value, res.Entries[0].Day)
		}
	})
}

func TestScoresAtMost(t *testing.T) {
	st := storeWith(
		done(1, "alex", 2), done(1, "bob", 4), done(1, "charles", 5),
		done(2, "alex", 3), done(2, "bob", 4),
		done(3, "alex", 4), done(3, "bob", 5), done(3, "charles", 6),
		dnf(4, "alex"),
	)

	t.Run("counts at or below threshold", func(t *testing.T) {
		res := mustQuery(t, st, Query{Kind: KindScoresAtMost, Selector: SelectorTopK, K: 10, ScoreAtMost: 4})
		want := map[string]float64{"alex": 3, "bob": 2, "charles": 0} // alex: 2,3,4 | bob: 4,4 | charles: none
		for name, v := range want {
			if got := valueOf(res.Entries, name); got != v {
				t.Errorf("%s: got %v, want %v", name, got, v)
			}
		}
	})

	t.Run("dnf (=7) excluded when threshold < 7", func(t *testing.T) {
		res := mustQuery(t, st, Query{Kind: KindScoresAtMost, Selector: SelectorPlayer, Player: "alex", ScoreAtMost: 6})
		if res.Entries[0].Value != 3 { // 2,3,4 count; dnf (7) excluded
			t.Errorf("got %v, want 3", res.Entries[0].Value)
		}
	})

	t.Run("threshold 0 yields zero for everyone", func(t *testing.T) {
		res := mustQuery(t, st, Query{Kind: KindScoresAtMost, Selector: SelectorPlayer, Player: "alex", ScoreAtMost: 0})
		if res.Entries[0].Value != 0 {
			t.Errorf("got %v, want 0", res.Entries[0].Value)
		}
	})
}

func TestNickMapResolution(t *testing.T) {
	nickMap := map[string]string{"old_alice": "U1", "also_alice": "U1"}
	resolver := staticResolver{"U1": "alice", "U2": "bob"}

	t.Run("mapped fixed nick resolves to display name", func(t *testing.T) {
		st := storeWithNickMap(nickMap, done(1, "old_alice", 3), done(1, "bob", 4))
		st.SetResolver(resolver)
		res := mustQuery(t, st, Query{Kind: KindAvgAllTime, Selector: SelectorPlayer, Player: "U1"})
		if res.Entries[0].Name != "alice" {
			t.Errorf("got %q, want %q", res.Entries[0].Name, "alice")
		}
	})

	t.Run("unmapped fixed nick passes through as raw name", func(t *testing.T) {
		st := storeWithNickMap(nickMap, done(1, "mystery", 3))
		st.SetResolver(resolver)
		res := mustQuery(t, st, Query{Kind: KindAvgAllTime, Selector: SelectorPlayer, Player: "mystery"})
		if res.Entries[0].Name != "mystery" {
			t.Errorf("got %q, want %q", res.Entries[0].Name, "mystery")
		}
	})

	t.Run("fixed nick results merge with userID results for same player", func(t *testing.T) {
		st := storeWithNickMap(nickMap,
			done(1, "old_alice", 2),    // fixed nick → U1 → alice
			done(2, "also_alice", 4),   // another fixed nick → U1 → alice
			doneUser(3, "U1", 6),       // direct snowflake → alice
		)
		st.SetResolver(resolver)
		res := mustQuery(t, st, Query{Kind: KindAvgAllTime, Selector: SelectorPlayer, Player: "U1"})
		if res.Entries[0].Value != 4 { // (2+4+6)/3
			t.Errorf("got %v, want 4", res.Entries[0].Value)
		}
	})
}
