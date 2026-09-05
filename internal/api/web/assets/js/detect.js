const statusEl = document.getElementById("status");
const videoEl = document.getElementById("videoInput");
const canvasOutput = document.getElementById("canvasOutput");
const canvasDetect = document.getElementById("canvasDetect");

const cameraEl = document.getElementById("camera");
cameraEl.addEventListener("change", function (e) {
  console.log("changed", e.target.value);

  const deviceId = e.target.value;
  startStream(deviceId);
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
  API_COOLDOWN: 500, //ms

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
   * Fetches card metadata based on the found image.
   * @param {cv.Mat} imgMat - The detected image.
   * @returns {Promise<void>}
   */
  async onImageDetect(imgMat) {
    const now = Date.now();
    if (now - this.lastApiCall <= this.API_COOLDOWN) {
      imgMat.delete();
      return;
    }

    this.detected = true;
    this.lastApiCall = now;

    const dataUrl = canvasDetect.toDataURL("image/jpeg", 0.8); // reduced quality

    try {
      cv.imshow("canvasDetect", imgMat);

      statusEl.innerText = "Checking match...";
      const rawBase64 = dataUrl.split(",")[1];
      const matches = await fetchMatchesByImage(rawBase64);
      console.log("matches", matches);

      if (matches.data.length === 0) {
        this.detected = false;
        return;
      }

      for (const match of matches.data) {
        // overwrite with image from server
        const ctx = canvasDetect.getContext("2d");
        const img = new Image();
        img.onload = function () {
          ctx.drawImage(img, 0, 0);
        };
        img.src = match.image;

        showDetectOverlay();

        // take first image
        statusEl.innerText =
          "Detected: " +
          match.name +
          " | " +
          match.set.name +
          " |" +
          match.set.code;
        statusEl.innerText += "Draw detected image " + img.src;
        return;
      }

      this.detected = false;
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
    console.log("requested permissions");
  } catch (err) {
    statusEl.innerText = "Error: Camera API not available. " + err;
    return;
  } finally {
    // unblock camera usage
    videoState.stopStream();
  }

  console.log("going to prepare camera list");
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

window.onbeforeunload = () => {
  videoState.stopStream();
};

async function fillCameraList() {
  statusEl.innerText = "Please select a camera ...";
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
  cameraEl.appendChild(optFn("select camera", null));
  vDevices.forEach((dev, index) => {
    const label = dev.label || `Camera ${index}`;

    cameraEl.appendChild(optFn(label, dev.deviceId));
  });
}

async function startStream(deviceId) {
  statusEl.innerText = "Starting stream...";

  try {
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
    statusEl.innerText = "Error: Camera API not available. " + err;
  }
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

    let detectedCard = detectCardFromImage(videoState.src);
    if (detectedCard) {
      videoState.onImageDetect(detectedCard);
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
 * @param {cv.Mat} src - Das unverarbeitete Kamerabild
 * @returns {cv.Mat|null} - Die freigestellte Karte oder null
 */
function detectCardFromImage(src) {
  // Vorbereitung der Matrizen (Entspricht den Variablen-Deklarationen in Go)
  let gray = new cv.Mat();
  let blurred = new cv.Mat();
  let edged = new cv.Mat();
  let contours = new cv.MatVector();
  let hierarchy = new cv.Mat();

  let warpedCard = null;

  try {
    // 1. Preprocessing: Graustufen & Weichzeichner (gocv.CvtColor / gocv.GaussianBlur)
    cv.cvtColor(src, gray, cv.COLOR_RGBA2GRAY);
    cv.GaussianBlur(gray, blurred, new cv.Size(5, 5), 0);

    // 2. Kanten erkennen (gocv.Canny)
    cv.Canny(blurred, edged, 75, 200);

    // Morphologischer Filter, um kleine Lücken in den Linien zu schließen
    let kernel = cv.getStructuringElement(cv.MORPH_RECT, new cv.Size(9, 9));
    cv.morphologyEx(edged, edged, cv.MORPH_CLOSE, kernel);
    kernel.delete();

    // 3. Konturen finden (gocv.FindContours)
    cv.findContours(
      edged,
      contours,
      hierarchy,
      cv.RETR_EXTERNAL,
      cv.CHAIN_APPROX_SIMPLE,
    );

    let maxArea = 0;
    let bestApprox = null;

    // 4. Konturen filtern (Die Schleife über alle gefundenen Formen)
    for (let i = 0; i < contours.size(); ++i) {
      let cnt = contours.get(i);
      let area = cv.contourArea(cnt);

      // Mindestgröße filtern (Go nutzt hier oft Schwellenwerte aus der Config)
      if (area > 10000) {
        let peri = cv.arcLength(cnt, true);
        let approx = new cv.Mat();

        // Kontur vereinfachen / Ecken zählen (gocv.ApproxPolyDP)
        cv.approxPolyDP(cnt, approx, 0.02 * peri, true);

        // Wenn die Form exakt 4 Ecken hat und die größte bisherige ist
        if (approx.rows === 4 && area > maxArea) {
          maxArea = area;
          if (bestApprox) bestApprox.delete(); // Alten Favoriten löschen
          bestApprox = approx;
        } else {
          approx.delete(); // Nicht gebrauchte Matrix sofort freigeben
        }
      }
    }

    // 5. Transformation durchführen, falls eine Karte gefunden wurde
    if (bestApprox) {
      warpedCard = transformPerspective(src, bestApprox);
      bestApprox.delete();
    }
  } catch (err) {
    statusEl.innerText = "Error: card detection failed. " + err;
  } finally {
    gray.delete();
    blurred.delete();
    edged.delete();
    contours.delete();
    hierarchy.delete();
  }

  return warpedCard;
}
/**
 * Sortiert die Ecken und schneidet die Karte reibungslos aus
 */
function transformPerspective(sourceMat, contour) {
  const targetWidth = 350;
  const targetHeight = 490;

  // 1. Punkte aus der OpenCV-Struktur in ein JS-Array extrahieren
  let pts = [];
  for (let i = 0; i < 4; i++) {
    pts.push({
      x: contour.data32S[i * 2],
      y: contour.data32S[i * 2 + 1],
    });
  }

  // 2. Mathematische Ecken-Sortierung (Summe & Differenz)
  let sums = pts.map((p) => p.x + p.y);
  let diffs = pts.map((p) => p.y - p.x);

  let tl = pts[sums.indexOf(Math.min(...sums))]; // Top-Left (kleinste Summe)
  let tr = pts[diffs.indexOf(Math.min(...diffs))]; // Top-Right (kleinste Differenz)
  let br = pts[sums.indexOf(Math.max(...sums))]; // Bottom-Right (größte Summe)
  let bl = pts[diffs.indexOf(Math.max(...diffs))]; // Bottom-Left (größte Differenz)

  // 3. Koordinaten-Matrizen für OpenCV erstellen
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

  // 4. Matrix berechnen und Bild verformen (gocv.GetPerspectiveTransform / gocv.WarpPerspective)
  let M = cv.getPerspectiveTransform(srcCoords, dstCoords);
  let result = new cv.Mat();
  cv.warpPerspective(
    sourceMat,
    result,
    M,
    new cv.Size(targetWidth, targetHeight),
  );

  // 5. Temporäre Berechnungsmatrizen löschen
  srcCoords.delete();
  dstCoords.delete();
  M.delete();

  return result;
}

async function fetchMatchesByImage(base64Image) {
  try {
    const response = await fetch("/detect", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ image: base64Image }),
    });

    if (!response.ok) {
      throw new Error("API Error");
    }

    return await response.json();
  } catch (e) {
    throw e;
  }
}
