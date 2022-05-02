import pathlib
from generator import Generator
import unittest


class GeneratorTest(unittest.TestCase):
    def test_generator(self):
        whl = pathlib.Path(
            pathlib.Path(__file__).parent, "testdata", "pytest-7.1.1-py3-none-any.whl"
        )
        gen = Generator(None, None)
        mapping = gen.dig_wheel(whl)
        self.assertLessEqual(
            {
                "_pytest": "pytest",
                "_pytest.__init__": "pytest",
                "_pytest._argcomplete": "pytest",
            }.items(),
            mapping.items(),
        )

    def test_stub_generator(self):
        whl = pathlib.Path(
            pathlib.Path(__file__).parent,
            "testdata",
            "django_types-0.15.0-py3-none-any.whl",
        )
        gen = Generator(None, None)
        mapping = gen.dig_wheel(whl)
        self.assertLessEqual(
            {"django_types": "django_types",}.items(), mapping.items(),
        )


if __name__ == "__main__":
    unittest.main()
