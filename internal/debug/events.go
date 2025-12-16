package debug

// RenderStartData contains information about the start of a render operation.
type RenderStartData struct {
	Text       string   `json:"text"`
	TextLength int      `json:"text_length"`
	CharHeight int      `json:"char_height"`
	Hardblank  rune     `json:"hardblank"`
	WidthLimit int      `json:"width_limit"`
	PrintDir   int      `json:"print_dir"`
	SmushMode  int      `json:"smush_mode"`
	SmushRules []string `json:"smush_rules"`
}

// RenderEndData contains information about the end of a render operation.
type RenderEndData struct {
	TotalLines   int   `json:"total_lines"`
	TotalRunes   int   `json:"total_runes"`
	TotalGlyphs  int   `json:"total_glyphs"`
	ElapsedMs    int64 `json:"elapsed_ms"`
	BytesWritten int   `json:"bytes_written"`
}

// GlyphData contains information about a processed glyph.
type GlyphData struct {
	Index        int  `json:"index"`
	Rune         rune `json:"rune"`
	Width        int  `json:"width"`
	SpaceGlyph   bool `json:"space_glyph"`
	UnknownSubst bool `json:"unknown_subst,omitempty"`
}

// SplitData contains information about a line split event.
type SplitData struct {
	Reason     string `json:"reason"` // "width", "wordbreak", "newline", "end"
	FSMPrev    int    `json:"fsm_prev"`
	FSMNext    int    `json:"fsm_next"`
	OutlineLen int    `json:"outline_len"`
	Position   int    `json:"position"`
}

// SmushDecisionData contains information about a smush decision.
type SmushDecisionData struct {
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Lch    rune   `json:"lch"`
	Rch    rune   `json:"rch"`
	Result rune   `json:"result"`
	Rule   string `json:"rule"`
}

// SmushAmountRowData contains per-row smush amount calculation details.
type SmushAmountRowData struct {
	GlyphIdx        int    `json:"glyph_idx"`
	Row             int    `json:"row"`
	LineBoundaryIdx int    `json:"line_boundary_idx"`
	CharBoundaryIdx int    `json:"char_boundary_idx"`
	Ch1             rune   `json:"ch1"`
	Ch2             rune   `json:"ch2"`
	AmountBefore    int    `json:"amount_before"`
	AmountAfter     int    `json:"amount_after"`
	Reason          string `json:"reason"` // "none", "ch1_null", "smushable"
	RTL             bool   `json:"rtl"`
}

// RowAppendData contains information about appending to a row.
type RowAppendData struct {
	Row       int `json:"row"`
	StartPos  int `json:"start_pos"`
	CharCount int `json:"char_count"`
	EndBefore int `json:"end_before"`
	EndAfter  int `json:"end_after"`
}

// FlushData contains information about a line flush.
type FlushData struct {
	LineNumber       int   `json:"line_number"`
	RowLengthsBefore []int `json:"row_lengths_before"`
	RowLengthsAfter  []int `json:"row_lengths_after"`
}

// LayoutMergeData contains information about layout merging decisions.
type LayoutMergeData struct {
	RequestedLayout int    `json:"requested_layout"`
	FontDefaults    int    `json:"font_defaults"`
	InjectedRules   int    `json:"injected_rules"`
	FinalLayout     int    `json:"final_layout"`
	FinalSmushMode  int    `json:"final_smush_mode"`
	Rationale       string `json:"rationale"`
}

// LayoutNormalizedData contains information about layout normalisation.
type LayoutNormalizedData struct {
	OldLayout     int    `json:"old_layout"`
	FullLayout    int    `json:"full_layout"`
	FullLayoutSet bool   `json:"full_layout_set"`
	NormalizedH   int    `json:"normalized_horz"`
	NormalizedV   int    `json:"normalized_vert"`
	Rationale     string `json:"rationale"`
}

// FontHeaderData contains parsed font header information.
type FontHeaderData struct {
	Hardblank    int `json:"hardblank"`
	Height       int `json:"height"`
	Baseline     int `json:"baseline"`
	MaxLength    int `json:"max_length"`
	OldLayout    int `json:"old_layout"`
	FullLayout   int `json:"full_layout"`
	PrintDir     int `json:"print_dir"`
	CommentLines int `json:"comment_lines"`
}

// GlyphStatsData contains glyph parsing statistics.
type GlyphStatsData struct {
	RequiredCount int      `json:"required_count"`
	OptionalCount int      `json:"optional_count"`
	MissingList   []string `json:"missing_list,omitempty"`
}

// OptionsData contains render options information.
type OptionsData struct {
	Layout         *int `json:"layout,omitempty"`
	Width          *int `json:"width,omitempty"`
	PrintDirection *int `json:"print_direction,omitempty"`
	UnknownRune    *int `json:"unknown_rune,omitempty"`
	TrimWhitespace bool `json:"trim_whitespace"`
}

// ErrorData contains error information.
type ErrorData struct {
	Type    string                 `json:"type"`
	Message string                 `json:"message"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// WriteRowData contains information about writing a row to output.
type WriteRowData struct {
	RowIdx                int  `json:"row_idx"`
	Trimmed               bool `json:"trimmed"`
	HardblankReplacements int  `json:"hardblank_replacements"`
}

// WriteDoneData contains information about completed write operation.
type WriteDoneData struct {
	BytesWritten int `json:"bytes_written"`
}
