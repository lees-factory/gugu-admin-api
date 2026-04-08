package batch

import (
	"context"
	"log"
	"time"
)

var scheduleLocation = loadScheduleLocation()

func startScheduleLoop(
	ctx context.Context,
	name string,
	interval time.Duration,
	alignToMidnight bool,
	run func(context.Context),
) {
	if interval <= 0 || run == nil {
		return
	}

	go func() {
		if !alignToMidnight {
			startTickerLoop(ctx, name, interval, run)
			return
		}

		startMidnightAlignedLoop(ctx, name, interval, run)
	}()
}

func startTickerLoop(ctx context.Context, name string, interval time.Duration, run func(context.Context)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("%s started: interval=%s", name, interval)

	for {
		select {
		case <-ctx.Done():
			log.Printf("%s stopped: %v", name, ctx.Err())
			return
		case <-ticker.C:
			run(ctx)
		}
	}
}

func startMidnightAlignedLoop(ctx context.Context, name string, interval time.Duration, run func(context.Context)) {
	nextRun := nextMidnightRun(time.Now(), scheduleLocation)
	initialDelay := time.Until(nextRun)
	if initialDelay < 0 {
		initialDelay = 0
	}

	timer := time.NewTimer(initialDelay)
	defer timer.Stop()

	log.Printf("%s started: interval=%s next_run=%s timezone=%s", name, interval, nextRun.Format(time.RFC3339), scheduleLocation.String())

	for {
		select {
		case <-ctx.Done():
			log.Printf("%s stopped: %v", name, ctx.Err())
			return
		case <-timer.C:
			run(ctx)
			timer.Reset(interval)
		}
	}
}

func shouldAlignToMidnight(interval time.Duration) bool {
	const day = 24 * time.Hour
	return interval >= day && interval%day == 0
}

func nextMidnightRun(now time.Time, loc *time.Location) time.Time {
	localNow := now.In(loc)
	return time.Date(localNow.Year(), localNow.Month(), localNow.Day()+1, 0, 0, 0, 0, loc)
}

func loadScheduleLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		log.Printf("schedule location load failed, falling back to Local: %v", err)
		return time.Local
	}
	return loc
}
