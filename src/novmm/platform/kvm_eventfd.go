package platform

type BoundEventFd struct {

    // Our system eventfd.
    *EventFd

    // Our VM reference.
    *Vm

    // Address information.
    paddr  Paddr
    size   uint
    is_pio bool

    // Value information.
    has_value bool
    value     uint64
}

func (vm *Vm) NewBoundEventFd(
    paddr Paddr,
    size uint,
    is_pio bool,
    has_value bool,
    value uint64) (*BoundEventFd, error) {

    // Are we enabled?
    if !vm.use_eventfds {
        return nil, nil
    }

    // Create our system eventfd.
    eventfd, err := NewEventFd()
    if err != nil {
        return nil, err
    }

    // Bind the eventfd.
    err = vm.SetEventFd(
        eventfd,
        paddr,
        size,
        is_pio,
        false,
        has_value,
        value)
    if err != nil {
        eventfd.Close()
        return nil, err
    }

    // Return our bound event.
    return &BoundEventFd{
        EventFd:   eventfd,
        Vm:        vm,
        paddr:     paddr,
        size:      size,
        is_pio:    is_pio,
        has_value: has_value,
        value:     value,
    }, nil
}

func (fd *BoundEventFd) Close() error {

    // Unbind the event.
    err := fd.Vm.SetEventFd(
        fd.EventFd,
        fd.paddr,
        fd.size,
        fd.is_pio,
        true,
        fd.has_value,
        fd.value)
    if err != nil {
        return err
    }

    // Close the eventfd.
    return fd.Close()
}

func (vm *Vm) EnableEventFds() {

    // Allow eventfds.
    vm.use_eventfds = true
}
