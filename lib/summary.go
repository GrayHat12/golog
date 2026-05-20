package lib

import "time"

type Activity struct {
	Name    string
	Tags    []string
	Started time.Time
	Ended   *time.Time
}

type Summary struct {
	Activities []Activity
}

func (summary *Summary) AddActivity(entry Entry) {
	start_time := entry.Time()
	if len(summary.Activities) > 0 && summary.Activities[len(summary.Activities)-1].Ended == nil {
		summary.Activities[len(summary.Activities)-1].Ended = &start_time
	}
	summary.Activities = append(summary.Activities, Activity{
		Name:    entry.Name,
		Tags:    entry.Labels,
		Started: start_time,
	})
}

func (summary *Summary) AddBreak(start_time time.Time) {
	if len(summary.Activities) > 0 && summary.Activities[len(summary.Activities)-1].Ended == nil {
		summary.Activities[len(summary.Activities)-1].Ended = &start_time
	}
}
