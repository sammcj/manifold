package documents

import (
	"errors"
	"regexp"
	"strings"
)

// RecursiveCharacterTextSplitter is a struct that represents a text splitter
// that splits text based on recursive character separators.
type RecursiveCharacterTextSplitter struct {
	Separators       []string
	KeepSeparator    bool
	IsSeparatorRegex bool
	ChunkSize        int
	OverlapSize      int
	LengthFunction   func(string) int
}

// SplitTextByCount splits the given text into chunks of the given size.
func SplitTextByCount(text string, size int) []string {
	// Slice the string into chunks of the specified size
	var chunks []string
	for i := 0; i < len(text); i += size {
		end := i + size
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[i:end])
	}
	return chunks
}

// SplitText splits the given text using a simple chunk-based approach if the language is not specifically defined.
func (r *RecursiveCharacterTextSplitter) SplitText(text string) []string {
	// Use a simple character count-based splitting mechanism
	chunks := SplitTextByCount(text, r.ChunkSize)

	// Enforce chunk size strictly
	chunks = r.enforceChunkSize(chunks)

	// Apply overlap to the chunks
	if r.OverlapSize > 0 {
		chunks = r.applyOverlap(chunks)
	}

	return chunks
}

// AdaptiveSplit splits text by paragraphs/headings, then enforces a chunk limit.
func (r *RecursiveCharacterTextSplitter) AdaptiveSplit(text string) []string {
	// 1) Split by double newlines (as a proxy for paragraphs)
	rawParagraphs := strings.Split(text, "\n\n")

	// 2) Accumulate paragraphs until we exceed r.ChunkSize tokens,
	// but always enforce the maximum chunk size even if exceeded.
	var chunks []string
	var buffer strings.Builder

	for _, para := range rawParagraphs {
		if r.LengthFunction == nil {
			r.LengthFunction = func(s string) int {
				return len([]rune(s)) // fallback: simple char count
			}
		}
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		var newCandidate string
		if buffer.Len() > 0 {
			newCandidate = buffer.String() + "\n\n" + para
		} else {
			newCandidate = para
		}

		if r.LengthFunction(newCandidate) > r.ChunkSize && buffer.Len() > 0 {
			// Finalize the accumulated buffer,
			// then enforce maximum chunk size on it.
			chunkStr := strings.TrimSpace(buffer.String())
			enforced := r.enforceChunkSize([]string{chunkStr})
			chunks = append(chunks, enforced...)

			// Reset the buffer and start with the current paragraph.
			buffer.Reset()
			buffer.WriteString(para)
		} else {
			if buffer.Len() == 0 {
				buffer.WriteString(para)
			} else {
				buffer.WriteString("\n\n" + para)
			}
		}
	}

	// Finalize the last buffer
	if buffer.Len() > 0 {
		chunkStr := strings.TrimSpace(buffer.String())
		enforced := r.enforceChunkSize([]string{chunkStr})
		chunks = append(chunks, enforced...)
	}

	// 3) Apply overlap to the chunks
	if r.OverlapSize > 0 {
		chunks = r.applyOverlap(chunks)
	}

	return chunks
}

// FromLanguage creates a RecursiveCharacterTextSplitter based on the given language.
// If the language is not a special case, it will default to simple chunk-based splitting.
func FromLanguage(language Language) (*RecursiveCharacterTextSplitter, error) {
	// If language is DEFAULT, create a splitter for general text
	if language == DEFAULT {
		return &RecursiveCharacterTextSplitter{
			ChunkSize:   1200, // slightly bigger default
			OverlapSize: 100,
			LengthFunction: func(s string) int {
				return len([]rune(s))
			},
		}, nil
	}

	// For specific languages, create a specialized splitter
	separators, err := GetSeparatorsForLanguage(language)
	if err != nil {
		return nil, err
	}
	return &RecursiveCharacterTextSplitter{
		Separators:       separators,
		IsSeparatorRegex: false,
		ChunkSize:        1000,
		OverlapSize:      100,
		LengthFunction: func(s string) int {
			return len([]rune(s))
		},
	}, nil
}

