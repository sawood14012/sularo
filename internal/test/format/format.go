package format

import (
	"io"

	"github.com/sawood14012/sularo/internal/test"
)

type Formatter interface {
	Write(w io.Writer, results []test.Result)
}
