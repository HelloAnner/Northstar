package exporter

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

func openEmbeddedMonthReportTemplate() (*excelize.File, error) {
	raw, err := decodeEmbeddedTemplateGzipBase64(monthReportTemplateGzipBase64)
	if err != nil {
		return nil, err
	}
	return excelize.OpenReader(bytes.NewReader(raw))
}

func decodeEmbeddedTemplateGzipBase64(b64 string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("decode embedded template base64 failed: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("open embedded template gzip failed: %w", err)
	}
	defer zr.Close()
	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("read embedded template gzip failed: %w", err)
	}
	return raw, nil
}
