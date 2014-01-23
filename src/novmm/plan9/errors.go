// Copyright 2009 The Go9p Authors.  All rights reserved.
// Copyright 2013 Adin Scannell.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the licenses/go9p file.
package plan9

import (
    "errors"
)

// Global errors.
var BufferInsufficient = errors.New("insufficient buffer?")
var InvalidMessage = errors.New("invalid 9pfs message?")
var XattrError = errors.New("unable to fetch xattr?")

// Internal errors.
var Eunknownfid error = &Error{"unknown fid", EINVAL}
var Enoauth error = &Error{"no authentication required", EINVAL}
var Einuse error = &Error{"fid already in use", EINVAL}
var Ebaduse error = &Error{"bad use of fid", EINVAL}
var Eopen error = &Error{"fid already opened", EINVAL}
var Enotdir error = &Error{"not a directory", ENOTDIR}
var Eperm error = &Error{"permission denied", EPERM}
var Etoolarge error = &Error{"i/o count too large", EINVAL}
var Ebadoffset error = &Error{"bad offset in directory read", EINVAL}
var Edirchange error = &Error{"cannot convert between files and directories", EINVAL}
var Enouser error = &Error{"unknown user", EINVAL}
var Enotimpl error = &Error{"not implemented", EINVAL}
var Eexist = &Error{"file already exists", EEXIST}
var Enoent = &Error{"file not found", ENOENT}
var Enotempty = &Error{"directory not empty", EPERM}
