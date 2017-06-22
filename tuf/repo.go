package tuf

import (
	"net/url"
	"os"
	"regexp"

	"github.com/pkg/errors"
)

const (
	tufURLScheme           = "https"
	tumAPIPattern          = `/v2/%s/_trust/tuf/%s.json`
	healthzPath            = `/_notary_server/health`
	roleRegex              = `^root$|^[1-9]*[0-9]+\.root$|^snapshot$|^timestamp$|^targets$`
	defaultMaxResponseSize = 5 * 1024 * 1024 // 5 Megabytes
	// http headers
	cacheControl       = "Cache-Control"
	cachePolicyNoStore = "no-store"
)

var errNotFound = errors.New("remote resource does not exist")

type repo interface {
	root(opts ...func() interface{}) (*Root, error)
	snapshot(opts ...func() interface{}) (*Snapshot, error)
	targets(opts ...func() interface{}) (*Targets, error)
	timestamp() (*Timestamp, error)
}

type remoteRepo interface {
	repo
	ping() error
}

type persistentRepo interface {
	repo
	save(role, interface{}) error
}

type localRepo struct {
	repoPath string
}

type notaryRepo struct {
	url             *url.URL
	skipVerify      bool
	gun             string
	maxResponseSize int64
}

func newLocalRepo(repoPath string) (*localRepo, error) {
	err := validatePath(repoPath)
	if err != nil {
		return nil, errors.Wrap(err, "new tuf repo")
	}
	repo := localRepo{
		repoPath: repoPath,
	}

	return &repo, nil
}

func newNotaryRepo(baseURL, gun string, maxResponseSize int64, skipVerify bool) (*notaryRepo, error) {
	var (
		repo notaryRepo
		err  error
	)
	repo.maxResponseSize = maxResponseSize
	repo.skipVerify = skipVerify
	repo.gun = gun
	repo.url, err = validateURL(baseURL)
	if err != nil {
		return nil, errors.Wrap(err, "new tuf remote repo")
	}
	return &repo, nil
}

func validateURL(repoURL string) (*url.URL, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return nil, errors.Wrap(err, "tuf remote repo url validation failed")
	}
	if u.Scheme != tufURLScheme {
		return nil, errors.Errorf("tuf url scheme must be %q", tufURLScheme)
	}
	return u, nil
}

// path must exist and be a directory, or a symlink to a directory
func validatePath(repoPath string) error {
	fi, err := os.Stat(repoPath)
	if os.IsNotExist(err) {
		return errors.Wrap(err, "tuf repo path validation failed")
	}
	if !fi.IsDir() {
		return errors.Errorf("tuf repo path %q must be a directory", repoPath)
	}
	return nil
}

func validateRole(r role) error {
	if !regexp.MustCompile(roleRegex).MatchString(string(r)) {
		return errors.Errorf("%q is not a valid role", r)
	}
	return nil
}

func isRoleCorrect(r role, s interface{}) {
	var hit bool
	switch s.(type) {
	case Root, *Root:
		hit = r == roleRoot
	case Targets, *Targets:
		hit = r == roleTargets
	case Timestamp, *Timestamp:
		hit = r == roleTimestamp
	case Snapshot, *Snapshot:
		hit = r == roleSnapshot
	}
	if !hit {
		panic("Programmer error! Role name and role type mismatch.")
	}
}