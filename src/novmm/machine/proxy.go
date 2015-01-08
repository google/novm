// Copyright 2014 Google Inc. All rights reserved.
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

package machine

import (
	"io"
)

//
// Proxy --
//
// The proxy is something that allows us to connect
// with the agent inside the VM. At the moment, this
// is only the virtio_console device. Theoretically,
// any of the devices could implement this interface
// if the agent supported it....
//

type Proxy interface {
	io.ReadWriteCloser
}
