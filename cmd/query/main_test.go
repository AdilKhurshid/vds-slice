package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSliceHappyHTTPResponse(t *testing.T) {
	testcases := []sliceTest{
		{
			baseTest{
				name:           "Valid GET Request",
				method:         http.MethodGet,
				expectedStatus: http.StatusOK,
			},
			testSliceRequest{
				Vds:       well_known,
				Direction: "i",
				Lineno:    0, //side-effect assurance that 0 is accepted
				Sas:       "n/a",
				Bounds: []testBound{
					{Direction: "inline", Lower: 1, Upper: 3},
				},
			},
		},
		{
			baseTest{
				name:           "Valid json POST Request",
				method:         http.MethodPost,
				expectedStatus: http.StatusOK,
			},
			testSliceRequest{
				Vds:       well_known,
				Direction: "crossline",
				Lineno:    10,
				Sas:       "n/a",
				Bounds: []testBound{
					{Direction: "inline", Lower: 1, Upper: 3},
				},
			},
		},
	}

	for _, testcase := range testcases {
		w := setupTest(t, testcase)

		requireStatus(t, testcase, w)
		parts := readMultipartData(t, w)

		require.Equalf(t, 2, len(parts),
			"Wrong number of multipart data parts in case '%s'", testcase.name)

		inlineAxis := testSliceAxis{
			Annotation: "Inline", Max: 3.0, Min: 1.0, Samples: 2, StepSize: 2, Unit: "unitless",
		}
		crosslineAxis := testSliceAxis{
			Annotation: "Crossline", Max: 11.0, Min: 10.0, Samples: 2, StepSize: 1, Unit: "unitless",
		}
		sampleAxis := testSliceAxis{
			Annotation: "Sample", Max: 16.0, Min: 4.0, Samples: 4, StepSize: 4, Unit: "ms",
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
		require.NoErrorf(t, err, "Failed json metadata extraction in case '%s'", testcase.name)
		require.EqualValuesf(t, expectedMetadata, metadata,
			"Metadata not equal in case '%s'", testcase.name)

		expectedDataLength := expectedMetadata.X.Samples *
			expectedMetadata.Y.Samples * 4 //4 bytes each
		require.Equalf(t, expectedDataLength, len(parts[1]),
			"Wrong number of bytes in data reply in case '%s'", testcase.name)
	}
}

func TestSliceErrorHTTPResponse(t *testing.T) {
	testcases := []endpointTest{
		sliceTest{
			baseTest{
				name:           "Invalid json GET request",
				method:         http.MethodGet,
				jsonRequest:    "help I am a duck",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "invalid character",
			},
			testSliceRequest{},
		},
		sliceTest{
			baseTest{
				name:           "GET request missing query parameter",
				method:         http.MethodGet,
				jsonRequest:    "MODIFIER: skip query parameter",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "GET request to specified endpoint requires a 'query' parameter",
			},
			testSliceRequest{},
		},
		sliceTest{
			baseTest{
				name:           "Invalid json POST request",
				method:         http.MethodPost,
				jsonRequest:    "help I am a duck",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "invalid character",
			},
			testSliceRequest{},
		},
		sliceTest{
			baseTest{
				name:   "Missing parameters GET request",
				method: http.MethodGet,
				jsonRequest: "{\"vds\":\"" + well_known +
					"\", \"direction\":\"i\", \"sas\": \"n/a\"}",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "Error:Field validation for 'Lineno'",
			},
			testSliceRequest{},
		},
		sliceTest{
			baseTest{
				name:   "Missing parameters POST Request",
				method: http.MethodPost,
				jsonRequest: "{\"vds\":\"" + well_known +
					"\", \"lineno\":1, \"sas\": \"n/a\"}",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "Error:Field validation for 'Direction'",
			},
			testSliceRequest{},
		},
		sliceTest{
			baseTest{
				name:   "Incomplete bounds parameters POST Request",
				method: http.MethodPost,
				jsonRequest: "{\"vds\":\"" + well_known +
					"\", \"lineno\":1, \"direction\": \"i\", \"sas\": \"n/a\", " +
					"\"bounds\": [{\"Upper\": 2 }]}",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "Error:Field validation for 'Direction'",
			},
			testSliceRequest{},
		},
		sliceTest{
			baseTest{
				name:           "Request with unknown axis",
				method:         http.MethodPost,
				expectedStatus: http.StatusBadRequest,
				expectedError:  "invalid direction 'unknown', valid options are",
			},
			testSliceRequest{
				Vds:       well_known,
				Direction: "unknown",
				Lineno:    1,
				Sas:       "n/a",
			},
		},
		sliceTest{
			baseTest{
				name:           "Request with out-of-bound lineno",
				method:         http.MethodPost,
				expectedStatus: http.StatusBadRequest,
				expectedError:  "Invalid lineno: 10, valid range: [0:2:1]",
			},
			testSliceRequest{
				Vds:       well_known,
				Direction: "i",
				Lineno:    10,
				Sas:       "n/a",
			},
		},
		sliceTest{
			baseTest{
				name:           "Datahandle error",
				method:         http.MethodPost,
				expectedStatus: http.StatusInternalServerError,
				expectedError:  "Could not open VDS",
			},
			testSliceRequest{
				Vds:       "unknown",
				Direction: "i",
				Lineno:    1,
				Sas:       "n/a",
			},
		},
	}
	testErrorHTTPResponse(t, testcases)
}

func TestFenceHappyHTTPResponse(t *testing.T) {
	testcases := []fenceTest{
		{
			baseTest{
				name:           "Valid GET Request",
				method:         http.MethodGet,
				expectedStatus: http.StatusOK,
			},

			testFenceRequest{
				Vds:              well_known,
				CoordinateSystem: "ilxl",
				Coordinates:      [][]float32{{3, 11}, {2, 10}},
				FillValue:        float32(-999.25),
				Sas:              "n/a",
			},
		},
		{
			baseTest{
				name:           "Valid json POST Request",
				method:         http.MethodPost,
				expectedStatus: http.StatusOK,
			},

			testFenceRequest{
				Vds:              well_known,
				CoordinateSystem: "ij",
				Coordinates:      [][]float32{{0, 1}, {1, 1}, {1, 0}},
				FillValue:        float32(-999.25),
				Sas:              "n/a",
			},
		},
	}

	for _, testcase := range testcases {
		w := setupTest(t, testcase)

		requireStatus(t, testcase, w)
		parts := readMultipartData(t, w)
		require.Equalf(t, 2, len(parts),
			"Wrong number of multipart data parts in case '%s'", testcase.name)

		metadata := string(parts[0])
		coordinatesLength := len(testcase.fence.Coordinates)
		expectedMetadata := `{
			"shape": [` + fmt.Sprint(coordinatesLength) + `, 4],
			"format": "<f4"
		}`
		require.JSONEqf(t, expectedMetadata, metadata,
			"Metadata not equal in case '%s'", testcase.name)

		expectedDataLength := coordinatesLength * 4 * 4 //4 bytes, 4 samples per each requested
		require.Equalf(t, expectedDataLength, len(parts[1]),
			"Wrong number of bytes in data reply in case '%s'", testcase.name)
	}
}

func TestFenceErrorHTTPResponse(t *testing.T) {
	testcases := []endpointTest{
		fenceTest{
			baseTest{
				name:           "Invalid json GET request",
				method:         http.MethodGet,
				jsonRequest:    "help I am a duck",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "invalid character",
			},
			testFenceRequest{},
		},
		fenceTest{
			baseTest{
				name:           "GET request missing query parameter",
				method:         http.MethodGet,
				jsonRequest:    "MODIFIER: skip query parameter",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "GET request to specified endpoint requires a 'query' parameter",
			},
			testFenceRequest{},
		},
		fenceTest{
			baseTest{
				name:           "Invalid json POST request",
				method:         http.MethodPost,
				jsonRequest:    "help I am a duck",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "invalid character",
			},
			testFenceRequest{},
		},
		fenceTest{
			baseTest{
				name:   "Missing parameters GET request",
				method: http.MethodGet,
				jsonRequest: "{\"vds\":\"" + well_known +
					"\", \"coordinateSystem\":\"ilxl\"," +
					"\"fillValue\": -999.25," +
					"\"coordinates\":[[0, 0]]}",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "No valid Sas token is found",
			},
			testFenceRequest{},
		},
		fenceTest{
			baseTest{
				name:   "Missing parameters POST Request",
				method: http.MethodPost,
				jsonRequest: "{\"vds\":\"" + well_known +
					"\", \"coordinateSystem\":\"ilxl\"," +
					"\"fillValue\": -999.25," +
					"\"sas\": \"n/a\"}",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "Error:Field validation for 'Coordinates'",
			},
			testFenceRequest{},
		},
		fenceTest{
			baseTest{
				name:           "Request with unknown coordinate system",
				method:         http.MethodPost,
				expectedStatus: http.StatusBadRequest,
				expectedError:  "coordinate system not recognized: 'unknown', valid options are",
			},
			testFenceRequest{
				Vds:              well_known,
				CoordinateSystem: "unknown",
				Coordinates:      [][]float32{{3, 12}, {2, 10}},
				Sas:              "n/a",
			},
		},
		fenceTest{
			baseTest{
				name:           "Request with incorrect coordinate pair length",
				method:         http.MethodGet,
				expectedStatus: http.StatusBadRequest,
				expectedError:  "invalid coordinate [2 10 3 4] at position 2, expected [x y] pair",
			},
			testFenceRequest{
				Vds:              well_known,
				CoordinateSystem: "cdp",
				Coordinates:      [][]float32{{3, 1001}, {200, 10}, {2, 10, 3, 4}, {1, 1}},
				Sas:              "n/a",
			},
		},
		fenceTest{
			baseTest{
				name:           "Datahandle error",
				method:         http.MethodPost,
				expectedStatus: http.StatusInternalServerError,
				expectedError:  "Could not open VDS",
			},
			testFenceRequest{
				Vds:              "unknown",
				CoordinateSystem: "ilxl",
				Coordinates:      [][]float32{{3, 12}, {2, 10}},
				Sas:              "n/a",
			},
		},
	}
	testErrorHTTPResponse(t, testcases)
}

func TestMetadataHappyHTTPResponse(t *testing.T) {
	testcases := []metadataTest{
		{
			baseTest{

				name:           "Valid GET Request",
				method:         http.MethodGet,
				expectedStatus: http.StatusOK,
			},
			testMetadataRequest{
				Vds: well_known,
				Sas: "n/a",
			},
		},
		{
			baseTest{
				name:           "Valid json POST Request",
				method:         http.MethodPost,
				expectedStatus: http.StatusOK,
			},
			testMetadataRequest{
				Vds: well_known,
				Sas: "n/a",
			},
		},
	}

	for _, testcase := range testcases {
		w := setupTest(t, testcase)

		requireStatus(t, testcase, w)
		metadata := w.Body.String()
		expectedMetadata := `{
			"axis": [
				{"annotation": "Inline", "max": 5.0, "min": 1.0, "samples" : 3, "stepsize":2, "unit": "unitless"},
				{"annotation": "Crossline", "max": 11.0, "min": 10.0, "samples" : 2, "stepsize":1, "unit": "unitless"},
				{"annotation": "Sample", "max": 16.0, "min": 4.0, "samples" : 4, "stepsize":4, "unit": "ms"}
			],
			"boundingBox": {
				"cdp": [[2,0],[14,8],[12,11],[0,3]],
				"ilxl": [[1, 10], [5, 10], [5, 11], [1, 11]],
				"ij": [[0, 0], [2, 0], [2, 1], [0, 1]]
			},
			"crs"            : "utmXX",
			"inputFileName"  : "well_known.segy",
			"importTimeStamp": "^\\d{4}-\\d{2}-\\d{2}[A-Z]\\d{2}:\\d{2}:\\d{2}\\.\\d{3}[A-Z]$"
		}`

		var expectedMap map[string]any
		var actualMap map[string]any

		json.Unmarshal([]byte(expectedMetadata), &expectedMap)
		json.Unmarshal([]byte(metadata), &actualMap)

		if _, ok := actualMap["importTimeStamp"]; !ok {
			t.Errorf("importTimeStampt is not found in case '%s'", testcase.name)
		}

		require.Regexp(t, expectedMap["importTimeStamp"], actualMap["importTimeStamp"])

		expectedMap["importTimeStamp"] = "dummy"
		actualMap["importTimeStamp"] = "dummy"

		require.Equal(t, expectedMap, actualMap, "Metadata not equal in case '%s'", testcase.name)
	}
}

func TestMetadataErrorHTTPResponse(t *testing.T) {
	testcases := []endpointTest{
		metadataTest{
			baseTest{
				name:           "Invalid json GET request",
				method:         http.MethodGet,
				jsonRequest:    "help I am a duck",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "invalid character",
			}, testMetadataRequest{},
		},
		metadataTest{
			baseTest{
				name:           "GET request missing query parameter",
				method:         http.MethodGet,
				jsonRequest:    "MODIFIER: skip query parameter",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "GET request to specified endpoint requires a 'query' parameter",
			},
			testMetadataRequest{},
		},
		metadataTest{
			baseTest{
				name:           "Invalid json POST request",
				method:         http.MethodPost,
				jsonRequest:    "help I am a duck",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "invalid character",
			}, testMetadataRequest{},
		},
		metadataTest{
			baseTest{
				name:           "Missing parameters GET request",
				method:         http.MethodGet,
				jsonRequest:    "{\"vds\":\"" + well_known + "\"}",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "No valid Sas token is found",
			}, testMetadataRequest{},
		},
		metadataTest{
			baseTest{
				name:           "Missing parameters POST Request",
				method:         http.MethodPost,
				jsonRequest:    "{\"sas\":\"somevalidsas\"}",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "Error:Field validation for 'Vds'",
			}, testMetadataRequest{},
		},
		metadataTest{
			baseTest{
				name:           "Datahandle error",
				method:         http.MethodPost,
				expectedStatus: http.StatusInternalServerError,
				expectedError:  "Could not open VDS",
			},

			testMetadataRequest{
				Vds: "unknown",
				Sas: "n/a",
			},
		},
	}
	testErrorHTTPResponse(t, testcases)
}

func TestAttributeOutOfBounds(t *testing.T) {
	newCase := func(name string, above, below, stepsize float32, status int) attributeAlongSurfaceTest {
		return attributeAlongSurfaceTest{
			baseTest{
				name:           name,
				method:         http.MethodPost,
				expectedStatus: status,
			},
			testAttributeAlongSurfaceRequest{
				Vds:        samples10,
				Values:     [][]float32{{20}},
				Sas:        "n/a",
				Above:      above,
				Below:      below,
				StepSize:   stepsize,
				Attributes: []string{"samplevalue"},
			},
		}
	}

	testCases := []endpointTest{
		newCase("Min values", 0, 0, 0, http.StatusOK),
		newCase("Above is too low", -1, 1, 1, http.StatusBadRequest),
		newCase("Above is too high", 250, 1, 1, http.StatusBadRequest),
		newCase("Below is too low", 1, -1, 1, http.StatusBadRequest),
		newCase("Below is too high", 1, 250, 1, http.StatusBadRequest),
		newCase("StepSize is too low", 1, 1, -1, http.StatusBadRequest),
	}

	for _, testcase := range testCases {
		w := setupTest(t, testcase)
		requireStatus(t, testcase, w)
	}
}

func TestAttributeHappyHTTPResponse(t *testing.T) {
	testcases := []attributeEndpointTest{
		attributeAlongSurfaceTest{
			baseTest{
				name:           "Valid json POST Request along surface",
				method:         http.MethodPost,
				expectedStatus: http.StatusOK,
			},

			testAttributeAlongSurfaceRequest{
				Vds:        samples10,
				Values:     [][]float32{{20, 20}, {20, 20}, {20, 20}},
				Sas:        "n/a",
				Above:      8.0,
				Below:      4.0,
				Attributes: []string{"samplevalue"},
			},
		},
		attributeBetweenSurfacesTest{
			baseTest{
				name:           "Valid json POST Request between surfaces",
				method:         http.MethodPost,
				expectedStatus: http.StatusOK,
			},

			testAttributeBetweenSurfacesRequest{
				Vds:             samples10,
				ValuesPrimary:   [][]float32{{20, 20}, {20, 20}, {20, 20}},
				ValuesSecondary: [][]float32{{20, 20}, {20, 20}, {20, 20}},
				Sas:             "n/a",
				Attributes:      []string{"samplevalue"},
			},
		},
	}

	for _, testcase := range testcases {
		w := setupTest(t, testcase)

		requireStatus(t, testcase, w)

		parts := readMultipartData(t, w)
		require.Equalf(t, 2, len(parts),
			"Wrong number of multipart data parts in case '%s'", testcase.base().name)

		metadata := string(parts[0])
		xLength := testcase.nrows()
		yLength := testcase.ncols()
		expectedMetadata := `{
			"shape": [` + fmt.Sprint(xLength) + `,` + fmt.Sprint(yLength) + `],
			"format": "<f4"
		}`
		require.JSONEqf(t, expectedMetadata, metadata,
			"Metadata not equal in case '%s'", testcase.base().name)

		expectedDataLength := xLength * yLength * 4 //4 bytes each
		require.Equalf(t, expectedDataLength, len(parts[1]),
			"Wrong number of bytes in data reply in case '%s'", testcase.base().name)
	}
}

func TestAttributeErrorHTTPResponse(t *testing.T) {
	testcases := []endpointTest{
		attributeAlongSurfaceTest{
			baseTest{
				name:           "Along: Invalid json POST request",
				method:         http.MethodPost,
				jsonRequest:    "help I am a duck",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "invalid character",
			},
			testAttributeAlongSurfaceRequest{},
		},
		attributeBetweenSurfacesTest{
			baseTest{
				name:           "Between: Invalid json POST request",
				method:         http.MethodPost,
				jsonRequest:    "help I am a duck",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "invalid character",
			},
			testAttributeBetweenSurfacesRequest{},
		},
		attributeAlongSurfaceTest{
			baseTest{
				name:   "Along: Missing parameters POST Request",
				method: http.MethodPost,
				jsonRequest: "{\"vds\":\"" + well_known +
					"\", \"interpolation\":\"cubic\", \"sas\": \"n/a\"}",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "Error:Field validation for",
			},
			testAttributeAlongSurfaceRequest{},
		},
		attributeBetweenSurfacesTest{
			baseTest{
				name:   "Between: Missing parameters POST Request",
				method: http.MethodPost,
				jsonRequest: "{\"vds\":\"" + well_known +
					"\", \"interpolation\":\"cubic\", \"sas\": \"n/a\"}",
				expectedStatus: http.StatusBadRequest,
				expectedError:  "Error:Field validation for",
			},
			testAttributeBetweenSurfacesRequest{},
		},
		attributeAlongSurfaceTest{
			baseTest{
				name:           "Along: Bad Request: wrong incorrect interpolation method",
				method:         http.MethodPost,
				expectedStatus: http.StatusBadRequest,
				expectedError:  "invalid interpolation method",
			},
			testAttributeAlongSurfaceRequest{
				Vds:           well_known,
				Values:        [][]float32{{4, 4}, {4, 4}, {4, 4}},
				Sas:           "n/a",
				Interpolation: "unsupported",
				Attributes:    []string{"samplevalue"},
			},
		},
		attributeBetweenSurfacesTest{
			baseTest{
				name:           "Between: Bad Request: wrong incorrect interpolation method",
				method:         http.MethodPost,
				expectedStatus: http.StatusBadRequest,
				expectedError:  "invalid interpolation method",
			},
			testAttributeBetweenSurfacesRequest{
				Vds:             well_known,
				ValuesPrimary:   [][]float32{{4, 4}, {4, 4}, {4, 4}},
				ValuesSecondary: [][]float32{{4, 4}, {4, 4}, {4, 4}},
				Sas:             "n/a",
				Interpolation:   "unsupported",
				Attributes:      []string{"samplevalue"},
			},
		},
		attributeAlongSurfaceTest{
			baseTest{
				name:           "Along: Datahandle error",
				method:         http.MethodPost,
				expectedStatus: http.StatusInternalServerError,
				expectedError:  "Could not open VDS",
			},
			testAttributeAlongSurfaceRequest{
				Vds:        "unknown",
				Values:     [][]float32{{4, 4}, {4, 4}, {4, 4}},
				Sas:        "n/a",
				Attributes: []string{"samplevalue"},
			},
		},
		attributeBetweenSurfacesTest{
			baseTest{
				name:           "Between: Datahandle error",
				method:         http.MethodPost,
				expectedStatus: http.StatusInternalServerError,
				expectedError:  "Could not open VDS",
			},
			testAttributeBetweenSurfacesRequest{
				Vds:             "unknown",
				ValuesPrimary:   [][]float32{{4, 4}, {4, 4}, {4, 4}},
				ValuesSecondary: [][]float32{{4, 4}, {4, 4}, {4, 4}},
				Sas:             "n/a",
				Attributes:      []string{"samplevalue"},
			},
		},
	}
	testErrorHTTPResponse(t, testcases)
}

func TestLogHasNoSas(t *testing.T) {
	var testcases []endpointTest
	addTests := func(method string) {
		// white box testing. we know all endpoints are handled the same
		okTest := sliceTest{
			baseTest{
				name:           fmt.Sprintf("%v OK Request", method),
				method:         method,
				expectedStatus: http.StatusOK,
			},
			testSliceRequest{
				Vds:       well_known,
				Direction: "crossline",
				Lineno:    10,
				Sas:       "SPARTA...T14:43:29Z%26se=2023",
			},
		}

		errorTest := metadataTest{
			baseTest{
				name:           fmt.Sprintf("%v Error Request", method),
				method:         method,
				expectedStatus: http.StatusInternalServerError,
			},
			testMetadataRequest{
				Vds: "unknown",
				Sas: "SPARTA...T14:43:29Z%26se=2023...",
			},
		}

		testcases = append(testcases, okTest)
		testcases = append(testcases, errorTest)
	}

	addTests(http.MethodGet)
	addTests(http.MethodPost)

	d := gin.DefaultWriter
	defer func() {
		gin.DefaultWriter = d
	}()

	for _, testcase := range testcases {
		buffer := new(bytes.Buffer)
		gin.DefaultWriter = buffer

		w := setupTest(t, testcase)

		requireStatus(t, testcase, w)

		require.NotContainsf(t, buffer.String(), "sas",
			"Test '%v'. Log should not contain SAS (sas)", testcase.base().name)
		// just in case also check for presence of parts of the token
		require.NotContainsf(t, buffer.String(), "se=",
			"Test '%v'. Log should not contain SAS (se=)", testcase.base().name)
		require.NotContainsf(t, buffer.String(), "se%3D",
			"Test '%v'. Log should not contain SAS (encoded se=)", testcase.base().name)
	}
}

func testErrorHTTPResponse(t *testing.T, testcases []endpointTest) {
	for _, testcase := range testcases {
		w := setupTest(t, testcase)

		requireStatus(t, testcase, w)

		testErrorInfo := &testErrorResponse{}
		err := json.Unmarshal(w.Body.Bytes(), testErrorInfo)
		require.NoError(t, err, "Test '%v'. Couldn't unmarshal data.", testcase.base().name)

		require.Containsf(t, testErrorInfo.Error, testcase.base().expectedError,
			"Test '%v'. Error string does not contain expected message.", testcase.base().name)
	}
}
