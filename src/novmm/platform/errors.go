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

package platform

import (
	"errors"
)

// Serialization.
var VcpuIncompatible = errors.New("Incompatible VCPU data?")
var PitIncompatible = errors.New("Incompatible PIT state?")
var IrqChipIncompatible = errors.New("Incompatible IRQ chip state?")
var LApicIncompatible = errors.New("Incompatible LApic state?")

// Register errors.
var UnknownRegister = errors.New("Unknown Register")

// Vcpu state errors.
var NotPaused = errors.New("Vcpu is not paused?")
var AlreadyPaused = errors.New("Vcpu is already paused.")
var UnknownState = errors.New("Unknown vcpu state?")
