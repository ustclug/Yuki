package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/errdefs"
	"github.com/labstack/echo/v4"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm/clause"
	"sigs.k8s.io/yaml"

	"github.com/ustclug/Yuki/pkg/api"
	"github.com/ustclug/Yuki/pkg/model"
	"github.com/ustclug/Yuki/pkg/set"
)

func (s *Server) handlerListRepos(c echo.Context) error {
	l := getLogger(c)
	l.Debug("Invoked")

	var repos []model.Repo
	err := s.getDB(c).
		Select("name", "cron", "image", "storage_dir").
		Find(&repos).Error
	if err != nil {
		const msg = "Fail to list Repos"
		l.Error(msg, slogErrAttr(err))
		return newHTTPError(http.StatusInternalServerError, msg)
	}

	resp := make(api.ListReposResponse, len(repos))
	for i, repo := range repos {
		resp[i] = api.ListReposResponseItem{
			Name:       repo.Name,
			Cron:       repo.Cron,
			Image:      repo.Image,
			StorageDir: repo.StorageDir,
		}
	}
	return c.JSON(http.StatusOK, resp)
}

func (s *Server) handlerGetRepo(c echo.Context) error {
	l := getLogger(c)
	l.Debug("Invoked")

	name, err := getRepoNameFromRoute(c)
	if err != nil {
		return err
	}

	var repo model.Repo
	res := s.getDB(c).
		Where(model.Repo{Name: name}).
		Limit(1).
		Find(&repo)
	if err != nil {
		const msg = "Fail to get Repo"
		l.Error(msg, slogErrAttr(err))
		return newHTTPError(http.StatusInternalServerError, msg)
	}
	if res.RowsAffected == 0 {
		return newHTTPError(http.StatusNotFound, "Repo not found")
	}

	return c.JSON(http.StatusOK, repo)
}

func (s *Server) handlerRemoveRepo(c echo.Context) error {
	l := getLogger(c)
	l.Debug("Invoked")

	name, err := getRepoNameFromRoute(c)
	if err != nil {
		return err
	}

	db := s.getDB(c)
	err = db.Where(model.Repo{Name: name}).Delete(&model.Repo{}).Error
	if err != nil {
		const msg = "Fail to delete Repo"
		l.Error(msg, slogErrAttr(err), slog.String("repo", name))
		return newHTTPError(http.StatusInternalServerError, msg)
	}
	err = db.Where(model.RepoMeta{Name: name}).Delete(&model.RepoMeta{}).Error
	if err != nil {
		l.Error("Fail to delete RepoMeta", slogErrAttr(err), slog.String("repo", name))
	}
	s.repoSchedules.Remove(name)
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) loadRepo(c echo.Context, logger *slog.Logger, dirs []string, file string) (*model.Repo, error) {
	l := logger.With(slog.String("config", file))

	var repo model.Repo
	errn := len(dirs)
	for _, dir := range dirs {
		data, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			errn--
			if errn > 0 && os.IsNotExist(err) {
				continue
			} else {
				return nil, newHTTPError(http.StatusNotFound, fmt.Sprintf("File not found: %q", file))
			}
		}
		err = yaml.Unmarshal(data, &repo)
		if err != nil {
			return nil, newHTTPError(http.StatusBadRequest, fmt.Sprintf("Fail to parse config: %q: %v", file, err))
		}
	}

	if err := s.e.Validator.Validate(&repo); err != nil {
		return nil, newHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid config: %q: %v", file, err))
	}

	logDir := filepath.Join(s.config.RepoLogsDir, repo.Name)
	err := os.MkdirAll(logDir, 0o755)
	if err != nil {
		return nil, newHTTPError(http.StatusInternalServerError, fmt.Sprintf("Fail to create log dir: %q", logDir))
	}

	db := s.getDB(c)
	err = db.
		Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&repo).Error
	if err != nil {
		const msg = "Fail to create Repo"
		l.Error(msg, slogErrAttr(err))
		return nil, newHTTPError(http.StatusInternalServerError, msg)
	}

	schedule, _ := cron.ParseStandard(repo.Cron)
	s.repoSchedules.Set(repo.Name, schedule)

	upstream := getUpstream(repo.Image, repo.Envs)
	nextRun := schedule.Next(time.Now()).Unix()
	err = db.
		Clauses(clause.OnConflict{
			DoUpdates: clause.Assignments(map[string]any{
				"upstream": upstream,
				"next_run": nextRun,
			}),
		}).
		Create(&model.RepoMeta{
			Name:     repo.Name,
			Upstream: upstream,
			Size:     s.getSize(repo.StorageDir),
			NextRun:  nextRun,
		}).Error
	if err != nil {
		const msg = "Fail to create RepoMeta"
		l.Error(msg, slogErrAttr(err))
		return nil, newHTTPError(http.StatusInternalServerError, msg)
	}
	return &repo, nil
}

