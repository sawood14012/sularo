package format

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/sawood14012/sularo/internal/test"
)

type JUnit struct{}

type xmlTestSuites struct {
	XMLName    xml.Name      `xml:"testsuites"`
	Name       string        `xml:"name,attr"`
	Tests      int           `xml:"tests,attr"`
	Failures   int           `xml:"failures,attr"`
	Skipped    int           `xml:"skipped,attr"`
	TestSuites []xmlSuite    `xml:"testsuite"`
}

type xmlSuite struct {
	Name     string        `xml:"name,attr"`
	Tests    int           `xml:"tests,attr"`
	Failures int           `xml:"failures,attr"`
	Skipped  int           `xml:"skipped,attr"`
	Cases    []xmlTestCase `xml:"testcase"`
}

type xmlTestCase struct {
	Name      string       `xml:"name,attr"`
	Classname string       `xml:"classname,attr"`
	Time      float64      `xml:"time,attr"`
	Failure   *xmlFailure  `xml:"failure,omitempty"`
	Skipped   *xmlSkipped  `xml:"skipped,omitempty"`
}

type xmlFailure struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

type xmlSkipped struct{}

func (j JUnit) Write(w io.Writer, results []test.Result) {
	var failures, skipped int
	cases := make([]xmlTestCase, 0, len(results))

	for _, r := range results {
		tc := xmlTestCase{
			Name:      r.Name,
			Classname: "sularo",
			Time:      r.Duration.Seconds(),
		}
		switch r.Status {
		case test.StatusFail:
			failures++
			tc.Failure = &xmlFailure{Message: "diff mismatch", Body: r.Message}
		case test.StatusSkip:
			skipped++
			tc.Skipped = &xmlSkipped{}
		}
		cases = append(cases, tc)
	}

	suite := xmlTestSuites{
		Name:     "sularo",
		Tests:    len(results),
		Failures: failures,
		Skipped:  skipped,
		TestSuites: []xmlSuite{{
			Name:     "sularo",
			Tests:    len(results),
			Failures: failures,
			Skipped:  skipped,
			Cases:    cases,
		}},
	}

	fmt.Fprint(w, xml.Header)
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(suite)
	fmt.Fprint(w, "\n")
}
