package compiler

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/gofiber/websocket/v2"
)

func (dm *DockerManager) RunLiveCode(lang, containerID string, conn *websocket.Conn) error {
	ctx := context.Background()
	opt, ok := LangImages[lang]
	if !ok {
		return fmt.Errorf("unsupported language: %s", lang)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read message: %w", err)
	}

	code := string(msg)

	var waitForMsg bool

	for {
		if waitForMsg {
			typ, msg, err := conn.ReadMessage()
			if err != nil || typ == websocket.CloseMessage {
				dm.DecreaseUser(containerID)
				return fmt.Errorf("failed to read message: %w", err)
			}
			code = string(msg)
		}

		if !strings.HasPrefix(code, "CODE:") {
			return fmt.Errorf("first message must be CODE")
		}
		tcode := strings.TrimPrefix(code, "CODE:")

		// Setup exec instance
		execConfig := container.ExecOptions{
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          true,
			Cmd:          opt.ExecCmd(tcode),
			User:         "nobody",
			Env: []string{
				"HOME=/tmp",
				"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			},
			WorkingDir: "/tmp",
			Privileged: false,
		}

		if opt.IsCompiled {
			fileName := opt.FileName(containerID)

			if err := os.WriteFile(CODE_FILES_DIR+"/"+fileName, []byte(tcode), 0644); err != nil {
				log.Printf("failed to write file: %v", err)

				if err := conn.WriteMessage(websocket.TextMessage, []byte("error: "+err.Error())); err != nil {
					return fmt.Errorf("failed to send message: %w", err)
				}
				waitForMsg = true
				continue
			}

			log.Print("File created: ", fileName)

			if opt.RunOnHost != nil {
				cmd := opt.RunOnHost(CODE_FILES_DIR + "/" + fileName)
				if out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput(); err != nil {
					log.Printf("failed to run command on host: %v", err)

					if err := conn.WriteMessage(websocket.TextMessage, []byte("error: "+string(out))); err != nil {
						return fmt.Errorf("failed to send message: %w", err)
					}
					waitForMsg = true
					continue
				}
				log.Print("Host commant ran bin has been created")
			} else {
				return fmt.Errorf("no Host command provided")
			}

			if lang == "ts" {
				fileName = fileName[:len(fileName)-3] + ".js"
			}
			if lang == "c" {
				fileName = fileName[:len(fileName)-2] + ".out"
			}
			if lang == "cpp" {
				fileName = fileName[:len(fileName)-4] + ".out"
			}
			if lang == "java" {
				fileName = fileName[:len(fileName)-5] + ".class"
			}

			execConfig.Cmd = opt.ExecCmd(CONTAINER_COMPILED_FILES + "/" + fileName)
		}

		execResp, err := dm.cli.ContainerExecCreate(ctx, containerID, execConfig)
		if err != nil {
			if err := conn.WriteMessage(websocket.TextMessage, []byte("error: "+err.Error())); err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}
			waitForMsg = true
			continue
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		hijackedResp, err := dm.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{Tty: true})
		if err != nil {
			if err := conn.WriteMessage(websocket.TextMessage, []byte("error: "+err.Error())); err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}
			waitForMsg = true
			continue
		}

		waitForMsg = false

		go func() {
			EXEC_TIMEOUT := 5 * time.Minute
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					inspect, err := dm.cli.ContainerExecInspect(ctx, execResp.ID)
					if err != nil || !inspect.Running {
						cancel()
						conn.WriteMessage(websocket.TextMessage, []byte("EXEC_TERMINATED"))
						return
					}
				case <-time.After(EXEC_TIMEOUT):
					cancel()
					conn.WriteMessage(websocket.TextMessage, []byte("EXEC_TIMEOUT"))
					return
				}
			}
		}()

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(hijackedResp.Reader)
			for scanner.Scan() {
				select {
				case <-ctx.Done():
					return
				default:
					conn.WriteMessage(websocket.TextMessage, scanner.Bytes())
				}
			}
		}()

		go func(code *string) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					typ, msg, err := conn.ReadMessage()
					if err != nil || typ == websocket.CloseMessage {
						dm.DecreaseUser(containerID)
						cancel()
						return
					}

					// Handle new CODE message
					if strMsg := string(msg); strings.HasPrefix(strMsg, "CODE:") {
						*code = strMsg
						cancel()
						return
					}

					if strMsg := string(msg); strMsg == "STOP" {
						cancel()
						return
					}

					// Forward input
					hijackedResp.Conn.Write(append(msg, '\n'))
				}
			}
		}(&code)

		wg.Wait()
		hijackedResp.Close()
	}
}
