const statusEl = document.getElementById("status");
const videoEl = document.getElementById("videoInput");
const overlayEl = document.getElementById("overlay");
const detectResultEl = document.querySelector("detect-result");

// the detected object is drawn here to encode it as image for the api call
const uploadCanvas = document.createElement("canvas");

// output size a detected object is warped to
const OBJECT_WARP_WIDTH = 350;
const OBJECT_WARP_HEIGHT = 490;
const OBJECT_RATIO = OBJECT_WARP_WIDTH / OBJECT_WARP_HEIGHT;

// detection runs on a copy of the frame whose longer side is at most this long,
// the warp uses the full frame
const DETECT_MAX_SIZE = 800;
// smallest object area in pixels, measured on the full frame
const MIN_OBJECT_AREA = 10000;
// allowed difference between a quad's aspect ratio and OBJECT_RATIO
const ASPECT_RATIO_TOLERANCE = 0.15;
// share of a contour that must run along its quad, rejects clutter joined to the object
const MIN_EDGE_COVERAGE = 0.8;
// the found quad usually misses the low-contrast outer border, so it is grown by this share
const BORDER_EXPANSION = 0.06;
// an aftermath card measured in sizes of its bottom half: the top half lies above it,
// a small border below and on the sides
const AFTERMATH_LAYOUT = Object.freeze({ above: 1.5, below: 0.22, side: 0.08 });
// height to width of the bottom half, about 0.6,
// the title and art of a normal card measure 0.73 and more
const AFTERMATH_HALF_RATIO =
  (1 + 2 * AFTERMATH_LAYOUT.side) /
  ((AFTERMATH_LAYOUT.above + 1 + AFTERMATH_LAYOUT.below) * OBJECT_RATIO);
// allowed difference between a landscape quad's ratio and AFTERMATH_HALF_RATIO
const AFTERMATH_HALF_TOLERANCE = 0.06;
// the guide box the card is held into, as share of the frame size
const GUIDE_SIZE = 0.8;
// share of the guide box area a detected object must cover
const MIN_GUIDE_FILL = 0.3;
// share of the guide box size the searched area reaches beyond the box on every side
const GUIDE_MARGIN = 0.1;

const DETECT_TIMEOUT_MILLIS = 5000;
const API_COOLDOWN_MILLIS = 500;
const RESUME_DELAY_MILLIS = 1000;
// time between two detection starts, the rest of it leaves the main thread free for the UI
const DETECT_INTERVAL_MILLIS = 100;

const CONTENT_TYPE_JPEG = "image/jpeg";

const txtSelectCamera = "Please select a camera ...";

/**
 * OpenCV.js namespace. Module-scoped (shadows the global `cv`), assigned in the load handler.
 * @type {any}
 */
let cv;

/**
 * Shows the given parts in the status line, errors as their message.
 * textContent doesn't force a layout like innerText does.
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
    updateStatusline(txtSelectCamera);

    return;
  }

  updateStatusline("Starting stream...");
  try {
    videoEl.srcObject = await camera.start(deviceId);
    // resolves once playback runs, so the video dimensions are known
    await videoEl.play();

    scanner.start();
  } catch (err) {
    updateStatusline("Error: Failed to start stream.", err);
  }
});

class DetectResultComponent extends HTMLElement {
  /** @type {Array<{id: string}>} Matches returned by the api. */
  #matches = [];
  /** @type {number} Index of the currently shown match. */
  #index = 0;
  #dialogEl;
  #cardEl;
  #prevEl;
  #nextEl;
  /** @type {AbortController|null} Removes the listeners when the element is disconnected. */
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
          // a click on the backdrop targets the dialog itself
          this.#dialogEl.close();
        }
      },
      { signal },
    );

    // the card was added
    this.#dialogEl.addEventListener(
      "htmx:afterRequest",
      (e) => {
        if (e.detail.successful && e.target.closest(".quick-add")) {
          this.#dialogEl.close();
        }
      },
      { signal },
    );

    // closed by backdrop, close button, quick add or escape key
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
   * Opens the overlay with the given matches, does nothing for an empty list.
   * @param {Array<{id: string}>} matches - The detected matches.
   * @returns {void}
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
   * @returns {HTMLElement} The first matching element inside, throws if there is none.
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
   * Loads the card detail of the match at the given index into the overlay.
   * @param {number} index - Index of the match to show.
   * @returns {Promise<void>} Resolves once the card is loaded or an error is shown, never rejects.
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
    // htmx inherits hx-vals, so print links inside the card detail keep the controls
    this.#cardEl.setAttribute("hx-vals", JSON.stringify(nav));

    // don't keep showing the previous card if loading this one fails
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

    // htmx doesn't swap error responses, so nothing was loaded
    if (!this.#cardEl.hasChildNodes()) {
      this.#cardEl.textContent = "Could not load the card.";
    }
  }

  /** Resets inner state and removes the inner overlay elements. */
  #reset() {
    this.#cardEl.replaceChildren();

    this.#matches = [];
    this.#index = 0;
  }
}

