package format

import "testing"

func TestFormatRate(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "zero",
			bytes: 0,
			want:  "0 Б/с",
		},
		{
			name:  "bytes",
			bytes: 512,
			want:  "512 Б/с",
		},
		{
			name:  "kilobytes",
			bytes: 1024,
			want:  "1.0 КБ/с",
		},
		{
			name:  "kilobytes decimal",
			bytes: 81920, // 80 KB
			want:  "80.0 КБ/с",
		},
		{
			name:  "megabytes",
			bytes: 1048576, // 1 MB
			want:  "1.0 МБ/с",
		},
		{
			name:  "megabytes decimal",
			bytes: 1258291, // ~1.2 MB
			want:  "1.2 МБ/с",
		},
		{
			name:  "gigabytes",
			bytes: 1073741824, // 1 GB
			want:  "1.0 ГБ/с",
		},
		{
			name:  "gigabytes decimal",
			bytes: 2147483648, // 2 GB
			want:  "2.0 ГБ/с",
		},
		{
			name:  "high megabytes near gigabyte",
			bytes: 536870912, // 512 MB
			want:  "512.0 МБ/с",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatRate(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatRate(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}
