"""Current and legacy payload builders 3."""


def active_payload():
    return {"name": "alpha", "retry": 2}


def active_parse(text):
    return text.strip()


def legacy_payload():
    return {"name": "alpha", "retry": 2}


def legacy_parse(text):
    return text.strip()
