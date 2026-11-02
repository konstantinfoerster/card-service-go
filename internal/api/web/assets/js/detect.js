const statusEl = document.getElementById("status");
const videoEl = document.getElementById("videoInput");
const overlayEl = document.getElementById("overlay");
const detectResultEl = document.querySelector("detect-result");

// encodes the detected card as JPEG for the api call
const uploadCanvas = document.createElement("canvas");

// size of the extracted card image
const CARD_WIDTH = 350;
const CARD_HEIGHT = 490;
const CARD_RATIO = CARD_WIDTH / CARD_HEIGHT;

const DETECT_TIMEOUT_MILLIS = 5000;
// pause after an api call starts
const API_COOLDOWN_MILLIS = 500;
// pause after the dialog closes
const RESUME_DELAY_MILLIS = 1000;
// min time between two detections
const DETECT_INTERVAL_MILLIS = 100;

const CONTENT_TYPE_JPEG = "image/jpeg";

const TXT_SELECT_CAMERA = "Please select a camera ...";

/**
 * OpenCV.js namespace, set once ready.
 * @type {any}
 */
let cv;

/**
 * Shows the parts in the status line, errors as their message.
 * @param {...(string|Error)} txt
 */
function updateStatusline(...txt) {
  statusEl.textContent = txt
    .map((part) => {
      if (part instanceof Error) {
        return part.message;
      }

      return part;
    })
    .join(" ");
}

const cameraEl = document.getElementById("camera");
cameraEl.addEventListener("change", async function (e) {
  const deviceId = e.target.value;

  if (!deviceId) {
    stopStream();
    updateStatusline(TXT_SELECT_CAMERA);

    return;
  }

  updateStatusline("Starting stream...");
  try {
    videoEl.srcObject = await camera.start(deviceId);
    // waits until the video dimensions are known
    await videoEl.play();

    scanner.start();
  } catch (err) {
    updateStatusline("Error: Failed to start stream.", err);
  }
});

class DetectResultComponent extends HTMLElement {
  /** @type {Array<{id: string}>} */
  #matches = [];
  /** @type {number} Index of the shown match. */
  #index = 0;
  #dialogEl;
  #cardEl;
  #prevEl;
  #nextEl;
  /** @type {AbortController|null} Removes the listeners on disconnect. */
  #abort = null;

  connectedCallback() {
    this.#dialogEl = this.#find("dialog");
    this.#cardEl = this.#find("[data-card]");
    this.#prevEl = this.#find(".match-prev");
    this.#nextEl = this.#find(".match-next");

    this.#abort = new AbortController();
    const { signal } = this.#abort;

    this.#dialogEl.addEventListener(
      "click",
      (e) => {
        if (e.target.closest(".match-prev")) {
          this.#show(this.#index - 1);
        } else if (e.target.closest(".match-next")) {
          this.#show(this.#index + 1);
        } else if (e.target === this.#dialogEl || e.target.closest(".close")) {
          // a backdrop click targets the dialog itself
          this.#dialogEl.close();
        }
      },
      { signal },
    );

