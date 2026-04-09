package repo

import "time"

const notificationRetentionPeriod = 30 * 24 * time.Hour

func NotificationRetentionCutoff(now time.Time) time.Time {
	return now.Add(-notificationRetentionPeriod)
}
