package main

import (
	"log"
	"github.com/hibiken/asynq"
)

func main() {
	redisOpt := asynq.RedisClientOpt{Addr: "localhost:6379"}
	client := asynq.NewClient(redisOpt)
	defer client.Close()

	tasks := []string{
		"scheduler:generate_invoices_all",
		"scheduler:auto_isolir_all",
		"scheduler:trigger_reminders_all",
	}

	for _, t := range tasks {
		task := asynq.NewTask(t, nil)
		if _, err := client.Enqueue(task); err != nil {
			log.Printf("Failed to enqueue %s: %v", t, err)
		} else {
			log.Printf("Enqueued %s", t)
		}
	}
}
