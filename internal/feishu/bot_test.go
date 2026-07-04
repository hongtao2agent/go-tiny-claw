package feishu

import "testing"

func TestParseTextContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "text json",
			content: `{"text":"帮我看 README.md"}`,
			want:    "帮我看 README.md",
		},
		{
			name:    "raw fallback",
			content: "plain text",
			want:    "plain text",
		},
		{
			name:    "empty",
			content: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTextContent(tt.content)
			if got != tt.want {
				t.Fatalf("parseTextContent() = %q, want %q", got, tt.want)
			}
		})
	}
}
