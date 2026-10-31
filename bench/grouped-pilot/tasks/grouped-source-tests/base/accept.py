import hashlib
from pathlib import Path

EXPECTED = {
    "unit_0.py": "f8b4780444b9c57d9a340449cd24edcff9ebfaa93f562a5c7c3a550f4f31d251",
    "test_unit_0.py": "581a02ba5bf503f7a23a19817bee055d554e269af4264bdedd0661cfa8c9047a",
    "unit_1.py": "19cf1ee06d88d911c0bc251647328bd5bb4057e263f31042f3efdf42661954db",
    "test_unit_1.py": "726c9c69715f53df86cb63b7e01cac74fefb79404bfd3669f103f6912b7e5899",
    "unit_2.py": "a433338466ff8900860f40c164725c27e9bcae424c1d60e4c26ae423499a628b",
    "test_unit_2.py": "f34f59d166c9e63e527776d249740227715581e82c5786fe1777f07e7cb7aba6",
    "unit_3.py": "f645a361bae77efc7273825a36f15490d8d8cecee19b01b50f1383f8cb08f80d",
    "test_unit_3.py": "0f05e9e86e4ecbe65d7ee4bbdb0b19769fa0223c7b9e7360cf7991146da35ffa",
    "unit_4.py": "685b303431a60c21528d77a8013193872d127bc81194dc46df082f9f2dacbba9",
    "test_unit_4.py": "aa15e7f269ecc3eac61e0c0d63ad3705698163f183b8b6d0b18d9066dcedbb72",
    ".gitignore": "32dae3052f331ee34d628ef535709b301259a45df7c7522c4d35dcf49873f00b",
    ".quality-harness.json": "f8bf0112a4ac05a3b1631697da9dead1f36b58ffd0b7964290ab936fda0720ab",
    "TASK.json": "7933e7ec324196b7e0e39848a498c6ba96f5a447ef8cb2e25da19bba99585afb"
}

for name, expected in EXPECTED.items():
    assert hashlib.sha256(Path(name).read_bytes()).hexdigest() == expected, name
print("accepted all registered changes and preserved decoys")
