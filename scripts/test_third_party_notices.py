import os
import sys
import re
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import third_party_notices as tpn  # noqa: E402


def module(root, name, files):
    d = os.path.join(root, name)
    os.makedirs(d)
    for fname, body in files.items():
        with open(os.path.join(d, fname), "w") as f:
            f.write(body)
    return d


class Render(unittest.TestCase):
    def test_any_number_of_modules_each_with_its_own_text(self):
        with tempfile.TemporaryDirectory() as t:
            mods = [(f"example.com/m{i}", f"v1.{i}.0", module(t, f"m{i}", {"LICENSE": f"licence text {i}\n"})) for i in range(5)]
            out = tpn.render(mods)
        for i in range(5):
            self.assertIn(f"example.com/m{i} v1.{i}.0", out)
            self.assertIn(f"licence text {i}", out)

    def test_every_licence_and_notice_file_is_carried(self):
        with tempfile.TemporaryDirectory() as t:
            d = module(t, "m", {"LICENSE": "the licence\n", "NOTICE": "the notice\n", "README": "not a licence\n"})
            out = tpn.render([("example.com/m", "v1.0.0", d)])
        self.assertIn("the licence", out)
        self.assertIn("the notice", out)
        self.assertNotIn("not a licence", out)

    def test_a_module_without_a_licence_file_is_refused_by_name(self):
        with tempfile.TemporaryDirectory() as t:
            ok = module(t, "ok", {"LICENSE": "x\n"})
            bare = module(t, "bare", {"README": "no licence here\n"})
            with self.assertRaises(tpn.MissingNotice) as cm:
                tpn.render([("example.com/ok", "v1.0.0", ok), ("example.com/bare", "v2.0.0", bare)])
        self.assertIn("example.com/bare@v2.0.0", str(cm.exception))
        self.assertNotIn("example.com/ok", str(cm.exception))


class Check(unittest.TestCase):
    def run_check(self, notices):
        with tempfile.TemporaryDirectory() as t:
            lic = module(t, "m", {"LICENSE": "licence text\n"})
            tpn_graph = tpn.graph
            tpn.graph = lambda root: [("example.com/m", "v1.0.0", lic)]
            try:
                root = os.path.join(t, "root")
                os.makedirs(os.path.join(root, "scripts", "third_party"))
                with open(os.path.join(root, "go.mod"), "w") as f:
                    f.write("module m\n\ngo 1.27.4\n")
                with open(os.path.join(root, tpn.GO_LICENCE), "w") as f:
                    f.write("Go licence\n")
                with open(os.path.join(root, tpn.GO_LICENCE + ".version"), "w") as f:
                    f.write("1.27.4\n")
                if notices is not None:
                    with open(os.path.join(root, tpn.NOTICES), "w", newline="") as f:
                        f.write(notices() if callable(notices) else notices)
                return tpn.main(["--check", "--root", root])
            finally:
                tpn.graph = tpn_graph

    def test_a_missing_file_fails(self):
        self.assertEqual(self.run_check(None), 1)

    def test_a_stale_file_fails(self):
        self.assertEqual(self.run_check("an old notices file\n"), 1)

    def test_a_file_that_matches_passes(self):
        with tempfile.TemporaryDirectory() as t:
            lic = module(t, "m", {"LICENSE": "licence text\n"})
            want = tpn.render([("example.com/m", "v1.0.0", lic)], tpn.GoLicence("1.27.4", "Go licence\n"))
        self.assertEqual(self.run_check(want), 0)


class Review762(unittest.TestCase):
    def test_a_notice_alone_is_not_a_licence(self):
        # A NOTICE file is additional text some licences ask to be carried; it holds no
        # licence terms, so a module with only that has shipped none (#762 review).
        with tempfile.TemporaryDirectory() as t:
            d = module(t, "m", {"NOTICE": "Copyright Example Authors\n"})
            with self.assertRaises(tpn.MissingNotice) as cm:
                tpn.render([("example.com/m", "v1.0.0", d)])
            self.assertIn("example.com/m@v1.0.0", str(cm.exception))
            both = module(t, "n", {"NOTICE": "Copyright Example Authors\n", "LICENSE": "terms\n"})
            self.assertIn("Copyright Example Authors", tpn.render([("example.com/n", "v1.0.0", both)]))

    def test_only_whole_licence_names_are_carried(self):
        with tempfile.TemporaryDirectory() as t:
            d = module(t, "m", {"LICENSE": "a\n", "LICENSE-MIT": "b\n", "NOTICE.txt": "c\n", "COPYING.md": "d\n",
                                "license.go": "package x\n", "notice_test.go": "package x\n", "licensing.py": "x\n"})
            self.assertEqual(tpn.licence_files(d), ["COPYING.md", "LICENSE", "LICENSE-MIT", "NOTICE.txt"])

    def test_a_replaced_module_names_what_replaced_it(self):
        with tempfile.TemporaryDirectory() as t:
            d = module(t, "m", {"LICENSE": "licence text\n"})
            out = tpn.render([("example.com/m", "v1.0.0", d, "example.com/fork v1.1.0")])
        self.assertIn("example.com/m v1.0.0 (replaced by example.com/fork v1.1.0)", out)

    def test_a_go_failure_says_what_go_said(self):
        with tempfile.TemporaryDirectory() as t:
            with self.assertRaises(tpn.MissingNotice) as cm:
                tpn.go(t, ["list", "-deps", "./cmd/mrw"])
        self.assertGreater(len(str(cm.exception)), len("go list -deps failed: "))

    def test_the_targets_are_the_release_build_matrix(self):
        root = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..")
        with open(os.path.join(root, ".github", "workflows", "ci.yml"), encoding="utf-8") as f:
            text = f.read()
        matrix = tuple((m.group(1), m.group(2)) for m in re.finditer(r"\{ goos: (\w+), goarch: (\w+) \}", text))
        self.assertEqual(tuple(sorted(matrix)), tuple(sorted(tpn.TARGETS)))