customElements.define("detect-result", DetectResultComponent);

// overlay closed, continue scanning
detectResultEl.addEventListener("close", () => scanner.resume());

class Camera {
  /** @type {MediaStream|null} The active camera stream. */
  #stream = null;

  /** Asks for the camera permission, the camera labels are empty without it. */
  async requestPermission() {
    try {
      this.#stream = await navigator.mediaDevices.getUserMedia({ video: true });
    } finally {
      this.stop();
    }
  }

  /**
   * Stops the current stream and opens a new one.
   * @param {string} deviceId - Camera to use.
   * @returns {Promise<MediaStream>}
   */
  async start(deviceId) {
    this.stop();

    this.#stream = await navigator.mediaDevices.getUserMedia({
      video: {
        // exact: don't silently fall back to another camera
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

// requestVideoFrameCallback fires once per new video frame instead of once per screen refresh
const HAS_VIDEO_FRAME_CALLBACK =
  "requestVideoFrameCallback" in HTMLVideoElement.prototype;

/** Owns the OpenCV buffers for the frames of the video element. */
class FrameBuffers {
  /** @type {HTMLVideoElement} */
  #video;
  /** @type {cv.VideoCapture|null} The video capture instance. */
  #vidCap = null;
  /** @type {cv.Mat|null} Source matrix (RGBA). */
  #src = null;

  /** @param {HTMLVideoElement} video */
  constructor(video) {
    this.#video = video;
  }

  /** @returns {cv.Mat|null} The last frame that was read (RGBA). */
  get frame() {
    return this.#src;
  }

  /**
   * Reads current video frame into the source matrix, the buffers follow the stream resolution.
   * @returns {boolean} true if the frame contains image data.
   */
  read() {
    if (this.#needsSync()) {
      this.#sync();
    }

    this.#vidCap.read(this.#src);

    return this.#hasImage();
  }

  /** @returns {boolean} true on the first read and when the stream resolution changed. */
  #needsSync() {
    return (
      !this.#src ||
      this.#video.videoWidth !== this.#video.width ||
      this.#video.videoHeight !== this.#video.height
    );
  }

  /**
   * Matches the video element and the buffers to the stream resolution, frees the old buffers.
   * @returns {void}
   */
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

  /**
   * Checks if the source matrix contains image data, by sampling a few bytes.
   * @returns {boolean} false if the frame looks empty/black, true otherwise.
   */
  #hasImage() {
    const data = this.#src.data;
    const samples = 12;
    const step = Math.floor(data.length / samples / 4) * 4;

    for (let i = 0; i < data.length; i += step) {
      if (data[i] !== 0) {
        return true;
      }
    }
    return false;
  }
}

/**
 * Draws the guide box and the outline of the detected object on a canvas that lies over
 * the video.
 */
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

