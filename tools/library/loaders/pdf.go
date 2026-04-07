package loader

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/gen2brain/go-fitz"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
)

// PDF loads text data from an io.ReaderAt.
type PDF struct {
	r        io.ReaderAt
	s        int64
	password string
}

// Loader interface verification
var _ documentloaders.Loader = PDF{}

// NewPDF creates a new PDF loader with an io.ReaderAt.
func NewPDF(r io.ReaderAt, size int64) PDF {
	pdf := PDF{
		r: r,
		s: size,
	}
	return pdf
}

func readerAtToBytes(r io.ReaderAt, size int64) ([]byte, error) {
	buf := make([]byte, size)
	if _, err := r.ReadAt(buf, 0); err != nil && err != io.EOF {
		return nil, err
	}
	return buf, nil
}

// Load reads from the io.ReaderAt for the PDF data and returns the documents with the data and with
// metadata attached of the page number and total number of pages of the PDF.
func (p PDF) Load(_ context.Context) ([]schema.Document, error) {
	pdfBytes, err := readerAtToBytes(p.r, p.s)
	if err != nil {
		return nil, err
	}

	doc, err := fitz.NewFromMemory(pdfBytes)
	if err != nil {
		return nil, err
	}
	defer doc.Close()

	pageCount := doc.NumPage()
	docs := []schema.Document{}

	for i := 0; i < pageCount; i++ {
		var textBuf bytes.Buffer

		pageText, err := doc.Text(i)
		if err != nil {
			docs = append(docs, schema.Document{
				PageContent: "",
				Metadata: map[string]any{
					"page":        i + 1,
					"total_pages": pageCount,
					"error":       err.Error(),
				},
			})
			continue
		}

		extractedText := strings.TrimSpace(pageText)
		if extractedText != "" {
			textBuf.WriteString(extractedText)
		}

		// Add the document to the doc list
		docs = append(docs, schema.Document{
			PageContent: textBuf.String(),
			Metadata: map[string]any{
				"page":        i + 1,
				"total_pages": pageCount,
			},
		})
	}

	return docs, nil
}

// LoadAndSplit reads PDF data from the io.ReaderAt and splits it into multiple
// documents using a text splitter.
func (p PDF) LoadAndSplit(ctx context.Context, splitter textsplitter.TextSplitter) ([]schema.Document, error) {
	docs, err := p.Load(ctx)
	if err != nil {
		return nil, err
	}
	return textsplitter.SplitDocuments(splitter, docs)
}
