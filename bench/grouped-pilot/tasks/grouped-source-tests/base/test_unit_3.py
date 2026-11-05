import unit_3


def test_active():
    assert unit_3.active_label() == "old_3"


def test_legacy():
    assert unit_3.legacy_label() == "old_3"