    // card added with quick add
    this.#dialogEl.addEventListener(
      "htmx:afterRequest",
      (e) => {
        if (e.detail.successful && e.target.closest(".quick-add")) {
          this.#dialogEl.close();
        }
      },
      { signal },
    );

    this.#dialogEl.addEventListener(
      "close",
      () => {
        this.#reset();
        this.dispatchEvent(new Event("close"));
      },
      { signal },
    );
  }

  disconnectedCallback() {
    this.#abort?.abort();
  }

  /** @returns {boolean} true while the dialog is open. */
  get isOpen() {
    return this.#dialogEl.open;
  }

  /**
   * Opens the dialog with the matches, ignores an empty list.
   * @param {Array<{id: string}>} matches
   */
  showMatches(matches) {
    if (matches.length === 0) {
      return;
    }

    this.#matches = matches;
    this.#show(0);

    this.#dialogEl.showModal();
  }

  /**
   * @param {string} selector
   * @returns {HTMLElement} First matching element, throws if none.
   */
  #find(selector) {
    const el = this.querySelector(selector);
    if (!el) {
      throw new Error(
        `<detect-result> needs an element matching "${selector}"`,
      );
    }

    return el;
  }

  /**
   * Loads the card of the match at index into the dialog, never rejects.
   * @param {number} index
   */
  async #show(index) {
    if (index < 0 || index >= this.#matches.length) {
      return;
    }

    this.#index = index;

    const single = this.#matches.length <= 1;
    this.#prevEl.classList.toggle("visually-hidden", single);
    this.#nextEl.classList.toggle("visually-hidden", single);

    this.#prevEl.disabled = index < 1;
    this.#nextEl.disabled = index >= this.#matches.length - 1;

    const nav = { quickAdd: true };
    // htmx inherits hx-vals, so links inside the card keep quick add
    this.#cardEl.setAttribute("hx-vals", JSON.stringify(nav));

    // clear the previous card in case loading fails
    this.#cardEl.replaceChildren();

    const id = this.#matches[index].id;
    try {
      await htmx.ajax("GET", `/cards/${id}`, {
        target: this.#cardEl,
        swap: "innerHTML",
        values: nav,
      });
    } catch (err) {
      console.error("Failed to load card", err);
    }

    // htmx doesn't swap error responses
    if (!this.#cardEl.hasChildNodes()) {
      this.#cardEl.textContent = "Could not load the card.";
    }
  }

  /** Resets the state and clears the card. */
  #reset() {
    this.#cardEl.replaceChildren();

    this.#matches = [];
    this.#index = 0;
  }
}

customElements.define("detect-result", DetectResultComponent);

detectResultEl.addEventListener("close", () => scanner.resume());

class Camera {
  /** @type {MediaStream|null} */
  #stream = null;

  /** Asks for the camera permission, without it the camera labels are empty. */
  async requestPermission() {
    try {
      this.#stream = await navigator.mediaDevices.getUserMedia({ video: true });
    } finally {
      this.stop();
    }
  }

  /**
   * Stops the current stream and opens the given camera.
   * @param {string} deviceId
   * @returns {Promise<MediaStream>}
   */
  async start(deviceId) {
    this.stop();

    this.#stream = await navigator.mediaDevices.getUserMedia({
      video: {
        // no fallback to another camera
        deviceId: { exact: deviceId },
        width: { ideal: 1280 },
        height: { ideal: 720 },
      },
    });

    return this.#stream;
  }

  stop() {
    this.#stream?.getTracks().forEach((track) => track.stop());
    this.#stream = null;
  }
}

const camera = new Camera();

/** Reads the video frames into an OpenCV Mat. */
class FrameReader {
  /** @type {HTMLVideoElement} */
  #video;
  /** @type {cv.VideoCapture|null} */
  #vidCap = null;
  /** @type {cv.Mat|null} RGBA */
  #src = null;

  /** @param {HTMLVideoElement} video */
  constructor(video) {
    this.#video = video;
  }

  /** @returns {cv.Mat|null} Last read frame (RGBA). */
  get frame() {
    return this.#src;
  }

  read() {
    if (this.#needsSync()) {
      this.#sync();
    }

    this.#vidCap.read(this.#src);
  }

  /** @returns {boolean} true on the first read or a resolution change. */
  #needsSync() {
    return (
      !this.#src ||
      this.#video.videoWidth !== this.#video.width ||
      this.#video.videoHeight !== this.#video.height
    );
  }

  /** Resizes the video element and the frame Mat to the stream resolution. */
  #sync() {
    const { videoWidth: w, videoHeight: h } = this.#video;
    this.#video.width = w;
    this.#video.height = h;

    const vidCap = new cv.VideoCapture(this.#video);
    const src = new cv.Mat(h, w, cv.CV_8UC4);

    this.#src?.delete();

    this.#vidCap = vidCap;
    this.#src = src;
  }
}

