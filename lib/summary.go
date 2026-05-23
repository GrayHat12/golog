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
		elapsed := start_time.Sub(summary.Activities[len(summary.Activities)-1].Started)
		if elapsed.Hours() > 4 {
			end := summary.Activities[len(summary.Activities)-1].Started.Add(time.Hour * 4)
			summary.Activities[len(summary.Activities)-1].Ended = &end
		}
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
