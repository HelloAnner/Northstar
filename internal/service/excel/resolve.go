package excel

import (
	"sort"

	"github.com/xuri/excelize/v2"

	"northstar/internal/model"
)

// ResolveOptions ResolveWorkbook 的选项
type ResolveOptions struct {
	Month     int
	Overrides map[string]model.SheetType
}

type resolveCandidate struct {
	sheetName string
	score     float64
	months    []int
	forced    bool
}

// ResolveWorkbook 根据“识别结果 + 用户 override + 月份”选择主表与快照
func ResolveWorkbook(wb *excelize.File, opts ResolveOptions) model.ResolveResult {
	rec := NewRecognizer()
	recognition := rec.RecognizeWorkbook(wb)

	applyOverrides(recognition, opts.Overrides)
	forced := forcedSheetsByType(opts.Overrides)

	mainTypes := []model.SheetType{
		model.SheetTypeWholesaleMain,
		model.SheetTypeRetailMain,
		model.SheetTypeAccommodationMain,
		model.SheetTypeCateringMain,
	}
	snapshotTypes := []model.SheetType{
		model.SheetTypeWholesaleRetailSnapshot,
		model.SheetTypeAccommodationCateringSnapshot,
	}

	result := model.ResolveResult{
		Month:          opts.Month,
		MainSheets:     make(map[model.SheetType]string),
		SnapshotSheets: make(map[model.SheetType]string),
		UnknownSheets:  []string{},
		UnusedSheets:   []string{},
	}

	selected := make(map[string]struct{})

	for _, t := range mainTypes {
		if picked := pickBestSheet(wb, rec, recognition, forced, t, 0); picked != "" {
			result.MainSheets[t] = picked
			selected[picked] = struct{}{}
		}
	}

	for _, t := range snapshotTypes {
		if picked := pickBestSheet(wb, rec, recognition, forced, t, opts.Month); picked != "" {
			result.SnapshotSheets[t] = picked
			selected[picked] = struct{}{}
		}
	}

	for sheetName, r := range recognition {
		if r.Type == model.SheetTypeUnknown {
			result.UnknownSheets = append(result.UnknownSheets, sheetName)
			continue
		}
		if _, ok := selected[sheetName]; ok {
			continue
		}
		result.UnusedSheets = append(result.UnusedSheets, sheetName)
	}

	sort.Strings(result.UnknownSheets)
	sort.Strings(result.UnusedSheets)
	return result
}

func applyOverrides(recognition map[string]model.SheetRecognition, overrides map[string]model.SheetType) {
	if len(overrides) == 0 {
		return
	}
	for sheetName, t := range overrides {
		if _, ok := recognition[sheetName]; !ok {
			continue
		}
		recognition[sheetName] = model.SheetRecognition{
			SheetName:     sheetName,
			Type:          t,
			Score:         1.0,
			MissingFields: []string{},
		}
	}
}

func forcedSheetsByType(overrides map[string]model.SheetType) map[model.SheetType]map[string]struct{} {
	out := make(map[model.SheetType]map[string]struct{})
	for sheetName, t := range overrides {
		if _, ok := out[t]; !ok {
			out[t] = make(map[string]struct{})
		}
		out[t][sheetName] = struct{}{}
	}
	return out
}

func pickBestSheet(wb *excelize.File, rec *Recognizer, recognition map[string]model.SheetRecognition, forced map[model.SheetType]map[string]struct{}, sheetType model.SheetType, month int) string {
	cands := make([]resolveCandidate, 0)
	for _, r := range recognition {
		if r.Type != sheetType {
			continue
		}
		_, isForced := forced[sheetType][r.SheetName]
		cands = append(cands, resolveCandidate{
			sheetName: r.SheetName,
			score:     r.Score,
			months:    sheetMonths(wb, rec, r.SheetName),
			forced:    isForced,
		})
	}
	if len(cands) == 0 {
		return ""
	}

	return bestCandidate(cands, month)
}

func bestCandidate(cands []resolveCandidate, month int) string {
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].forced != cands[j].forced {
			return cands[i].forced
		}
		if month > 0 {
			im := containsInt(cands[i].months, month)
			jm := containsInt(cands[j].months, month)
			if im != jm {
				return im
			}
		}
		if cands[i].score != cands[j].score {
			return cands[i].score > cands[j].score
		}
		return cands[i].sheetName < cands[j].sheetName
	})
	return cands[0].sheetName
}

func sheetMonths(wb *excelize.File, rec *Recognizer, sheetName string) []int {
	months := make(map[int]struct{})

	for _, m := range rec.extractMonthsFromText(sheetName) {
		months[m] = struct{}{}
	}

	for _, h := range readHeaderRow(wb, sheetName) {
		for _, m := range rec.extractMonthsFromText(h) {
			months[m] = struct{}{}
		}
	}

	out := make([]int, 0, len(months))
	for m := range months {
		out = append(out, m)
	}
	sort.Ints(out)
	return out
}

func containsInt(items []int, v int) bool {
	for _, it := range items {
		if it == v {
			return true
		}
	}
	return false
}