/** Draws the guide box and the card outline over the video. */
class Outline {
  /** @type {HTMLCanvasElement} */
  #canvas;
  /** @type {CanvasRenderingContext2D} */
  #ctx;
  /** @type {HTMLVideoElement} */
  #video;

  /**
   * @param {HTMLCanvasElement} canvas
   * @param {HTMLVideoElement} video
   */
  constructor(canvas, video) {
    this.#canvas = canvas;
    this.#ctx = canvas.getContext("2d");
    this.#video = video;
  }

  /** @param {Array<{x: number, y: number}>} corners - In pixels of the video frame. */
  draw(corners) {
    this.showGuide();

    const [first, ...rest] = corners;
    this.#ctx.beginPath();
    this.#ctx.moveTo(first.x, first.y);
    rest.forEach(({ x, y }) => this.#ctx.lineTo(x, y));
    this.#ctx.closePath();
    this.#ctx.strokeStyle = "#22c55e";
    this.#ctx.stroke();
  }

  /** Shows only the guide box. */
  showGuide() {
    // canvas has the frame's pixel size, css scales it with the video
    const { videoWidth: w, videoHeight: h } = this.#video;
    if (this.#canvas.width !== w || this.#canvas.height !== h) {
      this.#canvas.width = w;
      this.#canvas.height = h;
    }

    this.clear();

    const box = guideBox(w, h);
    this.#ctx.lineWidth = Math.max(2, w / 150);
    this.#ctx.strokeStyle = "#ffffff";
    this.#ctx.setLineDash([this.#ctx.lineWidth * 4, this.#ctx.lineWidth * 2]);
    this.#ctx.strokeRect(box.x, box.y, box.width, box.height);
    this.#ctx.setLineDash([]);
  }

  /** Clears the canvas. */
  clear() {
    this.#ctx.clearRect(0, 0, this.#canvas.width, this.#canvas.height);
  }
}

/** Runs the frame loop, detects cards and identifies them with the api. */
class Scanner {
  /** @type {HTMLVideoElement} */
  #video;
  /** @type {FrameReader} */
  #reader;
  /** @type {Outline} */
  #outline;
  /** @type {boolean} false while a request runs or the dialog is open. */
  #scanning = true;
  /** @type {number} No detection before this time. */
  #resumeAt = 0;
  /** @type {number} Time of the last detection. */
  #lastDetection = 0;
  /** @type {number|null} Pending video frame callback. */
  #frameRequest = null;

  /**
   * @param {HTMLVideoElement} video
   * @param {FrameReader} reader
   * @param {Outline} outline
   */
  constructor(video, reader, outline) {
    this.#video = video;
    this.#reader = reader;
    this.#outline = outline;
  }

  /** Starts the frame loop, replaces a running one. Call after the video plays. */
  start() {
    this.stop();
    this.#processVideo();
  }

  /** Stops the frame loop and clears the overlay. */
  stop() {
    this.#outline.clear();

    if (this.#frameRequest === null) {
      return;
    }

    this.#video.cancelVideoFrameCallback(this.#frameRequest);
    this.#frameRequest = null;
  }

  /** Continues scanning after a delay, so the same card isn't detected again. */
  resume() {
    this.#resumeAt = performance.now() + RESUME_DELAY_MILLIS;
    this.#scanning = true;
    this.#outline.showGuide();
    updateStatusline("");
  }

