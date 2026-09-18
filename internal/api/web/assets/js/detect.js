const statusEl = document.getElementById("status");
const videoEl = document.getElementById("videoInput");
const canvasOutput = document.getElementById("canvasOutput");
const canvasDetect = document.getElementById("canvasDetect");

// output size a detected object is warped to, matching an object's ~2.5:3.5 ratio
const OBJECT_WARP_WIDTH = 350;
const OBJECT_WARP_HEIGHT = 490;

const txtSelectCamera = "Please select a camera ...";

const cameraEl = document.getElementById("camera");
cameraEl.addEventListener("change", async function (e) {
  const deviceId = e.target.value;

  if (!deviceId) {
    stopStream();
    statusEl.innerText = txtSelectCamera;

    return;
  }

  statusEl.innerText = "Starting stream...";
  try {
    // TODO: clean up references when creating new stream
    videoEl.srcObject = await videoState.newStream(deviceId);
    videoEl.onloadedmetadata = () => {
      videoEl.width = videoEl.videoWidth;
      videoEl.height = videoEl.videoHeight;

      videoState.renew(
        new cv.VideoCapture(videoEl),
        new cv.Mat(videoEl.videoHeight, videoEl.videoWidth, cv.CV_8UC4),
        new cv.Mat(videoEl.videoHeight, videoEl.videoWidth, cv.CV_8UC1),
      );

      processVideo();
    };

    videoEl.play();
  } catch (err) {
    statusEl.innerText = "Error: Failed to start stream. " + err;
  }
});

canvasDetect.addEventListener("click", () => {
  if (!videoState || !videoState.detected) {
    return;
  }

  videoState.detected = false;
  hideDetectOverlay();

  statusEl.innerText = "";
});

/**
 * Shows the detect-overlay canvas.
 * @returns {void}
 */
function showDetectOverlay() {
  canvasDetect.classList.add("visible");
  canvasOutput.classList.add("hidden");
}

/**
 * Hides the detect-overlay canvas.
 * @returns {void}
 */
function hideDetectOverlay() {
  canvasDetect.classList.remove("visible");
  canvasOutput.classList.remove("hidden");
}

