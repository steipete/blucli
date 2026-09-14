import unittest

from check_macos_target import check_target


def commands(version="12.0", platform="1"):
    return f"""Load command 0
      cmd LC_BUILD_VERSION
  cmdsize 32
 platform {platform}
    minos {version}
      sdk 15.0
"""


class MacOSTargetTests(unittest.TestCase):
    def test_supported_native_and_universal(self):
        self.assertEqual(check_target(commands()), ["12.0"])
        self.assertEqual(check_target(commands() + commands("12.0.0")), ["12.0", "12.0.0"])

    def test_rejects_newer_older_and_mixed_slices(self):
        for source in (commands("15.0"), commands("11.0"), commands() + commands("15.0")):
            with self.subTest(source=source), self.assertRaises(ValueError):
                check_target(source)

    def test_rejects_missing_malformed_and_wrong_platform(self):
        for source in ("", commands("invalid"), commands(platform="2"), commands().replace("minos", "other")):
            with self.subTest(source=source), self.assertRaises(ValueError):
                check_target(source)

    def test_legacy_macos_command(self):
        self.assertEqual(check_target("Load command 0\n cmd LC_VERSION_MIN_MACOSX\n version 12.0\n"), ["12.0"])


if __name__ == "__main__":
    unittest.main()