  #scheduleNextFrame() {
    this.#frameRequest = this.#video.requestVideoFrameCallback(
      this.#processVideo,
    );
  }

  #processVideo = () => {
    if (this.#video.paused || this.#video.ended) {
      return;
    }

    if (this.#scanning) {
      try {
        this.#scanFrame();
      } catch (err) {
        console.error("Card detection failed:", err);
        updateStatusline("Error: card detection failed.", err);
      }
    }

    this.#scheduleNextFrame();
  };

  #scanFrame() {
    // skip while paused and throttle the detection
    const now = performance.now();
    if (
      now < this.#resumeAt ||
      now - this.#lastDetection < DETECT_INTERVAL_MILLIS
    ) {
      return;
    }
    this.#lastDetection = now;

    this.#reader.read();
    const detection = detectCard(this.#reader.frame);
    if (!detection) {
      this.#outline.showGuide();
      return;
    }

    this.#outline.draw(detection.corners);
    this.#identify(detection.image);
  }

  /**
   * Identifies the card with the api and shows the matches.
   * @param {cv.Mat} imgMat - Card image, deleted here.
   */
  async #identify(imgMat) {
    this.#scanning = false;
    this.#resumeAt = performance.now() + API_COOLDOWN_MILLIS;
    // clear the previous error
    updateStatusline("");

    try {
      cv.imshow(uploadCanvas, imgMat);
      const image = await new Promise((resolve) =>
        uploadCanvas.toBlob(resolve, CONTENT_TYPE_JPEG, 0.8),
      );

      const matches = await fetchMatchesByImage(image);
      detectResultEl.showMatches(matches.data);
    } catch (err) {
      console.error("detect api error", err);

      if (err.name === "TimeoutError") {
        updateStatusline("Error: api call timed out.");
      } else {
        updateStatusline("Error: api call", err);
      }
    } finally {
      imgMat.delete();

      // an open dialog resumes scanning when closed
      this.#scanning = !detectResultEl.isOpen;
    }
  }
}

const scanner = new Scanner(
  videoEl,
  new FrameReader(videoEl),
  new Outline(overlayEl, videoEl),
);

/**
 * Waits until OpenCV.js is ready.
 * @returns {Promise<{cv: any}>}
 */
async function waitForOpenCV() {
  const instance = globalThis.cv;
  if (!instance) {
    throw new Error("OpenCV.js is not loaded");
  }

  // wrapped, cv has a then() method
  if (instance.calledRun && instance.Mat && instance.getBuildInformation) {
    return { cv: instance };
  }

  return new Promise((resolve) => {
    instance.onRuntimeInitialized = () => resolve({ cv: instance });
  });
}

// after load, so the page is visible before the camera prompt
window.addEventListener("load", async () => {
  try {
    updateStatusline("Checking camera API permissions...");
    await camera.requestPermission();
  } catch (err) {
    updateStatusline("Error: Camera API not available.", err);
    return;
  }

  try {
    const result = await waitForOpenCV();
    cv = result.cv;
  } catch (err) {
    updateStatusline("Error: OpenCV not available.", err);
    return;
  }

  updateStatusline(TXT_SELECT_CAMERA);
  fillCameraList();

  // camera plugged in or removed
  navigator.mediaDevices.addEventListener("devicechange", fillCameraList);
});

function stopStream() {
  camera.stop();
  scanner.stop();
  videoEl.pause();
}

// pagehide also fires on mobile and for the back/forward cache
window.addEventListener("pagehide", stopStream);

// stop on htmx navigation, but not for swaps inside the detect result
document.addEventListener("htmx:beforeSwap", (e) => {
  if (detectResultEl.contains(e.detail.target)) {
    return;
  }

  stopStream();
});

/** Fills the camera select, keeps the selection if that camera still exists. */
async function fillCameraList() {
  const selected = cameraEl.value;

  const devices = await navigator.mediaDevices.enumerateDevices();
  const vDevices = devices.filter((dev) => dev.kind === "videoinput");

  // empty value -> no camera selected
  const placeholder = new Option("Select camera", "");
  const cameraOptions = vDevices.map((dev, index) => {
    const label = dev.label || `Camera ${index + 1}`;

    return new Option(label, dev.deviceId);
  });

  cameraEl.replaceChildren(placeholder, ...cameraOptions);

  if (!selected) {
    return;
  }

  if (vDevices.some((dev) => dev.deviceId === selected)) {
    cameraEl.value = selected;
  } else {
    // the selected camera was removed
    stopStream();
    updateStatusline(TXT_SELECT_CAMERA);
  }
}