let videoState = {
  /** @type {MediaStream|null} The active camera stream. */
  stream: null,
  /** @type {cv.VideoCapture|null} The video capture instance. */
  vidCap: null,
  /** @type {cv.Mat|null} Source matrix (RGBA). */
  src: null,
  /** @type {cv.Mat|null} Destination matrix (processed). */
  dest: null,
  processing: false,
  detected: false,

  lastApiCall: 0,
  API_COOLDOWN_MILLIS: 500,

  /**
   * Reads current video frame into the source matrix.
   * @returns {void}
   */
  readFrame() {
    this.vidCap.read(this.src);
  },

  /**
   * Checks if the source matrix lacks valid image data.
   * @returns {boolean} true if frame looks empty/black, false otherwise.
   */
  hasNoImage() {
    if (!this.src || !this.src.data || this.src.data.length === 0) {
      return true;
    }

    const data = this.src.data;
    const totalBytes = data.length;
    const samples = 12;
    const step = Math.floor(totalBytes / samples / 4) * 4 || 4;

    for (let i = 0; i < totalBytes; i += step) {
      if (data[i] !== 0) {
        return false;
      }
    }
    return true;
  },

  /**
   * Frees old matrices from memory and assigns new OpenCV instances.
   * @param {cv.VideoCapture} vidCap - New VideoCapture instance.
   * @param {cv.Mat} src - New source matrix.
   * @param {cv.Mat} dest - New destination matrix.
   * @returns {void}
   */
  renew(vidCap, src, dest) {
    if (this.src) {
      this.src.delete();
      this.src = null;
    }

    if (this.dest) {
      this.dest.delete();
      this.dest = null;
    }

    this.vidCap = vidCap;
    this.src = src;
    this.dest = dest;
  },

  isProcessing() {
    return this.processing || this.detected;
  },

  async newStream(deviceId) {
    this.stopStream();

    const settings = {
      video: true,
      audio: false,
    };

    if (deviceId) {
      settings.video = {
        deviceId: deviceId,
        width: { ideal: 1280 },
        height: { ideal: 720 },
        advanced: [
          { width: { exact: 1280 } },
          { width: { exact: 1024 } },
          { width: { exact: 640 } },
        ],
      };
    }

    this.stream = await navigator.mediaDevices.getUserMedia(settings);

    return this.stream;
  },

  stopStream() {
    if (this.stream) {
      this.stream.getTracks().forEach((track) => track.stop());
      this.stream = null;
    }
  },

  /**
   * Fetches object metadata based on the found image.
   * @param {cv.Mat} imgMat - The detected image.
   * @returns {Promise<void>}
   */
  async onImageDetect(imgMat) {
    const now = Date.now();
    if (now - this.lastApiCall <= this.API_COOLDOWN_MILLIS) {
      imgMat.delete();
      return;
    }

    this.detected = true;
    this.lastApiCall = now;

    try {
      cv.imshow("canvasDetect", imgMat);

      const dataUrl = canvasDetect.toDataURL("image/jpeg", 0.8);

      statusEl.innerText = "Checking match...";
      const rawBase64 = dataUrl.split(",")[1];
      const matches = await fetchMatchesByImage(rawBase64);

      if (matches.data.length === 0) {
        // drop the rejected object, it would otherwise flash on the next detection
        canvasDetect
          .getContext("2d")
          .clearRect(0, 0, canvasDetect.width, canvasDetect.height);
        this.detected = false;
        return;
      }

      const match = matches.data[0];

      showDetectOverlay();

      statusEl.innerText =
        "Detected: " +
        match.name +
        " | " +
        match.set.name +
        " |" +
        match.set.code;
    } catch (err) {
      console.log("fetch err", err);

      statusEl.innerText = "api err: " + err;
      this.detected = false;
    } finally {
      imgMat.delete();
    }
  },
};

window.onload = async () => {
  if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
    statusEl.innerText = "Error: Camera API not available.";
    return;
  }

  try {
    // create new stream to force an ask for permission
    await videoState.newStream();
  } catch (err) {
    statusEl.innerText = "Error: Camera API not available. " + err;
    return;
  } finally {
    // unblock camera usage
    videoState.stopStream();
  }

  if (typeof cv !== "undefined" && cv.getBuildInformation) {
    console.log("already loaded", cv.getBuildInformation());
    fillCameraList();
  } else if (cv instanceof Promise) {
    cv = await cv;
    console.log("load via promise", cv.getBuildInformation());
    fillCameraList();
  } else {
    cv["onRuntimeInitialized"] = () => {
      console.log("load via runtime", cv.getBuildInformation());
      fillCameraList();
    };
  }
};

function stopStream() {
  videoState.stopStream();
  videoEl.pause();
}

// stop stream on page leave
window.onbeforeunload = stopStream;

// stop camera on htmx swap
document.addEventListener("htmx:beforeSwap", stopStream);

async function fillCameraList() {
  statusEl.innerText = txtSelectCamera;
  const devices = await navigator.mediaDevices.enumerateDevices();
  console.log("found devices", devices);
  const vDevices = devices.filter((dev) => dev.kind === "videoinput");
  if (vDevices.length === 0) {
    return;
  }

  const optFn = (label, value) => {
    const option = document.createElement("option");
    option.value = value;
    option.appendChild(document.createTextNode(label));
    return option;
  };

  cameraEl.innerHTML = "";
  // an empty value keeps the "no camera selected"
  cameraEl.appendChild(optFn("Select camera", ""));
  vDevices.forEach((dev, index) => {
    const label = dev.label || `Camera ${index}`;

    cameraEl.appendChild(optFn(label, dev.deviceId));
  });
}

