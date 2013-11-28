"""
Pretty-printing.
"""
import types

def prettyprint(value, output):

    if isinstance(value, types.NoneType):
        # Print nothing.
        pass

    elif (isinstance(value, types.ListType) or 
        isinstance(value, types.DictType)):

        if len(value) == 0:
            # Empty list?
            return

        # Standardize lists and dictionaries.
        if isinstance(value, types.ListType):
            keys = range(len(value))
            values = value
        else:
            items = value.items()
            keys = [x[0] for x in items]
            values = [x[1] for x in items]

        # Get the first instance.
        # Standardize as a dictionary.
        proto = values[0]
        if not isinstance(proto, types.DictType):
            values = [{"value": x} for x in values]

        # Set a special element "id",
        # which in the case of a dictionary will
        # be the index into the list. In the case
        # of a dictionary, it'll be the key.
        # NOTE: We ensure below that the key "id"
        # is the first element in the sorted keys.
        for k, v in zip(keys, values):
            v["id"] = k

        def format_entry(entry):
            if isinstance(entry, types.ListType):
                return ",".join([str(x) for x in entry])
            else:
                return str(entry)

        # Compute column widths.
        max_width = {}
        for v in values:
            for k, kv in v.items():
                max_width[k] = max(
                    max_width.get(k, 0),
                    len(format_entry(kv)),
                    len(k))

        all_keys = max_width.keys()
        all_keys.remove("id")
        all_keys.insert(0, "id")

        def fmt_row(v):
            cols = " | ".join([
                ("%%-%ds" % max_width[k]) % format_entry(v.get(k))
                for k in all_keys])
            return "".join(["| ", cols, " |" ])

        def sep_row():
            return "-" * (4+sum(max_width.values())+3*(len(max_width)-1))

        # Dump our output.
        output.write(sep_row() + "\n")
        output.write(fmt_row(dict([(k, k) for k in max_width.keys()])) + "\n")
        output.write(sep_row() + "\n")
        for v in values:
            output.write(fmt_row(v) + "\n")
        output.write(sep_row() + "\n")

    else:
        # Default object.
        output.write(str(value) + "\n")
        return
