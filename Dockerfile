# Minimal Docker image containing only the composepack binary
# The binary is copied from the build artifacts in CI
FROM scratch

# Copy the composepack binary
COPY composepack /composepack

# Set as entrypoint
ENTRYPOINT ["/composepack"]

