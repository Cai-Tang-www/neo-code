//go:build !windows && !darwin

package infra

import (
	clipboardtext "github.com/atotto/clipboard"
)

func CopyText(text string) error {
	if handled, err := runCopyTextHook(text); handled {
		return err
	}
	return clipboardtext.WriteAll(text)
}

func ReadClipboardText() (string, error) {
	return clipboardtext.ReadAll()
}

func ReadClipboardImage() ([]byte, error) {
	return nil, errClipboardImageUnsupported
}
