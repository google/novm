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
Command-line entry point.
"""
import os
import sys
import inspect
import argparse
import json
import types
import traceback

from . import prettyprint

class Option(object):

    def __init__(self, description):
        self._description = description

    def __str__(self):
        return self._description

class IntOpt(Option):
    """ An integer argument. """
    def __int__(self):
        return 0

class StrOpt(Option):
    """ A string argument. """
    def __str__(self):
        return ""

class BoolOpt(Option):
    """ A simple boolean. """
    def truth(self):
        return False

class ListOpt(Option):
    """ Multiple specification (string). """
    def __len__(self):
        return 0

def alwaysjson(fn):
    # Add a simple attribute so that we can
    # easily treat this function specially
    # when deciding what to do with output.
    setattr(fn, '_alwaysjson_', True)
    return fn

def main(args):
    from . import shell
    cli = shell.NovmShell()

    # Build our options.
    commands = {}
    default_help = {}

    for attr in dir(cli):
        if attr.startswith('_'):
            continue

        # Grab our bound method.
        fn = getattr(cli, attr)

        # Filter static methods.
        if not isinstance(fn, types.MethodType):
            continue

        # Save the basic usage.
        _, line_number = inspect.getsourcelines(fn)
        simple_usage = fn.__doc__.strip().split("\n")[0]
        default_help[(line_number, attr)] = simple_usage

        # Build our local options.
        argspec = inspect.getargspec(fn)
        if argspec.defaults is not None:
            default_args = argspec.args[len(argspec.args)-len(argspec.defaults):]
            real_args = argspec.args[1:len(argspec.args)-len(argspec.defaults)]
            defaults = list(zip(default_args, argspec.defaults))
        else:
            default_args = []
            real_args = argspec.args[1:]
            defaults = {}
        parser = argparse.ArgumentParser(
            formatter_class=argparse.RawDescriptionHelpFormatter,
            epilog=fn.__doc__)

        for (option, spec) in defaults:
            env_value = os.environ.get("NOVM_%s" % option.upper())

            if isinstance(spec, BoolOpt):
                opt_action = "store_true"
                if env_value is not None:
                    opt_default = (env_value.lower() == "true")
                else:
                    opt_default = False
                opt_type = None
            elif isinstance(spec, ListOpt):
                opt_action = "append"
                if env_value is not None:
                    opt_default = env_value.split(",")
                else:
                    opt_default = []
                opt_type = None
            elif isinstance(spec, StrOpt):
                opt_action = "store"
                if env_value is not None:
                    opt_default = env_value
                else:
                    opt_default = None
                opt_type = None
            elif isinstance(spec, IntOpt):
                opt_action = "store"
                if env_value is not None:
                    opt_default = int(env_value)
                else:
                    opt_default = None
                opt_type = int
            else:
                # Shouldn't happen.
                assert False

            if opt_type is not None:
                parser.add_argument(
                    "--%s" % option,
                    action=opt_action,
                    default=opt_default,
                    dest=option,
                    type=opt_type,
                    help=str(spec))
            else:
                parser.add_argument(
                    "--%s" % option,
                    action=opt_action,
                    default=opt_default,
                    dest=option,
                    help=str(spec))

        # Save the command and parser.
        for arg in real_args:
            parser.add_argument(arg)
        if argspec.varargs is not None:
            parser.add_argument(
                argspec.varargs,
                nargs=argparse.REMAINDER)
        commands[attr] = (fn, argspec.args[1:], argspec.varargs, parser)

    # Build our top-level parser.
    command_text = [
        "available commands:"
        "",
    ]
    for (_, command), help_str in sorted(default_help.items()):
        command_text.append("   %-10s -- %s" % (command, help_str))

    top_parser = argparse.ArgumentParser(
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="\n".join(command_text))
    top_parser.add_argument("--debug",
        action="store_true",
        default=False,
        dest="debug",
        help="Show full stack trace on error.")
    top_parser.add_argument("--plain",
        action="store_true",
        default=False,
        dest="plain",
        help="Print result as plain text (where applicable).")
    top_parser.add_argument("--json",
        action="store_true",
        default=False,
        dest="json",
        help="Print result as JSON (where applicable).")
    top_parser.add_argument("command", nargs=argparse.REMAINDER)

    top_args = top_parser.parse_args(args)
    if (len(top_args.command) == 0 or
        not top_args.command[0] in commands):
        top_parser.print_help()
        sys.exit(1)

    (fn, args, varargs, parser) = commands[top_args.command[0]]
    command_args = parser.parse_args(top_args.command[1:])
    try:
        # Run our command.
        if varargs is not None:
            extra_args = getattr(command_args, varargs)
            delattr(command_args, varargs)
            built_args = [getattr(command_args, x) for x in args]
            built_args.extend(extra_args)
            result = fn(*built_args)
        else:
            result = fn(**vars(command_args))
    except Exception as e:
        if top_args.debug:
            traceback.print_exc()
        else:
            sys.stderr.write("error: %s\n" % str(e))
        sys.exit(1)
    except KeyboardInterrupt:
        sys.exit(2)

    # Print the result.
    # See decorator above, alwaysjson().
    if hasattr(fn, '_alwaysjson_') or top_args.json:
        print(json.dumps(result, indent=True))
    elif top_args.plain:
        prettyprint.plainprint(result, sys.stdout)
    else:
        prettyprint.prettyprint(result, sys.stdout)

    # In case we're being called, i.e. do().
    return result