// GetSeparatorsForLanguage returns the separators for the given language.
func GetSeparatorsForLanguage(language Language) ([]string, error) {
	switch language {
	case PYTHON:
		return []string{"\nclass ", "\ndef ", "\n\n", "\n", " ", ""}, nil
	case GO:
		return []string{"\nfunc ", "\nvar ", "\nif ", "\n\n", "\n", " ", ""}, nil
	case HTML:
		return []string{"<div", "<p", "<h1", "<br", "<table", "", "\n"}, nil
	case JS, TS:
		return []string{"\nfunction ", "\nconst ", "\nlet ", "\nclass ", "\n\n", "\n", " ", ""}, nil
	case MARKDOWN:
		return []string{"\n#{1,6} ", "\n---+\n", "\n", " ", ""}, nil
	case JSON:
		return []string{"}\n", ""}, nil
	default:
		return nil, errors.New("unsupported language")
	}
}

// escapeString is a helper function that escapes special characters in a string.
func escapeString(s string) string {
	return regexp.QuoteMeta(s)
}

// splitTextWithRegex is a helper function that splits text using a regular expression separator.
func splitTextWithRegex(text string, separator string, keepSeparator bool) []string {
	sepPattern := regexp.MustCompile(separator)
	splits := sepPattern.Split(text, -1)
	if keepSeparator {
		matches := sepPattern.FindAllString(text, -1)
		result := make([]string, 0, len(splits)+len(matches))
		for i, split := range splits {
			result = append(result, split)
			if i < len(matches) {
				result = append(result, matches[i])
			}
		}
		return result
	}
	return splits
}

// Enforce chunk size strictly by splitting each chunk further if needed.
func (r *RecursiveCharacterTextSplitter) enforceChunkSize(chunks []string) []string {
	var result []string
	for _, chunk := range chunks {
		if len(chunk) > r.ChunkSize {
			// Split the chunk into smaller pieces of size `ChunkSize`
			subChunks := SplitTextByCount(chunk, r.ChunkSize)
			result = append(result, subChunks...)
		} else {
			result = append(result, chunk)
		}
	}
	return result
}

// Apply overlap to the chunks.
func (r *RecursiveCharacterTextSplitter) applyOverlap(chunks []string) []string {
	overlappedChunks := make([]string, 0)
	for i := 0; i < len(chunks)-1; i++ {
		currentChunk := chunks[i]
		nextChunk := chunks[i+1]

		// Ensure overlap does not go out of range
		overlapLength := min(len(nextChunk), r.OverlapSize)
		if overlapLength > 0 && overlapLength < len(nextChunk) {
			nextChunkOverlap := nextChunk[:overlapLength]
			overlappedChunk := currentChunk + "\n" + nextChunkOverlap
			overlappedChunks = append(overlappedChunks, overlappedChunk)
		} else {
			overlappedChunks = append(overlappedChunks, currentChunk)
		}
	}

	// Add the last chunk without any overlap
	if len(chunks) > 0 {
		overlappedChunks = append(overlappedChunks, chunks[len(chunks)-1])
	}

	return overlappedChunks
}

// splitTextHelper is a recursive helper function that splits text using the given separators.
func (r *RecursiveCharacterTextSplitter) splitTextHelper(text string, separators []string) []string {
	finalChunks := make([]string, 0)

	if len(separators) == 0 {
		return []string{text}
	}

	// Determine the separator
	separator := separators[len(separators)-1]
	newSeparators := make([]string, 0)
	for i, sep := range separators {
		sepPattern := sep
		if !r.IsSeparatorRegex {
			sepPattern = escapeString(sep)
		}
		if regexp.MustCompile(sepPattern).MatchString(text) {
			separator = sep
			newSeparators = separators[i+1:]
			break
		}
	}

	// Split the text using the determined separator
	splits := splitTextWithRegex(text, separator, r.KeepSeparator)

	// Check each split
	for _, s := range splits {
		if r.LengthFunction(s) < r.ChunkSize {
			finalChunks = append(finalChunks, s)
		} else if len(newSeparators) > 0 {
			// If the split is too large, try to split it further using remaining separators
			recursiveSplits := r.splitTextHelper(s, newSeparators)
			finalChunks = append(finalChunks, recursiveSplits...)
		} else {
			// If no more separators left, add the large chunk as it is
			finalChunks = append(finalChunks, s)
		}
	}

	return finalChunks
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
