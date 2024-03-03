package report

import (
	"bytes"
	"sync"
	"testing"
)

func Test_parseBytes(t *testing.T) {
	type test struct {
		input  []byte
		expect []*reading
	}
	testCase := func(input string, expect ...*reading) (tc test) {
		tc.input = []byte(input)
		tc.expect = expect
		return
	}
	testCases := []test{
		testCase(
			"x;4.2\n", // shortest valid byte slice
			&reading{station: []byte("x"), subzero: false, temp: temperature{0, 4, 2}},
		),
		testCase(
			"Aix-en-Provence;0.0\n",
			&reading{station: []byte("Aix-en-Provence"), subzero: false, temp: temperature{0, 0, 0}},
		),
		testCase(
			"Aix-en-Provence;3.5\n",
			&reading{station: []byte("Aix-en-Provence"), subzero: false, temp: temperature{0, 3, 5}},
		),
		testCase(
			"Aix-en-Provence;30.5\n",
			&reading{station: []byte("Aix-en-Provence"), subzero: false, temp: temperature{3, 0, 5}},
		),
		testCase(
			"Aix-en-Provence;24.5\n",
			&reading{station: []byte("Aix-en-Provence"), subzero: false, temp: temperature{2, 4, 5}},
		),
		testCase(
			"Aix-en-Provence;-0.0\n",
			&reading{station: []byte("Aix-en-Provence"), subzero: true, temp: temperature{0, 0, 0}},
		),
		testCase(
			"Aix-en-Provence;-3.5\n",
			&reading{station: []byte("Aix-en-Provence"), subzero: true, temp: temperature{0, 3, 5}},
		),
		testCase(
			"Aix-en-Provence;-30.5\n",
			&reading{station: []byte("Aix-en-Provence"), subzero: true, temp: temperature{3, 0, 5}},
		),
		testCase(
			"Aix-en-Provence;-24.5\n",
			&reading{station: []byte("Aix-en-Provence"), subzero: true, temp: temperature{2, 4, 5}},
		),
		testCase(
			"\nAix-en-Provence;-24.5\n",
			&reading{station: []byte("Aix-en-Provence"), subzero: true, temp: temperature{2, 4, 5}},
		),
		testCase(
			"ignore;-24.5\nAix-en-Provence;-24.5\n",
			&reading{station: []byte("Aix-en-Provence"), subzero: true, temp: temperature{2, 4, 5}},
		),
		testCase(
			"\nAix-en-Provence;-24.5\nignore;-",
			&reading{station: []byte("Aix-en-Provence"), subzero: true, temp: temperature{2, 4, 5}},
		),
		testCase(
			"ignore;-24.5\nAix-en-Provence;-24.5\nignore;-",
			&reading{station: []byte("Aix-en-Provence"), subzero: true, temp: temperature{2, 4, 5}},
		),
		testCase(
			"Aix-en-Provence;-24.5\nDenver;0.0\n",
			&reading{station: []byte("Denver"), subzero: false, temp: temperature{0, 0, 0}},
		),
		testCase(
			"\nAix-en-Provence;-24.5\nDenver;0.0\n",
			&reading{station: []byte("Denver"), subzero: false, temp: temperature{0, 0, 0}},
			&reading{station: []byte("Aix-en-Provence"), subzero: true, temp: temperature{2, 4, 5}},
		),
		testCase(
			"ignore;-24.5\nAix-en-Provence;-24.5\nDenver;0.0\n",
			&reading{station: []byte("Denver"), subzero: false, temp: temperature{0, 0, 0}},
			&reading{station: []byte("Aix-en-Provence"), subzero: true, temp: temperature{2, 4, 5}},
		),
		testCase(
			"\nAix-en-Provence;-24.5\nDenver;0.0\nignore;-",
			&reading{station: []byte("Denver"), subzero: false, temp: temperature{0, 0, 0}},
			&reading{station: []byte("Aix-en-Provence"), subzero: true, temp: temperature{2, 4, 5}},
		),
		testCase(
			"ignore;-24.5\nAix-en-Provence;-24.5\nDenver;0.0\nignore;-",
			&reading{station: []byte("Denver"), subzero: false, temp: temperature{0, 0, 0}},
			&reading{station: []byte("Aix-en-Provence"), subzero: true, temp: temperature{2, 4, 5}},
		),
	}
	for _, tc := range testCases {
		t.Run(string(tc.input), func(t *testing.T) {
			readings := make(chan *reading)
			wg := &sync.WaitGroup{}
			wg.Add(1)
			var initialNL, terminalNL int
			go func() {
				initialNL, terminalNL = parseBytes(tc.input, readings)
				wg.Done()
			}()
			for _, expect := range tc.expect {
				actual := <-readings
				if actual == nil {
					if expect != nil {
						t.Errorf("expected reading; got nil")
					}
					return
				}
				if !bytes.Equal(expect.station, actual.station) {
					t.Errorf("expected: %s; got: %s", expect.station, actual.station)
				}
				if actual.subzero != expect.subzero {
					t.Errorf("expected subzero: %v; got: %v", expect.subzero, actual.subzero)
				}
				for i, v := range expect.temp {
					if actual.temp[i] != v {
						t.Errorf("expected offset %d to be: %d; got: %d", i, v, actual.temp[i])
					}
				}
			}
			if len(tc.expect) == 0 {
				select {
				case <-readings:
					t.Errorf("unexpected reading")
				}
			}
			wg.Wait()
			expectInitialNL := bytes.IndexByte(tc.input, '\n')
			expectTerminalNL := bytes.LastIndexByte(tc.input, '\n')
			if terminalNL != expectTerminalNL {
				t.Errorf("expected terminal newline at: %d; got %d", expectTerminalNL, terminalNL)
			}
			if expectInitialNL == expectTerminalNL {
				if initialNL != noNewline {
					t.Errorf("expected initial newline at: %d; got %d", noNewline, initialNL)
				}
			} else {
				if initialNL != expectInitialNL {
					t.Errorf("expected initial newline at: %d; got %d", expectInitialNL, initialNL)
				}
			}
		})
	}
}
