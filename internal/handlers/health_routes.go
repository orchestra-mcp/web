package handlers

import (
	"github.com/gofiber/fiber/v3"
)

// RegisterHealthRoutes registers all health debug routes on a Fiber route group.
//
//	GET    /api/health/profile          - get health profile
//	PUT    /api/health/profile          - update health profile
//	POST   /api/health/water            - log water intake
//	GET    /api/health/water            - list water logs
//	GET    /api/health/water/status     - get hydration status
//	POST   /api/health/meals            - log a meal
//	GET    /api/health/meals            - list meal logs
//	POST   /api/health/caffeine         - log caffeine intake
//	GET    /api/health/caffeine         - list caffeine logs
//	GET    /api/health/caffeine/score   - get caffeine score
//	POST   /api/health/pomodoro/start   - start pomodoro session
//	POST   /api/health/pomodoro/:id/end - end pomodoro session
//	GET    /api/health/pomodoro         - list pomodoro sessions
//	POST   /api/health/sleep            - log a sleep session
//	GET    /api/health/sleep            - list sleep logs
//	GET    /api/health/shutdown/status   - get shutdown status
//	POST   /api/health/shutdown/start   - start shutdown ritual
//	POST   /api/health/snapshots        - upsert health snapshot
//	GET    /api/health/snapshots        - list health snapshots
//	GET    /api/health/summary          - get daily health summary
func RegisterHealthRoutes(group fiber.Router, h *HealthHandler) {
	health := group.Group("/health")

	health.Get("/profile", h.GetProfile)
	health.Put("/profile", h.UpdateProfile)

	health.Post("/water", h.LogWater)
	health.Get("/water", h.ListWaterLogs)
	health.Get("/water/status", h.GetHydrationStatus)

	health.Post("/meals", h.LogMeal)
	health.Get("/meals", h.ListMealLogs)

	health.Post("/caffeine", h.LogCaffeine)
	health.Get("/caffeine", h.ListCaffeineLogs)
	health.Get("/caffeine/score", h.GetCaffeineScore)

	health.Post("/pomodoro/start", h.StartPomodoro)
	health.Post("/pomodoro/:id/end", h.EndPomodoro)
	health.Get("/pomodoro", h.ListPomodoroSessions)

	health.Post("/sleep", h.LogSleep)
	health.Get("/sleep", h.ListSleepLogs)

	health.Get("/shutdown/status", h.GetShutdownStatus)
	health.Post("/shutdown/start", h.StartShutdown)

	health.Post("/snapshots", h.UpsertSnapshot)
	health.Get("/snapshots", h.ListSnapshots)

	health.Get("/summary", h.GetHealthSummary)
}
