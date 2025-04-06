package main

import (
	"log"
	"os"
	"os/signal"
	"server/compiler"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func main() {
	dockerManager, err := compiler.NewDockerManager()
	if err != nil {
		log.Fatalf("Failed to initialize Docker manager: %v", err)
	}

	app := fiber.New()

	defer dockerManager.Shutdown()

	go dockerManager.MonitorResources()

	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		language := "cpp"

		containerID, err := dockerManager.FindContainer(language)
		if err != nil {
			log.Printf("Failed to start container: %v", err)
			c.WriteMessage(websocket.TextMessage, []byte("error: "+err.Error()))
			return
		}

		if err := c.WriteMessage(websocket.TextMessage, []byte("Container started: "+containerID)); err != nil {
			log.Printf("Failed to send message: %v", err)
			return
		}

		defer func() {
			if err := dockerManager.DecreaseUser(containerID); err != nil {
				log.Printf("Failed to remove container %s: %v", containerID, err)
			}
		}()

		if err := dockerManager.RunLiveCode(language, containerID, c); err != nil {
			log.Printf("Interactive session error: %v", err)
			c.WriteMessage(websocket.TextMessage, []byte("error: "+err.Error()))
		}
	}))

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdown
		log.Println("Shutting down server gracefully...")
		if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	}()

	log.Println("Starting server on :3000")
	if err := app.Listen(":3000"); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
