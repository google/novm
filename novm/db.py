"""
Database (instances, kernels, packs, etc.)

This is the simplest possible file database.
"""
import os
import json
import tempfile
import shutil
import urllib2
import hashlib
import time

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

    def file(self, name, *args):
        return os.path.join(self._root, name, *args)

    def list(self):
        # We store all data in the given directory.
        # All entries are simply stored as json files.
        entries = [os.path.splitext(x) for x in os.listdir(self._root)]
        return [x[0] for x in entries if x[1] == '.json']

    def show(self):
        return dict([(x, self.get(x)) for x in self.list()])

    def add(self, name, obj):
        obj["timestamp"] = time.time()
        with open(self.file("%s.json" % name), 'w') as outf:
            json.dump(
                obj, outf,
                check_circular=True,
                indent=True)

    def get(self, name):
        with open(self.file("%s.json" % name), 'r') as inf:
            return json.load(inf)

    def remove(self, name):
        os.remove(self.file("%s.json" % name))
        if os.path.exists(self.file(name)):
            shutil.rmtree(self.file(name))

    def find(self, **kwargs):
        found = None
        timestamp = 0
        for obj_id in self.list():
            obj_data = self.get(obj_id)
            obj_diff = [
                k for (k, v) in kwargs.items()
                if v != obj_data.get(k)
            ]
            if (not obj_diff and
                obj_data.get("timestamp") > timestamp):
                found = obj_id
                timestamp = obj_data.get("timestamp")
        return found

    def fetch(self, url, **kwargs):
        obj = {"url": url}
        obj.update(kwargs.items())
        sha1 = hashlib.new('sha1')

        with tempfile.NamedTemporaryFile() as tf:
            # Download the file.
            url_file = urllib2.urlopen(url)
            while True:
                data = url_file.read(65536)
                if not data:
                    break
                sha1.update(data)
                tf.write(data)
            tf.flush()

            # Get our computed id.
            obj_id = sha1.hexdigest()
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
