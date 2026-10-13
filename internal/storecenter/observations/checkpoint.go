package observations

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

func OrderWindows(start, end time.Time) ([]Window, error) {
	start = start.UTC().Truncate(time.Second)
	end = end.UTC().Truncate(time.Second)
	if start.IsZero() || end.IsZero() || start.After(end) || end.Sub(start) > 31*24*time.Hour {
		return nil, ErrInvalid
	}
	out := []Window{}
	for {
		next := start.Add(48 * time.Hour)
		if next.After(end) {
			next = end
		}
		out = append(out, Window{start, next})
		if next.Equal(end) {
			break
		}
		start = next
	}
	return out, nil
}
func copyCheckpoint(c Checkpoint) Checkpoint {
	c.Windows = append([]Window{}, c.Windows...)
	c.Notes = append([]string{}, c.Notes...)
	c.TemuTokenHashes = append([]string(nil), c.TemuTokenHashes...)
	if c.ExpectedTotal != nil {
		n := *c.ExpectedTotal
		c.ExpectedTotal = &n
	}
	return c
}

// E19 ends only when pagination.nextToken is absent. Reported totals and short
// pages cannot substitute for that signal. Token hashes fit the existing 64 KiB
// checkpoint; exhaustion preserves partial evidence in the same sync owner.
func advanceTemuCursor(s Sync, count, total int, next *string) (Checkpoint, string, error) {
	c := copyCheckpoint(s.Progress)
	if s.Kind != Products || c.Page < 1 || count < 0 || count > 10 || total < 0 || len(c.TemuTokenHashes) > 512 {
		return c, "", ErrInvalid
	}
	c.Pages++
	c.Seen += count
	if c.ExpectedTotal == nil {
		v := total
		c.ExpectedTotal = &v
	} else if *c.ExpectedTotal != total {
		c.Note("product_count_changed")
	}
	if next == nil {
		c.TemuPageToken = ""
		if c.Seen != *c.ExpectedTotal {
			c.Note("product_count_mismatch")
		}
		return finish(c)
	}
	if *next == "" || len(*next) > 8192 || !Text(*next, 8192) {
		return c, "", ErrInvalid
	}
	h := sha256.Sum256([]byte(*next))
	hash := hex.EncodeToString(h[:])
	for _, prior := range c.TemuTokenHashes {
		if prior == hash {
			c.Note("provider_cursor_cycle")
			return finish(c)
		}
	}
	if len(c.TemuTokenHashes) >= 512 || c.Pages >= 5000 {
		c.Note("provider_checkpoint_limit")
		return finish(c)
	}
	c.TemuTokenHashes = append(c.TemuTokenHashes, hash)
	c.TemuPageToken = *next
	c.Page++
	return c, "running", nil
}
func finish(c Checkpoint) (Checkpoint, string, error) {
	if c.Incomplete {
		return c, "partial", nil
	}
	return c, "completed", nil
}
func Advance(s Sync, count int, sourceTotal *int) (Checkpoint, string, error) {
	c := copyCheckpoint(s.Progress)
	max := 10
	if s.Kind == Orders {
		max = 30
	}
	if !s.Kind.Valid() || c.Page < 1 || count < 0 || count > max || sourceTotal != nil && *sourceTotal < 0 {
		return c, "", ErrInvalid
	}
	c.Pages++
	c.Seen += count
	if s.Kind == Products {
		if sourceTotal == nil {
			return c, "", ErrInvalid
		}
		if c.ExpectedTotal == nil {
			n := *sourceTotal
			c.ExpectedTotal = &n
		} else if *c.ExpectedTotal != *sourceTotal {
			c.Note("product_count_changed")
		}
		if count < 10 || c.Seen >= *c.ExpectedTotal {
			if c.Seen != *c.ExpectedTotal {
				c.Note("product_count_mismatch")
			}
			return finish(c)
		}
		if c.Pages >= 5000 {
			c.Note("page_limit")
			return finish(c)
		}
		c.Page++
		return c, "running", nil
	}
	if len(c.Windows) == 0 {
		return c, "", ErrInvalid
	}
	// SHEIN's optional count describes this response, not the window total.
	// Compare raw page rows before US filtering; changing page lengths is normal.
	if sourceTotal != nil && *sourceTotal != count {
		c.Note("order_response_count_mismatch")
	}
	// Only pagination exhaustion ends the window; a saturated window must be
	// split with overlap.
	if count == 30 && c.Page >= 333 {
		next, status := SplitWindow(c)
		return next, status, nil
	}
	if count < 30 {
		c.Windows = c.Windows[1:]
		c.Page = 1
		c.Seen = 0
		if len(c.Windows) == 0 {
			return finish(c)
		}
	} else {
		c.Page++
	}
	if c.Pages >= 30000 {
		c.Note("page_limit")
		return finish(c)
	}
	return c, "running", nil
}
func SplitWindow(c Checkpoint) (Checkpoint, string) {
	c = copyCheckpoint(c)
	if len(c.Windows) == 0 {
		return c, "partial"
	}
	w := c.Windows[0]
	if w.End.Sub(w.Start) < 2*time.Second || len(c.Windows) >= 512 {
		c.Note("order_window_saturated")
		return c, "partial"
	}
	midpoint := w.Start.Add(w.End.Sub(w.Start) / 2).Truncate(time.Second)
	c.Windows = append([]Window{{w.Start, midpoint}, {midpoint, w.End}}, c.Windows[1:]...)
	c.Page = 1
	c.Seen = 0
	c.ExpectedTotal = nil
	return c, "running"
}
