package contents

import (
	"errors"
	"github.com/jo7oem/hatsukari/store/config"
	"os"

	gogit "github.com/go-git/go-git/v5"
	goGitSSH "github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

func Start(cnf config.ContentConfig) error {
	cloneOpt := &gogit.CloneOptions{
		URL:      cnf.Remote.Address,
		Progress: os.Stdout,
	}

	switch cnf.Remote.AuthType {
	case "ssh":
		pubKey, err := goGitSSH.NewPublicKeysFromFile("git", cnf.Remote.AuthSSHKeyPath, "")
		if err != nil {
			return err
		}

		cloneOpt.Auth = pubKey

	default:
		panic("unsupported auth type")
	}

	// すでにcloneされているか確認する
	r, err := gogit.PlainOpen(cnf.Path)
	if errors.Is(err, gogit.ErrRepositoryNotExists) {
		r, err = gogit.PlainClone(cnf.Path, false, cloneOpt)
		if err != nil {
			return err
		}
	}

	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	if err := w.Pull(&gogit.PullOptions{
		RemoteName: "origin",
	}); err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return err
	}

	return nil
}
