package aime

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestDB_CreateDelete(t *testing.T) {
	db := DB{Dir: "./test"}
	db.Create("")
	db.Delete()
}

func TestDB_Report(t *testing.T) {
	db := DB{Dir: "./test"}
	db.Create("")
	defer db.Delete()

	rp := db.CreateReport("test@test.de", true)

	if rp.ID == "" {
		t.Fatal()
	}

	if !db.ExistsReport(rp.ID) {
		t.Fatal()
	}

	rep := db.GetReport(rp.ID)

	if rep.ID != rp.ID {
		t.Fatal()
	}

	if rep.CreatedAt.UnixNano()/int64(time.Minute) != time.Now().UnixNano()/int64(time.Minute) {
		t.Fatal()
	}

	if rep.UpdatedAt.UnixNano()/int64(time.Minute) != time.Now().UnixNano()/int64(time.Minute) {
		t.Fatal()
	}

	db.DeleteReport(rp.ID)

	if db.ExistsReport(rp.ID) {
		t.Fatal()
	}
}

func TestDB_Revision(t *testing.T) {
	db := DB{Dir: "./test"}
	db.Create("")
	defer db.Delete()

	rp := db.CreateReport("test@test.de", true)

	db.CreateRevision(rp.ID, "", []byte("{}"), rp.Token, false)

	rp = db.GetReport(rp.ID)
	if rp.Revisions != 1 {
		t.Fatal()
	}

	rev := db.GetRevision(rp.ID, 1)
	if rev.Version != 1 {
		t.Fatal()
	}
	if rev.ReportID != rp.ID {
		t.Fatal()
	}
	if string(rev.Answers) != "{}" {
		t.Fatal()
	}
	if rev.Public {
		t.Fatal()
	}

	db.CreateRevision(rp.ID, "", []byte("{}"), rp.Token, true)

	rp = db.GetReport(rp.ID)
	if rp.Revisions != 2 {
		t.Fatal()
	}

	rev = db.GetRevision(rp.ID, 2)
	if rev.Version != 2 {
		t.Fatal()
	}

	rev = db.LatestRevision(rp.ID)
	if rev.Version != 2 {
		t.Fatal()
	}

	if rev.CreatedAt.UnixNano()/int64(time.Minute) != time.Now().UnixNano()/int64(time.Minute) {
		t.Fatal()
	}

	if !rev.Public {
		t.Fatal()
	}
}

func TestDB_GetLatestRevisions(t *testing.T) {
	db := DB{Dir: "./test"}
	db.Create("")
	defer db.Delete()

	rp1 := db.CreateReport("test@test.de", false)
	db.CreateRevision(rp1.ID, "", []byte("{}"), rp1.Token, true)

	rp2 := db.CreateReport("test@test.de", false)
	db.CreateRevision(rp2.ID, "", []byte("{}"), rp2.Token, false)

	rp3 := db.CreateReport("test@test.de", false)
	db.CreateRevision(rp3.ID, "", []byte("{}"), rp3.Token, false)
	db.CreateRevision(rp3.ID, "", []byte("{}"), rp3.Token, true)

	rc := db.GetLatestRevisions(false)
	rvs := []*Revision{}
	for r := range rc {
		rvs = append(rvs, r)
	}

	if len(rvs) != 2 {
		t.Fatal()
	}

	rc = db.GetLatestRevisions(true)
	rvs = []*Revision{}
	for r := range rc {
		rvs = append(rvs, r)
	}

	if len(rvs) != 3 {
		t.Fatal()
	}
}

func TestDB_Document(t *testing.T) {
	db := DB{Dir: "./test"}
	db.Create("")
	defer db.Delete()

	fileName := db.UploadDocument([]byte{1, 2, 3})
	if len(fileName) < 5 || !strings.HasSuffix(fileName, ".pdf") {
		t.Fatal()
	}

	fileName2 := db.UploadDocument([]byte{5, 6})
	if fileName == fileName2 {
		t.Fatal()
	}

	fileBytes := db.ReadDocument(fileName)
	if len(fileBytes) != 3 || fileBytes[0] != 1 {
		t.Fatal()
	}

	fileBytes2 := db.ReadDocument(fileName2)
	if len(fileBytes2) != 2 || fileBytes2[0] != 5 {
		t.Fatal()
	}

	if db.ReadDocument("someinvaliddoc.pdf") != nil {
		t.Fatal()
	}
}

