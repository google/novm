# Copyright 2014 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
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
