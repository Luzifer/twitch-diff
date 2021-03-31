package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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

func init() {
	rconfig.AutoEnv(true)
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		log.Fatalf("Unable to parse commandline options: %s", err)
	}

	if cfg.VersionAndExit {
		fmt.Printf("twitch-diff %s\n", version)
		os.Exit(0)
	}

	if l, err := log.ParseLevel(cfg.LogLevel); err != nil {
		log.WithError(err).Fatal("Unable to parse log level")
	} else {
		log.SetLevel(l)
	}
}

func main() {
	repo, err := git.PlainOpen(cfg.TargetDir)
	if err != nil {
		if err != git.ErrRepositoryNotExists {
			log.WithError(err).Fatal("Unable to open repository")
		}

		if repo, err = initRepo(); err != nil {
			log.WithError(err).Fatal("Unable to initialize repo")
		}
	}

	worktree, err := repo.Worktree()
	if err != nil {
		log.WithError(err).Fatal("Unable to get working tree")
	}

	followers, err := twitch.GetFollowers()
	if err != nil {
		log.WithError(err).Fatal("Getting followers")
	}

	subs, err := twitch.GetSubscriptions()
	if err != nil {
		log.WithError(err).Fatal("Getting subscriptions")
	}

	for fn, r := range map[string]io.Reader{
		"followers.csv":   followers.ToCSV(),
		"subscribers.csv": subs.ToCSV(),
	} {
		f, err := os.Create(path.Join(cfg.TargetDir, fn))
		if err != nil {
			log.WithError(err).WithField("file", fn).Fatal("Unable to create file")
		}

		if _, err = io.Copy(f, r); err != nil {
			log.WithError(err).WithField("file", fn).Fatal("Unable to write file content")
		}

		f.Close()

		status, err := worktree.Status()
		if err != nil {
			log.WithError(err).Fatal("Unable to get worktree status")
		}
		if status.IsClean() {
			// Nothing to do
			continue
		}

		if _, err = worktree.Add(fn); err != nil {
			log.WithError(err).WithField("file", fn).Fatal("Unable to add file to worktree")
		}
	}

	status, err := worktree.Status()
	if err != nil {
		log.WithError(err).Fatal("Unable to get worktree status")
	}
	if status.IsClean() {
		// Nothing to do
		return
	}

	if _, err = worktree.Commit(
		"Automatic fetch of twitch data",
		&git.CommitOptions{Author: getSignature()},
	); err != nil {
		log.WithError(err).Fatal("Unable to commit changes")
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
