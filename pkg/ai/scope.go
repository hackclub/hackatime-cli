package ai

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hackclub/hackatime-cli/pkg/heartbeat"
)

// maxScopeDepth bounds the walk up the directory tree when looking for a
// project root marker.
const maxScopeDepth = 100

// activityScope is the folder that a heartbeat's activity belongs to. AI
// activity only turns human heartbeats into AI coding when both happen in
// related scopes, so an agent working in one project cannot relabel time spent
// in another.
type activityScope struct {
	// path is the cleaned folder, compared case-insensitively on Windows.
	path string
	// anchored is true when path is a known project folder, either because the
	// heartbeat named it or because it holds a project root marker. Only
	// anchored scopes contain the scopes nested beneath them.
	anchored bool
}

// scopeResolver resolves heartbeat scopes, caching the project root lookup
// for each folder.
type scopeResolver struct {
	roots map[string]activityScope
}

func newScopeResolver() *scopeResolver {
	return &scopeResolver{roots: make(map[string]activityScope)}
}

// scope returns the folder a heartbeat belongs to, preferring an explicit
// project folder over its file path. The second value is false when the
// heartbeat has no folder, such as an AI chat that never named a workspace.
func (r *scopeResolver) scope(h heartbeat.Heartbeat) (activityScope, bool) {
	if folder := firstNonEmptyString(h.ProjectPathOverride, h.ProjectPath); folder != "" {
		return activityScope{path: normalizeScopePath(folder), anchored: true}, true
	}

	if h.EntityType != heartbeat.FileType || h.Entity == "" || h.IsUnsavedEntity {
		return activityScope{}, false
	}

	entity := h.Entity
	if !filepath.IsAbs(filepath.FromSlash(entity)) && !isWindowsDrivePath(entity) {
		return activityScope{}, false
	}

	return r.root(filepath.Dir(filepath.FromSlash(entity))), true
}

// root walks up from dir to the nearest project root marker. Without one the
// folder itself is the scope, unanchored.
func (r *scopeResolver) root(dir string) activityScope {
	dir = filepath.Clean(dir)
	if cached, ok := r.roots[dir]; ok {
		return cached
	}

	scope := activityScope{path: normalizeScopePath(dir)}

	current := dir
	for range maxScopeDepth {
		if hasProjectRootMarker(current) {
			scope = activityScope{path: normalizeScopePath(current), anchored: true}
			break
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}

		current = parent
	}

	r.roots[dir] = scope

	return scope
}

// hasProjectRootMarker reports whether dir holds a file or folder that marks
// the root of a project, matching what project detection looks for first.
func hasProjectRootMarker(dir string) bool {
	for _, marker := range []string{".git", ".wakatime-project"} {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}

	return false
}

// related reports whether activity in one scope can describe work in the
// other: the same folder, or one nested inside an anchored other.
func (s activityScope) related(other activityScope) bool {
	if s.path == other.path {
		return true
	}

	return (s.anchored && scopeContains(s.path, other.path)) ||
		(other.anchored && scopeContains(other.path, s.path))
}

func scopeContains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}

	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func normalizeScopePath(path string) string {
	path = filepath.Clean(filepath.FromSlash(path))
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}

	return path
}

func isWindowsDrivePath(path string) bool {
	return len(path) >= 3 && path[1] == ':' && (path[2] == '/' || path[2] == '\\') &&
		((path[0] >= 'a' && path[0] <= 'z') || (path[0] >= 'A' && path[0] <= 'Z'))
}

// scopedHeartbeat is a heartbeat time paired with the scope it belongs to.
type scopedHeartbeat struct {
	scope  activityScope
	scoped bool
	time   float64
}

// timeWindow is the span of AI activity related to a scope.
type timeWindow struct {
	min float64
	max float64
}

// aiActivity is the scoped AI activity from one sync. AI events without a
// folder are left out, because they cannot be tied to the project a human
// heartbeat belongs to.
type aiActivity []scopedHeartbeat

func scopedAIActivity(aiHeartbeats []heartbeat.Heartbeat, scopes *scopeResolver) aiActivity {
	activity := make(aiActivity, 0, len(aiHeartbeats))

	for _, h := range aiHeartbeats {
		scope, ok := scopes.scope(h)
		if !ok {
			continue
		}

		activity = append(activity, scopedHeartbeat{scope: scope, scoped: true, time: h.Time})
	}

	return activity
}

func scopedHumanHeartbeats(humanHeartbeats []heartbeat.Heartbeat, scopes *scopeResolver) []scopedHeartbeat {
	scoped := make([]scopedHeartbeat, len(humanHeartbeats))

	for i, h := range humanHeartbeats {
		scope, ok := scopes.scope(h)
		scoped[i] = scopedHeartbeat{scope: scope, scoped: ok, time: h.Time}
	}

	return scoped
}

// windowFor returns the span of AI activity related to a human heartbeat's
// scope. It returns false when the heartbeat has no scope or no related AI
// activity, in which case the heartbeat keeps its category.
func (a aiActivity) windowFor(human scopedHeartbeat) (timeWindow, bool) {
	if !human.scoped {
		return timeWindow{}, false
	}

	var (
		window timeWindow
		found  bool
	)

	for _, ai := range a {
		if !ai.scope.related(human.scope) {
			continue
		}

		if !found {
			window = timeWindow{min: ai.time, max: ai.time}
			found = true

			continue
		}

		window.min = min(window.min, ai.time)
		window.max = max(window.max, ai.time)
	}

	return window, found
}

// firstRelatedHumanEdit returns the time of the first human edit in a related
// scope after the AI activity ended. Edits in unrelated projects say nothing
// about whether the developer took over from the agent here.
func firstRelatedHumanEdit(
	humanHeartbeats []heartbeat.Heartbeat,
	humanScopes []scopedHeartbeat,
	scope activityScope,
	after float64,
) *float64 {
	var first *float64

	for i, h := range humanHeartbeats {
		if !humanScopes[i].scoped || !humanScopes[i].scope.related(scope) {
			continue
		}

		if h.Time <= after+1 || h.HumanLineChanges == nil || *h.HumanLineChanges == 0 {
			continue
		}

		if first == nil || h.Time < *first {
			first = &humanHeartbeats[i].Time
		}
	}

	return first
}