/** Deletes the tracked OpenCV objects together. */
class MatScope {
  /** @type {Array<{delete: () => void}>} */
  #items = [];

  /**
   * @template {{delete: () => void}} T
   * @param {T} item
   * @returns {T}
   */
  track(item) {
    this.#items.push(item);
    return item;
  }

  dispose() {
    for (const item of this.#items) {
      item.delete();
    }
    this.#items.length = 0;
  }
}

/**
 * Finds a card in the frame.
 * @param {cv.Mat} src - Camera frame (RGBA).
 * @returns {{image: cv.Mat, corners: Array<{x: number, y: number}>}|null} The card image and
 *   its corners in frame pixels. The caller must delete the image.
 */
function detectCard(src) {
  const scope = new MatScope();

  try {
    // search only around the guide box, so neighbouring cards can't join the outline
    const area = searchArea(src.cols, src.rows);
    const searched = scope.track(src.roi(area));

    // edge detection on a smaller gray copy is much faster
    const maxSize = 800;
    const scale = Math.min(1, maxSize / Math.max(searched.cols, searched.rows));
    const fullGray = scope.track(new cv.Mat());
    cv.cvtColor(searched, fullGray, cv.COLOR_RGBA2GRAY);

    let gray = fullGray;
    if (scale < 1) {
      gray = scope.track(new cv.Mat());
      const size = new cv.Size(
        Math.round(searched.cols * scale),
        Math.round(searched.rows * scale),
      );
      cv.resize(fullGray, gray, size, 0, 0, cv.INTER_AREA);
    }

    const edged = findEdges(gray, scope);
    const contours = scope.track(new cv.MatVector());
    const hierarchy = scope.track(new cv.Mat());
    cv.findContours(
      edged,
      contours,
      hierarchy,
      cv.RETR_EXTERNAL,
      cv.CHAIN_APPROX_SIMPLE,
    );

    const bestQuad = findLargestQuad(
      contours,
      gray.cols,
      gray.rows,
      scale,
      scope,
    );
    if (!bestQuad) {
      return null;
    }

    const card = toWholeCard(orderCorners(bestQuad), gray.cols, gray.rows);
    if (!card) {
      return null;
    }

    // gray pixels to frame pixels, the extraction uses the full resolution
    const toFrame = (p) => ({
      x: p.x / scale + area.x,
      y: p.y / scale + area.y,
    });
    const corners = expandCorners(card.map(toFrame), src.cols, src.rows);
    if (!fillsGuideBox(corners, src.cols, src.rows)) {
      return null;
    }

    const image = extractCard(src, corners, scope);
    // reject blank surfaces
    if (!hasEnoughDetail(image, scope)) {
      image.delete();
      return null;
    }

    return { image, corners };
  } finally {
    scope.dispose();
  }
}

/**
 * Edge map of the gray image.
 * @param {cv.Mat} gray
 * @param {MatScope} scope - Owns the created Mats, including the result.
 * @returns {cv.Mat}
 */
function findEdges(gray, scope) {
  // reduces noise and fine texture
  const blurred = scope.track(new cv.Mat());
  cv.GaussianBlur(gray, blurred, new cv.Size(5, 5), 0);

  // thresholds follow the image brightness (auto Canny)
  const median = medianGray(blurred);
  const thresholdSpread = 0.33;
  const lower = Math.max(0, (1 - thresholdSpread) * median);
  const upper = Math.min(255, (1 + thresholdSpread) * median);
  const edged = scope.track(new cv.Mat());
  cv.Canny(blurred, edged, lower, upper);

  // closes small gaps in the edges, a bigger kernel would merge background texture
  const kernel = scope.track(
    cv.getStructuringElement(cv.MORPH_RECT, new cv.Size(3, 3)),
  );
  cv.morphologyEx(edged, edged, cv.MORPH_CLOSE, kernel);

  return edged;
}

