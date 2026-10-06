package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// idFragmentBytes — длина случайного фрагмента идентификатора (8 hex-символов).
const idFragmentBytes = 4

// newIDFragment генерирует криптослучайный hex-фрагмент для идентификаторов.
func newIDFragment() (string, error) {
	buf := make([]byte, idFragmentBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// newOrderID генерирует id заказа формата "FRM-<uuid-фрагмент>".
func newOrderID() (string, error) {
	fragment, err := newIDFragment()
	if err != nil {
		return "", err
	}
	return "FRM-" + strings.ToUpper(fragment), nil
}
