package server

import (
	"fmt"
	"os/exec"

	"github.com/knight42/Yuki/events"
	"github.com/sirupsen/logrus"
)

func (s *Server) registerPostSync() {
	cmds := s.config.PostSync
	events.On(events.SyncEnd, func(data events.Payload) {
		attrs := data.Attrs
		var env []string
		for k, v := range attrs {
			env = append(env, fmt.Sprintf("%s=%v", k, v))
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
				s.logger.WithFields(logrus.Fields{
					"command": cmd,
				}).Errorln(err)
			}
		}
	})
}
