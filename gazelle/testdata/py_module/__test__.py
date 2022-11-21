import unittest

from __init__ import foo
from __main__ import main


class FooTest(unittest.TestCase):
    def test_foo(self):
        self.assertEqual("foo", foo())


if __name__ == "__main__":
    unittest.main()
