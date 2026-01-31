package exporter

// ProgressEvent 导出进度事件（用于 UI 展示）
type ProgressEvent struct {
	Percent int
	Stage   string
}

func reportProgress(progress func(ProgressEvent), percent int, stage string) {
	if progress == nil {
		return
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	progress(ProgressEvent{
		Percent: percent,
		Stage:   stage,
	})
}