func TestDB_Keywords(t *testing.T) {
	db := DB{Dir: "./test"}
	db.Create("../../questionnaire.yaml")
	defer db.Delete()

	te := []struct {
		public   bool
		keywords []string
		category string
	}{
		{true, []string{"test"}, "cf"},
		{true, []string{"test2", "test"}, "cf"},
		{true, []string{"test", "test3"}, "cf"},
		{false, []string{"test3"}, "ce"},
		{false, []string{"test4"}, "cf"},
		{true, []string{"test5"}, "cl"},
	}

	for _, t := range te {
		kwJn := ""

		for i, k := range t.keywords {
			if i > 0 {
				kwJn += ","
			}
			kwJn += "{\"custom\":true,\"value\":\"" + k + "\"}"
		}

		jn := "{\"MD\":{\"5\":[" + kwJn + "]},\"P\":{\"3\":{\"1\":{\"custom\":false,\"value\":\"" + t.category + "\"}}}}"
		r := db.CreateReport("", true)
		db.CreateRevision(r.ID, "", json.RawMessage(jn), r.Token, t.public)
	}

	kw, ct := db.BuildKeywordList([]string{"MD", "5"}, []string{"P", "3", "1"})

	if kw != 4 {
		t.Fatal()
	}
	if ct != 2 {
		t.Fatal()
	}

	if len(db.GetKeyword("test").reports) != 3 {
		t.Fatal(len(db.GetKeyword("test").reports))
	}
	if len(db.GetKeyword("test2").reports) != 1 {
		t.Fatal()
	}
	if len(db.GetKeyword("test3").reports) != 1 {
		t.Fatal()
	}
	if db.GetKeyword("test4") != nil {
		t.Fatal()
	}
	if len(db.GetKeyword("test5").reports) != 1 {
		t.Fatal()
	}

	if len(db.GetCategory("Classification").reports) != 3 {
		t.Fatal()
	}
	if db.GetCategory("Continuous estimation / Regression") != nil {
		t.Fatal()
	}
	if len(db.GetCategory("Clustering").reports) != 1 {
		t.Fatal()
	}
}

func TestDB_Comments(t *testing.T) {
	db := DB{Dir: "./test"}
	db.Create("../../questionnaire.yaml")
	defer db.Delete()

	rep1 := db.CreateReport("a@b.c", true)
	rep2 := db.CreateReport("a2@y.z", true)

	c1 := db.CreateIssue(rep1.ID, "a", "x@y.z", []string{"MD", "1"}, "Test test", 0)

	db.CreateRevision(rep1.ID, "", json.RawMessage("true"), rep1.Token, true)
	c2 := db.CreateIssue(rep1.ID, "b", "x2@y.z", []string{"MD", "2"}, "ABC", 1)

	rep1 = db.GetReport(rep1.ID)
	rep2 = db.GetReport(rep2.ID)

	if rep1.Comments != 2 {
		t.Fatal()
	}
	if rep2.Comments != 0 {
		t.Fatal()
	}

	if !c1.Pending() {
		t.Fatal()
	}
	if c1.Verified {
		t.Fatal()
	}
	if !c1.VerifiedAt.IsZero() {
		t.Fatal()
	}
	if c1.Type != 0 {
		t.Fatal()
	}
	if c1.Content != "Test test" {
		t.Fatal()
	}
	if c1.Name != "a" {
		t.Fatal()
	}
	if c1.RevisionID != 0 {
		t.Fatal()
	}

	db.ValidateIssue(rep1.ID, c1.ID, "hgfh")
	c1 = db.GetIssue(rep1.ID, c1.ID)
	if c1.Verified {
		t.Fatal()
	}
	if !c1.VerifiedAt.IsZero() {
		t.Fatal()
	}

	db.ValidateIssue(rep1.ID, c1.ID, c1.Token)
	c1 = db.GetIssue(rep1.ID, c1.ID)
	if !c1.Verified {
		t.Fatal()
	}
	if c1.VerifiedAt.IsZero() {
		t.Fatal()
	}

	c2 = db.GetIssue(rep1.ID, c2.ID)

	if !c2.Pending() {
		t.Fatal()
	}
	if c2.Verified {
		t.Fatal()
	}
	if !c2.VerifiedAt.IsZero() {
		t.Fatal()
	}
	if c2.Type != 1 {
		t.Fatal()
	}
	if c2.Content != "ABC" {
		t.Fatal()
	}
	if c2.Name != "b" {
		t.Fatal()
	}
	if c2.RevisionID != 1 {
		t.Fatal()
	}

	rc := db.GetReportIssues(rep1.ID, false)
	rcs := []*Issue{}
	for r := range rc {
		rcs = append(rcs, r)
	}
	if len(rcs) != 0 {
		t.Fatal()
	}

	rc = db.GetReportIssues(rep1.ID, true)
	rcs = []*Issue{}
	for r := range rc {
		rcs = append(rcs, r)
	}
	if len(rcs) != 1 {
		t.Fatal()
	}
}

