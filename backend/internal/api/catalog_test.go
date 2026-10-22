package api_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// creator signs up a creator with a school and returns the session cookie and school id.
func creator(t *testing.T, sub string) (*http.Cookie, string) {
	t.Helper()
	body := fmt.Sprintf(`{"email":"%s@owner.test","password":"password123","name":"%s","schoolName":"%s","subdomain":"%s"}`, sub, sub, sub, sub)
	r := call(t, "POST", appHost, "/accounts/signup", body, nil, appOrig)
	expect(t, r, http.StatusCreated, "")
	return r.cookie, r.body["schools"].([]any)[0].(map[string]any)["schoolId"].(string)
}

// app calls a creator endpoint on the app host with a same-origin Origin header.
func app(t *testing.T, method, path, body string, cookie *http.Cookie) response {
	t.Helper()
	return call(t, method, appHost, path, body, cookie, appOrig)
}

func str(v any) string { return fmt.Sprint(v) }

func field(t *testing.T, m map[string]any, keys ...string) any {
	t.Helper()
	var v any = m
	for _, k := range keys {
		obj, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("no object at %q in %v", k, m)
		}
		v = obj[k]
	}
	return v
}

// outline renders sections as "S1[L1,L2] S2[]" for compact assertions.
func outline(sections []any) string {
	parts := make([]string, 0, len(sections))
	for _, s := range sections {
		sec := s.(map[string]any)
		titles := []string{}
		for _, l := range sec["lectures"].([]any) {
			titles = append(titles, str(l.(map[string]any)["title"]))
		}
		parts = append(parts, fmt.Sprintf("%s[%s]", sec["title"], strings.Join(titles, ",")))
	}
	return strings.Join(parts, " ")
}