class GoStandardLibrary(unittest.TestCase):
    """A binary carries the Go standard library and the golang.org/x code Go bundles,
    whose BSD-3 notice is Go's LICENSE; the text is vendored, since not every Go
    install has one (Homebrew's has none)."""

    def root_with(self, t, gomod, licence, version):
        root = os.path.join(t, "root")
        os.makedirs(os.path.join(root, "scripts", "third_party"))
        with open(os.path.join(root, "go.mod"), "w") as f:
            f.write(gomod)
        if licence is not None:
            with open(os.path.join(root, "scripts", "third_party", "go-LICENSE"), "w") as f:
                f.write(licence)
        if version is not None:
            with open(os.path.join(root, "scripts", "third_party", "go-LICENSE.version"), "w") as f:
                f.write(version)
        return root

    def check(self, root):
        tpn_graph = tpn.graph
        tpn.graph = lambda r: []
        try:
            return tpn.main(["--check", "--root", root])
        finally:
            tpn.graph = tpn_graph

    def test_the_committed_file_carries_go_s_licence(self):
        root = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..")
        with open(os.path.join(root, tpn.NOTICES), encoding="utf-8") as f:
            text = f.read()
        self.assertIn("Go standard library", text)
        self.assertIn("The Go Authors", text)

    def test_a_missing_vendored_licence_fails_the_check(self):
        with tempfile.TemporaryDirectory() as t:
            root = self.root_with(t, "module m\n\ngo 1.27.4\n", None, "1.27.4\n")
            with open(os.path.join(root, tpn.NOTICES), "w", newline="") as f:
                f.write(tpn.render([], tpn.GoLicence("1.27.4", "Go licence\n")))
            self.assertEqual(self.check(root), 1)

    def test_a_toolchain_the_vendored_text_was_not_checked_against_fails_the_check(self):
        with tempfile.TemporaryDirectory() as t:
            root = self.root_with(t, "module m\n\ngo 1.99.0\n", "Go licence\n", "1.27.4\n")
            # A notices file that matches what the vendored text renders: only the pin
            # between the toolchain and the text's version can fail the check.
            with open(os.path.join(root, tpn.NOTICES), "w", newline="") as f:
                f.write(tpn.render([], tpn.GoLicence("1.27.4", "Go licence\n")))
            self.assertEqual(self.check(root), 1)

    def test_a_matching_version_with_the_text_present_passes_the_notices_check_part(self):
        with tempfile.TemporaryDirectory() as t:
            root = self.root_with(t, "module m\n\ngo 1.27.4\n", "Go licence\n", "1.27.4\n")
            want = tpn.render([], tpn.GoLicence("1.27.4", "Go licence\n"))
            with open(os.path.join(root, tpn.NOTICES), "w", newline="") as f:
                f.write(want)
            self.assertEqual(self.check(root), 0)

    def test_the_vendored_text_is_the_one_the_installed_toolchain_ships(self):
        text = tpn.goroot_licence()
        if text is None:
            self.skipTest("this Go install ships no licence text to compare with")
        root = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..")
        with open(os.path.join(root, "scripts", "third_party", "go-LICENSE"), encoding="utf-8") as f:
            self.assertEqual(f.read(), text)


class Committed(unittest.TestCase):
    def test_the_committed_file_matches_go_mod(self):
        # The release gate, run as a test: THIRD_PARTY_NOTICES is the file the
        # graph of go.mod produces now, urfave/cli's notice included.
        root = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..")
        self.assertEqual(tpn.main(["--check", "--root", root]), 0)
        with open(os.path.join(root, tpn.NOTICES), encoding="utf-8") as f:
            self.assertIn("github.com/urfave/cli/v3", f.read())


if __name__ == "__main__":
    unittest.main()