function processVideo() {
  if (videoEl.paused || videoEl.ended) {
    return;
  }

  if (videoState.isProcessing()) {
    // skip frame if still processing
    requestAnimationFrame(() => {
      processVideo();
    });
    return;
  }

  videoState.processing = true;
  try {
    if (
      videoEl.videoWidth !== videoEl.width ||
      videoEl.videoHeight !== videoEl.height
    ) {
      console.log(
        `Resolution changed: ${videoEl.videoWidth}x${videoEl.videoHeight}`,
      );

      videoEl.width = videoEl.videoWidth;
      videoEl.height = videoEl.videoHeight;

      // delete and recreate mats with new resolution
      videoState.renew(
        new cv.VideoCapture(videoEl),
        new cv.Mat(videoEl.videoHeight, videoEl.videoWidth, cv.CV_8UC4),
        new cv.Mat(videoEl.videoHeight, videoEl.videoWidth, cv.CV_8UC1),
      );
    }

    videoState.readFrame();

    if (videoState.hasNoImage()) {
      console.log("Error: Camera has no image.");
      requestAnimationFrame(() => {
        processVideo();
      });
      return;
    }

    let detectedObject = detectObjectFromImage(videoState.src);
    if (detectedObject) {
      videoState.onImageDetect(detectedObject);
    }

    cv.cvtColor(videoState.src, videoState.dest, cv.COLOR_RGBA2GRAY);
    cv.imshow("canvasOutput", videoState.dest);
  } catch (err) {
    console.log("Error: Camera video error. ", err);
  } finally {
    videoState.processing = false;
  }

  requestAnimationFrame(() => {
    processVideo();
  });
}

/**
 * @param {cv.Mat} src - The raw camera frame
 * @returns {cv.Mat|null} - The warped object, or null
 */
function detectObjectFromImage(src) {
  let gray = new cv.Mat();
  let blurred = new cv.Mat();
  let edged = new cv.Mat();
  let contours = new cv.MatVector();
  let hierarchy = new cv.Mat();

  let warpedObject = null;

  try {
    findObjectEdges(src, gray, blurred, edged);

    cv.findContours(
      edged,
      contours,
      hierarchy,
      cv.RETR_EXTERNAL,
      cv.CHAIN_APPROX_SIMPLE,
    );

    const candidates = findObjectCandidates(contours, src.cols, src.rows);
    const bestApprox = selectBestCandidate(candidates, src.cols, src.rows);

    if (bestApprox) {
      warpedObject = transformPerspective(src, bestApprox);
      bestApprox.delete();

      // reject blank/featureless candidates (e.g. compression noise on an out-of-focus frame)
      if (warpedObject && !hasEnoughDetail(warpedObject)) {
        warpedObject.delete();
        warpedObject = null;
      }
    }
  } catch (err) {
    statusEl.innerText = "Error: object detection failed. " + err;
  } finally {
    gray.delete();
    blurred.delete();
    edged.delete();
    contours.delete();
    hierarchy.delete();
  }

  return warpedObject;
}

/**
 * Writes an edge map into the given (caller-owned) gray/blurred/edged Mats.
 * @param {cv.Mat} src
 * @param {cv.Mat} gray
 * @param {cv.Mat} blurred
 * @param {cv.Mat} edged
 * @returns {void}
 */
function findObjectEdges(src, gray, blurred, edged) {
  // wide blur suppresses background texture noise while keeping the object's own edges
  cv.cvtColor(src, gray, cv.COLOR_RGBA2GRAY);
  cv.GaussianBlur(gray, blurred, new cv.Size(9, 9), 0);

  // adaptive ("auto Canny") thresholds handle both bright and dark/textured backgrounds
  const median = medianGray(blurred);
  const sigma = 0.33;
  const lower = Math.max(0, (1 - sigma) * median);
  const upper = Math.min(255, (1 + sigma) * median);
  cv.Canny(blurred, edged, lower, upper);

  // small kernel: a wider one merges background texture into one frame-sized blob
  let kernel = cv.getStructuringElement(cv.MORPH_RECT, new cv.Size(3, 3));
  cv.morphologyEx(edged, edged, cv.MORPH_CLOSE, kernel);
  kernel.delete();
}

