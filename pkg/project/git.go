package project

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hackclub/hackatime-cli/pkg/file"
	"github.com/hackclub/hackatime-cli/pkg/log"
	"github.com/hackclub/hackatime-cli/pkg/regex"
)

var scpLikeRemoteRegex = regexp.MustCompile(`^[^/@:]+@[^:]+:.+$`)

type gitRemote struct {
	Repository            string
	RepositoryDescription string
}

// Git contains git data.
type Git struct {
	// Filepath contains the entity path.
	Filepath string
	// ProjectFromGitRemote when enabled uses the git remote as the project name instead of local git folder.
	ProjectFromGitRemote bool
	// SubmoduleDisabledPatterns will be matched against the submodule path and if matching, will skip it.
	SubmoduleDisabledPatterns []regex.Regex
	// SubmoduleProjectMapPatterns will be matched against the submodule path and if matching, will use the project map.
	SubmoduleProjectMapPatterns []MapPattern
}

// Detect gets information about the git project for a given file.
// It tries to return a project and branch name.
func (g Git) Detect(ctx context.Context) (Result, bool, error) {
	logger := log.Extract(ctx)
	fp := g.Filepath

	// Find for submodule takes priority if enabled
	gitdirSubmodule, ok, err := findSubmodule(ctx, fp, g.SubmoduleDisabledPatterns)
	if err != nil {
		return Result{}, false, fmt.Errorf("failed to find submodule: %s", err)
	}

	if ok {
		remote := readGitRemote(ctx, gitdirSubmodule)
		project := projectOrRemote(filepath.Base(gitdirSubmodule), g.ProjectFromGitRemote, remote)

		// If submodule has a project map, then use it.
		if result, ok := matchPattern(ctx, gitdirSubmodule, g.SubmoduleProjectMapPatterns); ok {
			project = result
		}

		branch, err := findGitBranch(ctx, filepath.Join(gitdirSubmodule, "HEAD"))
		if err != nil {
			logger.Errorf(
				"error finding branch from %q: %s",
				filepath.Join(filepath.Dir(gitdirSubmodule), "HEAD"),
				err,
			)
		}

		return Result{
			Project:               project,
			Branch:                branch,
			Folder:                filepath.Dir(gitdirSubmodule),
			Repository:            remote.Repository,
			RepositoryDescription: remote.RepositoryDescription,
		}, true, nil
	}

	// Find for .git file or directory
	dotGit, found := FindFileOrDirectory(ctx, fp, ".git")
	if !found {
		return Result{}, false, nil
	}

	// Find for gitdir path
	gitdir, err := findGitdir(ctx, dotGit)
	if err != nil {
		return Result{}, false, fmt.Errorf("error finding gitdir: %s", err)
	}

	// Commonly .git folder is present when it's a worktree but there's an exception where
	// worktree is present but .git folder is not present. In that case, we need to find
	// for worktree folder.
	// Find for commondir file
	commondir, ok, err := findCommondir(ctx, gitdir)
	if err != nil {
		return Result{}, false, fmt.Errorf("error finding commondir: %s", err)
	}

	// we found a commondir file so this is a worktree
	if ok {
		dir := filepath.Dir(commondir)

		// Commonly commondir file contains a .git folder but there's an exception where
		// commondir contains the actual git folder. It's common when repo is bare and
		// it's a worktree.
		if strings.LastIndex(commondir, ".git") == -1 {
			dir = commondir
		}

		remote := readGitRemote(ctx, commondir)
		project := projectOrRemote(filepath.Base(dir), g.ProjectFromGitRemote, remote)

		branch, err := findGitBranch(ctx, filepath.Join(gitdir, "HEAD"))
		if err != nil {
			logger.Errorf(
				"error finding branch from %q: %s",
				filepath.Join(filepath.Dir(dotGit), "HEAD"),
				err,
			)
		}

		return Result{
			Project:               project,
			Branch:                branch,
			Folder:                dir,
			Repository:            remote.Repository,
			RepositoryDescription: remote.RepositoryDescription,
		}, true, nil
	}

	// Otherwise it's only a plain .git file and not a submodule
	if gitdir != "" && !strings.Contains(gitdir, "modules") {
		remote := readGitRemote(ctx, gitdir)
		project := projectOrRemote(filepath.Base(filepath.Join(dotGit, "..")), g.ProjectFromGitRemote, remote)

		branch, err := findGitBranch(ctx, filepath.Join(gitdir, "HEAD"))
		if err != nil {
			logger.Errorf(
				"error finding branch from %q: %s",
				filepath.Join(filepath.Dir(gitdir), "HEAD"),
				err,
			)
		}

		return Result{
			Project:               project,
			Branch:                branch,
			Folder:                filepath.Join(gitdir, ".."),
			Repository:            remote.Repository,
			RepositoryDescription: remote.RepositoryDescription,
		}, true, nil
	}

	// Find for .git/config file
	gitConfigFile, found := FindFileOrDirectory(ctx, fp, filepath.Join(".git", "config"))

	if found {
		gitDir := filepath.Dir(gitConfigFile)
		projectDir := filepath.Join(gitDir, "..")

		branch, err := findGitBranch(ctx, filepath.Join(gitDir, "HEAD"))
		if err != nil {
			logger.Errorf(
				"error finding branch from %q: %s",
				filepath.Join(gitDir, "HEAD"),
				err,
			)
		}

		remote := readGitRemote(ctx, gitDir)
		project := projectOrRemote(filepath.Base(projectDir), g.ProjectFromGitRemote, remote)

		return Result{
			Project:               project,
			Branch:                branch,
			Folder:                projectDir,
			Repository:            remote.Repository,
			RepositoryDescription: remote.RepositoryDescription,
		}, true, nil
	}

	return Result{}, false, nil
}

