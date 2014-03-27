package control

import (
    "errors"
)

var InvalidControlSocket = errors.New("Invalid control socket?")
var InternalGuestError = errors.New("Internal guest error?")
