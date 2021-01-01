package buffer

type Default struct{}

func NewDefault() Handler { return Default{} }

func (Default) FlowConfig() FlowConfig                           { return FlowConfig{} }
func (Default) CheckBuffer(string) bool                          { return true }
func (Default) IsPaused() bool                                   { return false }
func (Default) HandleInput(input QueueItem) []CommandResponse    { return nil }
func (Default) HandleResponse(response string) []CommandResponse { return nil }
func (Default) PollCommand() string                              { return "" }