func (s *Server) handlerReloadAllRepos(c echo.Context) error {
	l := getLogger(c)
	l.Debug("Invoked")

	var repoNames []string
	db := s.getDB(c)
	err := db.Model(&model.Repo{}).Pluck("name", &repoNames).Error
	if err != nil {
		const msg = "Fail to list Repos"
		l.Error(msg, slogErrAttr(err))
		return newHTTPError(http.StatusInternalServerError, msg)
	}

	l.Debug("Reloading all repos")
	toDelete := set.New(repoNames...)
	for _, dir := range s.config.RepoConfigDir {
		infos, err := os.ReadDir(dir)
		if err != nil {
			if !os.IsNotExist(err) {
				l.Warn("Fail to list dir", slogErrAttr(err), slog.String("dir", dir))
			}
			continue
		}
		for _, info := range infos {
			fileName := info.Name()
			if info.IsDir() || fileName[0] == '.' || !strings.HasSuffix(fileName, suffixYAML) {
				continue
			}
			repo, err := s.loadRepo(c, l, s.config.RepoConfigDir, fileName)
			if err != nil {
				return err
			}
			toDelete.Del(repo.Name)
		}
	}

	toDeleteNames := toDelete.ToList()
	l.Debug("Deleting unnecessary repos", slog.Any("repos", toDeleteNames))
	err = db.Where("name IN ?", toDeleteNames).Delete(&model.Repo{}).Error
	if err != nil {
		const msg = "Fail to delete Repos"
		l.Error(msg, slogErrAttr(err))
		return newHTTPError(http.StatusInternalServerError, msg)
	}
	err = db.Where("name IN ?", toDeleteNames).Delete(&model.RepoMeta{}).Error
	if err != nil {
		const msg = "Fail to delete RepoMetas"
		l.Error(msg, slogErrAttr(err))
	}
	for name := range toDelete {
		s.repoSchedules.Remove(name)
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handlerReloadRepo(c echo.Context) error {
	l := getLogger(c)
	l.Debug("Invoked")

	name, err := getRepoNameFromRoute(c)
	if err != nil {
		return err
	}
	_, err = s.loadRepo(c, l.With(slog.String("repo", name)), s.config.RepoConfigDir, name+suffixYAML)
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handlerSyncRepo(c echo.Context) error {
	l := getLogger(c)
	l.Debug("Invoked")

	name, err := getRepoNameFromRoute(c)
	if err != nil {
		return err
	}
	l = l.With(slog.String("repo", name))

	debug := len(c.QueryParam("debug")) > 0
	err = s.syncRepo(c.Request().Context(), name, debug)
	if err != nil {
		if errors.Is(err, errNotFound) {
			return newHTTPError(http.StatusNotFound, "Repo not found")
		}
		// https://github.com/moby/moby/issues/47018
		var dkErr errdefs.ErrConflict
		if errors.As(err, &dkErr) {
			return newHTTPError(http.StatusConflict, "Repo is syncing")
		}
		const msg = "Fail to sync Repo"
		l.Error(msg, slogErrAttr(err))
		return newHTTPError(http.StatusInternalServerError, msg)
	}
	return c.NoContent(http.StatusCreated)
}
