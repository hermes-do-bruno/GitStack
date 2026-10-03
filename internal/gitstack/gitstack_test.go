package gitstack

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadConfigAndUpsert(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFilename)
	input := "# comment\n\nbranch1=master\nbranch2=branch1,branchX\n"
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Parents("branch2"); !reflect.DeepEqual(got, []string{"branch1", "branchX"}) {
		t.Fatalf("parents mismatch: %v", got)
	}

	cfg.Upsert("branch2", []string{"branch3"})
	cfg.Upsert("branch4", []string{"branch2", "branch3"})
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}

	reloaded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "# comment\n\nbranch1=master\nbranch2=branch3\nbranch4=branch2,branch3\n"
	if string(reloaded) != want {
		t.Fatalf("saved content mismatch:\nwant:\n%s\ngot:\n%s", want, string(reloaded))
	}
}

func TestSubtreeUsesChildOrder(t *testing.T) {
	cfg := &ConfigFile{EntryIndexes: map[string]int{}}
	cfg.Upsert("root", []string{"master"})
	cfg.Upsert("left", []string{"root"})
	cfg.Upsert("right", []string{"root"})
	cfg.Upsert("leaf", []string{"left"})

	order, err := cfg.Subtree("root")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"root", "left", "leaf", "right"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order mismatch: want %v got %v", want, order)
	}
}
