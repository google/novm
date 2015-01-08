// Original file Copyright 2009 The Go9p Authors. All rights reserved.
// Full license available in licenses/go9p.
//
// Modifications Copyright 2014 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
