package auth

import (
	"time"
)

type ClockService struct {
}

func NewTimeService() *ClockService {
	return &ClockService{}
}

func (s *ClockService) Now() time.Time {
	return time.Now()
}

type FakeClockService struct {
	time time.Time
}

func NewFakeTimeService(time time.Time) *FakeClockService {
	return &FakeClockService{
		time: time,
	}
}

func (s *FakeClockService) Now() time.Time {
	return s.time
}
