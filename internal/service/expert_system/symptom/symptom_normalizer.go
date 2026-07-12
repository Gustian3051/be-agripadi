package symptom

import (
	"regexp"
	"strings"
)

type NormalizerService struct {
	stopwords map[string]struct{}

	typoCorrections map[string]string
}

func NewNormalizerService() *NormalizerService {

	return &NormalizerService{
		stopwords: buildStopwords(),

		typoCorrections: buildTypoCorrections(),
	}
}

//
// ============================================================
// NORMALIZE TEXT
// ============================================================
//

func (s *NormalizerService) Normalize(
	input string,
) string {

	if input == "" {
		return ""
	}

	// ========================================================
	// LOWERCASE
	// ========================================================

	text := strings.ToLower(
		input,
	)

	// ========================================================
	// TRIM
	// ========================================================

	text = strings.TrimSpace(
		text,
	)

	// ========================================================
	// TYPO CORRECTION
	// ========================================================

	text = s.correctTypos(
		text,
	)

	// ========================================================
	// REMOVE SPECIAL CHARACTERS
	// ========================================================

	text = s.removeSpecialCharacters(
		text,
	)

	// ========================================================
	// NORMALIZE WHITESPACE
	// ========================================================

	text = s.normalizeWhitespace(
		text,
	)

	// ========================================================
	// TOKENIZATION
	// ========================================================

	tokens := strings.Fields(
		text,
	)

	// ========================================================
	// STOPWORD REMOVAL
	// ========================================================

	tokens = s.removeStopwords(
		tokens,
	)

	// ========================================================
	// FINAL JOIN
	// ========================================================

	return strings.Join(
		tokens,
		" ",
	)
}

//
// ============================================================
// NORMALIZE MANY
// ============================================================
//

func (s *NormalizerService) NormalizeMany(
	inputs []string,
) []string {

	results := make(
		[]string,
		0,
		len(inputs),
	)

	for _, item := range inputs {

		normalized :=
			s.Normalize(
				item,
			)

		if normalized == "" {
			continue
		}

		results = append(
			results,
			normalized,
		)
	}

	return results
}

//
// ============================================================
// TYPO CORRECTION
// ============================================================
//

func (s *NormalizerService) correctTypos(
	input string,
) string {

	words := strings.Fields(
		input,
	)

	results := make(
		[]string,
		0,
		len(words),
	)

	for _, word := range words {

		if corrected, ok :=
			s.typoCorrections[word]; ok {

			results = append(
				results,
				corrected,
			)

			continue
		}

		results = append(
			results,
			word,
		)
	}

	return strings.Join(
		results,
		" ",
	)
}

//
// ============================================================
// REMOVE STOPWORDS
// ============================================================
//

func (s *NormalizerService) removeStopwords(
	tokens []string,
) []string {

	results := make(
		[]string,
		0,
		len(tokens),
	)

	for _, token := range tokens {

		if _, exists :=
			s.stopwords[token]; exists {

			continue
		}

		results = append(
			results,
			token,
		)
	}

	return results
}

//
// ============================================================
// STEM TOKENS
// ============================================================
//

func (s *NormalizerService) removeSpecialCharacters(
	input string,
) string {

	reg := regexp.MustCompile(
		`[^a-zA-Z0-9\s]+`,
	)

	return reg.ReplaceAllString(
		input,
		"",
	)
}

//
// ============================================================
// NORMALIZE WHITESPACE
// ============================================================
//

func (s *NormalizerService) normalizeWhitespace(
	input string,
) string {

	return strings.Join(
		strings.Fields(input),
		" ",
	)
}

//
// ============================================================
// STOPWORDS
// ============================================================
//

func buildStopwords() map[string]struct{} {

	words := []string{
		"yang",
		"dan",
		"di",
		"ke",
		"dari",
		"untuk",
		"pada",
		"dengan",
		"ada",
		"itu",
		"ini",
		"karena",
		"atau",
		"saya",
		"tanaman",
		"padi",
		"terlihat",
		"bagian",
		"seperti",
		"jadi",
		"lebih",
		"area",
		"areal",
		"lahan",
		"sawah",
		"persawahan",
		"sekitar",
		"terjadi",
		"mengalami",
		"terdapat",
		"terlihat",
		"tampak",
		"kelihatan",
		"muncul",
		"mulai",
		"banyak",
		"jumlah",
		"bagian",
		"ada",
	}

	results := make(
		map[string]struct{},
		len(words),
	)

	for _, item := range words {

		results[item] = struct{}{}
	}

	return results
}

//
// ============================================================
// TYPO CORRECTIONS
// ============================================================
//

func buildTypoCorrections() map[string]string {

	return map[string]string{

		// ====================================================
		// LEAF
		// ====================================================

		"daun2": "daun",
		"daunn": "daun",

		// ====================================================
		// YELLOW
		// ====================================================

		"kunig":     "kuning",
		"kunning":   "kuning",
		"menguning": "kuning",
		"nguning":   "kuning",

		// ====================================================
		// SPOTS
		// ====================================================

		"bercak2":   "bercak",
		"bercakkk":  "bercak",
		"berbintik": "bintik",
		"bintik2":   "bintik",
		"bercak":    "bintik",

		// ====================================================
		// BROWN
		// ====================================================

		"cokelat":      "coklat",
		"kecoklatan":   "coklat",
		"kecokelatan":  "coklat",
		"coklatan":     "coklat",
		"cokelatan":    "coklat",
		"kehitaman":    "hitam",
		"menghitam":    "hitam",
		"menghitamkan": "hitam",

		// ====================================================
		// DRY
		// ====================================================

		"keringg": "kering",

		// ====================================================
		// PEST
		// ====================================================

		"hamaa": "hama",

		// ====================================================
		// STEM
		// ====================================================

		"batangg":    "batang",
		"berlubang":  "lubang",
		"bolong":     "lubang",
		"berbolong":  "lubang",
		"berlubang2": "lubang",
		"rapuh":      "kropos",

		// ====================================================
		// BORER LARVAE
		// ====================================================

		"uler":  "ulat",
		"ulet":  "ulat",
		"larva": "ulat",

		// ====================================================
		// EMPTY GRAIN
		// ====================================================

		"kopong": "hampa",
		"gabug":  "hampa",
		"gabuk":  "hampa",
		"gabug2": "hampa",

		// ====================================================
		// SMELL
		// ====================================================

		"menyenggat": "menyengat",
		"menyengatt": "menyengat",
		"bauu":       "bau",
	}
}
