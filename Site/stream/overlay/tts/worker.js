// Runs the phonemizer (dizione, through Pyodide) and the model (ONNX Runtime
// Web) off the main thread. Messages in: {type:"load"} and
// {type:"speak", id, text, voice, speed}. Messages out: status, ready,
// sentence, done, error.
//
// Everything is served from this directory so the overlay does not depend on
// Hugging Face, jsDelivr, or any other CDN at runtime.

const BASE = new URL("./", self.location.href).href;
const PYODIDE = BASE + "pyodide/";
const ORT = BASE + "ort/";
const ASSETS = BASE + "assets/";
const DATA = ["lexicon.tsv", "align.json", "g2p.json", "g2p.npz", "homographs.json"];

importScripts(PYODIDE + "pyodide.js", ORT + "ort.wasm.min.js");

let pyodide, phonemize, session, config;

const status = (text, progress) => postMessage({ type: "status", text, progress });

// Bump when the files under assets/ change, so browsers fetch them again.
const VERSION = "1";

// Downloaded files are kept in the browser's cache storage: a second visit
// starts without the network.
async function download(url, label) {
  const key = `${url}?v=${VERSION}`;
  let store = null;
  try {
    store = await caches.open("tiny-voca-" + VERSION);
    const hit = await store.match(key);
    if (hit) return new Uint8Array(await hit.arrayBuffer());
  } catch (error) {
    store = null;
  }
  const bytes = await fetchBytes(key, label);
  if (store) store.put(key, new Response(bytes)).catch(() => {});
  return bytes;
}

async function fetchBytes(url, label) {
  const response = await fetch(url);
  if (!response.ok) throw new Error(`${url}: ${response.status}`);
  const total = Number(response.headers.get("content-length")) || 0;
  if (!response.body || !total) return new Uint8Array(await response.arrayBuffer());
  const reader = response.body.getReader();
  const chunks = [];
  let received = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
    received += value.length;
    status(label, received / total);
  }
  const bytes = new Uint8Array(received);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.length;
  }
  return bytes;
}

async function loadPhonemizer() {
  status("Loading the phonemizer");
  pyodide = await loadPyodide({ indexURL: PYODIDE });
  await pyodide.loadPackage("numpy");
  const [source, ...files] = await Promise.all([
    download(ASSETS + "dizione.zip", "Loading the phonemizer"),
    ...DATA.map((name) => download(ASSETS + "phonemizer/" + name, "Loading the dictionary")),
  ]);
  status("Starting the phonemizer");
  pyodide.FS.mkdirTree("/dizione-data");
  DATA.forEach((name, index) => pyodide.FS.writeFile("/dizione-data/" + name, files[index]));
  pyodide.unpackArchive(source.buffer, "zip", { extractDir: "/dizione-src" });
  pyodide.runPython(`
import json, os, sys
os.environ["DIZIONE_DATA"] = "/dizione-data"
sys.path.insert(0, "/dizione-src")
import dizione

def phonemize_json(text):
    result = dizione.phonemize(text)
    return json.dumps({"ids": list(result.ids), "stress": list(result.stress), "text": str(result)})

phonemize_json("Warm up.")
`);
  phonemize = pyodide.globals.get("phonemize_json");
}

async function loadModel() {
  config = await (await fetch(ASSETS + "config.json")).json();
  const bytes = await download(ASSETS + "model.onnx", "Loading the model");
  status("Starting the model");
  ort.env.wasm.wasmPaths = ORT;
  // One thread avoids SharedArrayBuffer / COOP+COEP requirements. OBS does not
  // provide a cross-origin-isolated context, and a dedicated stream rig has
  // plenty of CPU for single-threaded inference.
  ort.env.wasm.numThreads = 1;
  session = await ort.InferenceSession.create(bytes, { executionProviders: ["wasm"] });
}

const ABBREVIATIONS = /(?:\b(?:mr|mrs|ms|dr|prof|sr|jr|st|vs|etc|no|inc|ltd|co|e\.g|i\.e|a\.m|p\.m)|\b[A-Z])\.$/i;

// Sentences, without breaking after titles, initials or inside numbers.
function sentences(text) {
  const parts = [];
  let current = "";
  for (const piece of text.replace(/\s+/g, " ").trim().split(/(?<=[.!?…]["')\]]?)\s+(?=["'(\[]?[A-Z0-9])/)) {
    current = current ? current + " " + piece : piece;
    if (!ABBREVIATIONS.test(current)) {
      parts.push(current);
      current = "";
    }
  }
  if (current) parts.push(current);
  return parts.filter((part) => /[\p{L}\p{N}]/u.test(part));
}

// One utterance holds at most max_tokens phonemes; longer sentences are cut at
// commas, then by words.
function fit(sentence) {
  const result = JSON.parse(phonemize(sentence));
  if (result.ids.length <= config.model.max_tokens - 12) return [result];
  const pieces = sentence.split(/(?<=[,;:—])\s+/);
  if (pieces.length > 1) return pieces.flatMap(fit);
  const words = sentence.split(" ");
  if (words.length < 2) return [];
  const half = Math.ceil(words.length / 2);
  return [...fit(words.slice(0, half).join(" ")), ...fit(words.slice(half).join(" "))];
}

async function synthesize(phones, voice, speed) {
  const count = phones.ids.length;
  const integers = (values) => new ort.Tensor("int64", BigInt64Array.from(values.map(BigInt)), [1, count]);
  const scalar = (value) => new ort.Tensor("float32", Float32Array.from([value]), [1]);
  const output = await session.run({
    tokens: integers(phones.ids),
    stress: integers(phones.stress),
    voice: new ort.Tensor("int64", BigInt64Array.from([BigInt(voice)]), [1]),
    speed: scalar(speed),
    pitch_shift: scalar(0),
    pitch_range: scalar(1),
  });
  return output.audio.data;
}

async function speak({ id, text, voice, speed }) {
  const started = performance.now();
  const list = sentences(String(text || ""));
  const chunks = [];
  let phonemeTime = 0;
  let modelTime = 0;
  for (let index = 0; index < list.length; index++) {
    let mark = performance.now();
    const utterances = fit(list[index]);
    phonemeTime += performance.now() - mark;
    for (const phones of utterances) {
      if (phones.ids.length === 0) continue;
      mark = performance.now();
      const audio = await synthesize(phones, voice, speed);
      modelTime += performance.now() - mark;
      chunks.push(audio);
      postMessage({ type: "sentence", id, index, total: list.length, phonemes: phones.text });
    }
  }
  const length = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
  const audio = new Float32Array(length);
  let offset = 0;
  for (const chunk of chunks) {
    audio.set(chunk, offset);
    offset += chunk.length;
  }
  postMessage({
    type: "done",
    id,
    audio,
    sampleRate: config.sample_rate,
    phonemeTime,
    modelTime,
    totalTime: performance.now() - started,
  }, [audio.buffer]);
}

onmessage = async ({ data }) => {
  try {
    if (data.type === "load") {
      await Promise.all([loadPhonemizer(), loadModel()]);
      postMessage({ type: "ready", voices: config.voices });
    } else if (data.type === "speak") {
      await speak(data);
    }
  } catch (error) {
    postMessage({ type: "error", id: data.id, message: String((error && error.message) || error) });
  }
};