  /**
   * @param {Array<{x: number, y: number}>} corners - In pixels of the video frame.
   * @returns {void}
   */
  draw(corners) {
    this.clear();

    const [first, ...rest] = corners;
    this.#ctx.beginPath();
    this.#ctx.moveTo(first.x, first.y);
    rest.forEach(({ x, y }) => this.#ctx.lineTo(x, y));
    this.#ctx.closePath();
    this.#ctx.strokeStyle = "#22c55e";
    this.#ctx.stroke();
  }

  /** Removes the outline, the guide box stays. */
  clear() {
    // the canvas has the pixel size of the frame, css scales it together with the video
    const { videoWidth: w, videoHeight: h } = this.#video;
    if (this.#canvas.width !== w || this.#canvas.height !== h) {
      this.#canvas.width = w;
      this.#canvas.height = h;
    }

    this.hide();

    const box = guideBox(w, h);
    this.#ctx.lineWidth = Math.max(2, w / 150);
    this.#ctx.strokeStyle = "#ffffff";
    this.#ctx.setLineDash([this.#ctx.lineWidth * 4, this.#ctx.lineWidth * 2]);
    this.#ctx.strokeRect(box.x, box.y, box.width, box.height);
    this.#ctx.setLineDash([]);
  }

  /** Removes everything. */
  hide() {
    this.#ctx.clearRect(0, 0, this.#canvas.width, this.#canvas.height);
  }
}

/** Runs the frame loop, finds objects and asks the api what they are. */
class Scanner {
  /** @type {HTMLVideoElement} */
  #video;
  /** @type {FrameBuffers} */
  #buffers;
  /** @type {Outline} */
  #outline;
  /** @type {boolean} false while a request runs or the result overlay is open. */
  #scanning = true;
  /** @type {number} No api call before this time (performance.now() timebase). */
  #resumeAt = 0;
  /** @type {number} Time of the last object detection, it runs less often than frames arrive. */
  #lastDetection = 0;
  /** @type {number|null} Pending frame callback of the processing loop. */
  #frameRequest = null;

  /**
   * @param {HTMLVideoElement} video
   * @param {FrameBuffers} buffers
   * @param {Outline} outline
   */
  constructor(video, buffers, outline) {
    this.#video = video;
    this.#buffers = buffers;
    this.#outline = outline;
  }

  /**
   * Starts the processing loop, a still running loop (e.g. of the previous camera) is stopped,
   * otherwise both loops would process every frame. Call after the video is playing.
   * @returns {void}
   */
  start() {
    this.stop();
    this.#processVideo();
  }

  /** Cancels the scheduled #processVideo call, if any. */
  stop() {
    this.#outline.hide();

    if (this.#frameRequest === null) {
      return;
    }

    if (HAS_VIDEO_FRAME_CALLBACK) {
      this.#video.cancelVideoFrameCallback(this.#frameRequest);
    } else {
      cancelAnimationFrame(this.#frameRequest);
    }
    this.#frameRequest = null;
  }

  /** Continues scanning after a short delay, so the same card isn't detected again. */
  resume() {
    this.#resumeAt = performance.now() + RESUME_DELAY_MILLIS;
    this.#scanning = true;
    this.#outline.clear();
    updateStatusline("");
  }

  /** Schedules #processVideo for the next video frame. */
  #scheduleNextFrame() {
    if (HAS_VIDEO_FRAME_CALLBACK) {
      this.#frameRequest = this.#video.requestVideoFrameCallback(
        this.#processVideo,
      );
    } else {
      this.#frameRequest = requestAnimationFrame(this.#processVideo);
    }
  }

  /** Processes the current video frame. */
  #processVideo = () => {
    if (this.#video.paused || this.#video.ended) {
      return;
    }

    if (this.#scanning) {
      try {
        this.#scanFrame();
      } catch (err) {
        console.error("Camera video error:", err);
      }
    }

