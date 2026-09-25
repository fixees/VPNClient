package format

import "fmt"

// FormatRate formats bytes/sec into Russian-friendly units (Б/с, КБ/с, МБ/с, ГБ/с).
// Mirrors the frontend fmtRate style from main.js.
func FormatRate(bytesPerSec int64) string {
	v := float64(bytesPerSec)
	if v >= 1024*1024*1024 {
		return fmt.Sprintf("%.1f ГБ/с", v/(1024*1024*1024))
	}
	if v >= 1024*1024 {
		return fmt.Sprintf("%.1f МБ/с", v/(1024*1024))
	}
	if v >= 1024 {
		return fmt.Sprintf("%.1f КБ/с", v/1024)
	}
	return fmt.Sprintf("%.0f Б/с", v)
}
