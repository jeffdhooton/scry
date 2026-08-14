package main

import (
	"reflect"
	"testing"
)

func TestInstallToolArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    []string
		wantErr bool
	}{
		{name: "defaults to both npm indexers", want: []string{"typescript", "python"}},
		{name: "accepts binary names", args: []string{"scip-typescript", "scip-python"}, want: []string{"typescript", "python"}},
		{name: "deduplicates", args: []string{"python", "scip-python"}, want: []string{"python"}},
		{name: "rejects unsupported tool", args: []string{"php"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := installToolArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tools = %#v, want %#v", got, tt.want)
			}
		})
	}
}
