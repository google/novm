package loader

import (
    "errors"
)

// Linux errors.
var InvalidSetupHeader = errors.New("Setup header past page boundary?")
