package platform

/*
#include <string.h>
*/
import "C"

import (
    "bytes"
    "encoding/json"
    "unsafe"
)

func (vm *Vm) LoadVcpus(data []byte) ([]*Vcpu, error) {

    // Decode a new object.
    json_decoder := json.NewDecoder(bytes.NewBuffer([]byte(data)))
    spec := make([]*VcpuInfo, 0, 0)
    err := json_decoder.Decode(&spec)
    if err != nil {
        return nil, err
    }

    // Make our array.
    vcpus := make([]*Vcpu, 0, len(spec))

    // Load all vcpus.
    for _, info := range spec {

        // Create a new vcpu.
        vcpu, err := vm.NewVcpu()
        if err != nil {
            return nil, err
        }

        // Ensure the registers are loaded.
        if info.Regs == nil || len(info.Regs) == 0 {
            // Nothing to do.
        } else if len(info.Regs) <= int(unsafe.Sizeof(vcpu.regs)) {
            // Load the data.
            C.memcpy(
                unsafe.Pointer(&vcpu.regs),
                unsafe.Pointer(&info.Regs[0]),
                C.size_t(len(info.Regs)))
            vcpu.regs_dirty = true
        } else {
            return nil, VcpuIncompatible
        }
        if info.Sregs == nil || len(info.Sregs) == 0 {
            // Nothing to do.
        } else if len(info.Sregs) <= int(unsafe.Sizeof(vcpu.sregs)) {
            C.memcpy(
                unsafe.Pointer(&vcpu.sregs),
                unsafe.Pointer(&info.Sregs[0]),
                C.size_t(len(info.Sregs)))
            vcpu.sregs_dirty = true
        } else {
            return nil, VcpuIncompatible
        }

        // Good to go.
        vcpus = append(vcpus, vcpu)
    }

    // We've okay.
    return vcpus, nil
}
