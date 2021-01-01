package buffer

type Handler interface {
	FlowConfig() FlowConfig

	CheckBuffer(cmd string) bool

	IsPaused() bool

	HandleInput(input QueueItem) []CommandResponse
	HandleMeta(cmd string) string
	HandleResponse(response string) []CommandResponse

	PollCommand() string
}