func TestDB_Answers(t *testing.T) {
	db := DB{Dir: "./test"}
	db.Create("../../questionnaire.yaml")
	defer db.Delete()

	rep1 := db.CreateReport("a@b.c", true)

	c1 := db.CreateIssue(rep1.ID, "a", "x@y.z", []string{"MD", "1"}, "Test test", 0)
	db.ValidateIssue(rep1.ID, c1.ID, c1.Token)

	ans1 := db.CreateAnswer(rep1.ID, c1.ID, "An answer", rep1.Token)
	if !ans1.Owner {
		t.Fatal()
	}

	c1 = db.GetIssue(rep1.ID, c1.ID)
	if c1.Pending() {
		t.Fatal()
	}
	if len(c1.Answers) != 1 {
		t.Fatal()
	}

	ans2 := db.CreateAnswer(rep1.ID, c1.ID, "An answer", c1.Token)
	if ans2.Owner {
		t.Fatal()
	}

	c1 = db.GetIssue(rep1.ID, c1.ID)
	if len(c1.Answers) != 2 {
		t.Fatal()
	}

	rc := db.GetReportIssues(rep1.ID, false)
	rcs := []*Issue{}
	for r := range rc {
		rcs = append(rcs, r)
	}
	if len(rcs) != 1 {
		t.Fatal()
	}
}

func TestKeywordGroups_transform_1(t *testing.T) {
	kwg := KeywordGroups{
		{
			Sources:       []string{"a"},
			CaseSensitive: false,
			Targets:       []string{"b"},
		},
	}

	test := []string{"a", "c"}
	test2 := kwg.transform(test)

	if len(test2) != 2 {
		t.Fatal()
	}
	if test2[0] != "b" || test2[1] != "c" {
		t.Fatal()
	}

	test = []string{"a", "A"}
	test2 = kwg.transform(test)

	if len(test2) != 1 {
		t.Fatal()
	}
	if test2[0] != "b" {
		t.Fatal()
	}
}

func TestKeywordGroups_transform_2(t *testing.T) {
	kwg := KeywordGroups{
		{
			Sources:       []string{"a"},
			CaseSensitive: true,
			Targets:       []string{"B"},
		},
	}

	test := []string{"A", "c"}
	test2 := kwg.transform(test)

	if len(test2) != 2 {
		t.Fatal()
	}
	if (test2[0] != "A" || test2[1] != "c") && (test2[1] != "A" || test2[0] != "c") {
		t.Fatal()
	}

	test = []string{"a", "A"}
	test2 = kwg.transform(test)

	if len(test2) != 2 {
		t.Fatal()
	}
	if (test2[0] != "B" || test2[1] != "A") && (test2[1] != "B" || test2[0] != "A") {
		t.Fatal()
	}
}

func TestKeywordGroups_transform_3(t *testing.T) {
	kwg := KeywordGroups{
		{
			Sources:       []string{"a"},
			CaseSensitive: true,
			Targets:       []string{"B"},
		},
		{
			Sources:       []string{"x"},
			CaseSensitive: true,
			Targets:       []string{"X", "Y"},
		},
		{
			Sources:       []string{"a"},
			CaseSensitive: false,
			Targets:       []string{"C", "D"},
		},
	}

	test := []string{"A"}
	test2 := kwg.transform(test)

	if len(test2) != 2 {
		t.Fatal()
	}
	if (test2[0] != "C" || test2[1] != "D") && (test2[1] != "C" || test2[0] != "D") {
		t.Fatal()
	}

	test = []string{"a", "x"}
	test2 = kwg.transform(test)

	if len(test2) != 5 {
		t.Fatal()
	}
}

func TestKeywordGroups_transform_4(t *testing.T) {
	var kwg KeywordGroups

	test := []string{"A", "B"}
	test2 := kwg.transform(test)

	if len(test2) != 2 {
		t.Fatal()
	}
}
