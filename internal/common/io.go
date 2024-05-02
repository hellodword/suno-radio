package common

import (
	"fmt"
	"io"
)

func WriteFull(w io.Writer, b []byte) error {
	l := len(b)
	written, err := w.Write(b)
	if err != nil {
		return err
	}

	if written != l {
		err = fmt.Errorf("write full failed %d/%d", written, l)
		return err
	}

	return nil
}
