import hashlib
from pathlib import Path

EXPECTED = {
    "parsers/parser_0.py": "e00d08aec31261f20389f67adcac9b0eecc524651bb0b2f9a592ded5826cce72",
    "parsers/parser_1.py": "8b6783a91c26e55071fee7c9636ca07f352ea3e675d5ed431fc9c2b95e879f27",
    "parsers/parser_2.py": "0d48d441afc94f791469b9fe7d6f9e68ee815395b1905b73c17e14809068feab",
    "parsers/parser_3.py": "3884096f81f58bf6415ba250669ee1a3e8445a4a11e1a01139c21dea58e3e99e",
    "parsers/parser_4.py": "b1539df9d74cf53cd268e31f561ebbb3c7ef5c2ad659d1ca1e6533835f112f9e",
    ".gitignore": "32dae3052f331ee34d628ef535709b301259a45df7c7522c4d35dcf49873f00b",
    ".quality-harness.json": "f8bf0112a4ac05a3b1631697da9dead1f36b58ffd0b7964290ab936fda0720ab",
    "TASK.json": "a01ce0a3abe307cf57f0e18e2b84ff6736d0df018753252a1b90cc325d121f98"
}

for name, expected in EXPECTED.items():
    assert hashlib.sha256(Path(name).read_bytes()).hexdigest() == expected, name
print("accepted all registered changes and preserved decoys")
