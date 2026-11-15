// Main-thread wrapper around the tiny-voca Web Worker. The worker does the
// phonemization and model inference; this module queues redemptions, turns the
// returned Float32Array into a WAV blob, and plays it through an Audio element.

const MAX_CHARS = 300
const MAX_QUEUE = 8

const wav = (samples, rate) => {
    const buffer = new ArrayBuffer(44 + samples.length * 2)
    const view = new DataView(buffer)
    const text = (offset, value) => {
        for (let i = 0; i < value.length; i++) view.setUint8(offset + i, value.charCodeAt(i))
    }

    text(0, "RIFF")
    view.setUint32(4, 36 + samples.length * 2, true)
    text(8, "WAVEfmt ")
    view.setUint32(16, 16, true)
    view.setUint16(20, 1, true)
    view.setUint16(22, 1, true)
    view.setUint32(24, rate, true)
    view.setUint32(28, rate * 2, true)
    view.setUint16(32, 2, true)
    view.setUint16(34, 16, true)
    text(36, "data")
    view.setUint32(40, samples.length * 2, true)

    for (let i = 0; i < samples.length; i++) {
        view.setInt16(44 + i * 2, Math.max(-1, Math.min(1, samples[i])) * 32767, true)
    }

    return new Blob([buffer], { type: "audio/wav" })
}

const cleanText = (text) => String(text ?? "")
    .replace(/[\u0000-\u001f\u007f]/g, " ")
    .replace(/\s+/g, " ")
    .trim()
    .slice(0, MAX_CHARS)

export const createTts = ({
    voice = 0, // 0 = F2, 1 = M2
    speed = 1.0,
    volume = 0.8,
    maxQueue = MAX_QUEUE,
    workerUrl = "/stream/overlay/tts/worker.js",
    onStatus = null,
    onError = null,
} = {}) => {
    const queue = []
    let ready = false
    let busy = false
    let failed = false
    let nextId = 0
    let currentId = 0
    let currentAudio = null

    const reportError = (message, error) => {
        console.error("[tts]", message, error ?? "")
        if (onError) onError(message, error)
    }

    let worker
    try {
        worker = new Worker(workerUrl)
    } catch (error) {
        failed = true
        reportError("failed to create worker", error)
        return { enqueue: () => false, isReady: () => false }
    }

    const pump = () => {
        if (failed || !ready || busy || queue.length === 0) return
        const job = queue.shift()
        currentId = job.id
        busy = true
        worker.postMessage({ type: "speak", id: job.id, text: job.text, voice, speed })
    }

    const finishPlayback = (audio, url) => {
        if (currentAudio !== audio) return
        currentAudio = null
        URL.revokeObjectURL(url)
        busy = false
        pump()
    }

    worker.onmessage = ({ data }) => {
        if (data.type === "status") {
            if (onStatus) onStatus(data.text, data.progress)
            return
        }

        if (data.type === "ready") {
            ready = true
            console.log("[tts] ready", data.voices ?? [])
            pump()
            return
        }

        if (data.type === "error") {
            reportError(data.message ?? "worker error")
            if (!ready) failed = true
            busy = false
            pump()
            return
        }

        if (data.type !== "done" || data.id !== currentId) return

        if (!data.audio || data.audio.length === 0) {
            busy = false
            pump()
            return
        }

        const url = URL.createObjectURL(wav(data.audio, data.sampleRate))
        const audio = new Audio(url)
        let finished = false
        const finish = () => {
            if (finished) return
            finished = true
            finishPlayback(audio, url)
        }

        currentAudio = audio
        audio.volume = volume
        audio.onended = finish
        audio.onerror = () => {
            reportError("audio playback failed")
            finish()
        }
        audio.play().catch((error) => {
            reportError("audio playback was blocked", error)
            finish()
        })
    }

    worker.onerror = (event) => {
        failed = true
        busy = false
        reportError("worker failed to start", event.message)
    }

    try {
        worker.postMessage({ type: "load" })
    } catch (error) {
        failed = true
        reportError("failed to start worker", error)
    }

    return {
        enqueue: (text) => {
            if (failed) return false
            const value = cleanText(text)
            if (!value) return false
            if (queue.length >= maxQueue) {
                const dropped = queue.shift()
                console.warn("[tts] queue full, dropping oldest message", dropped?.id)
            }
            queue.push({ id: ++nextId, text: value })
            pump()
            return true
        },
        isReady: () => ready,
    }
}

export default createTts