/**
 * Largest contour that simplifies to a quad with the card's aspect ratio, so inner regions
 * like the text box lose.
 * @param {cv.MatVector} contours
 * @param {number} imageWidth
 * @param {number} imageHeight
 * @param {number} scale - Size relative to the full frame.
 * @param {MatScope} scope - Owns the created Mats.
 * @returns {Array<{x: number, y: number}>|null} The quad's corners in contour pixels,
 *   unordered, or null if no contour fits.
 */
function findLargestQuad(contours, imageWidth, imageHeight, scale, scope) {
  // in pixels of the full frame
  const minCardArea = 10000;
  // allowed difference to CARD_RATIO
  const aspectRatioTolerance = 0.15;
  // fraction of a contour that must lie on its quad
  const minEdgeCoverage = 0.8;

  // smaller contours can't win, so they skip the checks
  let bestArea = minCardArea * scale * scale;
  // rejects blobs of merged background edges
  const maxAllowedArea = imageWidth * imageHeight * 0.9;

  let best = null;
  for (let i = 0; i < contours.size(); ++i) {
    // the hull closes borders with gaps
    const cnt = scope.track(contours.get(i));
    const hull = scope.track(new cv.Mat());
    cv.convexHull(cnt, hull);
    const area = cv.contourArea(hull);
    if (area <= bestArea || area > maxAllowedArea) {
      continue;
    }

    const epsilon = 0.02 * cv.arcLength(hull, true);
    const approx = scope.track(new cv.Mat());
    cv.approxPolyDP(hull, approx, epsilon, true);
    if (
      approx.rows !== 4 ||
      cardAspectRatioDiff(cv.minAreaRect(approx)) > aspectRatioTolerance ||
      edgeCoverage(cnt, approx, epsilon) < minEdgeCoverage
    ) {
      continue;
    }

    bestArea = area;
    best = matToPoints(approx);
  }

  return best;
}

/**
 * @param {cv.Mat} mat - CV_32SC2
 * @returns {Array<{x: number, y: number}>}
 */
function matToPoints(mat) {
  const d = mat.data32S;
  const pts = [];
  for (let i = 0; i < d.length; i += 2) {
    pts.push({ x: d[i], y: d[i + 1] });
  }

  return pts;
}

/**
 * Fraction of the contour's length on the quad's edges, low for clutter joined to the card.
 * @param {cv.Mat} contour - CV_32SC2
 * @param {cv.Mat} quad - CV_32SC2
 * @param {number} maxDist - Max distance of a point on an edge.
 * @returns {number} 0 to 1.
 */
function edgeCoverage(contour, quad, maxDist) {
  const pts = matToPoints(contour);
  const onEdge = pts.map(
    (p) => Math.abs(cv.pointPolygonTest(quad, p, true)) <= maxDist,
  );

  let total = 0;
  let along = 0;
  for (let i = 0; i < pts.length; i++) {
    // the last segment closes the contour
    const next = (i + 1) % pts.length;
    const len = dist(pts[i], pts[next]);
    total += len;
    if (onEdge[i] && onEdge[next]) {
      along += len;
    }
  }

  if (total === 0) {
    return 0;
  }

  return along / total;
}

/**
 * Of an aftermath card often only the landscape bottom half is found. Extends such a half to
 * the whole card.
 * @param {Array<{x: number, y: number}>} corners - top-left, top-right, bottom-right, bottom-left
 * @param {number} imageWidth
 * @param {number} imageHeight
 * @returns {Array<{x: number, y: number}>|null} The given corners for a portrait quad, the whole
 *   card for an aftermath half, null for other landscape quads or a top half out of the image.
 */
