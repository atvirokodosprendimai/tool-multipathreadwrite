import hashlib
from pathlib import Path

EXPECTED = {
    "guides/service_0.md": "6b10000cd6dd6e0f978196acdcc614a0848c59ec98f04573cd788a8887125ab8",
    "guides/service_1.md": "397244495b68d07a8ba44e99aa122c49e05d660cb848a743b4dc4e2d226f904e",
    "guides/service_2.md": "0bba176eb703f899d5d840dfe7c4535a245b9afadc1ac9e3b9d67c2b1b7cb09a",
    "guides/service_3.md": "d4c2ce05f6274ce29fac6586bbd5d757dc4feb3822b4d0dbb55791766eaa1eea",
    "guides/service_4.md": "9516ce256a49cee57b26dc415e50d48113069ff8ccbe3a93ee8c06b1c46c0f68",
    ".gitignore": "32dae3052f331ee34d628ef535709b301259a45df7c7522c4d35dcf49873f00b",
    ".quality-harness.json": "f8bf0112a4ac05a3b1631697da9dead1f36b58ffd0b7964290ab936fda0720ab",
    "TASK.json": "55a6921dc226838740a8fce0392d6d263928a735596134d9971c04da9f2657c8"
}

for name, expected in EXPECTED.items():
    assert hashlib.sha256(Path(name).read_bytes()).hexdigest() == expected, name
print("accepted all registered changes and preserved decoys")
