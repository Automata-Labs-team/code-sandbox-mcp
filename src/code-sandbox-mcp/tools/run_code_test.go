package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/Automata-Labs-team/code-sandbox-mcp/languages"
)

func TestRunInDocker(t *testing.T) {
	tests := []struct {
		name        string
		language    languages.Language
		code        string
		wantOutput  string
		wantErr     bool
		errContains string
	}{
		{
			name:     "simple javascript code",
			language: languages.NodeJS,
			code: `
				console.log('Hello from JavaScript!');
				const sum = (a, b) => a + b;
				console.log(sum(5, 3));
			`,
			wantOutput: "Hello from JavaScript!\n8\n",
			wantErr:    false,
		},
		{
			name:     "typescript with types",
			language: languages.NodeJS,
			code: `
				interface Point {
					x: number;
					y: number;
				}
				
				const calculateDistance = (p1: Point, p2: Point): number => {
					return Math.sqrt(Math.pow(p2.x - p1.x, 2) + Math.pow(p2.y - p1.y, 2));
				};
				
				const point1: Point = { x: 0, y: 0 };
				const point2: Point = { x: 3, y: 4 };
				console.log(calculateDistance(point1, point2));
			`,
			wantOutput: "5\n",
			wantErr:    false,
		},
		{
			name:     "go code with package",
			language: languages.Go,
			code: `
				package main

				import (
					"fmt"
					"strings"
				)

				func main() {
					message := "Hello from Go!"
					fmt.Println(strings.ToUpper(message))
				}
			`,
			wantOutput: "HELLO FROM GO!\n",
			wantErr:    false,
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := languages.SupportedLanguages[tt.language]
			output, _, err := runInDocker(ctx, config.RunCommand, config.Image, tt.code, tt.language)

			// Check error cases
			if (err != nil) != tt.wantErr {
				t.Errorf("runInDocker() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("runInDocker() error = %v, want error containing %v", err, tt.errContains)
				return
			}

			// Check output
			if !tt.wantErr {
				// Normalize line endings and trim spaces
				got := strings.TrimSpace(output)
				t.Logf("got: %q", got)
				want := strings.TrimSpace(tt.wantOutput)
				if got != want {
					t.Errorf("runInDocker() output = %q, want %q", got, want)
				}
			}
		})
	}
}
