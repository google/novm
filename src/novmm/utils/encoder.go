package utils

import (
    "encoding/json"
    "io"
)

func NewEncoder(writer io.Writer) *json.Encoder {
    encoder := json.NewEncoder(writer)
    return encoder
}
