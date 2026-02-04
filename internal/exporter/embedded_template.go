package exporter

import (
	"github.com/xuri/excelize/v2"
	"northstar/internal/reporttpl"
)

func openEmbeddedMonthReportTemplate() (*excelize.File, error) {
	return reporttpl.OpenEmbeddedMonthReportTemplate()
}

// OpenEmbeddedMonthReportTemplate 打开内置月报模板
func OpenEmbeddedMonthReportTemplate() (*excelize.File, error) {
	return openEmbeddedMonthReportTemplate()
}
