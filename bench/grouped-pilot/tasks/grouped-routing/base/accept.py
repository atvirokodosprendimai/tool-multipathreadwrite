import hashlib
from pathlib import Path

EXPECTED = {
    "routes/adapter_0.py": "2ab95814631f62f6d08e78f565bcd138e37d2c1b8fa74d13984fe7c7b13fd64a",
    "routes/adapter_1.py": "e93c3a3dec8233ef2d5a2adc728d4881ffc73dc13e229edda82858f5b6ef9e41",
    "routes/adapter_2.py": "8e2b2b6f0a2bf4e37471e59dcb309822dd5c26c73a6fe24e8dfa9cb4c6383cf4",
    "routes/adapter_3.py": "e828a1a739ba230020bb8e95c645b68fe1059441e7aa135b47201efe7bb03a32",
    "routes/adapter_4.py": "0111cdb2e62b4066ef1c19da359fd8305678d629c60e17e5efd09ab1f36dd70e",
    ".gitignore": "32dae3052f331ee34d628ef535709b301259a45df7c7522c4d35dcf49873f00b",
    ".quality-harness.json": "f8bf0112a4ac05a3b1631697da9dead1f36b58ffd0b7964290ab936fda0720ab",
    "TASK.json": "608a43989815b21f7c73bb1b365b74a08b9eb41cfb565ca945d9e7beae19e95b"
}

for name, expected in EXPECTED.items():
    assert hashlib.sha256(Path(name).read_bytes()).hexdigest() == expected, name
print("accepted all registered changes and preserved decoys")
