package docker

import (
	"fmt"
	"time"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/sirupsen/logrus"
)

type pullInfo struct {
	DownloadStarted bool
	ExtractStarted  bool
	Total           int64
	Downloaded      int64
	Extracted       int64
}

type pullLogFormatter struct {
	Order   []string
	Known   map[string]*pullInfo
	Backoff time.Time
}

// renderProgress generates human-readable and log-file-friendly progress messages.
//
// Every layer goes through the following stages:
// - 1 Pulling fs layer (ID but no size)
// - 1 Waiting (ID but no size)
// - 1+ Downloading
// - 1 Verifying Checksum
// - 1 Download Complete
// - 1+ Extracting
// - 1 Pull Complete
//
// You can't really estimate global progress because the log stream doesn't tell you how big the
// full download size is at any point, it only tells you how big each layer is, and only when that
// layer starts downloading.  The downloads are staggered, so when many layers are present you
// wouldn't know the full download size until you're basically done.
//
// Showing a per-layer status bar is practically impossible without an interactive terminal (as
// docker run would have).
//
// So instead we create a weighted-average status bar, where every layer's download and extraction
// count as equal parts.  The status bar ends up pretty jerky but it still gives a "sensation" of
// progress; things don't look frozen, the user has a rough idea of how far along you are, and the
// logs are still sane afterwards.
func (f *pullLogFormatter) RenderProgress() string {
	var downloaded int64
	var extracted int64
	progress := 0.0
	for _, id := range f.Order {
		info := f.Known[id]
		downloaded += info.Downloaded
		extracted += info.Extracted
		switch {
		case !info.DownloadStarted:
			// No progress on this layer.
		case info.Extracted == info.Total:
			// this layer is complete
			progress += 1.0
		case info.Downloaded == info.Total:
			// download complete, extraction in progress
			progress += 0.5 + 0.5*float64(info.Extracted)/float64(info.Total)
		default:
			progress += 0.5 * float64(info.Downloaded) / float64(info.Total)
		}
	}

	// Normalize by layer count.
	progress /= float64(len(f.Known))

	// 40-character progress bar.
	prog := int(40.0 * progress)

	bar := ""
	for i := 0; i < 40; i++ {
		if i <= prog {
			if prog == 40 || i+1 <= prog {
				// Download is full, or middle of bar.
				bar += "="
			} else {
				// Boundary between bar and spaces.
				bar += ">"
			}
		} else {
			bar += " "
		}
	}

	return fmt.Sprintf(
		"[%v] Downloaded: %.1fMB, Extracted %.1fMB",
		bar,
		float64(downloaded)/1e6,
		float64(extracted)/1e6,
	)
}

func (f *pullLogFormatter) backoffOrRenderProgress() *string {
	// Log at most one line every 1 second.
	now := time.Now().UTC()
	if now.Before(f.Backoff) {
		return nil
	}
	f.Backoff = now.Add(1 * time.Second)

	return ptrs.Ptr(f.RenderProgress())
}

// Update returns nil or a rendered progress update for the end user.
func (f *pullLogFormatter) Update(msg jsonmessage.JSONMessage) *string {
	if msg.Error != nil {
		logrus.Errorf("%d: %v", msg.Error.Code, msg.Error.Message)
		return nil
	}

	var info *pullInfo
	var ok bool

	switch msg.Status {
	case "Pulling fs layer", "Waiting":
		if _, ok = f.Known[msg.ID]; !ok {
			// New layer!
			f.Known[msg.ID] = &pullInfo{}
			f.Order = append(f.Order, msg.ID)
		}
		return nil

	case "Downloading":
		if info, ok = f.Known[msg.ID]; !ok {
			logrus.Error("message ID not found for downloading message!")
			return nil
		}
		if info.ExtractStarted {
			logrus.Error("got downloading message after extraction started!")
			return nil
		}
		info.Downloaded = msg.Progress.Current
		// The first "Downloading" msg is important, as it gives us the layer size.
		if !info.DownloadStarted {
			info.DownloadStarted = true
			info.Total = msg.Progress.Total
		}
		return f.backoffOrRenderProgress()

	case "Extracting":
		if info, ok = f.Known[msg.ID]; !ok {
			logrus.Error("message ID not found for extracting message!")
			return nil
		}
		info.Extracted = msg.Progress.Current
		if !info.ExtractStarted {
			info.ExtractStarted = true
			// Forcibly mark Downloaded as completed.
			info.Downloaded = info.Total
		}
		return f.backoffOrRenderProgress()

	case "Pull complete":
		if info, ok = f.Known[msg.ID]; !ok {
			logrus.Error("message ID not found for completed message!")
			return nil
		}
		// Forcibly mark Extracted as completed.
		info.Extracted = info.Total
		return f.backoffOrRenderProgress()
	}

	return nil
}
