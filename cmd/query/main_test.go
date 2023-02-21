package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSliceHappyHTTPResponse(t *testing.T) {
	testcases := []sliceTest{
		{
			name:   "Valid GET Request",
			method: http.MethodGet,
			slice: testSliceRequest{
				Vds:       well_known,
				Direction: "i",
				Lineno:    0, //side-effect assurance that 0 is accepted
				Sas:       "n/a",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Valid json POST Request",
			method: http.MethodPost,
			slice: testSliceRequest{
				Vds:       well_known,
				Direction: "crossline",
				Lineno:    10,
				Sas:       "n/a",
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, testcase := range testcases {
		w := httptest.NewRecorder()
		ctx, r := gin.CreateTestContext(w)
		setupTestServer(r)

		prepareSliceRequest(ctx, t, testcase)
		r.ServeHTTP(w, ctx.Request)

		if w.Result().StatusCode != testcase.expectedStatus {
			msg := "Got status %v; want %d %s in case '%s'"
			t.Errorf(
				msg,
				w.Result().Status,
				testcase.expectedStatus,
				http.StatusText(testcase.expectedStatus),
				testcase.name,
			)
		}
		parts := readMultipartData(t, w)
		if len(parts) != 2 {
			msg := "Got %d parts in reply; want it to always contain 2 in case '%s'"
			t.Errorf(
				msg,
				len(parts),
				testcase.name,
			)
		}

		inlineAxis := testSliceAxis{
			Annotation: "Inline", Max: 5.0, Min: 1.0, Samples: 3, Unit: "unitless",
		}
		crosslineAxis := testSliceAxis{
			Annotation: "Crossline", Max: 11.0, Min: 10.0, Samples: 2, Unit: "unitless",
		}
		sampleAxis := testSliceAxis{
			Annotation: "Sample", Max: 16.0, Min: 4.0, Samples: 4, Unit: "ms",
		}
		expectedFormat := "<f4"

		var expectedMetadata *testSliceMetadata
		switch testcase.slice.Direction {
		case "i":
			expectedMetadata = &testSliceMetadata{
				X:      sampleAxis,
				Y:      crosslineAxis,
				Format: expectedFormat}
		case "crossline":
			expectedMetadata = &testSliceMetadata{
				X:      sampleAxis,
				Y:      inlineAxis,
				Format: expectedFormat}
		default:
			t.Fatalf("Unhandled direction %s in case %s", testcase.slice.Direction, testcase.name)
		}

		metadata := &testSliceMetadata{}
		err := json.Unmarshal(parts[0], metadata)

		if err != nil {
			msg := "Failed json metadata extraction in case '%s'"
			t.Fatalf(
				msg,
				testcase.name,
			)
		}

		if !reflect.DeepEqual(metadata, expectedMetadata) {
			msg := "Got %v as metadata; want it to be %v in case '%s'"
			t.Fatalf(
				msg,
				metadata,
				expectedMetadata,
				testcase.name,
			)
		}

		expectedDataLength := expectedMetadata.X.Samples *
			expectedMetadata.Y.Samples * 4 //4 bytes each
		if len(parts[1]) != expectedDataLength {
			msg := "Got %d bytes in data reply; want it to be %d in case '%s'"
			t.Errorf(
				msg,
				len(parts[2]),
				expectedDataLength,
				testcase.name,
			)
		}
	}
}

func TestSliceErrorHTTPResponse(t *testing.T) {
	testcases := []sliceTest{
		{
			name:           "Invalid json GET request",
			method:         http.MethodGet,
			sliceRequest:   "help I am a duck",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid character",
		},
		{
			name:           "Invalid json POST request",
			method:         http.MethodPost,
			sliceRequest:   "help I am a duck",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid character",
		},
		{
			name:   "Missing parameters GET request",
			method: http.MethodGet,
			sliceRequest: "{\"vds\":\"" + well_known +
				"\", \"direction\":\"i\", \"sas\": \"n/a\"}",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Error:Field validation for 'Lineno'",
		},
		{
			name:   "Missing parameters POST Request",
			method: http.MethodPost,
			sliceRequest: "{\"vds\":\"" + well_known +
				"\", \"lineno\":1, \"sas\": \"n/a\"}",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Error:Field validation for 'Direction'",
		},
		{
			name:   "Request with unknown axis",
			method: http.MethodPost,
			slice: testSliceRequest{
				Vds:       well_known,
				Direction: "unknown",
				Lineno:    1,
				Sas:       "n/a",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid direction 'unknown', valid options are",
		},
		{
			name:   "Request which passed all input checks but still should fail",
			method: http.MethodPost,
			slice: testSliceRequest{
				Vds:       well_known,
				Direction: "i",
				Lineno:    10,
				Sas:       "n/a",
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Invalid lineno: 10, valid range: [0:2:1]",
		},
	}

	for _, testcase := range testcases {
		w := httptest.NewRecorder()
		ctx, r := gin.CreateTestContext(w)
		setupTestServer(r)

		prepareSliceRequest(ctx, t, testcase)
		r.ServeHTTP(w, ctx.Request)

		if w.Result().StatusCode != testcase.expectedStatus {
			msg := "Got status %v; want %d %s in case '%s'"
			t.Errorf(
				msg,
				w.Result().Status,
				testcase.expectedStatus,
				http.StatusText(testcase.expectedStatus),
				testcase.name,
			)
		}
		assertError(w, t, testcase.name, testcase.expectedError)
	}
}

func TestFenceHappyHTTPResponse(t *testing.T) {
	testcases := []fenceTest{
		{
			name:   "Valid GET Request",
			method: http.MethodGet,
			fence:  testFenceRequest{
				Vds:              well_known,
				CoordinateSystem: "ilxl",
				Coordinates:      [][]float32{{3, 11}, {2, 10}},
				Sas:              "n/a",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Valid json POST Request",
			method: http.MethodPost,
			fence:  testFenceRequest{
				Vds:              well_known,
				CoordinateSystem: "ij",
				Coordinates:      [][]float32{{0, 1}, {1, 1}, {1, 0}},
				Sas:              "n/a",
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, testcase := range testcases {
		w := httptest.NewRecorder()
		ctx, r := gin.CreateTestContext(w)
		setupTestServer(r)

		prepareFenceRequest(ctx, t, testcase)
		r.ServeHTTP(w, ctx.Request)

		if w.Result().StatusCode != testcase.expectedStatus {
			msg := "Got status %v; want %d %s in case '%s'"
			t.Errorf(
				msg,
				w.Result().Status,
				testcase.expectedStatus,
				http.StatusText(testcase.expectedStatus),
				testcase.name,
			)
		}
		parts := readMultipartData(t, w)
		if len(parts) != 2 {
			msg := "Got %d parts in reply; want it to always contain 3 in case '%s'"
			t.Errorf(
				msg,
				len(parts),
				testcase.name,
			)
		}

		metadata := string(parts[0])
		expectedMetadata := `{
			"shape": [` + fmt.Sprint(len(testcase.fence.Coordinates)) + `, 4],
			"format": "<f4"
		}`

		if metadata != expectedMetadata {
			msg := "Metadata not equal in case '%s'"
			require.JSONEq(t, expectedMetadata, metadata, fmt.Sprintf(msg, testcase.name))
		}

		expectedDataLength := len(testcase.fence.Coordinates) * 4 * 4 //4 bytes, 4 samples per each requested
		if len(parts[1]) != expectedDataLength {
			msg := "Got %d bytes in data reply; want it to be %d in case '%s'"
			t.Errorf(
				msg,
				len(parts[2]),
				expectedDataLength,
				testcase.name,
			)
		}
	}
}

func TestFenceErrorHTTPResponse(t *testing.T) {
	testcases := []fenceTest{
		{
			name:           "Invalid json GET request",
			method:         http.MethodGet,
			fenceRequest:   "help I am a duck",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid character",
		},
		{
			name:           "Invalid json POST request",
			method:         http.MethodPost,
			fenceRequest:   "help I am a duck",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid character",
		},
		{
			name:   "Missing parameters GET request",
			method: http.MethodGet,
			fenceRequest: "{\"vds\":\"" + well_known +
				"\", \"coordinateSystem\":\"ilxl\", \"coordinates\":[[0, 0]]}",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Error:Field validation for 'Sas'",
		},
		{
			name:   "Missing parameters POST Request",
			method: http.MethodPost,
			fenceRequest: "{\"vds\":\"" + well_known +
				"\", \"coordinateSystem\":\"ilxl\", \"sas\": \"n/a\"}",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Error:Field validation for 'Coordinates'",
		},
		{
			name:   "Request with unknown coordinate system",
			method: http.MethodPost,
			fence:  testFenceRequest{
				Vds:              well_known,
				CoordinateSystem: "unknown",
				Coordinates:      [][]float32{{3, 12}, {2, 10}},
				Sas:              "n/a",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "coordinate system not recognized: 'unknown', valid options are",
		},
		{
			name:   "Request which passed all input checks but still should fail",
			method: http.MethodPost,
			fence:  testFenceRequest{
				Vds:              "unknown",
				CoordinateSystem: "ilxl",
				Coordinates:      [][]float32{{3, 12}, {2, 10}},
				Sas:              "n/a",
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Could not open VDS",
		},
	}

	for _, testcase := range testcases {
		w := httptest.NewRecorder()
		ctx, r := gin.CreateTestContext(w)
		setupTestServer(r)

		prepareFenceRequest(ctx, t, testcase)
		r.ServeHTTP(w, ctx.Request)

		if w.Result().StatusCode != testcase.expectedStatus {
			msg := "Got status %v; want %d %s in case '%s'"
			t.Errorf(
				msg,
				w.Result().Status,
				testcase.expectedStatus,
				http.StatusText(testcase.expectedStatus),
				testcase.name,
			)
		}
		assertError(w, t, testcase.name, testcase.expectedError)
	}
}

func TestMetadataHappyHTTPResponse(t *testing.T) {
	testcases := []metadataTest{
		{
			name:   "Valid GET Request",
			method: http.MethodGet,
			metadata: testMetadataRequest{
				Vds: well_known,
				Sas: "n/a",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Valid json POST Request",
			method: http.MethodPost,
			metadata: testMetadataRequest{
				Vds: well_known,
				Sas: "n/a",
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, testcase := range testcases {
		w := httptest.NewRecorder()
		ctx, r := gin.CreateTestContext(w)
		setupTestServer(r)

		prepareMetadataRequest(ctx, t, testcase)
		r.ServeHTTP(w, ctx.Request)

		if w.Result().StatusCode != testcase.expectedStatus {
			msg := "Got status %v; want %d %s in case '%s'"
			t.Errorf(
				msg,
				w.Result().Status,
				testcase.expectedStatus,
				http.StatusText(testcase.expectedStatus),
				testcase.name,
			)
		}
		metadata := w.Body.String()
		expectedMetadata := `{
			"axis": [
				{"annotation": "Inline", "max": 5.0, "min": 1.0, "samples" : 3, "unit": "unitless"},
				{"annotation": "Crossline", "max": 11.0, "min": 10.0, "samples" : 2, "unit": "unitless"},
				{"annotation": "Sample", "max": 16.0, "min": 4.0, "samples" : 4, "unit": "ms"}
			],
			"boundingBox": {
				"cdp": [[5,0],[9,8],[4,11],[0,3]],
				"ilxl": [[1, 10], [5, 10], [5, 11], [1, 11]],
				"ij": [[0, 0], [2, 0], [2, 1], [0, 1]]
			},
			"crs": "utmXX",
			"format": "<f4"
		}`

		if metadata != expectedMetadata {
			msg := "Metadata not equal in case '%s'"
			require.JSONEq(t, expectedMetadata, metadata, fmt.Sprintf(msg, testcase.name))
		}
	}
}

func TestMetadataErrorHTTPResponse(t *testing.T) {
	testcases := []metadataTest{
		{
			name:            "Invalid json GET request",
			method:          http.MethodGet,
			metadataRequest: "help I am a duck",
			expectedStatus:  http.StatusBadRequest,
			expectedError:   "invalid character",
		},
		{
			name:            "Invalid json POST request",
			method:          http.MethodPost,
			metadataRequest: "help I am a duck",
			expectedStatus:  http.StatusBadRequest,
			expectedError:   "invalid character",
		},
		{
			name:            "Missing parameters GET request",
			method:          http.MethodGet,
			metadataRequest: "{\"vds\":\"" + well_known + "\"}",
			expectedStatus:  http.StatusBadRequest,
			expectedError:   "Error:Field validation for 'Sas'",
		},
		{
			name:            "Missing parameters POST Request",
			method:          http.MethodPost,
			metadataRequest: "{\"sas\":\"somevalidsas\"}",
			expectedStatus:  http.StatusBadRequest,
			expectedError:   "Error:Field validation for 'Vds'",
		},
		{
			name:     "Request which passed all input checks but still should fail",
			method:   http.MethodPost,
			metadata: testMetadataRequest{
				Vds: "unknown",
				Sas: "n/a",
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Could not open VDS",
		},
	}

	for _, testcase := range testcases {
		w := httptest.NewRecorder()
		ctx, r := gin.CreateTestContext(w)
		setupTestServer(r)

		prepareMetadataRequest(ctx, t, testcase)
		r.ServeHTTP(w, ctx.Request)

		if w.Result().StatusCode != testcase.expectedStatus {
			msg := "Got status %v; want %d %s in case '%s'"
			t.Errorf(
				msg,
				w.Result().Status,
				testcase.expectedStatus,
				http.StatusText(testcase.expectedStatus),
				testcase.name,
			)
		}
		assertError(w, t, testcase.name, testcase.expectedError)
	}
}
