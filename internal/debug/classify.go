package debug

// Smushing mode constants (matching renderer/types.go)
const (
	smSmush     = 128
	smKern      = 64
	smEqual     = 1
	smLowline   = 2
	smHierarchy = 4
	smPair      = 8
	smBigX      = 16
	smHardblank = 32
)

// FormatSmushRules returns human-readable names for the active smush rules.
func FormatSmushRules(smushMode int) []string {
	var rules []string

	if smushMode&smSmush != 0 {
		rules = append(rules, "SMSmush")
	}
	if smushMode&smKern != 0 {
		rules = append(rules, "SMKern")
	}
	if smushMode&smEqual != 0 {
		rules = append(rules, "Equal")
	}
	if smushMode&smLowline != 0 {
		rules = append(rules, "Lowline")
	}
	if smushMode&smHierarchy != 0 {
		rules = append(rules, "Hierarchy")
	}
	if smushMode&smPair != 0 {
		rules = append(rules, "Pair")
	}
	if smushMode&smBigX != 0 {
		rules = append(rules, "BigX")
	}
	if smushMode&smHardblank != 0 {
		rules = append(rules, "Hardblank")
	}

	if len(rules) == 0 {
		return []string{"None"}
	}
	return rules
}

// ClassifySmushRule returns the name of the rule that produced the given result.
// This analyses the input characters and result to determine which rule was applied.
func ClassifySmushRule(lch, rch, result rune, smushMode int) string {
	// Handle spaces first (always combine)
	if lch == ' ' || rch == ' ' {
		return "space"
	}

	// If not in smushing mode, no rule applies
	if smushMode&smSmush == 0 {
		return "kerning"
	}

	// Universal smushing (no specific rules set)
	if smushMode&63 == 0 {
		return "universal"
	}

	// Check controlled rules in order of precedence

	// Rule 6: Hardblank smushing
	// Note: We can't detect hardblank here without knowing the hardblank character
	// The renderer would need to pass it for accurate classification

	// Rule 1: Equal character smushing
	if smushMode&smEqual != 0 && lch == rch {
		return "equal"
	}

	// Rule 2: Underscore smushing
	if smushMode&smLowline != 0 {
		if lch == '_' && isUnderscoreBorder(rch) {
			return "underscore"
		}
		if rch == '_' && isUnderscoreBorder(lch) {
			return "underscore"
		}
	}

	// Rule 3: Hierarchy smushing
	if smushMode&smHierarchy != 0 {
		if isHierarchySmush(lch, rch, result) {
			return "hierarchy"
		}
	}

	// Rule 4: Opposite pair smushing
	if smushMode&smPair != 0 {
		if result == '|' && isPairSmush(lch, rch) {
			return "pair"
		}
	}

	// Rule 5: Big X smushing
	if smushMode&smBigX != 0 {
		if isBigXSmush(lch, rch, result) {
			return "bigx"
		}
	}

	// Unknown or no match
	return "unknown"
}

// isUnderscoreBorder checks if a character is a border character for underscore smushing.
func isUnderscoreBorder(r rune) bool {
	switch r {
	case '|', '/', '\\', '[', ']', '{', '}', '(', ')', '<', '>':
		return true
	}
	return false
}

// isHierarchySmush checks if the result follows hierarchy smushing rules.
func isHierarchySmush(lch, rch, result rune) bool {
	// '|' beats everything except itself
	if result == '|' && (lch == '|' || rch == '|') {
		return true
	}
	// '/\' beats '[]', '{}', '()', '<>'
	if (result == '/' || result == '\\') && (lch == '/' || lch == '\\' || rch == '/' || rch == '\\') {
		return true
	}
	// '[]' beats '{}', '()', '<>'
	if (result == '[' || result == ']') && (lch == '[' || lch == ']' || rch == '[' || rch == ']') {
		return true
	}
	// '{}' beats '()', '<>'
	if (result == '{' || result == '}') && (lch == '{' || lch == '}' || rch == '{' || rch == '}') {
		return true
	}
	// '()' beats '<>'
	if (result == '(' || result == ')') && (lch == '(' || lch == ')' || rch == '(' || rch == ')') {
		return true
	}
	return false
}

// isPairSmush checks if the characters form an opposite pair.
func isPairSmush(lch, rch rune) bool {
	// Opposite pairs that become '|'
	pairs := [][2]rune{
		{'[', ']'}, {']', '['},
		{'{', '}'}, {'}', '{'},
		{'(', ')'}, {')', '('},
	}
	for _, p := range pairs {
		if lch == p[0] && rch == p[1] {
			return true
		}
	}
	return false
}

// isBigXSmush checks if the result follows Big X smushing rules.
func isBigXSmush(lch, rch, result rune) bool {
	// /\ → |
	if lch == '/' && rch == '\\' && result == '|' {
		return true
	}
	// \/ → Y
	if lch == '\\' && rch == '/' && result == 'Y' {
		return true
	}
	// >< → X
	if lch == '>' && rch == '<' && result == 'X' {
		return true
	}
	return false
}
