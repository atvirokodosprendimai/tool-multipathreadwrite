import unit_2


def test_active():
    assert unit_2.active_label() == "old_2"


def test_legacy():
    assert unit_2.legacy_label() == "old_2"