/**
 * Contours with a plausible object-like aspect ratio. Caller owns each `approx` Mat.
 * @param {cv.MatVector} contours
 * @param {number} frameWidth
 * @param {number} frameHeight
 * @returns {Array<{approx: cv.Mat, area: number, center: {x: number, y: number}}>}
 */
function findObjectCandidates(contours, frameWidth, frameHeight) {
  // reject near-full-frame blobs (over-aggressive edge merging in findObjectEdges)
  const maxAllowedArea = frameWidth * frameHeight * 0.9;
  const aspectRatioTolerance = 0.15;

  let candidates = [];
  for (let i = 0; i < contours.size(); ++i) {
    let cnt = contours.get(i);
    let area = cv.contourArea(cnt);

    // filter by minimum size
    if (area > 10000 && area <= maxAllowedArea) {
      let peri = cv.arcLength(cnt, true);
      let approx = new cv.Mat();

      // simplify contour / count corners
      cv.approxPolyDP(cnt, approx, 0.02 * peri, true);

      if (
        approx.rows === 4 &&
        objectAspectRatioDiff(approx) <= aspectRatioTolerance
      ) {
        const rect = cv.minAreaRect(approx);
        candidates.push({ approx, area, center: rect.center });
      } else {
        approx.delete();
      }
    }
    cnt.delete();
  }

  return candidates;
}

/**
 * Picks the candidate closest to frame center (handles multiple objects in frame), then
 * the largest one at that spot (wins over a smaller inner sub-region like the text box).
 * Deletes every other candidate's `approx` Mat.
 * @param {Array<{approx: cv.Mat, area: number, center: {x: number, y: number}}>} candidates
 * @param {number} frameWidth
 * @param {number} frameHeight
 * @returns {cv.Mat|null}
 */
function selectBestCandidate(candidates, frameWidth, frameHeight) {
  let bestApprox = null;

  if (candidates.length > 0) {
    const frameCenter = { x: frameWidth / 2, y: frameHeight / 2 };
    let target = candidates[0];
    let bestCenterDist = Infinity;
    for (const c of candidates) {
      const d = dist(c.center, frameCenter);
      if (d < bestCenterDist) {
        bestCenterDist = d;
        target = c;
      }
    }

    const sameLocationRadius = Math.min(frameWidth, frameHeight) * 0.15;
    let bestArea = 0;
    for (const c of candidates) {
      if (
        dist(c.center, target.center) <= sameLocationRadius &&
        c.area > bestArea
      ) {
        bestArea = c.area;
        bestApprox = c.approx;
      }
    }
  }

  for (const c of candidates) {
    if (c.approx !== bestApprox) {
      c.approx.delete();
    }
  }

  return bestApprox;
}

/**
 * Median pixel value of a grayscale Mat, used for adaptive Canny thresholds.
 * @param {cv.Mat} mat - Grayscale image (CV_8UC1).
 * @returns {number} Median pixel value (0-255).
 */
function medianGray(mat) {
  const data = mat.data;
  const histogram = new Uint32Array(256);
  for (let i = 0; i < data.length; i++) {
    histogram[data[i]]++;
  }

  const half = data.length / 2;
  let cumulative = 0;
  for (let v = 0; v < 256; v++) {
    cumulative += histogram[v];
    if (cumulative >= half) {
      return v;
    }
  }

  return 255;
}

/**
 * Euclidean distance between two points.
 * @param {{x: number, y: number}} a
 * @param {{x: number, y: number}} b
 * @returns {number}
 */
function dist(a, b) {
  const dx = a.x - b.x;
  const dy = a.y - b.y;

  return Math.sqrt(dx * dx + dy * dy);
}

/**
 * How far a detected quad's aspect ratio is from an object's (~0.71, portrait).
 * @param {cv.Mat} approx - 4-point contour approximation (CV_32SC2).
 * @returns {number} Difference from the target ratio, or Infinity if degenerate.
 */
