package main

import (
	"io"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/Luzifer/rconfig/v2"
)

var (
	cfg = struct {
		LogLevel       string `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		TargetDir      string `flag:"target-dir" default:"./data" description:"Where to create the repo"`
		TwitchClientID string `flag:"twitch-client-id,c" default:"" description:"ClientID to access Twitch" validate:"nonzero"`
		TwitchToken    string `flag:"twitch-token,t" default:"" description:"Token generated for the given ClientID and account" validate:"nonzero"`
		VersionAndExit bool   `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	version = "dev"
)

func initApp() error {
	rconfig.AutoEnv(true)
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		return errors.Wrap(err, "parsing commandline options")
	}

	l, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return errors.Wrap(err, "parsing log level")
	}
	logrus.SetLevel(l)

	return nil
}

//nolint:gocyclo
func main() {
	var err error

	if err = initApp(); err != nil {
		logrus.WithError(err).Fatal("initializing app")
	}

	if cfg.VersionAndExit {
		logrus.WithField("version", version).Info("twitch-diff")
		os.Exit(0)
	}

	repo, err := git.PlainOpen(cfg.TargetDir)
	if err != nil {
		if err != git.ErrRepositoryNotExists {
			logrus.WithError(err).Fatal("opening repository")
		}

		if repo, err = initRepo(); err != nil {
			logrus.WithError(err).Fatal("initializing repo")
		}
	}

	worktree, err := repo.Worktree()
	if err != nil {
		logrus.WithError(err).Fatal("getting working tree")
	}

	followers, err := twitch.GetFollowers()
	if err != nil {
		logrus.WithError(err).Fatal("getting followers")
	}

	subs, err := twitch.GetSubscriptions()
	if err != nil {
		logrus.WithError(err).Fatal("getting subscriptions")
	}

	for fn, r := range map[string]io.Reader{
		"followers.csv":   followers.ToCSV(),
		"subscribers.csv": subs.ToCSV(),
	} {
		f, err := os.Create(path.Join(cfg.TargetDir, fn)) //#nosec:G304 // Opening that given dir is intentional
		if err != nil {
			logrus.WithError(err).WithField("file", fn).Fatal("creating file")
		}

		if _, err = io.Copy(f, r); err != nil {
			logrus.WithError(err).WithField("file", fn).Fatal("writing file content")
		}

		if err = f.Close(); err != nil {
			logrus.WithError(err).Error("closing file (leaked fd)")
		}

		status, err := worktree.Status()
		if err != nil {
			logrus.WithError(err).Fatal("getting worktree status")
		}
		if status.IsClean() {
			// Nothing to do
			continue
		}

		if _, err = worktree.Add(fn); err != nil {
			logrus.WithError(err).WithField("file", fn).Fatal("adding file to worktree")
		}
	}

	status, err := worktree.Status()
	if err != nil {
		logrus.WithError(err).Fatal("getting worktree status")
	}
	if status.IsClean() {
		// Nothing to do
		return
	}

	if _, err = worktree.Commit(
		"Automatic fetch of twitch data",
		&git.CommitOptions{Author: getSignature()},
	); err != nil {
		logrus.WithError(err).Fatal("committing changes")
	}
}

func initRepo() (*git.Repository, error) {
	repo, err := git.PlainInit(cfg.TargetDir, false)
	if err != nil {
		return nil, errors.Wrap(err, "initializing repo")
	}

	if _, err := repo.Branch("master"); err == git.ErrBranchNotFound {
		if err := repo.CreateBranch(&config.Branch{Name: "master"}); err != nil {
			return nil, errors.Wrap(err, "creating master branch")
		}
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, errors.Wrap(err, "getting worktree")
	}

	_, err = wt.Commit("Initial commit", &git.CommitOptions{Author: getSignature()})

	return repo, errors.Wrap(err, "creating initial commit")
}

func getSignature() *object.Signature {
	return &object.Signature{Name: "twitch-diff " + version, Email: "twitch-diff@luzifer.io", When: time.Now()}
}
