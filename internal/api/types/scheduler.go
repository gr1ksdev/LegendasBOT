package types

type CreateScheduleRequest struct {
	ChannelID     int64  `json:"channelId" binding:"required"`
	ScheduleType  string `json:"scheduleType" binding:"required,oneof=once daily weekly queue interval"`
	ScheduleTime  string `json:"scheduleTime"`
	ScheduledAt   string `json:"scheduledAt"`
	ScheduleDays  []int  `json:"scheduleDays"`
	RepeatUntil   string `json:"repeatUntil"`
	IntervalMin   int    `json:"intervalMin"`
	WindowStart   string `json:"windowStart"`
	WindowEnd     string `json:"windowEnd"`
	LoopQueue     bool   `json:"loopQueue"`
	PinMessage    bool   `json:"pinMessage"`
	AutoDeleteMin int    `json:"autoDeleteMin"`
}

type UpdateScheduleStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=paused pending cancelled"`
}

type EditScheduleRequest struct {
	NextRunAt     string  `json:"nextRunAt"`
	ScheduleTime  string  `json:"scheduleTime"`
	IntervalMin   *int    `json:"intervalMin"`
	WindowStart   *string `json:"windowStart"`
	WindowEnd     *string `json:"windowEnd"`
	PinMessage    *bool   `json:"pinMessage"`
	AutoDeleteMin *int    `json:"autoDeleteMin"`
}
