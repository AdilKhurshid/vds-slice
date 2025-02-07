import numpy as np
import pytest
import json
import re
from utils.cloud import *
from shared_test_functions import *


@pytest.mark.parametrize("method", [
    ("get"),
    ("post")
])
def test_metadata(method):
    metadata = dict(request_metadata(method))
    expected_metadata = {
        "axis": [
            {"annotation": "Inline",    "max": 5.0,  "min": 1.0,
                "samples": 3, "stepsize": 2.0, "unit": "unitless"},
            {"annotation": "Crossline", "max": 11.0, "min": 10.0,
                "samples": 2, "stepsize": 1.0, "unit": "unitless"},
            {"annotation": "Sample",    "max": 16.0, "min": 4.0,
                "samples": 4, "stepsize": 4.0, "unit": "ms"}
        ],
        "boundingBox": {
            "cdp": [[2, 0], [14, 8], [12, 11], [0, 3]],
            "ilxl": [[1, 10], [5, 10], [5, 11], [1, 11]],
            "ij": [[0, 0], [2, 0], [2, 1], [0, 1]]
        },
        "crs": "utmXX",
        "inputFileName": "well_known.segy",
        "importTimeStamp": "^\\d{4}-\\d{2}-\\d{2}[A-Z]\\d{2}:\\d{2}:\\d{2}\\.\\d{3}[A-Z]$"
    }

    expected_import_ts = expected_metadata.get("importTimeStamp")
    actual_import_ts = metadata.get("importTimeStamp")
    assert re.compile(expected_import_ts).match(
        actual_import_ts), f"Not a valid import Time Stamp {actual_import_ts} in metadata"
    expected_metadata["importTimeStamp"] = "dummy"
    metadata["importTimeStamp"] = "dummy"
    assert expected_metadata == metadata


@pytest.mark.parametrize("method", [
    ("get"),
    ("post")
])
def test_slice(method):
    meta, slice = request_slice(method, 5, 'inline')

    expected = np.array([[116, 117, 118, 119],
                         [120, 121, 122, 123]])
    assert np.array_equal(slice, expected)

    expected_meta = json.loads("""
    {
        "x": {"annotation": "Sample", "max": 16.0, "min": 4.0, "samples" : 4, "stepsize": 4.0, "unit": "ms"},
        "y": {"annotation": "Crossline", "max": 11.0, "min": 10.0, "samples" : 2, "stepsize": 1.0, "unit": "unitless"},
        "shape": [ 2, 4],
        "format": "<f4",
        "geospatial": [[14.0, 8.0], [12.0, 11.0]]
    }
    """)
    assert meta == expected_meta


@pytest.mark.parametrize("method", [
    ("get"),
    ("post")
])
def test_fence(method):
    meta, fence = request_fence(method, [[3, 10], [1, 11]], 'ilxl')

    expected = np.array([[108, 109, 110, 111],
                         [104, 105, 106, 107]])
    assert np.array_equal(fence, expected)

    expected_meta = json.loads("""
    {
        "shape": [ 2, 4],
        "format": "<f4"
    }
    """)
    assert meta == expected_meta


def test_attributes_along_surface():
    values = [
        [20, 20],
        [20, 20],
        [20, 20]
    ]
    meta, data = request_attributes_along_surface("post", values)

    expected = np.array([[-0.5, 0.5], [-8.5, 6.5], [16.5, -16.5]])
    assert np.array_equal(data, expected)

    expected_meta = json.loads("""
    {
        "shape": [3, 2],
        "format": "<f4"
    }
    """)
    assert meta == expected_meta


def test_attributes_between_surfaces():
    primary = [
        [12, 12],
        [12, 14],
        [22, 12]
    ]
    secondary = [
        [30,   28],
        [27.5, 29],
        [24,   12]
    ]
    meta, data = request_attributes_between_surfaces(
        "post", primary, secondary)

    expected = np.array([[1.5, 2.5], [-8.5, 7.5], [18.5, -8.5]])
    assert np.array_equal(data, expected)

    expected_meta = json.loads("""
    {
        "shape": [3, 2],
        "format": "<f4"
    }
    """)
    assert meta == expected_meta
