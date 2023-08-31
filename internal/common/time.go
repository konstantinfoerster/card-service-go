package common

import "time"

type TimeService interface {
	CurrentTime() time.Time
}

func NewTimeService() TimeService {
	return &clockService{}
}

type clockService struct {
}

func (s *clockService) CurrentTime() time.Time {
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

func (s *fakeClockService) CurrentTime() time.Time {
	return s.time
}