func findSubmodule(ctx context.Context, fp string, patterns []regex.Regex) (string, bool, error) {
	if !shouldTakeSubmodule(ctx, fp, patterns) {
		return "", false, nil
	}

	gitConfigFile, found := FindFileOrDirectory(ctx, fp, ".git")
	if !found {
		return "", false, nil
	}

	gitdir, err := findGitdir(ctx, gitConfigFile)
	if err != nil {
		return "", false,
			fmt.Errorf("error finding gitdir for submodule: %s", err)
	}

	if strings.Contains(gitdir, "modules") {
		return gitdir, true, nil
	}

	return "", false, nil
}

// shouldTakeSubmodule checks a filepath against the passed in regex patterns to determine,
// if submodule filepath should be taken.
func shouldTakeSubmodule(ctx context.Context, fp string, patterns []regex.Regex) bool {
	for _, p := range patterns {
		if p.MatchString(ctx, fp) {
			return false
		}
	}

	return true
}

func findGitdir(ctx context.Context, fp string) (string, error) {
	lines, err := file.ReadLines(ctx, fp, 1)
	if err != nil {
		return "", fmt.Errorf("failed while opening file %q: %s", fp, err)
	}

	if len(lines) > 0 && strings.HasPrefix(lines[0], "gitdir: ") {
		if arr := strings.Split(lines[0], "gitdir: "); len(arr) > 1 {
			return resolveGitdir(filepath.Join(fp, ".."), arr[1])
		}
	}

	return "", nil
}

func resolveGitdir(fp, gitdir string) (string, error) {
	subPath := strings.TrimSpace(gitdir)
	if !filepath.IsAbs(subPath) {
		subPath = filepath.Join(fp, subPath)
	}

	if fileOrDirExists(filepath.Join(subPath, "HEAD")) {
		return subPath, nil
	}

	return "", nil
}

func findCommondir(ctx context.Context, fp string) (string, bool, error) {
	if fp == "" {
		return "", false, nil
	}

	if filepath.Base(filepath.Dir(fp)) != "worktrees" {
		return "", false, nil
	}

	if fileOrDirExists(filepath.Join(fp, "commondir")) {
		return resolveCommondir(ctx, fp)
	}

	return "", false, nil
}

