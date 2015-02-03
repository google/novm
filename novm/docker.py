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
Docker utilities.
"""
import json
import random
import base64
import tempfile
import subprocess
import os
import sys

if sys.version_info[0] == 3:
    from http.client import HTTPSConnection
else:
    from httplib import HTTPSConnection

class RegistryClient(object):
    """
    A simple client for the docker registry.
    """

    def __init__(self,
            db,
            host=None,
            registry=None,
            username=None,
            password=None):

        if host is None:
            host = "index.docker.io"

        super(RegistryClient, self).__init__()

        self._db = db
        self._host = host
        self._token = None
        self._registries = [host]

        if username is not None and password is not None:
            self._auth = base64.encodestring(
                "%s:%s" % (username, password)).replace('\n', '')
        else:
            self._auth = None

    def tags(self, repository):
        data = self._request("v1/repositories/%s/tags" % repository)
        if isinstance(data, list):
            # Hmm... so much for semantic versionining.
            # The return here (a list) doesn't appear to
            # match their API at all.
            return dict([
                (x.get("name"), x.get("layer"))
                for x in data
            ])
        else:
            return data

    def tag_delete(self, repository, tag):
        self._request(
            "v1/repositories/%s/tags/%s" % (repository, tag),
            method="DELETE")

    def tag_create(self, repository, tag, image_id):
        self._request(
            "v1/repositories/%s/tags/%s" % (repository, tag),
            body=image_id)

    def images(self, repository):
        return [
            x.get("id")
            for x in self._request("v1/repositories/%s/images" % repository)
        ]

    def image_download(self, repository, image_id, output):
        (token, endpoint) = self._request(
            "v1/repositories/%s/images" % repository,
            auth=True)
        return self._request(
            "v1/images/%s/layer" % image_id,
            output=output,
            token=token,
            host=endpoint)

    def image_ancestry(self, repository, image_id):
        (token, endpoint) = self._request(
            "v1/repositories/%s/images" % repository,
            auth=True)
        return self._request(
            "v1/images/%s/ancestry" % image_id,
            token=token,
            host=endpoint)

    def image_info(self, repository, image_id):
        (token, endpoint) = self._request(
            "v1/repositories/%s/images" % repository,
            auth=True)
        return self._request(
            "v1/images/%s/json" % image_id,
            token=token,
            host=endpoint)

    def _request(self,
            url,
            method=None,
            body=None,
            output=None,
            host=None,
            token=None,
            auth=False):

        if method is None:
            if body is None:
                method = "GET"
            else:
                method = "PUT"
        if host is None:
            host = self._host

        # Build our headers.
        headers = {}
        headers["Accept"] = "application/json"
        headers["Content-Type"] = "application/json"
        if token is not None:
            headers["Authorization"] = "Token %s" % token
        elif auth and self._auth is not None:
            headers["Authorization"] = "Basic %s" % self._auth
            headers["X-Docker-Token"] = "true"

        # Open the requested URL.
        url = "https://%s/%s" % (host, url)
        con = HTTPSConnection(host)
        con.request(method, url, body=body, headers=headers)
        resp = con.getresponse()

        if int(resp.status / 100) != 2:
            # Return the given error.
            raise Exception(resp.read())

        if auth:
            # This is a special case where we are authenticating
            # with an index prior to querying the underlying registry.
            token = resp.getheader("X-Docker-Token")
            endpoints = resp.getheader("X-Docker-Endpoints")
            if endpoints:
                endpoint = random.choice(endpoints.split(","))
            else:
                endpoint = self._host
            return (token, endpoint)

        if output is not None:
            # Write out to the given stream.
            while True:
                content = resp.read(1024*1024)
                if not content:
                    break
                output.write(content)
            output.flush()
        else:
            # Interpret as JSON (since we requested it).
            content = resp.read()
            if content:
                return json.loads(content)

    def pull_image(self, repository, image_id):
        try:
            # Is this already in our database?
            obj = self._db.get(image_id)

            if "parent" in obj:
                parent = obj["parent"]
            else:
                parent = None

        except (IOError, KeyError):
            # Fetch object information.
            image_info = self.image_info(repository, image_id)

            # Do we have a parent?
            if "parent" in image_info:
                parent = image_info["parent"]
            else:
                parent = None

            # Make sure we're ready.
            image_dir = self._db.file(image_id)
            if not os.path.exists(image_dir):
                try:
                    os.makedirs(image_dir)
                except OSError:
                    pass

            with tempfile.NamedTemporaryFile() as tf:
                # Pull down the data.
                self.image_download(repository, image_id, tf)

                # Extract.
                subprocess.check_call([
                    "fakeroot",
                    "tar",
                    "-C", os.path.abspath(image_dir),
                    "-xf", os.path.abspath(tf.name),
                ])

            # Save the image data.
            self._db.add(image_id, image_info)

        if parent is not None:
            all_dirs = self.pull_image(repository, parent)
            all_dirs.append(self._db.file(image_id))
        else:
            all_dirs = [self._db.file(image_id)]

        return all_dirs

    def pull_repository(self, repository):
        if ":" in repository:
            (repository, tag) = repository.split(":", 1)
            image_id = self.tags(repository).get(tag)
            if not image_id:
                raise KeyError(tag)
        else:
            # Pick a random tag?
            tags = self.tags(repository)
            (tag, image_id) = list(tags.items())[0]

        # Fetch our image.
        return self.pull_image(repository, image_id)
