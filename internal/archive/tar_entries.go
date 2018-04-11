package archive

import (
	"archive/tar"
	"io"
)

// TarEntries interprets the given io.Reader as a tar archive, outputting a list
// of filenames contained within it
func TarEntries(r io.Reader) ([]string, error) {
	entries := []string{}
	tr := tar.NewReader(r)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		entries = append(entries, hdr.Name)
	}

	return entries, nil
}
