package tabular

import "strings"

// Row is one tabular record whose fields retain source order. The slice is
// mutable; copying a Row value aliases its array, while strings are immutable.
type Row []string

// NormalizationConfig describes explicit field-level data changes.
type NormalizationConfig struct {
	// TrimSpace removes leading/trailing Unicode whitespace before EmptyAs.
	TrimSpace bool
	// EmptyAs replaces exactly empty values after optional trimming; empty
	// disables replacement.
	EmptyAs string
}

// HeaderCase controls header-name case conversion.
type HeaderCase uint8

const (
	// HeaderCasePreserve retains source header casing.
	HeaderCasePreserve HeaderCase = iota
	// HeaderCaseLower converts headers to lowercase.
	HeaderCaseLower
	// HeaderCaseUpper converts headers to uppercase.
	HeaderCaseUpper
)

// HeaderConfig describes explicit header normalization and validation.
type HeaderConfig struct {
	// TrimSpace trims Unicode whitespace after the first-field BOM removal.
	TrimSpace bool
	// Case selects Unicode case mapping before Replace; unknown values preserve
	// casing.
	Case HeaderCase
	// Replace maps exact post-trim/post-case names to final names. Replacement
	// values are not trimmed/cased again; direct calls must keep the map
	// stable.
	Replace map[string]string
	// RejectEmpty rejects empty final names, after all transformations.
	RejectEmpty bool
	// RejectDuplicates rejects equal final names, including collisions caused
	// by replacement.
	RejectDuplicates bool
}

// NormalizeRow returns a new slice, trimming then replacing empty fields.
// Unchanged immutable strings may share storage. Nil input yields an empty
// allocated slice; no encoding validation, case folding or Unicode normalization
// is performed.
func NormalizeRow(row Row, config NormalizationConfig) Row {
	normalized := make(Row, len(row))
	for index, value := range row {
		if config.TrimSpace {
			value = strings.TrimSpace(value)
		}
		if value == "" && config.EmptyAs != "" {
			value = config.EmptyAs
		}
		normalized[index] = value
	}
	return normalized
}

// NormalizeHeader returns a normalized copy of header or a typed validation
// error. It removes a UTF-8 BOM from field one, then trims, changes case, applies
// exact replacements and validates final names. A replacement is not normalized
// again. It borrows config.Replace only during the call and returns no partial
// header on validation failure.
func NormalizeHeader(header Row, config HeaderConfig) (Row, error) {
	normalized := make(Row, len(header))
	seen := make(map[string]struct{}, len(header))
	for index, value := range header {
		if index == 0 {
			value = strings.TrimPrefix(value, "\ufeff")
		}
		if config.TrimSpace {
			value = strings.TrimSpace(value)
		}
		switch config.Case {
		case HeaderCasePreserve:
		case HeaderCaseLower:
			value = strings.ToLower(value)
		case HeaderCaseUpper:
			value = strings.ToUpper(value)
		}
		if replacement, ok := config.Replace[value]; ok {
			value = replacement
		}
		if config.RejectEmpty && value == "" {
			return nil, &Error{Kind: ErrorInvalidHeader, Op: "header.normalize", Field: index + 1}
		}
		if config.RejectDuplicates {
			if _, ok := seen[value]; ok {
				return nil, &Error{Kind: ErrorDuplicateHeader, Op: "header.normalize", Field: index + 1}
			}
			seen[value] = struct{}{}
		}
		normalized[index] = value
	}
	return normalized, nil
}