func resolveCommondir(ctx context.Context, fp string) (string, bool, error) {
	lines, err := file.ReadLines(ctx, filepath.Join(fp, "commondir"), 1)
	if err != nil {
		return "", false,
			fmt.Errorf("failed while opening file %q: %s", fp, err)
	}

	if len(lines) == 0 {
		return "", false, nil
	}

	gitdir, err := filepath.Abs(filepath.Join(fp, lines[0]))
	if err != nil {
		return "", false,
			fmt.Errorf("failed to get absolute path: %s", err)
	}

	return gitdir, true, nil
}

func findGitBranch(ctx context.Context, fp string) (string, error) {
	if !fileOrDirExists(fp) {
		return "master", nil
	}

	lines, err := file.ReadLines(ctx, fp, 1)
	if err != nil {
		return "", fmt.Errorf("failed while opening file %q: %s", fp, err)
	}

	logger := log.Extract(ctx)

	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "ref: ") {
		parts := strings.SplitN(lines[0], "/", 3)
		if len(parts) < 3 {
			logger.Warnf("invalid branch from %q: %s", fp, lines[0])

			return "", nil
		}

		return strings.TrimSpace(strings.SplitN(lines[0], "/", 3)[2]), nil
	}

	return "", nil
}

func projectOrRemote(projectName string, projectFromGitRemote bool, remote gitRemote) string {
	if projectFromGitRemote && remote.RepositoryDescription != "" {
		return remote.RepositoryDescription
	}

	return projectName
}

func readGitRemote(ctx context.Context, dotGitFolder string) gitRemote {
	logger := log.Extract(ctx)
	configFile := filepath.Join(dotGitFolder, "config")

	remoteURL, err := findGitRemoteURL(ctx, configFile)
	if err != nil {
		logger.Errorf("error finding git remote from %q: %s", configFile, err)

		return gitRemote{}
	}

	remote, err := parseGitRemote(remoteURL)
	if err != nil {
		logger.Errorf("error parsing git remote %q from %q: %s", remoteURL, configFile, err)

		return gitRemote{}
	}

	return remote
}

func findGitRemoteURL(ctx context.Context, fp string) (string, error) {
	if !fileOrDirExists(fp) {
		return "", nil
	}

	lines, err := file.ReadLines(ctx, fp, 1000)
	if err != nil {
		return "", fmt.Errorf("failed while opening file %q: %s", fp, err)
	}

	for i, line := range lines {
		if strings.Trim(line, "\n\r\t") != "[remote \"origin\"]" {
			continue
		}

		if i >= len(lines) {
			continue
		}

		for _, subline := range lines[i+1:] {
			if strings.HasPrefix(subline, "[") {
				break
			}

			if strings.HasPrefix(strings.TrimSpace(subline), "url = ") {
				remote := strings.Trim(subline, "\n\r\t")

				parts := strings.SplitN(remote, "=", 2)
				if len(parts) != 2 {
					return "", fmt.Errorf("invalid origin url from %q: %s", fp, subline)
				}

				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "", nil
}

func parseGitRemote(remoteURL string) (gitRemote, error) {
	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return gitRemote{}, nil
	}

	if scpLikeRemoteRegex.MatchString(remoteURL) {
		parts := strings.SplitN(remoteURL, ":", 2)
		hostParts := strings.SplitN(parts[0], "@", 2)
		host := hostParts[len(hostParts)-1]

		return gitRemoteFromHostPath(host, parts[1]), nil
	}

	parsed, err := url.Parse(remoteURL)
	if err != nil {
		return gitRemote{}, fmt.Errorf("invalid remote URL: %w", err)
	}

	if parsed.Scheme == "" && parsed.Host == "" {
		return gitRemoteFromHostPath("", remoteURL), nil
	}

	return gitRemoteFromHostPath(parsed.Host, parsed.Path), nil
}

func gitRemoteFromHostPath(host, path string) gitRemote {
	host = strings.TrimSpace(host)
	path = strings.Trim(strings.TrimSpace(path), "/")
	path = strings.TrimSuffix(path, ".git")

	if path == "" {
		return gitRemote{}
	}

	if host == "" {
		return gitRemote{
			Repository:            path,
			RepositoryDescription: path,
		}
	}

	return gitRemote{
		Repository:            "https://" + host + "/" + path,
		RepositoryDescription: path,
	}
}

// ID returns its id.
func (Git) ID() DetectorID {
	return GitDetector
}
