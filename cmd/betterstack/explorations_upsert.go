package betterstack

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tommymorgan/betterstack-cli/internal/client"
	"github.com/tommymorgan/betterstack-cli/internal/errs"
	"github.com/tommymorgan/betterstack-cli/internal/output"
	"github.com/tommymorgan/betterstack-cli/internal/payload"
)

// runExplorationsUpsert implements the --upsert flow. It lists explorations,
// matches on name, and:
//   - 0 matches → POST, action=created
//   - 1 match with differing shorthand fields → PATCH, action=updated
//   - 1 match with identical shorthand fields → no write, action=unchanged
//   - 2+ matches → error (exit 3)
//
// The upsert is best-effort: a concurrent actor creating a duplicate between
// list and POST, or deleting the match between list and PATCH, is reported via
// specific error prefixes.
func runExplorationsUpsert(cmd *cobra.Command, c *client.TelemetryClient, s payload.ExplorationShorthand) error {
	ctx := context.Background()
	all, err := c.ListExplorations(ctx, 0)
	if err != nil {
		return wrapAPIErr(err)
	}

	matches, parseErr := findExplorationsByName(all, s.Name)
	if parseErr != nil {
		return errs.Wrap(errs.ExitUpstream,
			fmt.Errorf("upsert cannot match explorations: malformed envelope: %w", parseErr))
	}
	switch len(matches) {
	case 0:
		body, err := payload.BuildExplorationCreate(s)
		if err != nil {
			return err
		}
		result, err := c.CreateExploration(ctx, body)
		if err != nil {
			return wrapAPIErr(err)
		}
		return renderUpsert(cmd, "created", result)

	case 1:
		existing := matches[0]
		equal, parseErr := explorationShorthandEquals(existing.raw, s)
		if parseErr != nil {
			return errs.Wrap(errs.ExitUpstream,
				fmt.Errorf("could not verify match for upsert: malformed exploration envelope: %w", parseErr))
		}
		if equal {
			return renderUpsert(cmd, "unchanged", existing.raw)
		}
		body, err := payload.BuildExplorationPatch(s)
		if err != nil {
			return err
		}
		result, err := c.UpdateExploration(ctx, existing.id, body)
		if err != nil {
			if isNotFound(err) {
				fmt.Fprintf(cmd.ErrOrStderr(), "error: upsert_match_deleted_after_precheck: explorations get %s\n", existing.id)
				return errs.SilentExit(errs.ExitUpstream)
			}
			return wrapAPIErr(err)
		}
		return renderUpsert(cmd, "updated", result)

	default:
		return errs.New(errs.ExitUserInput,
			"%d explorations match name %q; use a unique --name or rename duplicates via `explorations update`",
			len(matches), s.Name)
	}
}

type explorationSummary struct {
	id   string
	name string
	raw  json.RawMessage
}

// findExplorationsByName returns every exploration in list whose name
// matches. A parse failure on any envelope is surfaced as an error so upsert
// does not silently miss a match and POST a duplicate.
func findExplorationsByName(list []json.RawMessage, name string) ([]explorationSummary, error) {
	var out []explorationSummary
	for _, item := range list {
		var env struct {
			ID         string `json:"id"`
			Attributes struct {
				Name string `json:"name"`
			} `json:"attributes"`
		}
		if err := json.Unmarshal(item, &env); err != nil {
			return nil, err
		}
		if env.Attributes.Name == name {
			out = append(out, explorationSummary{id: env.ID, name: env.Attributes.Name, raw: item})
		}
	}
	return out, nil
}

// explorationShorthandEquals checks whether the existing resource already has
// the provided-shorthand values. Equality is over only the provided fields.
// Returns (false, err) on parse failure so upsert does not silently treat an
// unreadable match as "unchanged".
func explorationShorthandEquals(raw json.RawMessage, s payload.ExplorationShorthand) (bool, error) {
	var e output.Exploration
	if err := json.Unmarshal(raw, &e); err != nil {
		return false, err
	}
	// Name is always provided in an upsert (matching logic requires it).
	if s.Name != "" && e.Attributes.Name != s.Name {
		return false, nil
	}
	if s.SourceID != "" {
		if len(e.Attributes.Queries) == 0 || e.Attributes.Queries[0].SourceVariable != s.SourceID {
			return false, nil
		}
	}
	if s.Pattern != "" {
		// The where_condition is built from the pattern; compare the resulting literal.
		want := fmt.Sprintf(`message CONTAINS %q`, s.Pattern)
		if len(e.Attributes.Queries) == 0 || e.Attributes.Queries[0].WhereCondition != want {
			return false, nil
		}
	}
	return true, nil
}

func renderUpsert(cmd *cobra.Command, action string, resource json.RawMessage) error {
	w := cmd.OutOrStdout()
	if jsonFlag(cmd) {
		return output.RenderJSON(w, map[string]any{
			"action":   action,
			"resource": resource,
		})
	}
	fmt.Fprintf(w, "%s:\n", action)
	return output.RenderExplorationDetail(w, resource)
}

// isNotFound reports whether an error is a 404 API error.
func isNotFound(err error) bool {
	apiErr, ok := err.(*client.APIError)
	if !ok {
		return false
	}
	return apiErr.StatusCode == 404
}
