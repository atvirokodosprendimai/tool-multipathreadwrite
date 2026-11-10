package guide

import (
	"strings"
	"testing"
)

// ADR-196: urfave/cli v3.14.0 hands mrw every path and attached flag value as
// typed, so the guide stops teaching ADR-069's `--` escape and the refusal it
// answered. Both texts an agent reads say a path reaches mrw as typed; neither
// says a padded path must go after --, nor that a padded attached value is
// refused, nor that a lone - is moved.
func TestTheGuideSaysAPathReachesMrwAsTyped(t *testing.T) {
	for name, text := range map[string]string{"instructions": CLI(), "--core": Core()} {
		if !strings.Contains(text, "reaches mrw as typed") {
			t.Errorf("%s does not say a path reaches mrw as typed", name)
		}
		for _, gone := range []string{"goes after --", "mrw read -- 'x '", "as its own argument", "mrw moves it last"} {
			if strings.Contains(text, gone) {
				t.Errorf("%s still says %q", name, gone)
			}
		}
	}
}
