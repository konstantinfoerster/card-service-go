package common

import "time"

type TimeService interface {
	Now() time.Time
}

func NewTimeService() TimeService {
	return &clockService{}
}

type clockService struct {
}

func (s *clockService) Now() time.Time {
	return time.Now()
}

func NewFakeTimeService(time time.Time) TimeService {
	return &fakeClockService{
		time: time,
	}
}

type fakeClockService struct {
	time time.Time
}

func (s *fakeClockService) Now() time.Time {
	return s.time
}
