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
Database (instances, kernels, packs, etc.)

This is the simplest possible file database.
"""
import os
import json
import tempfile
import shutil
import time
import sys

if sys.version_info[0] == 3:
    from urllib.request import urlopen
else:
    from urllib2 import urlopen

from . import utils

class Nodb(object):

    """
    A directory of JSON objects.

    This is used as very simple "database".
    """

    def __init__(self, root):
        self._root = root
        self._create()

    def _create(self):
        try:
            os.makedirs(self._root)
        except OSError:
            if not os.path.isdir(self._root):
                raise

    def file(self, obj_id, *args):
        return os.path.join(self._root, obj_id, *args)

    def list(self):
        # We store all data in the given directory.
        # All entries are simply stored as json files.
        entries = [os.path.splitext(x) for x in os.listdir(self._root)]
        return [x[0] for x in entries if x[1] == '.json']

    def show(self):
        keys = self.list()
        result = {}
        for key in keys:
            try:
                result[key] = self.get(obj_id=key)
            except KeyError:
                # Race condition, deleted.
                # Ignore and continue.
                continue
        return result

    def add(self, obj_id, obj):
        obj["timestamp"] = time.time()
        with open(self.file("%s.json" % obj_id), 'w') as outf:
            json.dump(
                obj, outf,
                check_circular=True,
                indent=True)

    def get(self, obj_id=None, **kwargs):
        obj_id = self.find(obj_id=obj_id, **kwargs)
        with open(self.file("%s.json" % obj_id), 'r') as inf:
            return json.load(inf)

    def remove(self, obj_id=None, **kwargs):
        obj_id = self.find(obj_id=obj_id, **kwargs)
        os.remove(self.file("%s.json" % obj_id))
        if os.path.exists(self.file(obj_id)):
            shutil.rmtree(self.file(obj_id))

    def find(self, obj_id=None, **kwargs):
        if obj_id is not None:
            return obj_id

        found = None
        timestamp = 0
        for obj_id in self.list():
            obj_data = self.get(obj_id)
            obj_diff = [
                k for (k, v) in list(kwargs.items())
                if v != obj_data.get(k)
            ]
            if (not obj_diff and
                obj_data.get("timestamp") > timestamp):
                found = obj_id
                timestamp = obj_data.get("timestamp")

        if found:
            return found
        else:
            raise KeyError(str(kwargs))

    def fetch(self, url, **kwargs):
        obj = {"url": url}
        obj.update(list(kwargs.items()))

        with tempfile.NamedTemporaryFile() as tf:
            # Download the file.
            url_file = urlopen(url)
            obj_id = utils.copy(tf, url_file, hash=True)

            # Does this already exist?
            if obj_id in self.list():
                return obj_id

            # Create the path.
            obj_path = self.file(obj_id)
            os.makedirs(obj_path)

            # Extract it.
            utils.unpackdir(tf.name, obj_path)

        # Save the path.
        self.add(obj_id, obj)
        return obj_id