    this.#scheduleNextFrame();
  };

  #scanFrame() {
    // no detection while paused (api cooldown, resume delay) and not more often than needed,
    // the video itself plays natively
    const now = performance.now();
    if (
      now < this.#resumeAt ||
      now - this.#lastDetection < DETECT_INTERVAL_MILLIS
    ) {
      return;
    }
    this.#lastDetection = now;

    if (!this.#buffers.read()) {
      return;
    }

    const detection = detectObjectFromImage(this.#buffers.frame);
    if (!detection) {
      this.#outline.clear();
      return;
    }

    this.#outline.draw(detection.corners);
    this.#identify(detection.image);
  }

  /**
   * Fetches object metadata based on the found image.
   * @param {cv.Mat} imgMat - The detected image.
   * @returns {Promise<void>}
   */
  async #identify(imgMat) {
    this.#scanning = false;
    // also throttles retries after a failed request or an empty result
    this.#resumeAt = performance.now() + API_COOLDOWN_MILLIS;
    // an error of the previous request is outdated now
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

      // the overlay decides what comes next
      this.#scanning = !detectResultEl.isOpen;
    }
  }
}

const scanner = new Scanner(
  videoEl,
  new FrameBuffers(videoEl),
  new Outline(overlayEl, videoEl),
);

/**
 * The OpenCV namespace, wrapped in an object: it has a then() method, so a promise resolved
 * with it directly never settles.
 * @returns {Promise<{cv: any}>}
 */
async function loadOpenCV() {
  const instance = globalThis.cv;
  if (!instance) {
    throw new Error("OpenCV.js is not loaded");
  }

  if (instance.calledRun && instance.Mat && instance.getBuildInformation) {
    return { cv: instance };
  }

  return new Promise((resolve) => {
    instance.onRuntimeInitialized = () => resolve({ cv: instance });
  });
}

// "load" (not DOMContentLoaded) also waits for the async opencv.js script
window.addEventListener("load", async () => {
  try {
    updateStatusline("Checking camera API permissions...");
    await camera.requestPermission();
  } catch (err) {
    updateStatusline("Error: Camera API not available.", err);
    return;
  }

  try {
    const result = await loadOpenCV();
    cv = result.cv;
  } catch (err) {
    updateStatusline("Error: OpenCV not available.", err);
    return;
  }

  updateStatusline(txtSelectCamera);
  fillCameraList();

  // a camera was plugged in or removed
  navigator.mediaDevices.addEventListener("devicechange", fillCameraList);
});

function stopStream() {
  camera.stop();
  scanner.stop();
  videoEl.pause();
}

// stop stream on page leave (pagehide also fires on mobile and with the back/forward cache)
window.addEventListener("pagehide", stopStream);

// stop camera on htmx swap, e.g. navigating to another page,
// but not for swaps inside the detect result like loading the card detail
document.addEventListener("htmx:beforeSwap", (e) => {
  if (detectResultEl.contains(e.detail.target)) {
    return;
  }

  stopStream();
});

/**
 * Fills the camera select, also called when a camera is plugged in or removed.
 * Keeps the current selection if that camera is still there.
 */
async function fillCameraList() {
  const selected = cameraEl.value;

  const devices = await navigator.mediaDevices.enumerateDevices();
  const vDevices = devices.filter((dev) => dev.kind === "videoinput");

  // an empty value keeps the "no camera selected"
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
    // the active camera was removed
    stopStream();
    updateStatusline(txtSelectCamera);
  }
}

/**
 * Tracks OpenCV objects (Mat, MatVector, ...) and frees them all in one place.
 * Use `try { ... } finally { scope.dispose(); }`.
 */
class MatScope {
  /** @type {Array<{delete: () => void}>} */
  #items = [];

