package multibase

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSpec(t *testing.T) {
	file, err := os.Open("spec/multibase.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.LazyQuotes = false
	reader.FieldsPerRecord = 3
	reader.TrimLeadingSpace = true

	values, err := reader.ReadAll()
	if err != nil {
		t.Error(err)
	}
	expectedEncodings := make(map[Encoding]string, len(values)-1)
	for _, v := range values[1:] {
		encoding := v[0]
		codeStr := v[1]

		var code Encoding
		if strings.HasPrefix(codeStr, "0x") {
			i, err := strconv.ParseUint(codeStr[2:], 16, 64)
			if err != nil {
				t.Errorf("invalid multibase byte %q", codeStr)
				continue
			}
			code = Encoding(i)
		} else {
			codeRune, length := utf8.DecodeRuneInString(codeStr)
			if code == utf8.RuneError {
				t.Errorf("multibase %q wasn't valid utf8", codeStr)
				continue
			}
			if length != len(codeStr) {
				t.Errorf("multibase %q wasn't a single character", codeStr)
				continue
			}
			code = Encoding(codeRune)
		}
		expectedEncodings[code] = encoding
	}

	for name, enc := range Encodings {
		expectedName, ok := expectedEncodings[enc]
		if !ok {
			t.Errorf("encoding %q (%c) not defined in the spec", name, enc)
			continue
		}
		if expectedName != name {
			t.Errorf("encoding %q (%c) has unexpected name %q", expectedName, enc, name)
		}
	}
}
func TestSpecVectors(t *testing.T) {
	files, err := filepath.Glob("spec/tests/test[0-9]*.csv")
	if err != nil {
		t.Fatal(err)
	}
	for _, fname := range files {
		t.Run(fname, func(t *testing.T) {
			file, err := os.Open(fname)
			if err != nil {
				t.Error(err)
				return
			}
			defer file.Close()
			reader := csv.NewReader(file)
			reader.LazyQuotes = false
			reader.FieldsPerRecord = 2
			reader.TrimLeadingSpace = true

			values, err := reader.ReadAll()
			if err != nil {
				t.Error(err)
			}
			if len(values) == 0 {
				t.Error("no test values")
				return
			}
			header := values[0]

			var decodeOnly bool
			switch header[0] {
			case "encoding":
			case "non-canonical encoding":
				decodeOnly = true
			default:
				t.Errorf("invalid test spec %q", fname)
				return
			}

			testValue, err := strconv.Unquote("\"" + header[1] + "\"")
			if err != nil {
				t.Error("failed to unquote testcase:", err)
				return
			}

			for _, testCase := range values[1:] {
				encodingName := testCase[0]
				expected := testCase[1]

				t.Run(encodingName, func(t *testing.T) {
					encoder, err := EncoderByName(encodingName)
					if err != nil {
						t.Skipf("skipping %s: not supported", encodingName)
						return
					}
					if !decodeOnly {
						t.Logf("encoding %q with %s", testValue, encodingName)
						actual := encoder.Encode([]byte(testValue))
						if expected != actual {
							t.Errorf("expected %q, got %q", expected, actual)
						}
					}
					t.Logf("decoding %q", expected)
					encoding, decoded, err := Decode(expected)
					if err != nil {
						t.Error("failed to decode:", err)
						return
					}
					expectedEncoding := Encodings[encodingName]
					if encoding != expectedEncoding {
						t.Errorf("expected encoding to be %c, got %c", expectedEncoding, encoding)
					}
					if string(decoded) != testValue {
						t.Errorf("failed to decode %q to %q, got %q", expected, testValue, string(decoded))
					}
				})

			}
		})
	}
}
