"""
Basic CPU state.
"""
from . import state

class Cpu(state.State):

    """ Basic CPU state. """

    # A CPU can be modeled directly using its state,
    # without any external links (like files or network
    # tap devices, for example). Therefore, it is much
    # simpler than the device model.
