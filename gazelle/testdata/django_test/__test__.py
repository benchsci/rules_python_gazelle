import unittest

from __init__ import foo
from django.test import TestCase


class FooTest(TestCase):
    def test_foo(self):
        self.assertEqual("foo", foo())


if __name__ == "__main__":
    unittest.main()
