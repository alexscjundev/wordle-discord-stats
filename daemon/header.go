package daemon

import (
	"fmt"
	"time"
)

const headerGrid = "🟩 ⬛ ⬛ ⬛ ⬛\n🟩 🟩 🟨 ⬛ ⬛\n🟩 🟩 🟩 🟩 🟩"

func buildHeader(t time.Time) string {
	return fmt.Sprintf("%s\n**%s Stats**\n\n", headerGrid, t.Weekday())
}
