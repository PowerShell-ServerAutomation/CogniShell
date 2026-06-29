package github

import (
	"reflect"
	"testing"
)

func TestParseCODEOWNERS(t *testing.T) {
	content := `# Default owner
*       @global-owner

# Script specific owners
/scripts/hello-world/   @vsrivastava @johndoe
/scripts/test-secrets/  @secops-team
`

	tests := []struct {
		name       string
		scriptPath string
		want       []string
	}{
		{
			name:       "Match hello-world directory",
			scriptPath: "scripts/hello-world/hello-world.ps1",
			want:       []string{"@vsrivastava", "@johndoe"},
		},
		{
			name:       "Match test-secrets directory",
			scriptPath: "scripts/test-secrets/test-secrets.ps1",
			want:       []string{"@secops-team"},
		},
		{
			name:       "Fallback to wildcard",
			scriptPath: "scripts/other/test.ps1",
			want:       []string{"@global-owner"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCODEOWNERS(content, tt.scriptPath)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCODEOWNERS() = %v, want %v", got, tt.want)
			}
		})
	}
}
