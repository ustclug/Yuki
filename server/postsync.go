package server

import (
	"fmt"
	"os/exec"

	"github.com/knight42/Yuki/events"
)

func (s *Server) registerPostSync() {
	cmds := s.config.PostSync
	events.On(events.SyncEnd, func(data events.Payload) {
		attrs := data.Attrs
		var env []string
		for k, v := range attrs {
			switch v.(type) {
			case string:
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			case int:
				env = append(env, fmt.Sprintf("%s=%d", k, v))
			}
		}
		id := attrs["ID"].(string)
		name := attrs["Name"].(string)
		dir := attrs["Dir"].(string)
		code := attrs["ExitCode"].(int)

		s.c.RemoveContainer(id)
		s.c.UpsertRepoMeta(name, dir, code)
		for _, cmd := range cmds {
			prog := exec.Command("sh", "-c", cmd)
			prog.Env = env
			if err := prog.Run(); err != nil {
				s.logger.Error(err.Error())
			}
		}
	})
}
