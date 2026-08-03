package systemdunit

import "testing"

func TestQuote(t *testing.T) {
	got, err := Quote(`C:\path with "quotes"\100%`)
	if err != nil {
		t.Fatal(err)
	}
	want := `"C:\\path with \"quotes\"\\100%%"`
	if got != want {
		t.Fatalf("Quote() = %q, want %q", got, want)
	}
}

func TestCommandUsesSystemdEscapes(t *testing.T) {
	got, err := Command("/opt/my app/bin", "$HOME", "100%", "a'b")
	if err != nil {
		t.Fatal(err)
	}
	want := `"/opt/my app/bin" "$$HOME" "100%%" "a'b"`
	if got != want {
		t.Fatalf("Command() = %q, want %q", got, want)
	}
}

func TestRejectsControlCharacters(t *testing.T) {
	if _, err := Quote("safe\nInjected=yes"); err == nil {
		t.Fatal("expected newline rejection")
	}
	if _, err := Command(); err == nil {
		t.Fatal("expected empty command rejection")
	}
}
