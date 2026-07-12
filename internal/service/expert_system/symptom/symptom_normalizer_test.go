package symptom

import "testing"

func TestNormalizeRemovesFieldContextWords(t *testing.T) {
	normalizer := NewNormalizerService()

	got := normalizer.Normalize(
		"bau menyengat di area persawahan",
	)

	want := "bau menyengat"

	if got != want {
		t.Fatalf("Normalize() = %q, want %q", got, want)
	}
}

func TestNormalizeCorrectsCommonTypos(t *testing.T) {
	normalizer := NewNormalizerService()

	got := normalizer.Normalize(
		"daunn kunig dan bauu menyengatt cokelat berbintik",
	)

	want := "daun kuning bau menyengat coklat bintik"

	if got != want {
		t.Fatalf("Normalize() = %q, want %q", got, want)
	}
}
