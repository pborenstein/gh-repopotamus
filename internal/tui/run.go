package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/pborenstein/gh-repopotamus/internal/move"
)

// progressMsg reports one finished transfer. plan indexes the app's plans.
type progressMsg struct {
	plan int
	item move.Item
}

type transfersDoneMsg struct{}

// startTransfers runs every plan in a goroutine and returns the channel it
// reports on. The goroutine works on its own copy of the plans, so the UI's
// copy is only ever changed by Update. The channel closes when all plans
// are done.
func startTransfers(plans []move.Plan, transfer move.TransferFunc) chan progressMsg {
	work := make([]move.Plan, len(plans))
	for i, p := range plans {
		work[i] = p
		work[i].Items = append([]move.Item(nil), p.Items...)
	}
	ch := make(chan progressMsg)
	go func() {
		defer close(ch)
		for i := range work {
			move.Execute(&work[i], transfer, func(it move.Item) {
				ch <- progressMsg{plan: i, item: it}
			})
		}
	}()
	return ch
}

// waitProgress is a tea.Cmd that blocks until the next report. Update
// re-issues it after each progressMsg, so reports arrive one at a time.
func waitProgress(ch chan progressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return transfersDoneMsg{}
		}
		return msg
	}
}

// apply records a progress report in plans.
func apply(plans []move.Plan, msg progressMsg) {
	items := plans[msg.plan].Items
	for i := range items {
		if items[i].Name == msg.item.Name {
			items[i] = msg.item
			return
		}
	}
}
