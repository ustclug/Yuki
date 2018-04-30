package server

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/sessions"
	"github.com/knight42/Yuki/auth"
	"github.com/knight42/Yuki/core"
	"github.com/knight42/Yuki/tail"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/sirupsen/logrus"
)

func badRequest(msg interface{}) error {
	return &echo.HTTPError{
		Code:    http.StatusBadRequest,
		Message: msg,
	}
}

func unauthorized(msg interface{}) error {
	return &echo.HTTPError{
		Code:    http.StatusUnauthorized,
		Message: msg,
	}
}

func notFound(msg interface{}) error {
	return &echo.HTTPError{
		Code:    http.StatusNotFound,
		Message: msg,
	}
}

func notAcceptable(msg interface{}) error {
	return &echo.HTTPError{
		Code:    http.StatusNotAcceptable,
		Message: msg,
	}
}

func conflict(msg interface{}) error {
	return &echo.HTTPError{
		Code:    http.StatusConflict,
		Message: msg,
	}
}

func forbidden(msg interface{}) error {
	return &echo.HTTPError{
		Code:    http.StatusForbidden,
		Message: msg,
	}
}

func (s *Server) listRepos(c echo.Context) error {
	type repo struct {
		Name     string `bson:"_id" json:"name"`
		Interval string `bson:"interval" json:"interval"`
		Image    string `bson:"image" json:"image"`
	}
	var repos []repo
	s.c.FindRepository(nil).Select(bson.M{
		"interval": 1,
		"image":    1,
	}).Sort("_id").All(&repos)
	return c.JSON(http.StatusOK, repos)
}

func (s *Server) addRepo(c echo.Context) error {
	repo := new(core.Repository)
	if err := c.Bind(repo); err != nil {
		return badRequest(err)
	}
	if err := c.Validate(repo); err != nil {
		return badRequest(err)
	}
	name := c.Param("name")
	if repo.Name == "" {
		repo.Name = name
	}
	err := s.c.AddRepository(repo)
	if err != nil {
		s.addLogField(c, "repo", name)
		if mgo.IsDup(err) {
			return conflict(err)
		}
		return err
	}
	return c.NoContent(http.StatusCreated)
}

func (s *Server) getRepo(c echo.Context) error {
	name := c.Param("name")
	repo, err := s.c.GetRepository(name)
	if err != nil {
		s.addLogField(c, "repo", name)
		return notFound(err)
	}
	return c.JSON(http.StatusOK, repo)
}

func (s *Server) getRepoLogs(c echo.Context) error {
	type repoLogsOptions struct {
		N     int  `query:"n"`
		Tail  int  `query:"tail"`
		Stats bool `query:"stats"`
	}
	type fileInfo struct {
		Name  string    `json:"name"`
		Size  int64     `json:"size"`
		Mtime time.Time `json:"mtime"`
	}

	opts := repoLogsOptions{}
	if err := c.Bind(&opts); err != nil {
		return badRequest(err)
	}

	logdir := path.Join(s.config.LogDir, c.Param("name"))
	if err := os.MkdirAll(logdir, os.ModePerm); err != nil {
		return fmt.Errorf("not a directory: %s", logdir)
	}

	files, err := ioutil.ReadDir(logdir)
	if err != nil {
		return err
	}

	if opts.Stats {
		infos := []fileInfo{}
		for _, f := range files {
			name := f.Name()
			if !strings.HasPrefix(name, "result.log.") {
				continue
			}
			infos = append(infos, fileInfo{
				Name:  name,
				Size:  f.Size(),
				Mtime: f.ModTime(),
			})
		}
		sort.Slice(infos, func(i, j int) bool {
			return infos[j].Mtime.After(infos[i].Mtime)
		})
		return c.JSON(http.StatusOK, infos)
	}

	if opts.Tail < 0 {
		opts.Tail = 0
	}

	wantedName := fmt.Sprintf("result.log.%d", opts.N)
	fileName := ""
	for _, f := range files {
		if strings.HasPrefix(f.Name(), wantedName) {
			// result.log.0
			// result.log.1.gz
			// result.log.2.gz
			fileName = f.Name()
			break
		}
	}

	if fileName == "" {
		return notFound(fmt.Sprintf("no such file: %s", wantedName))
	}

	content, err := os.Open(path.Join(logdir, fileName))
	if err != nil {
		return err
	}
	defer content.Close()

	var t *tail.Tail

	switch path.Ext(fileName) {
	case ".gz":
		gr, err := gzip.NewReader(content)
		if err != nil {
			return err
		}
		defer gr.Close()
		tmpfile, err := ioutil.TempFile(logdir, "extracted")
		if err != nil {
			return err
		}
		defer os.Remove(tmpfile.Name())
		defer tmpfile.Close()
		_, err = io.Copy(tmpfile, gr)
		if err != nil {
			return err
		}
		_, err = tmpfile.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}
		t = tail.New(tmpfile, opts.Tail)
	default:
		t = tail.New(content, opts.Tail)
	}

	_, err = t.WriteTo(c.Response())
	return err
}

