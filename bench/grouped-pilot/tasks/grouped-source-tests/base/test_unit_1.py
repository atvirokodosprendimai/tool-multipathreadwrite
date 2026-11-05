import unit_1


def test_active():
    assert unit_1.active_label() == "old_1"


def test_legacy():
    assert unit_1.legacy_label() == "old_1"
