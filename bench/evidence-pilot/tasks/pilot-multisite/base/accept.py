import json
from pathlib import Path
import producer
import worker
import router
import reader
import scheduler
import legacy

modules = (producer, worker, router, reader, scheduler)
assert all(module.ACTIVE_STATUS == "pending" for module in modules)
assert all(module.LEGACY_STATUS == "queued" for module in modules)
assert legacy.STATUS == "queued"
config = json.loads(Path("config.json").read_text())
assert config == {"active_status": "pending", "legacy_status": "queued"}
print("accepted active migration and preserved legacy decoys")