func convertUpdate(update bson.M) bson.M {
	if _, ok := update["$set"]; !ok {
		update["$set"] = make(map[string]interface{})
	}
	set := update["$set"].(map[string]interface{})
	for k, v := range update {
		if !strings.HasPrefix(k, "$") {
			set[k] = v
			delete(update, k)
		}
	}
	return update
}

func (s *Server) updateRepo(c echo.Context) error {
	t := bson.M{}
	decoder := json.NewDecoder(c.Request().Body)
	if err := decoder.Decode(&t); err != nil {
		return badRequest(err)
	}
	t = convertUpdate(t)
	set := t["$set"].(map[string]interface{})
	myva := s.e.Validator.(*myValidator)
	if err := myva.CheckMap(set, core.Repository{}); err != nil {
		return badRequest(err)
	}
	name := c.Param("name")
	if err := s.c.UpdateRepository(name, t); err != nil {
		return err
	}
	r, err := s.c.GetRepository(name)
	if err != nil {
		return err
	}
	s.logger.Infof("Rescheduled %s", name)
	if err := s.cron.AddJob(r.Name, r.Interval, s.newJob(r.Name)); err != nil {
		s.logger.WithField("repo", r.Name).Errorln(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) removeRepo(c echo.Context) error {
	name := c.Param("name")
	if err := s.c.RemoveRepository(name); err != nil {
		return err
	}
	if err := s.c.RemoveMeta(name); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) listCts(c echo.Context) error {
	type container struct {
		ID      string `json:"Id"`
		Image   string `json:"Image"`
		Created int64  `json:"Created"`
		State   string `json:"State"`
		Status  string `json:"Status"`
		Name    string `json:"Name"`
	}
	opts := docker.ListContainersOptions{
		All: true,
		Filters: map[string][]string{
			"label": {"org.ustcmirror.syncing=true"},
		},
	}
	apiCts, err := s.c.Docker.ListContainers(opts)
	if err != nil {
		return err
	}
	cts := []container{}
	for _, ct := range apiCts {
		cts = append(cts, container{
			ID:      ct.ID[:10],
			Image:   ct.Image,
			Created: ct.Created,
			State:   ct.State,
			Status:  ct.Status,
			Name:    strings.TrimLeft(ct.Names[0], "/"),
		})
	}
	return c.JSON(http.StatusOK, cts)
}

func (s *Server) sync(c echo.Context) error {
	name := c.Param("name")
	debug, err := strconv.ParseBool(c.QueryParam("debug"))
	if err != nil {
		debug = false
	}

	s.logger.Infof("Syncing %s", name)
	ct, err := s.c.Sync(core.SyncOptions{
		Name:          name,
		NamePrefix:    s.config.NamePrefix,
		LogDir:        s.config.LogDir,
		DefaultOwner:  s.config.Owner,
		MountDir:      true,
		Debug:         debug,
		DefaultBindIP: s.config.BindIP,
	})
	if err != nil {
		s.addLogField(c, "repo", name)
		if err == docker.ErrContainerAlreadyExists {
			return conflict(err)
		}
		return err
	}

	go s.c.WaitForSync(*ct)
	return c.NoContent(http.StatusCreated)
}

func (s *Server) getCtID(name string) string {
	if s.cron.HasJob(name) {
		// repo name
		return s.config.NamePrefix + name
	}
	// container ID
	return name
}

func (s *Server) getCtLogs(c echo.Context) error {
	type logsOptions struct {
		Follow bool   `query:"follow"`
		Tail   string `query:"tail"`
	}
	id := s.getCtID(c.Param("name"))
	opts := logsOptions{}
	if err := c.Bind(&opts); err != nil {
		return badRequest(err)
	}
	fw := NewFlushWriter(c.Response())
	return s.c.GetContainerLogs(core.LogsOptions{
		ID:          id,
		Stream:      fw,
		Tail:        opts.Tail,
		Follow:      opts.Follow,
		CloseNotify: c.Response().CloseNotify(),
	})
}

func (s *Server) stopCt(c echo.Context) error {
	id := s.getCtID(c.Param("name"))
	if err := s.c.StopContainer(id); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) removeCt(c echo.Context) error {
	id := s.getCtID(c.Param("name"))
	err := s.c.RemoveContainer(id)
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

type metaWithSyncStatus struct {
	*core.Meta
	Syncing bool `json:"syncing"`
}

func (s *Server) expandMeta(m *core.Meta) metaWithSyncStatus {
	_, ok := s.syncStatus.Load(m.Name)
	return metaWithSyncStatus{m, ok}
}

func (s *Server) listMetas(c echo.Context) error {
	ms := s.c.ListAllMetas()
	var result []metaWithSyncStatus
	for i := 0; i < len(ms); i++ {
		result = append(result, s.expandMeta(&ms[i]))
	}
	return c.JSON(http.StatusOK, result)
}

func (s *Server) getMeta(c echo.Context) error {
	name := c.Param("name")
	m, err := s.c.GetMeta(name)
	if err != nil {
		return notFound(err)
	}
	return c.JSON(http.StatusOK, s.expandMeta(m))
}

func (s *Server) exportConfig(c echo.Context) error {
	var query bson.M
	names := c.QueryParam("names")
	if names != "" {
		nameLst := strings.Split(names, ",")
		query = bson.M{
			"_id": bson.M{"$in": nameLst},
		}
	}
	var repos []bson.M
	s.c.FindRepository(query).Select(bson.M{"updatedAt": 0, "createdAt": 0}).Sort("_id").All(&repos)
	for i := 0; i < len(repos); i++ {
		r := repos[i]
		r["name"] = r["_id"]
		delete(r, "_id")
		delete(r, "__v")
	}
	return c.JSON(http.StatusOK, repos)
}

func (s *Server) importConfig(c echo.Context) error {
	var repos []*core.Repository
	if err := c.Bind(&repos); err != nil {
		return err
	}
	if err := s.c.AddRepository(repos...); err != nil {
		return err
	}
	for _, r := range repos {
		if err := s.cron.AddJob(r.Name, r.Interval, s.newJob(r.Name)); err != nil {
			return err
		}
	}
	return c.NoContent(http.StatusNoContent)
}

func fromBrowser(c echo.Context) bool {
	return strings.Contains(c.Request().UserAgent(), "Mozilla")
}

func (s *Server) createSession(c echo.Context) error {
	name := c.Get("user").(string)
	s.logger.WithField("name", name).Infoln("login")
	if fromBrowser(c) {
		// web browser
		sess, _ := session.Get("session", c)
		sess.Options = &sessions.Options{
			Domain:   s.config.CookieDomain,
			Path:     "/api/v1/",
			HttpOnly: true,
			MaxAge:   int(s.config.SessionAge / time.Second),
			Secure:   s.config.SecureCookie,
		}
		now := time.Now()
		sess.Values["expireAt"] = now.Add(s.config.SessionAge).Unix()
		sess.Values["user"] = name
		sess.Save(c.Request(), c.Response())
		return c.NoContent(http.StatusCreated)
	}

	type token struct {
		Token string `json:"token"`
	}
	tok, err := s.c.CreateSession(name)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, token{tok})
}

func (s *Server) removeSession(c echo.Context) error {
	if fromBrowser(c) {
		sess, _ := session.Get("session", c)
		sess.Options = &sessions.Options{
			Secure:   s.config.SecureCookie,
			Domain:   s.config.CookieDomain,
			Path:     "/api/v1/",
			HttpOnly: true,
			MaxAge:   -1,
		}
		sess.Save(c.Request(), c.Response())
	} else {
		tok := c.Get("user").(string)
		s.c.RemoveSession(tok)
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) registerAPIs(g *echo.Group) {
	cfg := auth.Config{
		Validator:   s.config.Authenticator.Authenticate,
		LookupToken: s.c.LookupToken,
	}
	g.GET("metas", s.listMetas)
	g.GET("metas/:name", s.getMeta)

	g.Use(auth.Middleware(cfg))
	g.GET("repositories", s.listRepos)
	g.POST("repositories/:name", s.addRepo)
	g.GET("repositories/:name", s.getRepo)
	g.PUT("repositories/:name", s.updateRepo)
	g.DELETE("repositories/:name", s.removeRepo)
	g.GET("repositories/:name/logs", s.getRepoLogs)

	g.GET("containers", s.listCts)
	g.POST("containers/:name", s.sync)
	g.POST("containers/:name/stop", s.stopCt)
	g.DELETE("containers/:name", s.removeCt)
	g.GET("containers/:name/logs", s.getCtLogs)

	g.GET("config", s.exportConfig)
	g.POST("config", s.importConfig)

	g.POST("sessions", s.createSession)
	g.DELETE("sessions", s.removeSession)
}

func (s *Server) addLogField(c echo.Context, k string, v interface{}) {
	current := c.Get(ctxLogFieldKey).(logrus.Fields)
	current[k] = v
	c.Set(ctxLogFieldKey, current)
}

func (s *Server) addLogFields(c echo.Context, fields logrus.Fields) {
	current := c.Get(ctxLogFieldKey).(logrus.Fields)
	for k, v := range fields {
		current[k] = v
	}
	c.Set(ctxLogFieldKey, current)
}

func (s *Server) HTTPErrorHandler(err error, c echo.Context) {
	var (
		code    = http.StatusInternalServerError
		msg     string
		respMsg echo.Map
	)

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		msg = fmt.Sprintf("%v", he.Message)
		if he.Internal != nil {
			msg = fmt.Sprintf("%v, %v", err, he.Internal)
		}
	} else if de, ok := err.(*docker.Error); ok {
		code = de.Status
		msg = de.Message
	} else {
		msg = err.Error()
	}
	respMsg = echo.Map{"message": msg}

	s.logger.WithFields(c.Get(ctxLogFieldKey).(logrus.Fields)).WithFields(logrus.Fields{
		"remote_ip": c.RealIP(),
		"status":    code,
		"method":    c.Request().Method,
		"uri":       c.Request().RequestURI,
	}).Error(msg)

	// Send response
	if !c.Response().Committed {
		if c.Request().Method == echo.HEAD { // Issue echo#608
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, respMsg)
		}
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"method": c.Request().Method,
				"uri":    c.Request().RequestURI,
			}).Errorln(err)
		}
	}
}
