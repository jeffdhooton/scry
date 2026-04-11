package php

import (
	"reflect"
	"testing"
	"time"
)

func TestScannerCollectsUsesAndClassRefs(t *testing.T) {
	src := []byte(`<?php

use App\Http\Controllers\UserController;
use App\Models\{User, Post as Article};
use Illuminate\Support\Facades\Route;
use function array_keys;

Route::get('/users', [UserController::class, 'index']);
Route::get('/users/{user}', [UserController::class, 'show']);
Route::resource('posts', \App\Http\Controllers\PostController::class);

// A line comment with UserController::class that should NOT be picked up.
$comment = 'a string with UserController::class inside should NOT match';

Route::post('/articles', [Article::class, 'store']);
`)

	s := newPhpScanner(src)
	res := s.collect()
	uses, refs := res.uses, res.classRefs

	wantUses := map[string]string{
		"UserController": "App\\Http\\Controllers\\UserController",
		"User":           "App\\Models\\User",
		"Article":        "App\\Models\\Post",
		"Route":          "Illuminate\\Support\\Facades\\Route",
	}
	for alias, want := range wantUses {
		if got := uses[alias]; got != want {
			t.Errorf("uses[%q] = %q, want %q", alias, got, want)
		}
	}

	// 4 valid ::class refs (3 UserController + Route + 1 absolute PostController + Article)
	// = 1 Route bound use + 4 ::class hits.
	// But we only emit class refs (Route::get is a method call, not ::class).
	// Expected ::class refs:
	//   1. UserController on line 8
	//   2. UserController on line 9
	//   3. \App\Http\Controllers\PostController on line 10
	//   4. Article on line 15
	if len(refs) != 4 {
		for i, r := range refs {
			t.Logf("ref[%d] = %+v", i, r)
		}
		t.Fatalf("got %d refs, want 4", len(refs))
	}

	wantNames := []string{
		"UserController",
		"UserController",
		"\\App\\Http\\Controllers\\PostController",
		"Article",
	}
	gotNames := make([]string, len(refs))
	for i, r := range refs {
		gotNames[i] = r.name
	}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Errorf("ref names = %v, want %v", gotNames, wantNames)
	}
}

func TestResolveName(t *testing.T) {
	uses := map[string]string{
		"UserController": "App\\Http\\Controllers\\UserController",
		"Route":          "Illuminate\\Support\\Facades\\Route",
	}
	cases := []struct {
		in, want string
	}{
		{"UserController", "App\\Http\\Controllers\\UserController"},
		{"\\App\\Http\\Controllers\\PostController", "App\\Http\\Controllers\\PostController"},
		{"Route\\Foo", "Illuminate\\Support\\Facades\\Route\\Foo"},
		{"Closure", ""}, // unqualified, no use entry → unresolvable
		{"Foo\\Bar", "Foo\\Bar"},
	}
	for _, c := range cases {
		if got := resolveName(c.in, uses); got != c.want {
			t.Errorf("resolveName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestScannerCollectsStringCallRefs(t *testing.T) {
	src := []byte(`<?php

return [
    'view' => view('users.show', ['user' => $user]),
    'config' => config('mail.from.address'),
    'translated' => __('messages.welcome'),
    'computed' => view($dynamic), // not a literal — must NOT be picked up
    'with_escaping' => config('foo\'bar.baz'),
    'interpolated' => "value $foo bar", // double-quoted with interpolation, not a string ref
];

// view('commented.out') in a line comment must NOT match
$noise = 'config(\\'should.not.match\\') inside string';
`)
	s := newPhpScanner(src)
	refs := s.collect().stringRefs

	wantPairs := []struct {
		fn, val string
	}{
		{"view", "users.show"},
		{"config", "mail.from.address"},
		{"__", "messages.welcome"},
		{"config", "foo'bar.baz"},
	}
	if len(refs) != len(wantPairs) {
		for i, r := range refs {
			t.Logf("ref[%d] = %+v", i, r)
		}
		t.Fatalf("got %d string refs, want %d", len(refs), len(wantPairs))
	}
	for i, want := range wantPairs {
		if refs[i].funcName != want.fn || refs[i].value != want.val {
			t.Errorf("ref[%d] = (%q,%q), want (%q,%q)", i, refs[i].funcName, refs[i].value, want.fn, want.val)
		}
	}
}

func TestScannerHandlesIncompleteUTF8(t *testing.T) {
	// Regression: a real Laravel command file from hoopless_crm hung the
	// scanner because a `$this->line(...)` interpolated string contained the
	// `→` Unicode arrow (\xE2\x86\x92), and an early termination of the
	// scanner inside the string left s.pos at one of those bytes. The main
	// loop then dispatched to scanIdentifierOrKeyword via Latin-1 widening
	// (rune(0xE2) = `â`, which IS a unicode letter), but utf8.DecodeRune on
	// the multibyte sequence returned RuneError, so the identifier reader
	// produced an empty string and returned without advancing — infinite
	// loop. The fix is twofold: (1) decode UTF-8 properly when deciding
	// whether a byte starts an identifier, and (2) force-advance one byte
	// if scanIdentifierOrKeyword returns without making progress.
	//
	// This test asserts both: a file containing the arrow inside an
	// interpolated string scans to completion in well under a second.
	src := []byte("<?php\n$x->line(\"hello {$y} → world\");\n$z = 1;\n")
	done := make(chan struct{})
	go func() {
		s := newPhpScanner(src)
		_ = s.collect()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("scanner failed to terminate within 1 second on a file with interpolated unicode")
	}

	// Truncated UTF-8 (last byte of an arrow is missing) is the worst case;
	// we should still terminate cleanly.
	truncated := []byte{'<', '?', 'p', 'h', 'p', '\n', 'a', '(', '"', 'b', 0xE2, 0x86}
	done2 := make(chan struct{})
	go func() {
		s := newPhpScanner(truncated)
		_ = s.collect()
		close(done2)
	}()
	select {
	case <-done2:
	case <-time.After(1 * time.Second):
		t.Fatal("scanner failed to terminate within 1 second on truncated UTF-8")
	}
}

func TestScannerSkipsStringsAndComments(t *testing.T) {
	src := []byte(`<?php
// UserController::class in line comment
/* UserController::class in block comment */
$single = 'UserController::class in single quotes';
$double = "UserController::class in double quotes";
$heredoc = <<<EOT
UserController::class inside heredoc
EOT;
use App\Models\Real;
Real::class;
`)
	s := newPhpScanner(src)
	refs := s.collect().classRefs
	if len(refs) != 1 {
		for i, r := range refs {
			t.Logf("ref[%d] = %+v", i, r)
		}
		t.Fatalf("got %d refs, want 1", len(refs))
	}
	if refs[0].name != "Real" {
		t.Errorf("ref[0].name = %q, want %q", refs[0].name, "Real")
	}
}
