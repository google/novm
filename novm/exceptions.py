"""
Exceptions.
"""

class CommandInvalid(Exception):

    def __init__(self):
        super(CommandInvalid, self).__init__("Invalid command")