function objectAspectRatioDiff(approx) {
  const rect = cv.minAreaRect(approx);
  const w = rect.size.width;
  const h = rect.size.height;
  if (w === 0 || h === 0) {
    return Infinity;
  }

  const ratio = Math.min(w, h) / Math.max(w, h);
  const objectRatio = OBJECT_WARP_WIDTH / OBJECT_WARP_HEIGHT;

  return Math.abs(ratio - objectRatio);
}

/**
 * Whether a candidate has enough pixel variation to plausibly be a real object.
 * @param {cv.Mat} mat - Warped candidate object image.
 * @returns {boolean}
 */
function hasEnoughDetail(mat) {
  let gray = new cv.Mat();
  let mean = new cv.Mat();
  let stddev = new cv.Mat();

  try {
    cv.cvtColor(mat, gray, cv.COLOR_RGBA2GRAY);
    cv.meanStdDev(gray, mean, stddev);

    const minStdDev = 20; // blank frames measure near 0
    return stddev.data64F[0] >= minStdDev;
  } finally {
    gray.delete();
    mean.delete();
    stddev.delete();
  }
}

/**
 * Sorts the corners and warps the object into a straightened, fixed-size image.
 */
function transformPerspective(sourceMat, contour) {
  const targetWidth = OBJECT_WARP_WIDTH;
  const targetHeight = OBJECT_WARP_HEIGHT;

  // 1. Extract points from the OpenCV structure into a JS array
  let pts = [];
  for (let i = 0; i < 4; i++) {
    pts.push({
      x: contour.data32S[i * 2],
      y: contour.data32S[i * 2 + 1],
    });
  }

  // 1b. the quad usually only captures the inner frame, not the object's own border
  // (often too low-contrast to detect) - expand outward to approximate the true edge
  const borderExpansion = 0.06;
  const cx = pts.reduce((sum, p) => sum + p.x, 0) / pts.length;
  const cy = pts.reduce((sum, p) => sum + p.y, 0) / pts.length;
  pts = pts.map((p) => ({
    x: Math.min(
      Math.max(cx + (p.x - cx) * (1 + borderExpansion), 0),
      sourceMat.cols - 1,
    ),
    y: Math.min(
      Math.max(cy + (p.y - cy) * (1 + borderExpansion), 0),
      sourceMat.rows - 1,
    ),
  }));

  // 2. Sort corners mathematically (sum & difference)
  let sums = pts.map((p) => p.x + p.y);
  let diffs = pts.map((p) => p.y - p.x);

  let tl = pts[sums.indexOf(Math.min(...sums))]; // top-left (smallest sum)
  let tr = pts[diffs.indexOf(Math.min(...diffs))]; // top-right (smallest difference)
  let br = pts[sums.indexOf(Math.max(...sums))]; // bottom-right (largest sum)
  let bl = pts[diffs.indexOf(Math.max(...diffs))]; // bottom-left (largest difference)

  // 3. Build coordinate matrices
  let srcCoords = cv.matFromArray(4, 1, cv.CV_32FC2, [
    tl.x,
    tl.y,
    tr.x,
    tr.y,
    br.x,
    br.y,
    bl.x,
    bl.y,
  ]);

  let dstCoords = cv.matFromArray(4, 1, cv.CV_32FC2, [
    0,
    0,
    targetWidth,
    0,
    targetWidth,
    targetHeight,
    0,
    targetHeight,
  ]);

  // 4. Compute the matrix and warp the image
  let M = cv.getPerspectiveTransform(srcCoords, dstCoords);
  let result = new cv.Mat();
  cv.warpPerspective(
    sourceMat,
    result,
    M,
    new cv.Size(targetWidth, targetHeight),
  );

  srcCoords.delete();
  dstCoords.delete();
  M.delete();

  return result;
}

async function fetchMatchesByImage(base64Image) {
  const response = await fetch("/detect", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ image: base64Image }),
  });

  if (!response.ok) {
    throw new Error("API Error: " + response.status);
  }

  return await response.json();
}
