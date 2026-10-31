import unit_4


def test_active():
    assert unit_4.active_label() == "old_4"


def test_legacy():
    assert unit_4.legacy_label() == "old_4"
