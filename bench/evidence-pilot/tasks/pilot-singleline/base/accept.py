import settings
assert settings.TIMEOUT_SECONDS == 45
assert settings.RETRY_SECONDS == 30
print("accepted timeout and unchanged retry")