function toWholeCard(corners, imageWidth, imageHeight) {
  const [tl, tr, br, bl] = corners;
  const width = dist(tl, tr) + dist(bl, br);
  const height = dist(tl, bl) + dist(tr, br);
  if (width <= height) {
    return corners;
  }

  // aftermath card in units of its bottom half
  const above = 1.5;
  const below = 0.22;
  const side = 0.08;
  // height to width of the bottom half, about 0.6
  const halfRatio = (1 + 2 * side) / ((above + 1 + below) * CARD_RATIO);
  const halfTolerance = 0.06;
  if (Math.abs(height / width - halfRatio) > halfTolerance) {
    return null;
  }

  const lerp = (a, b, t) => ({
    x: a.x + (b.x - a.x) * t,
    y: a.y + (b.y - a.y) * t,
  });
  // point relative to the half, u along the top edge, v along the left edge
  const at = (u, v) => lerp(lerp(tl, tr, u), lerp(bl, br, u), v);

  const inImage = (p) =>
    p.x >= 0 && p.y >= 0 && p.x < imageWidth && p.y < imageHeight;
  if (!inImage(at(0, -above)) || !inImage(at(1, -above))) {
    return null;
  }

  return [
    at(-side, -above),
    at(1 + side, -above),
    at(1 + side, 1 + below),
    at(-side, 1 + below),
  ];
}

/**
 * @param {cv.Mat} mat - CV_8UC1
 * @returns {number} Median pixel value.
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
 * @param {{x: number, y: number}} a
 * @param {{x: number, y: number}} b
 * @returns {number}
 */
function dist(a, b) {
  return Math.hypot(a.x - b.x, a.y - b.y);
}

/**
 * @param {{size: {width: number, height: number}}} rect - minAreaRect of the quad.
 * @returns {number} Difference to CARD_RATIO, Infinity for an empty rect.
 */
function cardAspectRatioDiff({ size: { width, height } }) {
  if (width === 0 || height === 0) {
    return Infinity;
  }

  const ratio = Math.min(width, height) / Math.max(width, height);

  return Math.abs(ratio - CARD_RATIO);
}

/**
 * Whether the image has enough contrast to be a card.
 * @param {cv.Mat} mat - RGBA
 * @param {MatScope} scope - Owns the created Mats.
 * @returns {boolean}
 */
function hasEnoughDetail(mat, scope) {
  const gray = scope.track(new cv.Mat());
  const mean = scope.track(new cv.Mat());
  const stddev = scope.track(new cv.Mat());

  cv.cvtColor(mat, gray, cv.COLOR_RGBA2GRAY);
  cv.meanStdDev(gray, mean, stddev);

  const minStdDev = 20; // blank surfaces measure near 0
  return stddev.data64F[0] >= minStdDev;
}

/**
 * @param {Array<{x: number, y: number}>} quad - 4 points in any order.
 * @returns {Array<{x: number, y: number}>} top-left, top-right, bottom-right, bottom-left
 */
function orderCorners(quad) {
  // sort corners by sum and difference of the coordinates
  const sums = quad.map((p) => p.x + p.y);
  const diffs = quad.map((p) => p.y - p.x);

  const tl = quad[sums.indexOf(Math.min(...sums))]; // top-left (smallest sum)
  const tr = quad[diffs.indexOf(Math.min(...diffs))]; // top-right (smallest difference)
  const br = quad[sums.indexOf(Math.max(...sums))]; // bottom-right (largest sum)
  const bl = quad[diffs.indexOf(Math.max(...diffs))]; // bottom-left (largest difference)

  return [tl, tr, br, bl];
}

/**
 * Grows the corners around their center, clamped to the frame.
 * @param {Array<{x: number, y: number}>} corners
 * @param {number} frameWidth
 * @param {number} frameHeight
 * @returns {Array<{x: number, y: number}>}
 */
function expandCorners(corners, frameWidth, frameHeight) {
  // the quad misses the card's outer border
  const grow = 1.06;
  const c = centerOf(corners);
  const clamp = (v, size) => Math.min(Math.max(v, 0), size - 1);

  return corners.map((p) => ({
    x: clamp(c.x + (p.x - c.x) * grow, frameWidth),
    y: clamp(c.y + (p.y - c.y) * grow, frameHeight),
  }));
}