func TestCourseBuilder(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	owner, schoolID := creator(t, "cb-main")
	base := "/admin/schools/" + schoolID
	site := "cb-main.localhost:3000"

	// Create: slug from title, de-duplicated; Arabic-only titles fall back to "course".
	r := app(t, "POST", base+"/courses", `{"title":"Intro to Go!","subtitle":"Basics","locale":"en"}`, owner)
	expect(t, r, http.StatusCreated, "")
	if r.body["slug"] != "intro-to-go" || r.body["status"] != "draft" || r.body["locale"] != "en" {
		t.Fatalf("created course = %v", r.body)
	}
	courseID := str(r.body["id"])
	course := base + "/courses/" + courseID

	r = app(t, "POST", base+"/courses", `{"title":"Intro to Go"}`, owner)
	expect(t, r, http.StatusCreated, "")
	if r.body["slug"] != "intro-to-go-2" {
		t.Errorf("duplicate title slug = %v", r.body["slug"])
	}
	emptyCourse := base + "/courses/" + str(r.body["id"])
	r = app(t, "POST", base+"/courses", `{"title":"مقدمة في البرمجة"}`, owner)
	expect(t, r, http.StatusCreated, "")
	if r.body["slug"] != "course" || r.body["locale"] != "ar" {
		t.Errorf("arabic course = %v", r.body)
	}
	expect(t, app(t, "POST", base+"/courses", `{"title":"X","slug":"intro-to-go"}`, owner), http.StatusConflict, "slug_taken")
	expect(t, app(t, "POST", base+"/courses", `{"title":"X","slug":"Bad Slug"}`, owner), http.StatusBadRequest, "invalid_request")
	expect(t, app(t, "POST", base+"/courses", `{"title":"   "}`, owner), http.StatusBadRequest, "invalid_title")
	expect(t, app(t, "PATCH", emptyCourse, `{"slug":"intro-to-go"}`, owner), http.StatusConflict, "slug_taken")

	r = app(t, "GET", base+"/courses", "", owner)
	expect(t, r, http.StatusOK, "")
	if len(r.list) != 3 {
		t.Fatalf("courses = %d, want 3", len(r.list))
	}

	// Update.
	r = app(t, "PATCH", course, `{"title":"Go for Beginners","description":"Learn Go."}`, owner)
	expect(t, r, http.StatusOK, "")
	if r.body["title"] != "Go for Beginners" || r.body["description"] != "Learn Go." || r.body["slug"] != "intro-to-go" {
		t.Errorf("updated course = %v", r.body)
	}

	// Curriculum.
	newSection := func(title string) string {
		r := app(t, "POST", course+"/sections", fmt.Sprintf(`{"title":%q}`, title), owner)
		expect(t, r, http.StatusCreated, "")
		return str(r.body["id"])
	}
	newLecture := func(sectionID, title string) string {
		r := app(t, "POST", course+"/sections/"+sectionID+"/lectures", fmt.Sprintf(`{"title":%q}`, title), owner)
		expect(t, r, http.StatusCreated, "")
		if r.body["status"] != "published" || r.body["isFreePreview"] != false {
			t.Fatalf("new lecture = %v", r.body)
		}
		return str(r.body["id"])
	}
	s1, s2 := newSection("S1"), newSection("S2")
	l1, l2, l3 := newLecture(s1, "L1"), newLecture(s1, "L2"), newLecture(s2, "L3")

	r = app(t, "GET", course, "", owner)
	expect(t, r, http.StatusOK, "")
	if got := outline(r.body["sections"].([]any)); got != "S1[L1,L2] S2[L3]" {
		t.Fatalf("outline = %s", got)
	}
	r = app(t, "PATCH", course+"/sections/"+s1, `{"title":"Getting started"}`, owner)
	expect(t, r, http.StatusOK, "")
	if r.body["title"] != "Getting started" || len(r.body["lectures"].([]any)) != 2 {
		t.Errorf("renamed section = %v", r.body)
	}

	// Blocks: text is sanitized, embeds must be https, order is explicit.
	lecture := course + "/lectures/" + l1
	r = app(t, "POST", lecture+"/blocks", `{"kind":"text","data":{"html":"<p dir=\"rtl\">مرحبا <strong>بك</strong></p><script>alert(1)</script><img src=x onerror=alert(1)>"}}`, owner)
	expect(t, r, http.StatusCreated, "")
	textBlock := str(r.body["id"])
	html := str(field(t, r.body, "data", "html"))
	if strings.Contains(html, "script") || strings.Contains(html, "onerror") || !strings.Contains(html, `<p dir="rtl">مرحبا <strong>بك</strong></p>`) {
		t.Errorf("sanitized html = %s", html)
	}
	expect(t, app(t, "POST", lecture+"/blocks", `{"kind":"embed","data":{"url":"http://example.com/v"}}`, owner), http.StatusBadRequest, "invalid_block")
	expect(t, app(t, "POST", lecture+"/blocks", `{"kind":"quiz"}`, owner), http.StatusBadRequest, "invalid_request")
	r = app(t, "POST", lecture+"/blocks", `{"kind":"embed","data":{"url":"https://www.youtube.com/embed/abc","html":"ignored"}}`, owner)
	expect(t, r, http.StatusCreated, "")
	embedBlock := str(r.body["id"])
	if field(t, r.body, "data", "html") != nil || r.body["position"] != float64(1) {
		t.Errorf("embed block = %v", r.body)
	}
	r = app(t, "POST", lecture+"/reorder", fmt.Sprintf(`{"blockIds":[%q,%q]}`, embedBlock, textBlock), owner)
	expect(t, r, http.StatusOK, "")
	if str(r.list[0].(map[string]any)["id"]) != embedBlock {
		t.Errorf("blocks after reorder = %v", r.list)
	}
	expect(t, app(t, "POST", lecture+"/reorder", fmt.Sprintf(`{"blockIds":[%q]}`, embedBlock), owner), http.StatusBadRequest, "invalid_order")
	r = app(t, "PATCH", lecture+"/blocks/"+textBlock, `{"data":{"html":"<h2>Updated</h2>"}}`, owner)
	expect(t, r, http.StatusOK, "")
	if field(t, r.body, "data", "html") != "<h2>Updated</h2>" {
		t.Errorf("updated block = %v", r.body)
	}
	expect(t, app(t, "DELETE", lecture+"/blocks/"+embedBlock, "", owner), http.StatusNoContent, "")
	expect(t, app(t, "DELETE", lecture+"/blocks/"+embedBlock, "", owner), http.StatusNotFound, "block_not_found")
	r = app(t, "GET", lecture, "", owner)
	expect(t, r, http.StatusOK, "")
	if n := len(r.body["blocks"].([]any)); n != 1 || field(t, r.body, "lecture", "blockCount") != float64(1) {
		t.Errorf("lecture detail = %v", r.body)
	}

	// Reorder: sections swap and L2 moves to S2.
	order := fmt.Sprintf(`{"sections":[{"id":%q,"lectureIds":[%q,%q]},{"id":%q,"lectureIds":[%q]}]}`, s2, l2, l3, s1, l1)
	r = app(t, "POST", course+"/reorder", order, owner)
	expect(t, r, http.StatusOK, "")
	if got := outline(r.body["sections"].([]any)); got != "S2[L2,L3] Getting started[L1]" {
		t.Fatalf("outline after reorder = %s", got)
	}
	missing := fmt.Sprintf(`{"sections":[{"id":%q,"lectureIds":[%q]},{"id":%q,"lectureIds":[%q]}]}`, s2, l2, s1, l1)
	expect(t, app(t, "POST", course+"/reorder", missing, owner), http.StatusBadRequest, "invalid_order")
	dup := fmt.Sprintf(`{"sections":[{"id":%q,"lectureIds":[%q,%q,%q]},{"id":%q,"lectureIds":[%q]}]}`, s2, l2, l3, l3, s1, l1)
	expect(t, app(t, "POST", course+"/reorder", dup, owner), http.StatusBadRequest, "invalid_order")

	// Publishing needs a published lecture; drafts stay off the storefront.
	expect(t, app(t, "POST", emptyCourse+"/publish", "", owner), http.StatusConflict, "course_empty")
	r = call(t, "GET", site, "/storefront/courses", "", nil, "")
	expect(t, r, http.StatusOK, "")
	if len(r.list) != 0 {
		t.Fatalf("storefront lists drafts: %v", r.list)
	}
	expect(t, call(t, "GET", site, "/storefront/courses/intro-to-go", "", nil, ""), http.StatusNotFound, "course_not_found")

	expect(t, app(t, "PATCH", course+"/lectures/"+l3, `{"status":"draft"}`, owner), http.StatusOK, "")
	r = app(t, "PATCH", course+"/lectures/"+l2, `{"isFreePreview":true,"title":"L2 preview"}`, owner)
	expect(t, r, http.StatusOK, "")
	if r.body["isFreePreview"] != true || r.body["title"] != "L2 preview" || r.body["status"] != "published" {
		t.Errorf("updated lecture = %v", r.body)
	}
	r = app(t, "POST", course+"/publish", "", owner)
	expect(t, r, http.StatusOK, "")
	if r.body["status"] != "published" || r.body["publishedAt"] == nil {
		t.Fatalf("published course = %v", r.body)
	}

	r = call(t, "GET", site, "/storefront/courses", "", nil, "")
	expect(t, r, http.StatusOK, "")
	if len(r.list) != 1 {
		t.Fatalf("storefront courses = %v", r.list)
	}
	if c := r.list[0].(map[string]any); c["slug"] != "intro-to-go" || c["lectureCount"] != float64(2) {
		t.Errorf("storefront course = %v", c)
	}
	r = call(t, "GET", site, "/storefront/courses/intro-to-go", "", nil, "")
	expect(t, r, http.StatusOK, "")
	if got := outline(r.body["sections"].([]any)); got != "S2[L2 preview] Getting started[L1]" {
		t.Errorf("storefront outline = %s", got)
	}
	preview := r.body["sections"].([]any)[0].(map[string]any)["lectures"].([]any)[0].(map[string]any)
	if preview["isFreePreview"] != true {
		t.Errorf("free preview flag missing: %v", preview)
	}
	if r.body["description"] != "Learn Go." {
		t.Errorf("storefront description = %v", r.body["description"])
	}

	// Unpublish hides it again; deleting cascades.
	expect(t, app(t, "POST", course+"/unpublish", "", owner), http.StatusOK, "")
	expect(t, call(t, "GET", site, "/storefront/courses/intro-to-go", "", nil, ""), http.StatusNotFound, "course_not_found")
	expect(t, app(t, "DELETE", course+"/sections/"+s2, "", owner), http.StatusNoContent, "")
	expect(t, app(t, "GET", course+"/lectures/"+l2, "", owner), http.StatusNotFound, "lecture_not_found")
	expect(t, app(t, "DELETE", course+"/lectures/"+l1, "", owner), http.StatusNoContent, "")
	expect(t, app(t, "DELETE", course, "", owner), http.StatusNoContent, "")
	expect(t, app(t, "GET", course, "", owner), http.StatusNotFound, "course_not_found")
}

func TestCourseBuilderAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ownerA, schoolA := creator(t, "acc-a")
	ownerB, schoolB := creator(t, "acc-b")
	baseA, baseB := "/admin/schools/"+schoolA, "/admin/schools/"+schoolB

	r := app(t, "POST", baseA+"/courses", `{"title":"Secret","slug":"secret"}`, ownerA)
	expect(t, r, http.StatusCreated, "")
	courseA := str(r.body["id"])
	r = app(t, "POST", baseA+"/courses/"+courseA+"/sections", `{"title":"S"}`, ownerA)
	expect(t, r, http.StatusCreated, "")
	sectionA := str(r.body["id"])
	r = app(t, "POST", baseA+"/courses/"+courseA+"/sections/"+sectionA+"/lectures", `{"title":"L"}`, ownerA)
	expect(t, r, http.StatusCreated, "")
	lectureA := str(r.body["id"])
	expect(t, app(t, "POST", baseA+"/courses/"+courseA+"/publish", "", ownerA), http.StatusOK, "")

	// Authentication and host.
	expect(t, app(t, "GET", baseA+"/courses", "", nil), http.StatusUnauthorized, "unauthorized")
	expect(t, call(t, "GET", "acc-a.localhost:3000", baseA+"/courses", "", ownerA, ""), http.StatusNotFound, "not_found")

	// A non-member cannot see that the school exists.
	expect(t, app(t, "GET", baseA+"/courses", "", ownerB), http.StatusNotFound, "school_not_found")
	expect(t, app(t, "POST", baseA+"/courses", `{"title":"x"}`, ownerB), http.StatusNotFound, "school_not_found")
	expect(t, app(t, "GET", baseA+"/courses/"+courseA, "", ownerB), http.StatusNotFound, "school_not_found")
	expect(t, app(t, "DELETE", baseA+"/courses/"+courseA, "", ownerB), http.StatusNotFound, "school_not_found")

	// Ids of school A are invisible through school B.
	expect(t, app(t, "GET", baseB+"/courses/"+courseA, "", ownerB), http.StatusNotFound, "course_not_found")
	expect(t, app(t, "PATCH", baseB+"/courses/"+courseA, `{"title":"pwned"}`, ownerB), http.StatusNotFound, "course_not_found")
	expect(t, app(t, "DELETE", baseB+"/courses/"+courseA, "", ownerB), http.StatusNotFound, "course_not_found")
	expect(t, app(t, "POST", baseB+"/courses/"+courseA+"/publish", "", ownerB), http.StatusNotFound, "course_not_found")
	expect(t, app(t, "POST", baseB+"/courses/"+courseA+"/sections", `{"title":"x"}`, ownerB), http.StatusNotFound, "course_not_found")
	expect(t, app(t, "POST", baseB+"/courses/"+courseA+"/sections/"+sectionA+"/lectures", `{"title":"x"}`, ownerB), http.StatusNotFound, "section_not_found")
	expect(t, app(t, "GET", baseB+"/courses/"+courseA+"/lectures/"+lectureA, "", ownerB), http.StatusNotFound, "lecture_not_found")
	expect(t, app(t, "POST", baseB+"/courses/"+courseA+"/lectures/"+lectureA+"/blocks", `{"kind":"text"}`, ownerB), http.StatusNotFound, "lecture_not_found")
	r = app(t, "POST", baseB+"/courses", `{"title":"B course"}`, ownerB)
	expect(t, r, http.StatusCreated, "")
	courseB := str(r.body["id"])
	// A section of another course (and school) cannot be referenced in a reorder.
	foreign := fmt.Sprintf(`{"sections":[{"id":%q,"lectureIds":[%q]}]}`, sectionA, lectureA)
	expect(t, app(t, "POST", baseB+"/courses/"+courseB+"/reorder", foreign, ownerB), http.StatusBadRequest, "invalid_order")

	// Storefront of school B does not serve school A's course.
	r = call(t, "GET", "acc-b.localhost:3000", "/storefront/courses", "", nil, "")
	expect(t, r, http.StatusOK, "")
	if len(r.list) != 0 {
		t.Fatalf("school B storefront lists %v", r.list)
	}
	expect(t, call(t, "GET", "acc-b.localhost:3000", "/storefront/courses/secret", "", nil, ""), http.StatusNotFound, "course_not_found")
	expect(t, call(t, "GET", "acc-a.localhost:3000", "/storefront/courses/secret", "", nil, ""), http.StatusOK, "")

	// Roles: support may view but not edit; authors may edit.
	staff, _ := creator(t, "acc-staff")
	setRole := func(role string) {
		t.Helper()
		conn, err := pgx.Connect(context.Background(), pgEnv.OwnerURL)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = conn.Close(context.Background()) }()
		_, err = conn.Exec(context.Background(), `
			INSERT INTO school_members (school_id, account_id, role)
			SELECT $1, id, $2 FROM accounts WHERE email = 'acc-staff@owner.test'
			ON CONFLICT (school_id, account_id) DO UPDATE SET role = excluded.role`, schoolA, role)
		if err != nil {
			t.Fatal(err)
		}
	}
	setRole("support")
	expect(t, app(t, "GET", baseA+"/courses", "", staff), http.StatusOK, "")
	expect(t, app(t, "GET", baseA+"/courses/"+courseA, "", staff), http.StatusOK, "")
	expect(t, app(t, "POST", baseA+"/courses", `{"title":"x"}`, staff), http.StatusForbidden, "forbidden")
	expect(t, app(t, "PATCH", baseA+"/courses/"+courseA, `{"title":"x"}`, staff), http.StatusForbidden, "forbidden")
	expect(t, app(t, "POST", baseA+"/courses/"+courseA+"/unpublish", "", staff), http.StatusForbidden, "forbidden")
	setRole("author")
	expect(t, app(t, "POST", baseA+"/courses", `{"title":"By author"}`, staff), http.StatusCreated, "")

	// The membership check ignores the school a session was created with.
	expect(t, app(t, "GET", baseB+"/courses", "", staff), http.StatusNotFound, "school_not_found")
}
