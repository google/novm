"""
Serializable state.

This is a class which accepts arbitrary
arguments and implements a basic interface
to get/set state and retreive it. It is
essentially a dictionary.
"""

class State(object):

    def __init__(self, **kwargs):
        self._state = kwargs

    def state(self):
        """ Returns the full state. """
        return self._state

    def get(self, key, default=None):
        """ Return a single data item. """
        return self._state.get(key, default)
