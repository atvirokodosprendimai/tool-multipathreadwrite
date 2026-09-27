package mcp

// maxInstructionsChars bounds the handshake document. A host that supports the
// field puts it in front of the model once per session, whether or not a tool
// is ever called, so the length is paid by every session rather than by the
// callers who benefit. The bound is what keeps this from growing into a second
// copy of the repository's agent instructions. It is an assertion, not runtime
// behaviour (ADR-037): these tests and the contract hold the built handshake to
// it, so it lives here rather than in production (ADR-088).
const maxInstructionsChars = 4096
