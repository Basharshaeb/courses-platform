package catalog

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Intro to Go!":            "intro-to-go",
		"  Hello,   World  ":      "hello-world",
		"مقدمة في البرمجة":        "course",
		"Python 101 — للمبتدئين":  "python-101",
		"---":                     "course",
		strings.Repeat("ab ", 50): strings.TrimSuffix(strings.Repeat("ab-", 20), "-"),
	}
	for in, want := range cases {
		got := Slugify(in)
		if got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
		if err := ValidateSlug(got); err != nil {
			t.Errorf("Slugify(%q) = %q is not a valid slug", in, got)
		}
	}
}

func TestValidateSlug(t *testing.T) {
	for _, ok := range []string{"a", "intro-to-go", "go-101"} {
		if err := ValidateSlug(ok); err != nil {
			t.Errorf("ValidateSlug(%q) = %v", ok, err)
		}
	}
	for _, bad := range []string{"", "Intro", "a--b", "-a", "a-", "a b", "دورة", strings.Repeat("a", 81)} {
		if ValidateSlug(bad) == nil {
			t.Errorf("ValidateSlug(%q) accepted", bad)
		}
	}
}

func TestNormalizeBlock(t *testing.T) {
	d, err := normalizeBlock(KindText, BlockData{
		HTML: `<p dir="rtl" style="color:red" onclick="x()">نص <a href="https://example.com">link</a></p><script>alert(1)</script>`,
		URL:  "https://dropped.example",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(d.HTML, "script") || strings.Contains(d.HTML, "onclick") || strings.Contains(d.HTML, "style") {
		t.Errorf("unsafe html kept: %s", d.HTML)
	}
	if !strings.Contains(d.HTML, `dir="rtl"`) || !strings.Contains(d.HTML, `rel="nofollow noreferrer noopener"`) || d.URL != "" {
		t.Errorf("text block = %+v", d)
	}

	for _, bad := range []string{"", "http://example.com", "javascript:alert(1)", "https://"} {
		if _, err := normalizeBlock(KindEmbed, BlockData{URL: bad}); err == nil {
			t.Errorf("embed %q accepted", bad)
		}
	}
	if d, err := normalizeBlock(KindEmbed, BlockData{URL: " https://player.vimeo.com/video/1 "}); err != nil || d.URL != "https://player.vimeo.com/video/1" {
		t.Errorf("embed = %+v, %v", d, err)
	}
	if _, err := normalizeBlock("quiz", BlockData{}); err == nil {
		t.Error("unknown kind accepted")
	}
}
