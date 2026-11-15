# Vendored tiny-voca TTS

This directory contains the browser-side TTS stack used by
`/stream/overlay/`. Everything here is served from this origin; the worker
does not fetch the model, phonemizer, ONNX Runtime, or Pyodide from a CDN.

## Versions

| Component | Version | Source | License |
|---|---|---|---|
| tiny-voca model | HF revision `5131e959f6da837cb855b02694341efbdba6fdf0` | https://huggingface.co/n4ze3m/tiny-voca | Apache-2.0 |
| tiny-voca web code | commit `28e0722c6ad040d74a1f6e63d77621272477a71f` | https://github.com/n4ze3m/tiny-voca | Apache-2.0 |
| dizione phonemizer | commit `f8629fe989f3d0db5f440f9b3c6f0154f2e6110f` | https://github.com/n4ze3m/dizione | Apache-2.0 |
| ONNX Runtime Web | 1.20.1 | https://www.npmjs.com/package/onnxruntime-web | MIT |
| Pyodide | 0.26.4 | https://pyodide.org/ | MPL-2.0 |
| numpy (Pyodide wheel) | 1.26.4 | https://numpy.org/ | BSD-3-Clause |

The model is `assets/model.onnx` (21,386,372 bytes, SHA-256
`fe6d918791a15b82e37465981d7f27a070e46b46635e5a3482fc8dff87a1ae46`).
The numpy wheel is
`numpy-1.26.4-cp312-cp312-pyodide_2024_0_wasm32.whl` (SHA-256
`4a2f5303a88a0747c5e6c80701e0ff4ff5667e487c0c9f9c73875e4d88f4cf9a`).
The F2 voice is index `0`; M2 is index `1`.

## Layout

```
worker.js                  phonemizer + ONNX inference worker
../tts.js                  main-thread queue/playback wrapper
assets/model.onnx          tiny-voca ONNX model
assets/config.json         model config (sample rate, max tokens, voices)
assets/dizione.zip         dizione Python source for Pyodide
assets/phonemizer/*        dizione dictionary/model data
ort/                       ONNX Runtime Web JS + WASM
pyodide/                   Pyodide runtime, lock file, numpy wheel
licenses/                  upstream licenses
```

## Updating

1. Replace the files from the upstream releases. The model and phonemizer
   data come from the Hugging Face repository; `dizione.zip` is the
   `dizione/src/dizione` tree zipped without `__pycache__`, `*.pyc`, or
   `built/`.
2. Bump `VERSION` in `worker.js`. This changes the Cache API key used for
   `assets/`, so OBS/browsers fetch the new files.
3. In OBS, open the browser source properties and use
   **Refresh cache of current page** once.
4. Keep `ort/ort.wasm.min.js`, `ort/ort-wasm-simd-threaded.mjs`, and
   `ort/ort-wasm-simd-threaded.wasm` in sync with the ONNX Runtime version.
5. Keep `pyodide/pyodide.js`, `pyodide.asm.js`, `pyodide.asm.wasm`,
   `python_stdlib.zip`, `pyodide-lock.json`, and the numpy wheel in sync with
   the Pyodide version. `loadPackage("numpy")` resolves the wheel from the
   local `indexURL`.

The worker deliberately sets `ort.env.wasm.numThreads = 1`. OBS does not
provide a cross-origin-isolated context, so multi-threaded ORT would require
COOP/COEP headers and would risk breaking the rest of the overlay.

The assets are about 62 MB. If repository size becomes a problem, move them
to Git LFS or download them during deployment instead of committing them.
