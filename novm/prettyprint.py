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
Pretty-printing.
"""
import time
import types

def prettyprint(value, output):

    if isinstance(value, type(None)):
        # Print nothing.
        pass

    elif (isinstance(value, list) or
        isinstance(value, dict)):

        if len(value) == 0:
            # Empty list?
            return

        # Standardize lists and dictionaries.
        if isinstance(value, list):
            keys = list(range(len(value)))
            values = value
        else:
            def try_int(k):
                try:
                    return int(k)
                except ValueError:
                    return k

            items = sorted([(try_int(k), v) for (k, v) in list(value.items())])
            keys = [x[0] for x in items]
            values = [x[1] for x in items]

        # Get the first instance.
        # Standardize as a dictionary.
        proto = values[0]
        if not isinstance(proto, dict):
            values = [{"value": x} for x in values]

        # Set a special element "id",
        # which in the case of a dictionary will
        # be the index into the list. In the case
        # of a dictionary, it'll be the key.
        # NOTE: We ensure below that the key "id"
        # is the first element in the sorted keys.
        for k, v in zip(keys, values):
            v["id"] = k

        def format_entry(k, v):
            if isinstance(v, float) and k == "timestamp":
                # Hack to print the time.
                return time.ctime(v)
            elif isinstance(v, list):
                return ",".join([str(x) for x in v])
            elif v is not None:
                return str(v)
            else:
                return ""

        # Compute column widths.
        max_width = {}
        for entry in values:
            for k, v in list(entry.items()):
                max_width[k] = max(
                    max_width.get(k, 0),
                    len(format_entry(k, v)),
                    len(k))

        all_keys = list(max_width.keys())
        all_keys.remove("id")
        all_keys.insert(0, "id")

        def fmt_row(entry):
            cols = " | ".join([
                ("%%-%ds" % max_width[k]) % format_entry(k, entry.get(k))
                for k in all_keys])
            return "".join(["| ", cols, " |" ])

        def sep_row():
            return "-" * (4+sum(max_width.values())+3*(len(max_width)-1))

        # Dump our output.
        output.write(sep_row() + "\n")
        output.write(fmt_row(dict([(k, k) for k in list(max_width.keys())])) + "\n")
        output.write(sep_row() + "\n")
        for entry in values:
            output.write(fmt_row(entry) + "\n")
        output.write(sep_row() + "\n")

    else:
        # Default object.
        output.write(str(value) + "\n")
        return

def plainprint(value, output):

    if isinstance(value, type(None)):
        # Print nothing.
        pass

    elif (isinstance(value, list) or
        isinstance(value, dict)):
        # Print individual values.
        for subvalue in value:
            output.write("%s\n" % subvalue)

    else:
        # Print the single value.
        output.write("%s\n" % value)