/**
 * @param {Array<{x: number, y: number}>} corners
 * @returns {{x: number, y: number}}
 */
function centerOf(corners) {
  return {
    x: corners.reduce((sum, p) => sum + p.x, 0) / corners.length,
    y: corners.reduce((sum, p) => sum + p.y, 0) / corners.length,
  };
}

/**
 * Centered box with the card's aspect ratio.
 * @param {number} frameWidth
 * @param {number} frameHeight
 * @returns {{x: number, y: number, width: number, height: number}}
 */
function guideBox(frameWidth, frameHeight) {
  // relative to the largest card-shaped box that fits in the frame
  const size = 0.8;
  const height = Math.min(frameHeight, frameWidth / CARD_RATIO) * size;
  const width = height * CARD_RATIO;

  return {
    x: (frameWidth - width) / 2,
    y: (frameHeight - height) / 2,
    width,
    height,
  };
}

/**
 * Guide box grown by a margin on every side, clamped to the frame.
 * @param {number} frameWidth
 * @param {number} frameHeight
 * @returns {cv.Rect}
 */
function searchArea(frameWidth, frameHeight) {
  // fraction of the box size
  const margin = 0.1;
  const box = guideBox(frameWidth, frameHeight);
  const marginX = box.width * margin;
  const marginY = box.height * margin;

  const left = Math.max(0, Math.floor(box.x - marginX));
  const top = Math.max(0, Math.floor(box.y - marginY));
  const right = Math.min(frameWidth, Math.ceil(box.x + box.width + marginX));
  const bottom = Math.min(frameHeight, Math.ceil(box.y + box.height + marginY));

  return new cv.Rect(left, top, right - left, bottom - top);
}

/**
 * Whether the quad covers enough of the guide box.
 * @param {Array<{x: number, y: number}>} corners - In order around the quad.
 * @param {number} frameWidth
 * @param {number} frameHeight
 * @returns {boolean}
 */
function fillsGuideBox(corners, frameWidth, frameHeight) {
  const minFill = 0.3;
  const box = guideBox(frameWidth, frameHeight);

  // shoelace formula
  let area = 0;
  corners.forEach((p, i) => {
    const next = corners[(i + 1) % corners.length];
    area += p.x * next.y - next.x * p.y;
  });
  area = Math.abs(area) / 2;

  return area >= box.width * box.height * minFill;
}

/**
 * Cuts the card out of the frame, straightened to a CARD_WIDTH x CARD_HEIGHT image.
 * @param {cv.Mat} sourceMat
 * @param {Array<{x: number, y: number}>} corners - top-left, top-right, bottom-right, bottom-left
 * @param {MatScope} scope - Owns the created Mats, except the result.
 * @returns {cv.Mat} The caller must delete it.
 */
function extractCard(sourceMat, corners, scope) {
  const w = CARD_WIDTH;
  const h = CARD_HEIGHT;

  const srcCoords = scope.track(
    cv.matFromArray(
      4,
      1,
      cv.CV_32FC2,
      corners.flatMap((p) => [p.x, p.y]),
    ),
  );
  const dstCoords = scope.track(
    cv.matFromArray(4, 1, cv.CV_32FC2, [0, 0, w, 0, w, h, 0, h]),
  );

  const M = scope.track(cv.getPerspectiveTransform(srcCoords, dstCoords));
  const result = new cv.Mat();
  cv.warpPerspective(sourceMat, result, M, new cv.Size(w, h));

  return result;
}

/**
 * @param {Blob} image - JPEG
 * @returns {Promise<{data: Array<{id: string}>}>} The matches, best first.
 */
async function fetchMatchesByImage(image) {
  const response = await fetch("/detect", {
    method: "POST",
    headers: { "Content-Type": CONTENT_TYPE_JPEG },
    body: image,
    signal: AbortSignal.timeout(DETECT_TIMEOUT_MILLIS),
  });

  if (!response.ok) {
    throw new Error(`API error: ${response.status}`);
  }

  return response.json();
}
