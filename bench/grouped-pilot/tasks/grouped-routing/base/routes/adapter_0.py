"""Routing adapter 0, including the legacy compatibility path."""

ACTIVE_ROUTE = "/v1/submit"
LEGACY_ROUTE = "/v1/submit"

SUPPORTED_METHODS = ("POST", "GET")

ACTIVE_LIMIT = 20
LEGACY_LIMIT = 20


def active_request():
    return ACTIVE_ROUTE, ACTIVE_LIMIT


def legacy_request():
    return LEGACY_ROUTE, LEGACY_LIMIT
