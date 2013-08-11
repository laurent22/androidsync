package androidsync

import (
	"testing"
)

func TestPatternMatchesFile(t *testing.T) {
	synchronizer := New()
	
	type PatternMatchesFileTest struct {
		pattern string
		path string
		separator string
		ok bool
	}

	var tests = []PatternMatchesFileTest{
		{"/some/folder/", "/abc/some/folder/", "/", false},
		{"/some/folder/", "/some/folder/", "/", true},
		{"/some/folder/", "/some/folder", "/", false},
		{"/*/some/folder/", "/abc/some/folder/", "/", true},
		{"/*/some/folder/", "/some/folder/", "/", false},
		{"/*/some/file", "/some/file/", "/", false},
		{"/*/some/file", "/some/file", "/", false},
		{"/*/some/file", "/abcd/some/file", "/", true},
		{"/*/some/file", "/abcd/some/file/", "/", false},
		{"/*/some/file_*.cfg", "/abcd/some/file_.cfg", "/", false},
		{"/*/some/file_*.cfg", "/abcd/some/file_123.cfg", "/", true},
		{"/*/some/file_*.cfg", "/abcd/some/file_1234cfg", "/", false},
		{"/*/some/folder_*/", "/abcd/some/folder_/", "/", false},
		{"/*/some/folder_*/", "/abcd/some/folder_123/", "/", true},
		{"/*/some/folder_*/", "/abcd/some/folder_123", "/", false},
		{"/*/some/folder_*/", "/abcd/some/folder_123", "/", false},
		{"/escape/this/*[test]", "/escape/this/abcd[test]", "/", true},
		{"\\abcd\\efgh\\*", "\\abcd\\efgh\\wintesting", "\\", true},
		{"*.avi", "/some/path/film.avi", "/", true},
	}
	
	for _, d := range tests {
		synchronizer.PathSeparator = d.separator
		output, err := synchronizer.PatternMatchesFile(d.pattern, d.path)
		if err != nil {
			t.Error("Expected nil error; got ", err)
			continue
		}
		if output != d.ok {
			t.Errorf("Expected %t; got %t for %s  %s", d.ok, output, d.pattern, d.path)
		}
	}
}