  /**
   * @template {{delete: () => void}} T
   * @param {T} item
   * @returns {T} the same item, for inline use
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
 * @param {cv.Mat} src - The raw camera frame
 * @returns {{image: cv.Mat, corners: Array<{x: number, y: number}>}|null} - The warped object
 *   (caller owns `image`) and its corners in pixels of the frame, or null
 */
function detectObjectFromImage(src) {
  const scope = new MatScope();
  let warpedObject = null;
  let corners = null;

  try {
    // only the guide box with a margin is searched, so neighbouring objects
    // (e.g. the other cards of a binder page) can't join the object's outline
    const area = detectionArea(src.cols, src.rows);
    const searched = scope.track(src.roi(area));

    // the edge detection cost grows with the pixel count, so search on a smaller copy,
    // converted to gray first, resizing one channel moves a quarter of the data
    const scale = Math.min(
      1,
      DETECT_MAX_SIZE / Math.max(searched.cols, searched.rows),
    );
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

    const blurred = scope.track(new cv.Mat());
    const edged = scope.track(new cv.Mat());
    const contours = scope.track(new cv.MatVector());
    const hierarchy = scope.track(new cv.Mat());

    findObjectEdges(gray, blurred, edged);

    cv.findContours(
      edged,
      contours,
      hierarchy,
      cv.RETR_EXTERNAL,
      cv.CHAIN_APPROX_SIMPLE,
    );

    const candidates = findObjectCandidates(
      contours,
      gray.cols,
      gray.rows,
      scale,
      scope,
    );
    const bestApprox = selectBestCandidate(candidates);

    if (bestApprox) {
      // the corners are mapped back to the full frame, the warp keeps the full resolution
      const found = quadCorners(bestApprox, scale);
      const card = extendAftermathHalf(found, searched.cols, searched.rows);
      if (card) {
        const inFrame = card.map((p) => ({
          x: p.x + area.x,
          y: p.y + area.y,
        }));
        corners = expandCorners(inFrame, src.cols, src.rows);
      }
    }

    if (corners && fitsGuideBox(corners, src.cols, src.rows)) {
      warpedObject = transformPerspective(src, corners);

      // reject blank/featureless candidates (e.g. compression noise on an out-of-focus frame)
      if (!hasEnoughDetail(warpedObject)) {
        warpedObject.delete();
        warpedObject = null;
      }
    }
  } catch (err) {
    warpedObject?.delete();
    warpedObject = null;

    updateStatusline("Error: object detection failed.", err);
  } finally {
    scope.dispose();
  }

  if (!warpedObject) {
    return null;
  }

  return { image: warpedObject, corners };
}

/**
 * Writes an edge map of the gray image into the given (caller-owned) blurred/edged Mats.
 * @param {cv.Mat} gray - Gray image, possibly scaled down from the full frame.
 * @param {cv.Mat} blurred
 * @param {cv.Mat} edged
 * @returns {void}
 */
function findObjectEdges(gray, blurred, edged) {
  const scope = new MatScope();

  try {
    // a wide blur suppresses background texture noise while keeping the object's own edges
    cv.GaussianBlur(gray, blurred, new cv.Size(5, 5), 0);

    // adaptive ("auto Canny") thresholds handle both bright and dark/textured backgrounds
    const median = medianGray(blurred);
    const sigma = 0.33;
    const lower = Math.max(0, (1 - sigma) * median);
    const upper = Math.min(255, (1 + sigma) * median);
    cv.Canny(blurred, edged, lower, upper);

    // small kernel: a wider one merges background texture into one frame-sized blob
    const kernel = scope.track(
      cv.getStructuringElement(cv.MORPH_RECT, new cv.Size(3, 3)),
    );
    cv.morphologyEx(edged, edged, cv.MORPH_CLOSE, kernel);
  } finally {
    scope.dispose();
  }
}

/**
 * Contours with a plausible object-like aspect ratio.
 * @param {cv.MatVector} contours
 * @param {number} frameWidth
 * @param {number} frameHeight
 * @param {number} scale - Size of the frame relative to the full frame, 1 for the full frame.
 * @param {MatScope} scope - Owns every Mat created here, including each `approx`.
 * @returns {Array<{approx: cv.Mat, area: number}>}
 */
function findObjectCandidates(contours, frameWidth, frameHeight, scale, scope) {
  const minArea = MIN_OBJECT_AREA * scale * scale;
  // reject near-full-frame blobs (over-aggressive edge merging in findObjectEdges)
  const maxAllowedArea = frameWidth * frameHeight * 0.9;

  const candidates = [];
  for (let i = 0; i < contours.size(); ++i) {
    // the hull closes an outer border with gaps (dark border on dark background), otherwise
    // only a closed inner region like one half of a two-part card is found
    const cnt = scope.track(contours.get(i));
    const hull = scope.track(new cv.Mat());
    cv.convexHull(cnt, hull);
    const area = cv.contourArea(hull);

    // filter by minimum size
    if (area > minArea && area <= maxAllowedArea) {
      const epsilon = 0.02 * cv.arcLength(hull, true);
      const approx = scope.track(new cv.Mat());

      // simplify contour / count corners
      cv.approxPolyDP(hull, approx, epsilon, true);

      if (approx.rows !== 4) {
        continue;
      }

      const rect = cv.minAreaRect(approx);
      if (
        objectAspectRatioDiff(rect) <= ASPECT_RATIO_TOLERANCE &&
        edgeCoverage(cnt, approx, epsilon) >= MIN_EDGE_COVERAGE
      ) {
        candidates.push({ approx, area });
      }
    }
  }

  return candidates;
}

/**
 * Picks the largest candidate, it wins over a smaller inner region like the text box or a
 * piece of a neighbouring object in the margin of the searched area. Does not free any Mat,
 * the scope passed to findObjectCandidates owns them.
 * @param {Array<{approx: cv.Mat, area: number}>} candidates
 * @returns {cv.Mat|null}
 */
function selectBestCandidate(candidates) {
  let best = null;
  for (const c of candidates) {
    if (!best || c.area > best.area) {
      best = c;
    }
  }

  if (!best) {
    return null;
  }

  return best.approx;
}

/**
 * Share of the contour's length that runs along the quad's edges. A border with gaps runs
 * along its hull, clutter joined to the object (e.g. a neighbouring card) runs away from it.
 * @param {cv.Mat} contour - CV_32SC2.
 * @param {cv.Mat} quad - 4-point approximation of the contour's hull (CV_32SC2).
 * @param {number} maxDist - Distance in pixels up to which a point lies on an edge.
 * @returns {number} 0 to 1.
 */
function edgeCoverage(contour, quad, maxDist) {
  const d = contour.data32S;
  const pts = [];
  for (let i = 0; i < d.length; i += 2) {
    pts.push(new cv.Point(d[i], d[i + 1]));
  }
  const onEdge = pts.map(
    (p) => Math.abs(cv.pointPolygonTest(quad, p, true)) <= maxDist,
  );

  let total = 0;
  let along = 0;
  for (let i = 0; i < pts.length; i++) {
    // the segment from the last point back to the first closes the contour
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
 * Of an aftermath card often only the bottom half has a closed contour. The half is a landscape
 * quad, so with the card held upright a landscape quad with the half's ratio is extended to the
 * whole card. Other landscape quads, like the title and art of a normal card whose outline
 * wasn't found, are rejected.
 * @param {Array<{x: number, y: number}>} corners - top-left, top-right, bottom-right, bottom-left
 * @param {number} frameWidth
 * @param {number} frameHeight
 * @returns {Array<{x: number, y: number}>|null} The corners of the whole card (may reach out of
 *   the frame by its border), the given ones for a portrait quad, null for a landscape quad
 *   that is no aftermath half or whose top half is out of the frame.
 */
function extendAftermathHalf(corners, frameWidth, frameHeight) {
  const [tl, tr, br, bl] = corners;
  const width = dist(tl, tr) + dist(bl, br);
  const height = dist(tl, bl) + dist(tr, br);
  if (width <= height) {
    return corners;
  }

  if (
    Math.abs(height / width - AFTERMATH_HALF_RATIO) > AFTERMATH_HALF_TOLERANCE
  ) {
    return null;
  }

  const lerp = (a, b, t) => ({
    x: a.x + (b.x - a.x) * t,
    y: a.y + (b.y - a.y) * t,
  });
  // point of the half extrapolated beyond 0..1, u along the top edge, v along the left edge
  const at = (u, v) => lerp(lerp(tl, tr, u), lerp(bl, br, u), v);
  const { above, below, side } = AFTERMATH_LAYOUT;

  const inFrame = (p) =>
    p.x >= 0 && p.y >= 0 && p.x < frameWidth && p.y < frameHeight;
  if (!inFrame(at(0, -above)) || !inFrame(at(1, -above))) {
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
  return Math.hypot(a.x - b.x, a.y - b.y);
}

/**
 * How far a detected quad's aspect ratio is from an object's (~0.71, portrait).
 * @param {{size: {width: number, height: number}}} rect - minAreaRect of the quad.
 * @returns {number} Difference from the target ratio, or Infinity if degenerate.
 */
function objectAspectRatioDiff({ size: { width, height } }) {
  if (width === 0 || height === 0) {
    return Infinity;
  }

  const ratio = Math.min(width, height) / Math.max(width, height);

  return Math.abs(ratio - OBJECT_RATIO);
}

/**
 * Whether a candidate has enough pixel variation to plausibly be a real object.
 * @param {cv.Mat} mat - Warped candidate object image.
 * @returns {boolean}
 */
function hasEnoughDetail(mat) {
  const scope = new MatScope();

  try {
    const gray = scope.track(new cv.Mat());
    const mean = scope.track(new cv.Mat());
    const stddev = scope.track(new cv.Mat());

    cv.cvtColor(mat, gray, cv.COLOR_RGBA2GRAY);
    cv.meanStdDev(gray, mean, stddev);

    const minStdDev = 20; // blank frames measure near 0
    return stddev.data64F[0] >= minStdDev;
  } finally {
    scope.dispose();
  }
}

/**
 * Corners of a quad in pixels of the full frame, ordered clockwise from the top left.
 * @param {cv.Mat} quad - 4-point contour approximation (CV_32SC2).
 * @param {number} scale - Size of the frame the quad was found in, relative to the full frame.
 * @returns {Array<{x: number, y: number}>} top-left, top-right, bottom-right, bottom-left
 */
function quadCorners(quad, scale) {
  const d = quad.data32S;
  const pts = [0, 2, 4, 6].map((i) => ({
    x: d[i] / scale,
    y: d[i + 1] / scale,
  }));

  // sort corners by sum and difference of the coordinates
  const sums = pts.map((p) => p.x + p.y);
  const diffs = pts.map((p) => p.y - p.x);

  const tl = pts[sums.indexOf(Math.min(...sums))]; // top-left (smallest sum)
  const tr = pts[diffs.indexOf(Math.min(...diffs))]; // top-right (smallest difference)
  const br = pts[sums.indexOf(Math.max(...sums))]; // bottom-right (largest sum)
  const bl = pts[diffs.indexOf(Math.max(...diffs))]; // bottom-left (largest difference)

  return [tl, tr, br, bl];
}

/**
 * Grows the corners around their center by BORDER_EXPANSION to reach the object's outer
 * border, and clamps them to the frame. The order is kept.
 * @param {Array<{x: number, y: number}>} corners
 * @param {number} frameWidth
 * @param {number} frameHeight
 * @returns {Array<{x: number, y: number}>}
 */
function expandCorners(corners, frameWidth, frameHeight) {
  const grow = 1 + BORDER_EXPANSION;
  const c = centerOf(corners);
  const clamp = (v, size) => Math.min(Math.max(v, 0), size - 1);

  return corners.map((p) => ({
    x: clamp(c.x + (p.x - c.x) * grow, frameWidth),
    y: clamp(c.y + (p.y - c.y) * grow, frameHeight),
  }));
}

/**
 * Mean point of the corners.
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
 * The centered box with the object's aspect ratio the user holds the object into.
 * @param {number} frameWidth
 * @param {number} frameHeight
 * @returns {{x: number, y: number, width: number, height: number}}
 */
function guideBox(frameWidth, frameHeight) {
  const height = Math.min(frameHeight, frameWidth / OBJECT_RATIO) * GUIDE_SIZE;
  const width = height * OBJECT_RATIO;

  return {
    x: (frameWidth - width) / 2,
    y: (frameHeight - height) / 2,
    width,
    height,
  };
}

/**
 * The part of the frame that is searched for the object: the guide box grown by GUIDE_MARGIN
 * of its size on every side, cut to the frame.
 * @param {number} frameWidth
 * @param {number} frameHeight
 * @returns {cv.Rect}
 */
function detectionArea(frameWidth, frameHeight) {
  const box = guideBox(frameWidth, frameHeight);
  const marginX = box.width * GUIDE_MARGIN;
  const marginY = box.height * GUIDE_MARGIN;

  const left = Math.max(0, Math.floor(box.x - marginX));
  const top = Math.max(0, Math.floor(box.y - marginY));
  const right = Math.min(frameWidth, Math.ceil(box.x + box.width + marginX));
  const bottom = Math.min(frameHeight, Math.ceil(box.y + box.height + marginY));

  return new cv.Rect(left, top, right - left, bottom - top);
}

/**
 * Whether the object lies in the guide box: its center is inside and it covers at least
 * MIN_GUIDE_FILL of the box. Rejects e.g. the text box of a card held too close.
 * @param {Array<{x: number, y: number}>} corners - In order around the quad.
 * @param {number} frameWidth
 * @param {number} frameHeight
 * @returns {boolean}
 */
function fitsGuideBox(corners, frameWidth, frameHeight) {
  const box = guideBox(frameWidth, frameHeight);

  // shoelace formula
  let area = 0;
  corners.forEach((p, i) => {
    const next = corners[(i + 1) % corners.length];
    area += p.x * next.y - next.x * p.y;
  });
  area = Math.abs(area) / 2;

  return (
    isInside(centerOf(corners), box) &&
    area >= box.width * box.height * MIN_GUIDE_FILL
  );
}

/**
 * Whether the point lies in the box.
 * @param {{x: number, y: number}} p
 * @param {{x: number, y: number, width: number, height: number}} box
 * @returns {boolean}
 */
function isInside(p, box) {
  return (
    p.x >= box.x &&
    p.x <= box.x + box.width &&
    p.y >= box.y &&
    p.y <= box.y + box.height
  );
}

/**
 * Warps the object into a straightened, fixed-size image.
 * @param {cv.Mat} sourceMat
 * @param {Array<{x: number, y: number}>} corners - top-left, top-right, bottom-right, bottom-left
 * @returns {cv.Mat} The warped image, the caller owns it.
 */
function transformPerspective(sourceMat, corners) {
  const w = OBJECT_WARP_WIDTH;
  const h = OBJECT_WARP_HEIGHT;

  const scope = new MatScope();
  try {
    const srcCoords = scope.track(
      cv.matFromArray(4, 1, cv.CV_32FC2, corners.flatMap((p) => [p.x, p.y])),
    );
    const dstCoords = scope.track(
      cv.matFromArray(4, 1, cv.CV_32FC2, [0, 0, w, 0, w, h, 0, h]),
    );

    const M = scope.track(cv.getPerspectiveTransform(srcCoords, dstCoords));
    const result = new cv.Mat();
    cv.warpPerspective(sourceMat, result, M, new cv.Size(w, h));

    return result;
  } finally {
    scope.dispose();
  }
}

/**
 * Sends the given image to the detect api.
 * @param {Blob} image - JPEG encoded image, sent as raw body.
 * @returns {Promise<{data: Array<{id: string}>}>} The matches, best match first.
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
