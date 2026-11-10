package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/pborenstein/gh-repopotamus/internal/clone"
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

// cloneProgressMsg reports one finished clone.
type cloneProgressMsg struct{ item clone.Item }

type clonesDoneMsg struct{}

// startClones is startTransfers for a clone plan: it clones in a goroutine
// working on its own copy and reports each item on the returned channel.
func startClones(plan clone.Plan, cloneFn clone.CloneFunc) chan cloneProgressMsg {
	work := plan
	work.Items = append([]clone.Item(nil), plan.Items...)
	ch := make(chan cloneProgressMsg)
	go func() {
		defer close(ch)
		clone.Execute(&work, cloneFn, func(it clone.Item) {
			ch <- cloneProgressMsg{it}
		})
	}()
	return ch
}

func waitClone(ch chan cloneProgressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return clonesDoneMsg{}
		}
		return msg
	}
}

// applyClone records a clone report in plan.
func applyClone(plan *clone.Plan, msg cloneProgressMsg) {
	for i := range plan.Items {
		if plan.Items[i].FullName() == msg.item.FullName() {
			plan.Items[i] = msg.item
			return
		}
	}
}
