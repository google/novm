package utils

import (
    "encoding/json"
    "io"
)

func NewDecoder(reader io.Reader) *json.Decoder {
    decoder := json.NewDecoder(reader)
    decoder.UseNumber()
    return decoder
}
