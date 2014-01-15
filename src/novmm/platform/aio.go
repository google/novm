package platform

/*
#include <linux/aio_abi.h>
*/
import "C"

import (
    "log"
)

type Aio struct {
    context C.aio_context_t
}

// We create a global AIO context for the entire
// process. In reality, we will be making blocking
// calls and employing goroutines so having a super
// efficient AIO implementation probably wont matter.
var aio *Aio

func NewAio() (*Aio, error) {
    return new(Aio), nil
}

func init() {
    var err error
    aio, err = NewAio()
    if err != nil {
        log.Fatal(err)
    }
}
