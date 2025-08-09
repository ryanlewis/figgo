package parser

import (
	"bufio"
	"strings"
	"testing"
)

// BenchmarkStripTrailingRun benchmarks the stripTrailingRun function with various inputs
func BenchmarkStripTrailingRun(b *testing.B) {
	tests := []struct {
		name  string
		input string
	}{
		{"ASCII_short", "hello@@"},
		{"ASCII_long_run", "hello" + strings.Repeat("@", 100)},
		{"ASCII_no_run", "hello@"},
		{"Unicode", "hello世界界界"},
		{"Invalid_UTF8", "hello\xff\xff\xff"},
		{"Empty", ""},
		{"Only_endmarks", "@@@@@"},
		{"Very_long_line", strings.Repeat("a", 1000) + "@@@@"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _, _ = stripTrailingRun(tt.input)
			}
		})
	}
}

// BenchmarkParseGlyph benchmarks parsing a single glyph
func BenchmarkParseGlyph(b *testing.B) {
	tests := []struct {
		name   string
		height int
		lines  []string
	}{
		{
			name:   "Small_ASCII",
			height: 3,
			lines: []string{
				"  ##  @@",
				" #  # @@",
				"##### @@",
			},
		},
		{
			name:   "Large_ASCII",
			height: 8,
			lines: []string{
				"     ###     @@",
				"    ## ##    @@",
				"   ##   ##   @@",
				"  ##     ##  @@",
				"  #########  @@",
				"  ##     ##  @@",
				"  ##     ##  @@",
				"  ##     ##  @@",
			},
		},
		{
			name:   "Wide_glyph",
			height: 3,
			lines: []string{
				strings.Repeat("#", 50) + "@@",
				strings.Repeat("#", 50) + "@@",
				strings.Repeat("#", 50) + "@@",
			},
		},
		{
			name:   "Unicode_endmark",
			height: 3,
			lines: []string{
				"  ##  世世",
				" #  # 世世",
				"##### 世世",
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			content := strings.Join(tt.lines, "\n")
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				scanner := strings.NewReader(content)
				s := bufio.NewScanner(scanner)
				_, _ = parseGlyph(s, tt.height, 100)
			}
		})
	}
}

// BenchmarkParse benchmarks parsing a complete font
func BenchmarkParse(b *testing.B) {
	// Create a minimal valid font for benchmarking
	standardFont := `flf2a$ 8 6 14 0 3
Standard by Glenn Chappell & Ian Chai 3/93 -- based on Frank's .sig
Includes ISO Latin-1
figlet release 2.1 -- 12 Aug 1994
Permission is hereby given to modify this font, as long as the
modifier's name is placed on a comment line.

Modified by Paul Burton <solution@earthlink.net> 12/96 to include new parameter
supported by FIGlet and FIGWin.  May also be slightly modified for better use
of new full-width/kern/smush alternatives, but default output is NOT changed.
  
 $$
 $$
 $$
 $$
 $$
 $$
 $$
 $$
 _ @
| |@
| |@
|_|@
(_)@
   @
   @
   @`

	// Add ASCII 33-40 for a more realistic test
	for i := 0; i < 8; i++ {
		standardFont += `
### @
 ## @
 ## @
 ## @
 ## @
 ## @
### @
    @`
	}

	tests := []struct {
		name    string
		content string
	}{
		{"Standard_partial", standardFont},
		{"Empty_glyphs", strings.Replace(standardFont, "###", "", -1)},
		{"Unicode_hardblank", strings.Replace(standardFont, "flf2a$", "flf2a世", -1)},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				r := strings.NewReader(tt.content)
				_, _ = Parse(r)
			}
		})
	}
}

// BenchmarkParseHeader benchmarks header parsing only
func BenchmarkParseHeader(b *testing.B) {
	headers := []struct {
		name    string
		content string
	}{
		{
			"Minimal",
			"flf2a$ 8 6 14 0 0\n",
		},
		{
			"With_comments",
			"flf2a$ 8 6 14 0 3\nComment 1\nComment 2\nComment 3\n",
		},
		{
			"Full_optional",
			"flf2a$ 8 6 14 0 0 1 24463 229\n",
		},
		{
			"Unicode_hardblank",
			"flf2a世 8 6 14 0 0\n",
		},
	}

	for _, h := range headers {
		b.Run(h.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				r := strings.NewReader(h.content)
				_, _ = ParseHeader(r)
			}
		})
	}
}