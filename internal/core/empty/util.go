package empty

import (
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

// GetSessionId returns current session id based on: startId - session id to start from, startBlock - block
// where session with startId started, current - current block, sessionType - type of the session.
// Example:
// Lets take duration = 23 blocks and start = 10 block. So first three session will be on 10-33 34-57 58-81 blocks
// id = (current - start) / 24 + 1
// current = 10 => id = (10 - 10) / 24 + 1 = 1
// current = 33 => id = (33 - 10) / 24 + 1 = 1
// current = 34 => id = (34 - 10) / 24 + 1 = 1 + 1 = 1
func GetSessionId(current, startId, startBlock uint64, sessionType types.SessionType) uint64 {
	switch sessionType {
	case types.SessionType_DefaultSession:
		return (current-startBlock)/(core.DefaultSessionDuration+1) + startId
	case types.SessionType_KeygenSession:
		return 1
	case types.SessionType_ReshareSession:
		return (current-startBlock)/(core.ReshareSessionDuration+1) + startId
	}

	// Should not appear
	panic("Invalid session type")
}

// GetSessionEnd returns session end based on: sessionId - current session id, startBlock - block
// where the first session started, sessionType - type of the session.
// Example:
// Lets take duration = 23 blocks and start = 10 block. So first three session will be on 10-33 34-57 58-81 blocks
// end = id*24 + start - 1
// id = 1 => end = 1 * 24 + 10 - 1 = 33
// id = 2 => end = 2 * 24 + 10 - 1 = 57
// id = 3 => end = 3 * 24 + 10 - 1 = 81
func GetSessionEnd(sessionId, startBlock uint64, sessionType types.SessionType) uint64 {
	switch sessionType {
	case types.SessionType_DefaultSession:
		return sessionId*(core.DefaultSessionDuration+1) + startBlock - 1
	case types.SessionType_KeygenSession:
		return startBlock + core.KeygenSessionDuration
	case types.SessionType_ReshareSession:
		return sessionId*(core.ReshareSessionDuration+1) + startBlock - 1
	}

	// Should not appear
	panic("Invalid session type")
}
